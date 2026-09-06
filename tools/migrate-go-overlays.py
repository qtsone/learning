#!/usr/bin/env python3
"""One-off migration of the Go slice of every shared lesson into its overlay.

For each shared lesson (content/shared/<stage>/<lesson>/):

- exercises/go/  -> content/go/shared/<stage>/<lesson>/exercise/
- solutions/go/  -> content/go/shared/<stage>/<lesson>/solution/
- every "In Go" block in LESSON.md -> .../snippets/<slot>.md, with a
  <!-- lang: <slot> --> anchor left where the block was.

The engine renders the anchor back into the snippet's exact bytes, so a Go
learner's workspace is unchanged. The script proves that: it fingerprints
every Go-path lesson the way the pre-overlay engine scaffolded it, runs the
migration, fingerprints again through the new engine, and fails loudly on any
difference. Run from the repo root. A lesson already in the overlay layout is
left alone and fingerprinted through the engine on both sides, so re-running
on a migrated tree is a no-op.
"""

import hashlib
import importlib.util
import re
import shutil
import sys
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]
SKILL = REPO / "skills" / "tutor"
CONTENT = SKILL / "curriculum" / "content"
ENGINE = SKILL / "scripts" / "tutor.py"
LANGUAGE = "go"

MARKER_RE = re.compile(r"^(?P<indent>\s*)(?P<quote>>\s*)?(\*\*)?In Go\b")
FENCE_RE = re.compile(r"^\s*(>\s*)?```")
HEADING_RE = re.compile(r"^#{1,6}\s+(.*)$")
LIST_RE = re.compile(r"^\s*([-*+]|\d+[.)])\s+")
CODE_SPAN_RE = re.compile(r"`([^`]+)`")
GO_HINT_RE = re.compile(r"\bGo\b|`go\b|\bS[1357]\b|gofmt|goroutine")


def load_engine():
    spec = importlib.util.spec_from_file_location("tutor", ENGINE)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def sha(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def legacy_fingerprint(lesson_dir: Path, language: str) -> dict:
    """What the pre-overlay engine scaffolded: rel path -> sha256."""
    junk = {".DS_Store", "__pycache__", ".ruff_cache", ".pytest_cache", ".mypy_cache"}
    out = {}
    for src in sorted(lesson_dir.rglob("*")):
        if not src.is_file():
            continue
        rel = src.relative_to(lesson_dir)
        parts = rel.parts
        if parts[0] in {"solution", "solutions"} or rel.name in {
            "TUTOR.md",
            "quiz.json",
        }:
            continue
        if junk.intersection(parts):
            continue
        if parts[0] == "exercises":
            if len(parts) < 3 or parts[1] != language:
                continue
            rel = Path("exercise").joinpath(*parts[2:])
        out[str(rel)] = sha(src.read_bytes())
    return out


def slugify(text: str) -> str:
    text = re.sub(r"`|\*|_", "", text).lower()
    text = re.sub(r"[^a-z0-9]+", "-", text).strip("-")
    if len(text) > 40:
        text = text[:40].rsplit("-", 1)[0]
    return text or "intro"


def paragraph_end(lines: list, start: int) -> int:
    """Index one past the last line of the paragraph or quote starting at start."""
    quoted = lines[start].lstrip().startswith(">")
    i = start
    while i < len(lines) and lines[i].strip():
        if quoted and not lines[i].lstrip().startswith(">"):
            break
        i += 1
    return i


def fence_end(lines: list, start: int) -> int:
    i = start + 1
    while i < len(lines) and not FENCE_RE.match(lines[i]):
        i += 1
    return min(i + 1, len(lines))


def next_nonblank(lines: list, i: int) -> int:
    while i < len(lines) and not lines[i].strip():
        i += 1
    return i


def paragraph_text(lines: list, start: int) -> str:
    return " ".join(lines[start : paragraph_end(lines, start)])


def introduces_code(lines: list, j: int) -> bool:
    """A paragraph ending in ':' whose next chunk is a fence or a list."""
    if not paragraph_text(lines, j).rstrip().endswith(":"):
        return False
    k = next_nonblank(lines, paragraph_end(lines, j))
    return k < len(lines) and bool(FENCE_RE.match(lines[k]) or LIST_RE.match(lines[k]))


def comments_on(lines: list, j: int, fence) -> bool:
    """Prose right after a fence that names something from it (`heap.Pop`, `name`)."""
    if fence is None:
        return False
    code = "\n".join(lines[fence[0] : fence[1]])
    return any(span in code for span in CODE_SPAN_RE.findall(paragraph_text(lines, j)))


def lead_in_extent(lines: list, start: int, end: int) -> int:
    """Grow a block whose marker paragraph ends in ':' and so owns what follows.

    The first chunk is the marker's by construction; after it, fences,
    indented list continuations, Go-hinted prose, inner lead-ins and
    commentary on the code just shown stay in the block. A heading, a quote
    or a neutral paragraph ends it.
    """
    first = True
    fence = None
    j = next_nonblank(lines, end)
    while j < len(lines):
        line = lines[j]
        if HEADING_RE.match(line) or line.lstrip().startswith(">"):
            break
        if FENCE_RE.match(line):
            fence = (j, fence_end(lines, j))
            end = fence[1]
        else:
            keep = (
                first
                or line[0].isspace()
                or GO_HINT_RE.search(paragraph_text(lines, j))
                or introduces_code(lines, j)
                or comments_on(lines, j, fence)
            )
            if not keep:
                break
            end = paragraph_end(lines, j)
            fence = None
        first = False
        j = next_nonblank(lines, end)
    return end


def block_extent(lines: list, start: int) -> int:
    """One past the last line of the Go block whose marker line is `start`."""
    quoted = lines[start].lstrip().startswith(">")
    end = paragraph_end(lines, start)
    if quoted:
        # A quoted block may wrap a fence: keep consuming quoted runs.
        j = next_nonblank(lines, end)
        while j < len(lines) and lines[j].lstrip().startswith(">"):
            end = paragraph_end(lines, j)
            j = next_nonblank(lines, end)
        return end
    if paragraph_text(lines, start).rstrip().endswith(":"):
        return lead_in_extent(lines, start, end)
    j = next_nonblank(lines, end)
    if j < len(lines) and FENCE_RE.match(lines[j]):
        end = fence_end(lines, j)
        j = next_nonblank(lines, end)
        # Trailing prose that explains the code stays with it when it is
        # plainly about Go; neutral prose stops the block.
        if (
            j < len(lines)
            and not HEADING_RE.match(lines[j])
            and not FENCE_RE.match(lines[j])
            and not lines[j].lstrip().startswith((">", "-", "*", "|", "1."))
            and GO_HINT_RE.search(" ".join(lines[j : paragraph_end(lines, j)]))
        ):
            end = paragraph_end(lines, j)
    return end


def extract_blocks(text: str):
    """Yield (start, end, heading) for every Go block, in file order."""
    lines = text.split("\n")
    heading = "intro"
    i = 0
    in_fence = False
    while i < len(lines):
        line = lines[i]
        if FENCE_RE.match(line) and not line.lstrip().startswith(">"):
            in_fence = not in_fence
            i += 1
            continue
        if in_fence:
            i += 1
            continue
        m = HEADING_RE.match(line)
        if m:
            heading = m.group(1)
        if MARKER_RE.match(line):
            end = block_extent(lines, i)
            yield i, end, heading
            i = end
            continue
        i += 1


def migrate_lesson(lesson_dir: Path, overlay: Path, report: list) -> bool:
    moved = False
    for old, new in (("exercises", "exercise"), ("solutions", "solution")):
        src = lesson_dir / old / LANGUAGE
        if src.is_dir():
            dest = overlay / new
            if dest.exists():
                sys.exit(f"refusing to overwrite {dest}")
            dest.parent.mkdir(parents=True, exist_ok=True)
            shutil.move(str(src), str(dest))
            moved = True
        parent = lesson_dir / old
        if parent.is_dir():
            leftovers = [p.name for p in parent.iterdir()]
            if leftovers:
                sys.exit(f"{parent} still holds {leftovers}; migrate them first")
            parent.rmdir()

    lesson_md = lesson_dir / "LESSON.md"
    text = lesson_md.read_text(encoding="utf-8")
    blocks = list(extract_blocks(text))
    if not blocks:
        return moved
    lines = text.split("\n")
    used = {}
    replacements = []
    for start, end, heading in blocks:
        base = slugify(heading)
        used[base] = used.get(base, 0) + 1
        slot = base if used[base] == 1 else f"{base}-{used[base]}"
        replacements.append((start, end, slot))
    snippets = overlay / "snippets"
    snippets.mkdir(parents=True, exist_ok=True)
    out = []
    cursor = 0
    for start, end, slot in replacements:
        out.extend(lines[cursor:start])
        body = "\n".join(lines[start:end]) + "\n"
        (snippets / f"{slot}.md").write_text(body, encoding="utf-8")
        out.append(f"<!-- lang: {slot} -->")
        report.append(
            f"{lesson_dir.relative_to(CONTENT)}: {slot} <- lines {start + 1}-{end}"
        )
        cursor = end
    out.extend(lines[cursor:])
    lesson_md.write_text("\n".join(out), encoding="utf-8")
    return True


def needs_migration(lesson_dir: Path) -> bool:
    if any((lesson_dir / old).is_dir() for old in ("exercises", "solutions")):
        return True
    text = (lesson_dir / "LESSON.md").read_text(encoding="utf-8")
    return any(True for _ in extract_blocks(text))


def main():
    tutor = load_engine()
    reg = tutor.Registry.load()
    pending = [
        lid
        for lid in reg.ordered_lessons()
        if reg.pool(lid) == "shared"
        and (reg.content_dir(lid) / "LESSON.md").exists()
        and needs_migration(reg.content_dir(lid))
    ]
    if not pending:
        print("nothing to migrate (tree already in overlay layout)")
        return

    def engine_fingerprint(lid: str) -> dict:
        return {
            rel: tutor.source_hash(src)
            for rel, src in reg.scaffold_map(lid, LANGUAGE).items()
        }

    go_lessons = [lid for lid, _ in tutor.iter_language_lessons(reg, LANGUAGE)]
    before = {}
    for lid in go_lessons:
        already_overlaid = reg.pool(lid) == "shared" and lid not in pending
        before[lid] = (
            engine_fingerprint(lid)
            if already_overlaid
            else legacy_fingerprint(reg.content_dir(lid), LANGUAGE)
        )

    report = []
    for lid in pending:
        migrate_lesson(reg.content_dir(lid), reg.overlay_dir(lid, LANGUAGE), report)

    mismatches = []
    for lid in go_lessons:
        after = engine_fingerprint(lid)
        if after != before[lid]:
            gone = sorted(set(before[lid]) - set(after))
            new = sorted(set(after) - set(before[lid]))
            changed = sorted(
                r for r in before[lid] if r in after and after[r] != before[lid][r]
            )
            mismatches.append(f"{lid}: missing {gone} new {new} changed {changed}")

    for line in report:
        print(line)
    print(
        f"migrated {len(pending)} shared lesson(s), {len(report)} snippet(s) extracted"
    )
    if mismatches:
        print("BYTE IDENTITY BROKEN for the Go path:")
        for m in mismatches:
            print("  " + m)
        sys.exit(1)
    print(f"byte identity verified for {len(go_lessons)} Go-path lessons")


if __name__ == "__main__":
    main()
