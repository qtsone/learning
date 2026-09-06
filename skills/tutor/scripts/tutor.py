#!/usr/bin/env python3
"""tutor.py — deterministic curriculum engine for the tutor skill.

Owns all learner-workspace state. The tutoring LLM never edits state files by
hand; it calls these subcommands:

    init <language> [--focus a,b] [--carry-over DIR]
                                    create/refresh a workspace in CWD (idempotent)
    sync                            re-scaffold + JSON diff report
    status [--json]                 session-start briefing
    mark <lesson-id> <status> [--grade A..F] [--note ...]
    verify <lesson-id>              run the lesson's automated checks
    guidance <guided|standard|spartan>
    custom add <slug> --title T     register a tutor-generated custom lesson
    graph [--language X] [--focus ..] [--format tree|mermaid|json]
    validate [--strict]             repo-level registry/content validation
    ci [--filter substr] [--language X]
                                    validate + run every solution against its tests

Python 3 stdlib only. Curriculum is resolved relative to this file; the
workspace is the current directory (or --workspace).
"""

import argparse
import copy
import fnmatch
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from datetime import datetime, timezone
from pathlib import Path
from typing import NoReturn

SKILL_DIR = Path(__file__).resolve().parents[1]
CONTENT_DIR = SKILL_DIR / "curriculum" / "content"
REGISTRY_PATH = SKILL_DIR / "curriculum" / "registry.json"

STATE_SCHEMA = 1
MANIFEST_SCHEMA = 1
STATUSES = ("todo", "in_progress", "passed", "skipped", "needs_review")
GUIDANCE_MODES = ("guided", "standard", "spartan")
GRADE_RE = re.compile(r"^[A-F][+-]?$")
TUTOR_ONLY_FILES = {"TUTOR.md", "quiz.json"}
TUTOR_ONLY_DIRS = {"solution", "snippets"}
# The pre-overlay layout. Scaffold ignores it and validate rejects it so a
# stale exercises/go/ can never shadow a language overlay.
RETIRED_DIRS = {"exercises", "solutions"}
POOL_DIRS = ("shared", "focus")
OVERLAY_ENTRIES = {"snippets", "exercise", "solution", "TUTOR.md", "quiz.json"}
# Editor and tool droppings that land in a curriculum checkout but are never
# content. Without this list a .DS_Store next to LESSON.md would be scaffolded
# and hashed into the learner's manifest.
JUNK_NAMES = {
    ".DS_Store",
    ".git",
    "__pycache__",
    ".ruff_cache",
    ".pytest_cache",
    ".mypy_cache",
    ".venv",
    "venv",
    ".hypothesis",
    ".coverage",
    ".tox",
    ".nox",
}
VERIFY_COMMANDS = {
    "gotest": ["go", "test", "-race", "./..."],
    "pytest": ["uv", "run", "-q", "--locked", "pytest", "-q"],
    "script": ["bash", "./check.sh"],
}
TOOLCHAIN_HINTS = {
    "go": "install the Go toolchain (1.22 or newer): https://go.dev/dl/",
    "uv": (
        "install uv: https://docs.astral.sh/uv/getting-started/installation/ "
        "then run: uv python install 3.14"
    ),
    "bash": "install bash; the script runner needs it on PATH",
}
TEST_FILE_GLOBS = {"gotest": "*_test.go", "pytest": "test_*.py"}
# Shared and pack lessons declare this and the workspace language's runner
# (languages.<code>.verify.type) decides what actually runs.
RESOLVED_VERIFY = "tests"
VERIFY_TYPES = (*VERIFY_COMMANDS, RESOLVED_VERIFY, "discussion")
PYTEST_PROJECT_FILES = ("pyproject.toml", "uv.lock", ".python-version")
ANCHOR_RE = re.compile(rb"^<!-- lang: ([a-z0-9][a-z0-9-]*) -->\r?\n?$")
# Anything that looks like an anchor but is not one byte-exact would ship to
# learners as a stray HTML comment, so validate rejects it.
LOOSE_ANCHOR_RE = re.compile(rb"^\s*<!--\s*lang\b.*$")
FENCE_RE = re.compile(rb"^\s*```")
SLOT_RE = re.compile(r"^[a-z0-9][a-z0-9-]*$")
EXERCISE_TOOLCHAIN_RE = re.compile(
    r"\b(go (?:test|build|run|vet|mod)|gofmt|pytest|uv run)\b"
)
# The capstone harnesses tell learners how to reach projects/capstone at the
# workspace root from a scaffolded exercise directory. That depth belongs to
# lesson_workspace_dir(), so validate re-derives it and rejects stale text.
CAPSTONE_REL_RE = re.compile(r"(?:\.\./)+projects/capstone")
CAPSTONE_TEXT_SUFFIXES = (".md", ".go", ".py", ".sh", ".json", ".toml")
WORKSPACE_IGNORE = (
    ".venv/",
    "venv/",
    "__pycache__/",
    ".pytest_cache/",
    ".mypy_cache/",
    ".ruff_cache/",
    ".hypothesis/",
    ".coverage",
    ".tox/",
    ".nox/",
    "*.egg-info/",
)


def now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def sha256_file(path: Path) -> str:
    return sha256_bytes(path.read_bytes())


def die(msg: str, code: int = 1) -> NoReturn:
    print(f"error: {msg}", file=sys.stderr)
    sys.exit(code)


def warn(msg: str):
    print(f"warning: {msg}", file=sys.stderr)


def canonical(data) -> str:
    return json.dumps(data, sort_keys=True)


def write_json_atomic(path: Path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile(
        "w", dir=path.parent, delete=False, suffix=".tmp", encoding="utf-8"
    ) as tmp:
        try:
            json.dump(data, tmp, indent=2, ensure_ascii=False)
            tmp.write("\n")
        except BaseException:
            Path(tmp.name).unlink(missing_ok=True)
            raise
    Path(tmp.name).replace(path)


def ensure_toolchain(cmd: list):
    tool = cmd[0]
    if shutil.which(tool) is None:
        die(f"'{tool}' not found on PATH — {TOOLCHAIN_HINTS.get(tool, 'install it')}")


def content_files(base: Path) -> dict:
    """relative path -> file under base, junk skipped; empty when base is absent."""
    out = {}
    if not base.is_dir():
        return out
    for src in sorted(base.rglob("*")):
        rel = src.relative_to(base)
        if src.is_file() and not JUNK_NAMES.intersection(rel.parts):
            out[str(rel)] = src
    return out


# A scaffold source is either a curriculum file (copied as-is) or bytes the
# engine rendered (LESSON.md with the language's snippets spliced in).


def source_bytes(src) -> bytes:
    return src if isinstance(src, bytes) else src.read_bytes()


def source_hash(src) -> str:
    return sha256_bytes(source_bytes(src))


def place(src, dest: Path):
    dest.parent.mkdir(parents=True, exist_ok=True)
    if isinstance(src, bytes):
        dest.write_bytes(src)
    else:
        shutil.copy2(src, dest)


# ---------------------------------------------------------------- registry


class Registry:
    def __init__(self, data: dict):
        self.data = data
        self.languages: dict = data["languages"]
        self.stages: dict = data["stages"]
        self.packs: dict = data["packs"]
        self.lessons: dict = data["lessons"]
        self.version: str = data.get("version", "0")
        self.owner = {}
        for key, stage in self.stages.items():
            for lid in stage["lessons"]:
                self.owner[lid] = ("stage", key)
        for key, pack in self.packs.items():
            for ins in pack.get("insertions", []):
                for lid in ins["lessons"]:
                    self.owner[lid] = ("pack", key)

    @classmethod
    def load(cls) -> "Registry":
        if not REGISTRY_PATH.exists():
            die(f"registry not found: {REGISTRY_PATH}")
        try:
            return cls(json.loads(REGISTRY_PATH.read_text(encoding="utf-8")))
        except json.JSONDecodeError as e:
            die(f"registry is not valid JSON: {e}")

    def slug(self, lid: str) -> str:
        return lid.rsplit(".", 1)[-1]

    def owner_dir(self, lid: str) -> str:
        kind, key = self.owner.get(lid, (None, None))
        if kind == "stage":
            return self.stages[key]["dir"]
        if kind == "pack":
            return self.packs[key]["dir"]
        die(f"lesson {lid} not referenced by any stage or pack")

    def content_dir(self, lid: str) -> Path:
        return CONTENT_DIR / self.owner_dir(lid) / self.slug(lid)

    def pool(self, lid: str) -> str:
        kind, key = self.owner[lid]
        return self.stages[key]["pool"] if kind == "stage" else "focus"

    def is_language_pool(self, lid: str) -> bool:
        return self.pool(lid) in self.languages

    def ordered_lessons(self) -> list:
        out = []
        for stage in self.stages.values():
            out.extend(stage["lessons"])
        for pack in self.packs.values():
            for ins in pack.get("insertions", []):
                out.extend(ins["lessons"])
        return out

    def languages_for(self, lid: str) -> list:
        """Language codes whose path or pack list carries this lesson."""
        kind, key = self.owner[lid]
        if kind == "stage":
            return [c for c, lang in self.languages.items() if key in lang["path"]]
        return [c for c in self.packs[key].get("languages", []) if c in self.languages]

    def runner(self, language: str) -> str:
        vtype = self.languages[language].get("verify", {}).get("type")
        if vtype not in TEST_FILE_GLOBS:
            die(
                f"language '{language}' declares no test runner "
                f"(languages.{language}.verify.type must be one of "
                f"{', '.join(TEST_FILE_GLOBS)})"
            )
        return vtype

    def resolve_verify(self, lid: str, language: str) -> str:
        vtype = self.lessons[lid]["verify"]["type"]
        return self.runner(language) if vtype == RESOLVED_VERIFY else vtype

    def overlay_dir(self, lid: str, language: str):
        """content/<language>/<owner dir>/<slug>; None for language-pool lessons."""
        if self.is_language_pool(lid):
            return None
        return CONTENT_DIR / language / self.owner_dir(lid) / self.slug(lid)

    def exercise_files(self, lid: str, language: str) -> dict:
        """The exercise a learner of `language` gets: agnostic files, overlay on top."""
        files = content_files(self.content_dir(lid) / "exercise")
        overlay = self.overlay_dir(lid, language)
        if overlay is not None:
            files.update(content_files(overlay / "exercise"))
        return files

    def solution_dir(self, lid: str, language: str):
        overlay = self.overlay_dir(lid, language)
        candidates = [overlay / "solution"] if overlay is not None else []
        candidates.append(self.content_dir(lid) / "solution")
        return next(
            (
                d
                for d in candidates
                if d.is_dir() and any(p.name not in JUNK_NAMES for p in d.iterdir())
            ),
            None,
        )

    def has_lesson_text(self, lid: str) -> bool:
        return (self.content_dir(lid) / "LESSON.md").exists()

    def is_authored(self, lid: str, language: str) -> bool:
        """LESSON.md exists and the language has whatever its verify type needs."""
        if not self.has_lesson_text(lid):
            return False
        vtype = self.resolve_verify(lid, language)
        if vtype == "discussion":
            return True
        files = self.exercise_files(lid, language)
        if vtype == "script":
            return "check.sh" in files
        glob = TEST_FILE_GLOBS[vtype]
        return any(fnmatch.fnmatch(Path(rel).name, glob) for rel in files)

    def lesson_anchors(self, lid: str) -> list:
        return [slot for slot, _ in self.anchor_lines(lid)]

    def anchor_lines(self, lid: str) -> list:
        """(slot, problem) per anchor-looking line; problem is None when it is well formed."""
        text = (self.content_dir(lid) / "LESSON.md").read_bytes()
        out = []
        in_fence = False
        for line in text.splitlines(keepends=True):
            if FENCE_RE.match(line):
                in_fence = not in_fence
                continue
            m = ANCHOR_RE.match(line)
            if m:
                slot = m.group(1).decode()
                out.append(
                    (slot, "sits inside a fenced code block" if in_fence else None)
                )
            elif LOOSE_ANCHOR_RE.match(line):
                shown = line.decode("utf-8", errors="replace").rstrip("\r\n")
                out.append((None, f"malformed anchor line {shown!r}"))
        return out

    def render_lesson(self, lid: str, language: str) -> bytes:
        """LESSON.md with each anchor replaced by the language's snippet, or dropped.

        A dropped anchor takes the blank line after it along (when one precedes it
        too), so paragraph spacing — and every other language's rendered bytes —
        survive an anchor that only one language fills.
        """
        text = (self.content_dir(lid) / "LESSON.md").read_bytes()
        overlay = self.overlay_dir(lid, language)
        snippets = overlay / "snippets" if overlay is not None else None
        lines = text.splitlines(keepends=True)
        out = []
        prev_blank = False
        i = 0
        while i < len(lines):
            line = lines[i]
            i += 1
            m = ANCHOR_RE.match(line)
            if not m:
                out.append(line)
                prev_blank = line.strip() == b""
                continue
            snippet = snippets / f"{m.group(1).decode()}.md" if snippets else None
            if snippet is not None and snippet.is_file():
                blob = snippet.read_bytes()
                out.append(blob)
                prev_blank = blob.endswith(b"\n\n")
            elif prev_blank and i < len(lines) and lines[i].strip() == b"":
                i += 1
        return b"".join(out)

    def compose(self, language: str, focuses: list) -> list:
        """Ordered groups of {key, slug, title, lessons} for (language, focuses)."""
        lang = self.languages.get(language)
        if lang is None:
            die(f"unknown language '{language}'. Known: {', '.join(self.languages)}")
        for f in focuses:
            pack = self.packs.get(f)
            if pack is None:
                die(f"unknown focus pack '{f}'. Known: {', '.join(self.packs)}")
            if language not in pack.get("languages", []):
                die(f"focus pack '{f}' does not support language '{language}'")
        groups = []
        for stage_key in lang["path"]:
            stage = self.stages[stage_key]
            groups.append(
                {
                    "key": stage_key,
                    "slug": stage["slug"],
                    "title": stage["title"],
                    "lessons": list(stage["lessons"]),
                }
            )
            for f in focuses:
                for ins in self.packs[f].get("insertions", []):
                    if ins["after"] == stage_key:
                        groups.append(
                            {
                                "key": f"{f}:{ins['slug']}",
                                "slug": ins["slug"],
                                "title": ins["title"],
                                "lessons": list(ins["lessons"]),
                            }
                        )
        return groups

    def composition_hash(self, language: str, focuses: list) -> str:
        """Fingerprint of everything a workspace derives from the registry.

        Registry edits outside this slice (another language's lessons) must not
        tell a learner to sync.
        """
        groups = self.compose(language, focuses)
        payload = {
            "version": self.version,
            "language": self.languages[language],
            "groups": groups,
            "lessons": {lid: self.lessons[lid] for g in groups for lid in g["lessons"]},
        }
        return sha256_bytes(canonical(payload).encode("utf-8"))[:12]

    def scaffold_map(self, lid: str, language: str) -> dict:
        """relative dest path -> source (Path or rendered bytes) for every learner-facing file."""
        src_root = self.content_dir(lid)
        if not self.has_lesson_text(lid):
            return {}
        out = {}
        for src in sorted(src_root.rglob("*")):
            if not src.is_file():
                continue
            rel = src.relative_to(src_root)
            parts = rel.parts
            if parts[0] in TUTOR_ONLY_DIRS | RETIRED_DIRS | {"exercise"}:
                continue
            if rel.name in TUTOR_ONLY_FILES or JUNK_NAMES.intersection(parts):
                continue
            out[str(rel)] = src
        for rel, src in self.exercise_files(lid, language).items():
            out[f"exercise/{rel}"] = src
        out["LESSON.md"] = self.render_lesson(lid, language)
        return out


# ---------------------------------------------------------------- workspace


class Workspace:
    def __init__(self, root: Path):
        self.root = root
        self.tutor_dir = root / ".tutor"
        self.state_path = self.tutor_dir / "state.json"
        self.manifest_path = self.tutor_dir / "manifest.json"
        self.journal_path = self.tutor_dir / "journal.md"
        self.state: dict = {}
        self.manifest: dict = {}
        self.loaded_state = None
        self.loaded_manifest = None

    def exists(self) -> bool:
        return self.state_path.exists()

    def load(self):
        try:
            self.state = json.loads(self.state_path.read_text(encoding="utf-8"))
            self.manifest = json.loads(self.manifest_path.read_text(encoding="utf-8"))
        except FileNotFoundError:
            die("no tutor workspace here — run: tutor.py init <language>")
        except json.JSONDecodeError as e:
            die(f"corrupt state under .tutor/ ({e}); restore from git or re-init")
        self.loaded_state = canonical(self.state)
        self.loaded_manifest = canonical(self.manifest)
        return self

    def save(self):
        """Write only what changed: a no-op sync must leave the workspace byte-identical."""
        if canonical(self.state) != self.loaded_state:
            self.state["updated"] = now()
            write_json_atomic(self.state_path, self.state)
        if canonical(self.manifest) != self.loaded_manifest:
            write_json_atomic(self.manifest_path, self.manifest)

    def lesson_state(self, lid: str) -> dict:
        entry = self.state["lessons"].setdefault(lid, {"status": "todo"})
        entry.setdefault("attempts", 0)
        return entry


def find_workspace(args) -> Workspace:
    return Workspace(Path(args.workspace).resolve())


def workspace_warnings(ws: Workspace, reg: Registry) -> list:
    out = []
    language = ws.state["language"]
    path = reg.languages[language]["path"]
    for f in ws.state["focuses"]:
        inserted = sum(
            len(ins["lessons"])
            for ins in reg.packs[f].get("insertions", [])
            if ins["after"] in path
        )
        if inserted == 0:
            out.append(f"focus pack '{f}' adds no lessons to the {language} path")
    if reg.runner(language) == "pytest":
        for parent in (ws.root, *ws.root.parents):
            if (parent / "pyproject.toml").exists():
                out.append(
                    f"{parent}/pyproject.toml is above this workspace; uv discovers "
                    "projects upward, so move the workspace outside that project"
                )
                break
    return out


# ---------------------------------------------------------------- scaffold/sync


def numbered(idx: int, slug: str) -> str:
    return f"{idx:02d}-{slug}"


def lesson_workspace_dir(group_idx, group_slug, lesson_idx, lesson_slug) -> str:
    return (
        f"lessons/{numbered(group_idx, group_slug)}/{numbered(lesson_idx, lesson_slug)}"
    )


def capstone_rel_from_exercise() -> str:
    """Relative path from a scaffolded exercise dir to projects/capstone."""
    depth = lesson_workspace_dir(1, "g", 1, "l").count("/") + 2  # + exercise/
    return "../" * depth + "projects/capstone"


def sync_workspace(ws: Workspace, reg: Registry, apply: bool = True) -> dict:
    """Diff the workspace against the curriculum and, if apply, bring it up to date.

    With apply=False nothing on disk changes and the report is what a real
    sync would do. The in-memory state and manifest are still walked, so
    dry runs go through preview_sync, which hands in a throwaway copy.
    """
    report = {
        "added": [],
        "updated": [],
        "conflicts": [],
        "renamed": [],
        "removed": [],
        "removed_files": [],
        "needs_review": [],
        "pending_content": [],
    }
    language = ws.state["language"]
    groups = reg.compose(language, ws.state["focuses"])
    man_lessons = ws.manifest["lessons"]
    seen = set()

    for gi, group in enumerate(groups, start=1):
        for li, lid in enumerate(group["lessons"], start=1):
            ws.lesson_state(lid)
            if not reg.is_authored(lid, language):
                report["pending_content"].append(lid)
                continue
            seen.add(lid)
            expected_dir = lesson_workspace_dir(gi, group["slug"], li, reg.slug(lid))
            entry = man_lessons.get(lid)
            new_lesson = entry is None
            changed = False

            if new_lesson:
                entry = {"dir": expected_dir, "files": {}}
                man_lessons[lid] = entry
                report["added"].append(lid)
            elif entry["dir"] != expected_dir:
                old, new = ws.root / entry["dir"], ws.root / expected_dir
                if apply and old.exists():
                    new.parent.mkdir(parents=True, exist_ok=True)
                    old.rename(new)
                report["renamed"].append(
                    {"lesson": lid, "from": entry["dir"], "to": expected_dir}
                )
                entry["dir"] = expected_dir

            dest_root = ws.root / entry["dir"]
            smap = reg.scaffold_map(lid, language)

            for rel, src in smap.items():
                src_hash = source_hash(src)
                man_hash = entry["files"].get(rel)
                dest = dest_root / rel
                ws_hash = sha256_file(dest) if dest.exists() else None
                if src_hash == man_hash:
                    continue
                if ws_hash is None or ws_hash == man_hash:
                    if apply:
                        place(src, dest)
                    if not new_lesson:
                        report["updated"].append(f"{lid}:{rel}")
                elif ws_hash == src_hash:
                    pass
                else:
                    sidecar = dest.with_name(dest.name + ".upstream")
                    if apply:
                        place(src, sidecar)
                    report["conflicts"].append(
                        {
                            "lesson": lid,
                            "file": rel,
                            "sidecar": str(sidecar.relative_to(ws.root)),
                        }
                    )
                entry["files"][rel] = src_hash
                changed = changed or not new_lesson

            for rel in [r for r in entry["files"] if r not in smap]:
                dest = dest_root / rel
                if apply and dest.exists() and sha256_file(dest) == entry["files"][rel]:
                    dest.unlink()
                report["removed_files"].append(f"{lid}:{rel}")
                del entry["files"][rel]
                changed = True

            lstate = ws.lesson_state(lid)
            if changed and lstate["status"] in ("passed", "skipped"):
                lstate.setdefault("prior_status", lstate["status"])
                lstate["status"] = "needs_review"
                report["needs_review"].append(lid)

    for lid in [x for x in man_lessons if x not in seen]:
        entry = man_lessons.pop(lid)
        src = ws.root / entry["dir"]
        if apply and src.exists():
            name = Path(entry["dir"]).name
            attic = ws.tutor_dir / "attic" / name
            attic.parent.mkdir(parents=True, exist_ok=True)
            generation = 1
            while attic.exists():
                attic = attic.with_name(f"{name}.{generation}")
                generation += 1
            src.rename(attic)
        report["removed"].append(lid)

    ws.manifest["curriculum_version"] = reg.version
    ws.manifest.pop("registry_hash", None)
    ws.manifest["composition_hash"] = reg.composition_hash(
        language, ws.state["focuses"]
    )
    if apply:
        prune_empty_dirs(ws.root / "lessons")
        write_roadmap(ws, reg)
        ws.save()
    return report


SYNC_ACTIONS = (
    "added",
    "updated",
    "conflicts",
    "renamed",
    "removed",
    "removed_files",
    "needs_review",
)


def preview_sync(ws: Workspace, reg: Registry) -> dict:
    """The report a sync would produce right now, touching neither disk nor ws."""
    scratch = Workspace(ws.root)
    scratch.state = copy.deepcopy(ws.state)
    scratch.manifest = copy.deepcopy(ws.manifest)
    return sync_workspace(scratch, reg, apply=False)


def prune_empty_dirs(root: Path):
    if not root.exists():
        return
    for d in sorted((p for p in root.rglob("*") if p.is_dir()), reverse=True):
        if not any(d.iterdir()):
            d.rmdir()


STATUS_ICONS = {
    "passed": "x",
    "skipped": "x",
    "todo": " ",
    "in_progress": " ",
    "needs_review": " ",
}


def write_roadmap(ws: Workspace, reg: Registry):
    st = ws.state
    language = st["language"]
    groups = reg.compose(language, st["focuses"])
    counts = {s: 0 for s in STATUSES}
    total = 0
    for g in groups:
        for lid in g["lessons"]:
            total += 1
            counts[ws.lesson_state(lid)["status"]] += 1
    lang_name = reg.languages[language]["name"]
    focuses = ", ".join(st["focuses"]) or "none"
    lines = [
        f"# Roadmap — {lang_name}",
        "",
        (
            f"> Generated by tutor.py — do not edit. Focuses: {focuses} · "
            f"Guidance: {st['guidance']} · Curriculum {reg.version}"
        ),
        ">",
        (
            f"> Progress: **{counts['passed']}/{total} passed** · "
            f"{counts['in_progress']} in progress · "
            f"{counts['needs_review']} need review · {counts['skipped']} skipped"
        ),
        "",
    ]
    man = ws.manifest["lessons"]
    for gi, g in enumerate(groups, start=1):
        lines.append(f"## {gi:02d} · {g['title']}")
        lines.append("")
        for li, lid in enumerate(g["lessons"], start=1):
            meta = reg.lessons[lid]
            lstate = ws.lesson_state(lid)
            status = lstate["status"]
            mark = STATUS_ICONS[status]
            suffix = ""
            if status == "passed":
                grade = lstate.get("grade")
                suffix = f" — passed{f' ({grade})' if grade else ''}"
            elif status == "skipped":
                suffix = " — skipped ⚠"
            elif status == "in_progress":
                suffix = " — **in progress**"
            elif status == "needs_review":
                suffix = " — **needs review** ⚠ (content changed since you passed)"
            if not reg.is_authored(lid, language):
                suffix += " — ⏳ content pending"
                link = ""
            else:
                entry = man.get(lid)
                link = f" · [`{entry['dir']}`]({entry['dir']}/)" if entry else ""
            lines.append(
                f"- [{mark}] {li:02d} **{meta['title']}**"
                f" ({meta['duration']}){suffix}{link}"
            )
        lines.append("")
    if st.get("custom_lessons"):
        lines.append("## Custom lessons")
        lines.append("")
        for c in st["custom_lessons"]:
            lstate = ws.lesson_state(c["id"])
            mark = STATUS_ICONS[lstate["status"]]
            lines.append(f"- [{mark}] **{c['title']}** · [`{c['dir']}`]({c['dir']}/)")
        lines.append("")
    (ws.root / "ROADMAP.md").write_text("\n".join(lines), encoding="utf-8")


JOURNAL_SEED = """# Tutor journal

LLM-owned session notes. Structured curriculum observations use:
`- [<lesson-id>] <issue|gap|errata|difficulty> — <observation> — suggested: <fix>`

## Observations

## Session notes
"""

GITIGNORE_HEADER = (
    "# Seeded by tutor.py: tool caches and build outputs that never belong in "
    "your learning repo.\n"
)


def seed_gitignore(ws: Workspace, reg: Registry, language: str) -> bool:
    path = ws.root / ".gitignore"
    if path.exists():
        return False
    patterns = [*WORKSPACE_IGNORE, *reg.languages[language].get("workspace_ignore", [])]
    path.write_text(GITIGNORE_HEADER + "\n".join(patterns) + "\n", encoding="utf-8")
    return True


def load_carry_over_source(src_dir: str, language: str) -> Workspace:
    src = Workspace(Path(src_dir).resolve())
    if not src.exists():
        die(f"--carry-over: no tutor workspace at {src_dir}")
    src.load()
    if src.state["language"] == language:
        die("--carry-over expects a workspace of another language")
    return src


def carry_over(ws: Workspace, reg: Registry, src: Workspace) -> list:
    """Note prior passes of shared lessons from another language's workspace.

    Status stays todo: the exercise is per language, so the gate is re-run;
    the tutor fast-tracks the reading from the note.
    """
    carried = []
    ordered = [
        lid
        for g in reg.compose(ws.state["language"], ws.state["focuses"])
        for lid in g["lessons"]
    ]
    for lid in ordered:
        other = src.state["lessons"].get(lid)
        if not other or other.get("status") != "passed":
            continue
        lstate = ws.lesson_state(lid)
        if lstate["status"] != "todo" or "carried over" in lstate.get("notes", ""):
            continue
        grade = other.get("grade")
        when = (other.get("updated_at") or "")[:10]
        lstate["notes"] = (
            f"carried over from {src.state['language']} workspace: passed"
            f"{f' ({grade})' if grade else ''}{f' on {when}' if when else ''}"
        )
        carried.append(lid)
    return carried


# ---------------------------------------------------------------- commands


def cmd_init(args):
    reg = Registry.load()
    ws = find_workspace(args)
    if (ws.root / "skills" / "tutor").exists():
        die(
            "this looks like the curriculum repo itself — "
            "run init in a separate, empty learning folder"
        )
    lang = reg.languages.get(args.language)
    if lang is None:
        die(f"unknown language '{args.language}'. Known: {', '.join(reg.languages)}")
    if lang.get("status") != "available":
        die(
            f"language '{args.language}' has no authored content yet "
            f"(status: {lang.get('status')})"
        )
    focuses = [f.strip() for f in (args.focus or "").split(",") if f.strip()]
    carry_src = (
        load_carry_over_source(args.carry_over, args.language)
        if args.carry_over
        else None
    )

    if ws.exists():
        ws.load()
        if ws.state["language"] != args.language:
            die(
                f"workspace already initialized for '{ws.state['language']}' — "
                "one workspace per language; use a different folder"
            )
        added = [f for f in focuses if f not in ws.state["focuses"]]
        ws.state["focuses"].extend(added)
        report = sync_workspace(ws, reg)
        report["init"] = "existing workspace synced" + (
            f"; focuses added: {', '.join(added)}" if added else ""
        )
    else:
        reg.compose(args.language, focuses)
        ws.tutor_dir.mkdir(parents=True, exist_ok=True)
        (ws.root / "projects").mkdir(exist_ok=True)
        if not ws.journal_path.exists():
            ws.journal_path.write_text(JOURNAL_SEED, encoding="utf-8")
        ws.state = {
            "schema": STATE_SCHEMA,
            "language": args.language,
            "focuses": focuses,
            "guidance": "guided",
            "created": now(),
            "updated": now(),
            "lessons": {},
            "custom_lessons": [],
        }
        ws.manifest = {
            "schema": MANIFEST_SCHEMA,
            "curriculum_version": reg.version,
            "lessons": {},
        }
        report = sync_workspace(ws, reg)
        report["init"] = "new workspace created"
    if seed_gitignore(ws, reg, args.language):
        report["gitignore"] = "seeded .gitignore"
    if carry_src is not None:
        report["carried_over"] = carry_over(ws, reg, carry_src)
        ws.save()
    report["warnings"] = workspace_warnings(ws, reg)
    for w in report["warnings"]:
        warn(w)
    print(json.dumps(report, indent=2))


def cmd_sync(args):
    reg = Registry.load()
    ws = find_workspace(args).load()
    print(json.dumps(sync_workspace(ws, reg), indent=2))


def compute_next(ws: Workspace, reg: Registry):
    language = ws.state["language"]
    groups = reg.compose(language, ws.state["focuses"])
    ordered = [lid for g in groups for lid in g["lessons"]]
    for bucket in ("needs_review", "in_progress"):
        for lid in ordered:
            if ws.lesson_state(lid)["status"] == bucket:
                return lid, bucket
    for lid in ordered:
        if ws.lesson_state(lid)["status"] == "todo" and reg.is_authored(lid, language):
            return lid, "todo"
    return None, None


def cmd_status(args):
    reg = Registry.load()
    ws = find_workspace(args).load()
    st = ws.state
    language = st["language"]
    groups = reg.compose(language, st["focuses"])
    ordered = [lid for g in groups for lid in g["lessons"]]
    counts = {s: 0 for s in STATUSES}
    for lid in ordered:
        counts[ws.lesson_state(lid)["status"]] += 1
    authored = sum(1 for lid in ordered if reg.is_authored(lid, language))
    next_id, next_reason = compute_next(ws, reg)
    conflicts = (
        sorted(
            str(p.relative_to(ws.root))
            for p in (ws.root / "lessons").rglob("*.upstream")
        )
        if (ws.root / "lessons").exists()
        else []
    )
    pending = preview_sync(ws, reg)
    pending_actions = [k for k in SYNC_ACTIONS if pending[k]]
    stale = ws.manifest.get("composition_hash") != reg.composition_hash(
        language, st["focuses"]
    ) or bool(pending_actions)
    warnings = workspace_warnings(ws, reg)

    data = {
        "workspace": str(ws.root),
        "language": language,
        "focuses": st["focuses"],
        "guidance": st["guidance"],
        "curriculum_version": reg.version,
        "sync_needed": stale,
        "pending": pending,
        "progress": {**counts, "composed": len(ordered), "authored": authored},
        "needs_review": [
            x for x in ordered if ws.lesson_state(x)["status"] == "needs_review"
        ],
        "in_progress": [
            x for x in ordered if ws.lesson_state(x)["status"] == "in_progress"
        ],
        "conflicts": conflicts,
        "warnings": warnings,
        "next": {"lesson": next_id, "reason": next_reason},
        "next_dir": ws.manifest["lessons"].get(next_id, {}).get("dir")
        if next_id
        else None,
        "recent": [
            {
                "lesson": lid,
                **{k: v for k, v in ws.lesson_state(lid).items() if k != "history"},
            }
            for lid in ordered
            if ws.lesson_state(lid).get("updated_at")
        ][-5:],
    }
    if args.json:
        print(json.dumps(data, indent=2))
        return
    p = data["progress"]
    print(f"workspace  {data['workspace']}")
    print(
        f"language   {language}   "
        f"focuses: {', '.join(st['focuses']) or 'none'}   guidance: {st['guidance']}"
    )
    line = f"curriculum {reg.version}"
    if stale:
        summary = ", ".join(
            f"{len(pending[k])} {k.replace('_', ' ')}" for k in pending_actions
        )
        line += (
            f"   ⚠ SYNC NEEDED ({summary or 'registry changed'} — run: tutor.py sync)"
        )
    print(line)
    print(
        f"progress   {p['passed']}/{p['composed']} passed · "
        f"{p['in_progress']} in progress · {p['needs_review']} need review · "
        f"{p['skipped']} skipped · authored {p['authored']}/{p['composed']}"
    )
    if data["needs_review"]:
        print(f"⚠ review   {', '.join(data['needs_review'])}")
    if conflicts:
        print(f"⚠ conflict {', '.join(conflicts)}")
    for w in warnings:
        print(f"⚠ warning  {w}")
    if next_id:
        print(f"next       {next_id} ({next_reason}) → {data['next_dir']}")
    else:
        print("next       nothing actionable — roadmap complete or content pending")


def cmd_mark(args):
    reg = Registry.load()
    ws = find_workspace(args).load()
    lid = args.lesson
    known = lid in reg.lessons or any(
        c["id"] == lid for c in ws.state["custom_lessons"]
    )
    if not known:
        die(f"unknown lesson id '{lid}'")
    lstate = ws.lesson_state(lid)
    if args.status == "resolved":
        prior = lstate.pop("prior_status", None)
        if lstate["status"] != "needs_review" or prior is None:
            die(f"'{lid}' is not awaiting review resolution")
        lstate["status"] = prior
    else:
        if args.status not in STATUSES:
            die(f"status must be one of {STATUSES} or 'resolved'")
        lstate["status"] = args.status
        lstate.pop("prior_status", None)
    if args.grade:
        if not GRADE_RE.match(args.grade):
            die("grade must look like A, B+, C- …")
        lstate["grade"] = args.grade
    if args.note:
        lstate["notes"] = args.note
    lstate["updated_at"] = now()
    hist = lstate.setdefault("history", [])
    hist.append(
        {
            "at": lstate["updated_at"],
            "status": lstate["status"],
            **({"grade": args.grade} if args.grade else {}),
        }
    )
    del hist[:-20]
    write_roadmap(ws, reg)
    ws.save()
    print(
        json.dumps(
            {"lesson": lid, **{k: v for k, v in lstate.items() if k != "history"}},
            indent=2,
        )
    )


def cmd_verify(args):
    reg = Registry.load()
    ws = find_workspace(args).load()
    lid = args.lesson
    meta = reg.lessons.get(lid)
    if meta is None:
        die(
            f"unknown lesson id '{lid}' "
            "(custom lessons are verified by the tutor directly)"
        )
    language = ws.state["language"]
    vtype = reg.resolve_verify(lid, language)
    if vtype == "discussion":
        print(
            json.dumps(
                {
                    "lesson": lid,
                    "verify": "discussion",
                    "result": "no automated check — tutor gates via conversation",
                }
            )
        )
        return
    on_path = {
        x for g in reg.compose(language, ws.state["focuses"]) for x in g["lessons"]
    }
    if lid not in on_path:
        die(f"'{lid}' is not on the {language} path of this workspace")
    entry = ws.manifest["lessons"].get(lid)
    if entry is None:
        die(f"'{lid}' is not scaffolded here (content pending for {language})")
    cwd = ws.root / entry["dir"] / "exercise"
    if not cwd.is_dir():
        die(
            f"'{lid}' has no exercise directory in this workspace "
            f"(content pending for {language}?)"
        )
    cmd = VERIFY_COMMANDS[vtype]
    ensure_toolchain(cmd)
    if vtype == "pytest":
        for w in workspace_warnings(ws, reg):
            if "pyproject.toml" in w:
                warn(w)
    proc = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True, check=False)
    lstate = ws.lesson_state(lid)
    lstate["attempts"] = lstate.get("attempts", 0) + 1
    ws.save()
    print(
        json.dumps(
            {
                "lesson": lid,
                "verify": vtype,
                "cwd": str(cwd.relative_to(ws.root)),
                "command": " ".join(cmd),
                "exit_code": proc.returncode,
                "passed": proc.returncode == 0,
                "output": (proc.stdout + proc.stderr)[-4000:],
            },
            indent=2,
        )
    )
    sys.exit(proc.returncode)


def cmd_guidance(args):
    reg = Registry.load()
    ws = find_workspace(args).load()
    ws.state["guidance"] = args.mode
    write_roadmap(ws, reg)
    ws.save()
    print(json.dumps({"guidance": args.mode}))


def cmd_custom(args):
    reg = Registry.load()
    ws = find_workspace(args).load()
    lid = f"custom.{args.slug}"
    if any(c["id"] == lid for c in ws.state["custom_lessons"]):
        die(f"custom lesson '{lid}' already exists")
    rel = f"lessons/90-custom/{args.slug}"
    (ws.root / rel).mkdir(parents=True, exist_ok=True)
    ws.state["custom_lessons"].append({"id": lid, "title": args.title, "dir": rel})
    ws.lesson_state(lid)
    write_roadmap(ws, reg)
    ws.save()
    print(
        json.dumps(
            {
                "id": lid,
                "dir": rel,
                "note": "tutor authors content here; excluded from sync/diffing",
            },
            indent=2,
        )
    )


def cmd_graph(args):
    reg = Registry.load()
    if not args.language:
        rows = []
        for code, lang in reg.languages.items():
            path = lang["path"]
            lessons = [lid for k in path for lid in reg.stages[k]["lessons"]]
            packs = [p for p, v in reg.packs.items() if code in v.get("languages", [])]
            rows.append(
                {
                    "language": code,
                    "name": lang["name"],
                    "status": lang["status"],
                    "stages": len(path),
                    "lessons": len(lessons),
                    "authored": sum(1 for lid in lessons if reg.is_authored(lid, code)),
                    "packs": packs,
                }
            )
        if args.format == "json":
            print(json.dumps(rows, indent=2))
        else:
            for r in rows:
                print(
                    f"{r['language']:<8} {r['status']:<10} stages {r['stages']:>2} · "
                    f"lessons {r['authored']}/{r['lessons']} authored · "
                    f"packs: {', '.join(r['packs']) or '—'}"
                )
        return

    language = args.language
    focuses = [f.strip() for f in (args.focus or "").split(",") if f.strip()]
    groups = reg.compose(language, focuses)
    if args.format == "json":
        out = [
            {
                "group": g["title"],
                "slug": g["slug"],
                "lessons": [
                    {
                        "id": lid,
                        "title": reg.lessons[lid]["title"],
                        "duration": reg.lessons[lid]["duration"],
                        "verify": reg.resolve_verify(lid, language),
                        "authored": reg.is_authored(lid, language),
                        "objectives": reg.lessons[lid]["objectives"],
                    }
                    for lid in g["lessons"]
                ],
            }
            for g in groups
        ]
        print(json.dumps(out, indent=2))
    elif args.format == "mermaid":
        print("flowchart TD")
        prev = None
        for gi, g in enumerate(groups, start=1):
            node = f"g{gi}"
            n = len(g["lessons"])
            print(f'    {node}["{gi:02d} {g["title"]} ({n} lessons)"]')
            if prev:
                print(f"    {prev} --> {node}")
            prev = node
    else:
        for gi, g in enumerate(groups, start=1):
            print(f"{gi:02d} · {g['title']}")
            for li, lid in enumerate(g["lessons"], start=1):
                meta = reg.lessons[lid]
                pending = (
                    "" if reg.is_authored(lid, language) else "  [content pending]"
                )
                print(
                    f"   {li:02d} {meta['title']}  ({lid}, {meta['duration']}){pending}"
                )


# ---------------------------------------------------------------- validate/ci


def iter_language_lessons(reg: Registry, language: str):
    """(lesson-id, ordered-position) over a language's full path incl. all its packs."""
    focuses = [p for p, v in reg.packs.items() if language in v.get("languages", [])]
    groups = reg.compose(language, focuses)
    pos = 0
    for g in groups:
        for lid in g["lessons"]:
            yield lid, pos
            pos += 1


def exercise_section(text: str) -> str:
    lines = text.splitlines()
    start = next(
        (i for i, line in enumerate(lines) if line.startswith("## Exercise")), None
    )
    if start is None:
        return ""
    end = next(
        (i for i in range(start + 1, len(lines)) if lines[i].startswith("## ")),
        len(lines),
    )
    return "\n".join(lines[start:end])


def validate_registry(reg: Registry, errors: list, warnings: list):
    for key, stage in reg.stages.items():
        for field in ("title", "slug", "pool", "dir", "lessons"):
            if field not in stage:
                errors.append(f"stage {key}: missing field '{field}'")
    for code, lang in reg.languages.items():
        for k in lang.get("path", []):
            if k not in reg.stages:
                errors.append(f"language {code}: path references unknown stage '{k}'")
        runner = lang.get("verify", {}).get("type")
        if runner not in TEST_FILE_GLOBS:
            errors.append(
                f"language {code}: verify.type must be one of "
                f"{', '.join(TEST_FILE_GLOBS)}, got '{runner}'"
            )
    for pname, pack in reg.packs.items():
        for ins in pack.get("insertions", []):
            if ins.get("after") not in reg.stages:
                errors.append(
                    f"pack {pname}: insertion after unknown stage '{ins.get('after')}'"
                )
        for code in pack.get("languages", []):
            if code not in reg.languages:
                errors.append(f"pack {pname}: unknown language '{code}'")

    seen = {}
    for lid, (kind, key) in reg.owner.items():
        if lid not in reg.lessons:
            errors.append(f"{kind} {key}: references undefined lesson '{lid}'")
    for key, stage in reg.stages.items():
        for lid in stage["lessons"]:
            if lid in seen:
                errors.append(
                    f"lesson '{lid}' appears in both {seen[lid]} and stage {key}"
                )
            seen[lid] = f"stage {key}"
    for pname, pack in reg.packs.items():
        for ins in pack.get("insertions", []):
            for lid in ins["lessons"]:
                if lid in seen:
                    errors.append(
                        f"lesson '{lid}' appears in both {seen[lid]} and pack {pname}"
                    )
                seen[lid] = f"pack {pname}"
    for lid in reg.lessons:
        if lid not in reg.owner:
            warnings.append(
                f"lesson '{lid}' defined but not referenced by any stage/pack"
            )
    for lid, meta in reg.lessons.items():
        for field in ("title", "duration", "objectives", "verify"):
            if field not in meta:
                errors.append(f"lesson {lid}: missing field '{field}'")
        vtype = meta.get("verify", {}).get("type")
        if vtype not in VERIFY_TYPES:
            errors.append(f"lesson {lid}: unknown verify type '{vtype}'")
            continue
        if lid not in reg.owner or lid not in reg.lessons:
            continue
        if reg.is_language_pool(lid):
            code = reg.pool(lid)
            runner = reg.languages[code].get("verify", {}).get("type")
            if vtype == RESOLVED_VERIFY:
                errors.append(
                    f"lesson {lid}: '{RESOLVED_VERIFY}' is for shared and pack lessons; "
                    f"a {code} lesson declares its runner ('{runner}')"
                )
            elif vtype in TEST_FILE_GLOBS and vtype != runner:
                errors.append(
                    f"lesson {lid}: runner '{vtype}' is not {code}'s runner '{runner}'"
                )
        elif vtype in TEST_FILE_GLOBS:
            errors.append(
                f"lesson {lid}: a {reg.pool(lid)} lesson must declare "
                f"'{RESOLVED_VERIFY}' (resolved per language), not '{vtype}'"
            )

    if errors:
        return
    for code in reg.languages:
        order = {}
        for lid, pos in iter_language_lessons(reg, code):
            order[lid] = pos
        for lid, pos in order.items():
            for pre in reg.lessons.get(lid, {}).get("prereqs", []):
                if pre not in reg.lessons:
                    errors.append(f"lesson {lid}: unknown prereq '{pre}'")
                elif pre in order and order[pre] >= pos:
                    errors.append(
                        f"language {code}: prereq '{pre}' of '{lid}' "
                        "does not precede it"
                    )


def validate_content_tree(errors: list):
    capstone_rel = capstone_rel_from_exercise()
    for path in CONTENT_DIR.rglob("*"):
        rel = path.relative_to(CONTENT_DIR)
        if JUNK_NAMES.intersection(rel.parts):
            continue
        if path.is_dir():
            if path.name in RETIRED_DIRS:
                errors.append(
                    f"{rel}/: '{path.name}/' is retired — language variants live "
                    "under content/<lang>/<shared or focus>/<stage>/<lesson>/"
                )
            continue
        if path.stat().st_size > 512 * 1024:
            errors.append(f"content file over 512KB (built artifact?): {rel}")
        elif os.access(path, os.X_OK) and path.suffix not in (".sh", ".py"):
            errors.append(
                f"executable file without script extension (built binary?): {rel}"
            )
        if path.suffix in CAPSTONE_TEXT_SUFFIXES:
            text = path.read_text(encoding="utf-8", errors="ignore")
            for stale in sorted(set(CAPSTONE_REL_RE.findall(text)) - {capstone_rel}):
                errors.append(
                    f"{rel}: '{stale}' does not reach the workspace root from a "
                    f"scaffolded exercise dir — expected '{capstone_rel}'"
                )


def validate_overlays(reg: Registry, errors: list, warnings: list):
    """Anchors, snippets and overlay folders agree with the registry."""
    known_overlays = {}
    for lid in reg.owner:
        if lid not in reg.lessons or reg.is_language_pool(lid):
            continue
        for code in reg.languages:
            overlay = reg.overlay_dir(lid, code)
            known_overlays[overlay] = (lid, code)

    for lang_dir in sorted(CONTENT_DIR.iterdir()):
        if lang_dir.name in JUNK_NAMES or lang_dir.name in POOL_DIRS:
            continue
        if lang_dir.name not in reg.languages:
            errors.append(
                f"content/: unexpected entry '{lang_dir.name}'; content/ holds only "
                f"shared/, focus/ and one dir per registry language "
                f"({', '.join(reg.languages)})"
            )
            continue
        stage_dirs = {
            Path(stage["dir"]).name
            for stage in reg.stages.values()
            if Path(stage["dir"]).parts[:1] == (lang_dir.name,)
        }
        for child in sorted(lang_dir.iterdir()):
            if child.name in JUNK_NAMES or child.name in (*POOL_DIRS, *stage_dirs):
                continue
            errors.append(
                f"{lang_dir.name}/: unexpected entry '{child.name}'; a language dir "
                "holds only shared/, focus/ and the language's stage dirs "
                f"({', '.join(sorted(stage_dirs))})"
            )
        for pool in POOL_DIRS:
            base = lang_dir / pool
            if not base.is_dir():
                continue
            for lesson_dir in sorted(p for p in base.glob("*/*") if p.is_dir()):
                rel = lesson_dir.relative_to(CONTENT_DIR)
                if lesson_dir not in known_overlays:
                    errors.append(f"{rel}/: overlay matches no registered lesson")
                    continue
                for entry in sorted(lesson_dir.iterdir()):
                    if entry.name in JUNK_NAMES or entry.name in OVERLAY_ENTRIES:
                        continue
                    errors.append(
                        f"{rel}/: unexpected entry '{entry.name}'; an overlay holds "
                        "only snippets/, exercise/, solution/, TUTOR.md, quiz.json"
                    )

    for lid in reg.ordered_lessons():
        if lid not in reg.lessons or not reg.has_lesson_text(lid):
            continue
        anchors = reg.lesson_anchors(lid)
        for slot, problem in reg.anchor_lines(lid):
            if problem:
                where = f"anchor <!-- lang: {slot} --> " if slot else ""
                errors.append(f"{lid}: {where}{problem}")
        for slot in sorted({a for a in anchors if anchors.count(a) > 1}):
            errors.append(f"{lid}: duplicate anchor <!-- lang: {slot} --> in LESSON.md")
        tutor_md = reg.content_dir(lid) / "TUTOR.md"
        if tutor_md.exists() and b"<!-- lang:" in tutor_md.read_bytes():
            warnings.append(
                f"{lid}: TUTOR.md carries a language anchor; TUTOR.md is never rendered"
            )
        if reg.is_language_pool(lid):
            if anchors:
                warnings.append(
                    f"{lid}: LESSON.md carries language anchors but is a "
                    f"{reg.pool(lid)} lesson; they render as nothing"
                )
            continue
        text = (reg.content_dir(lid) / "LESSON.md").read_text(
            encoding="utf-8", errors="ignore"
        )
        hits = sorted(set(EXERCISE_TOOLCHAIN_RE.findall(exercise_section(text))))
        if hits:
            warnings.append(
                f"{lid}: '## Exercise' names toolchain commands ({', '.join(hits)}); "
                "language-bound steps belong in the overlay's exercise/README.md"
            )
        for code in reg.languages:
            overlay = reg.overlay_dir(lid, code)
            if overlay is None or not overlay.is_dir():
                continue
            label = f"{lid} [{code}]"
            for snippet in sorted((overlay / "snippets").glob("*")):
                if snippet.name in JUNK_NAMES:
                    continue
                slot = snippet.stem
                if snippet.suffix != ".md" or not SLOT_RE.match(slot):
                    errors.append(
                        f"{label}: snippet '{snippet.name}' must be <slot>.md"
                    )
                    continue
                if slot not in anchors:
                    errors.append(
                        f"{label}: snippet '{slot}' has no <!-- lang: {slot} --> "
                        "anchor in LESSON.md"
                    )
                if not snippet.read_bytes().endswith(b"\n"):
                    errors.append(f"{label}: snippet '{slot}' lacks a trailing newline")
            if (overlay / "exercise").is_dir() and not (
                overlay / "exercise" / "README.md"
            ).exists():
                warnings.append(f"{label}: overlay exercise has no README.md")
            quiz = overlay / "quiz.json"
            if quiz.exists():
                try:
                    json.loads(quiz.read_text(encoding="utf-8"))
                except json.JSONDecodeError as e:
                    errors.append(f"{label}: overlay quiz.json invalid JSON: {e}")


def lesson_completeness(reg: Registry, lid: str, code: str) -> list:
    """What an authored lesson still lacks for `code`; [] when complete."""
    vtype = reg.resolve_verify(lid, code)
    if vtype == "discussion":
        return []
    files = reg.exercise_files(lid, code)
    problems = []
    if vtype == "script":
        if "check.sh" not in files:
            problems.append("no check.sh under exercise dir")
        return problems
    glob = TEST_FILE_GLOBS[vtype]
    if not any(fnmatch.fnmatch(Path(rel).name, glob) for rel in files):
        problems.append(f"no {glob} under exercise dir")
    if reg.solution_dir(lid, code) is None:
        problems.append("no solution dir")
    if vtype == "pytest":
        for name in PYTEST_PROJECT_FILES:
            if name not in files:
                problems.append(f"no {name} beside the pytest exercise")
    return problems


def run_validate(strict: bool):
    errors, warnings = [], []
    reg = Registry.load()
    validate_registry(reg, errors, warnings)
    if errors:
        for e in errors:
            print(f"ERROR: {e}")
        print(f"validate: {len(errors)} error(s), {len(warnings)} warning(s)")
        return reg, errors
    validate_content_tree(errors)
    validate_overlays(reg, errors, warnings)

    for lid in reg.ordered_lessons():
        cdir = reg.content_dir(lid)
        if not reg.has_lesson_text(lid):
            continue
        for f in ("TUTOR.md", "quiz.json"):
            if not (cdir / f).exists():
                warnings.append(f"{lid}: missing {f}") if not strict else errors.append(
                    f"{lid}: missing {f}"
                )
            elif f == "quiz.json":
                try:
                    json.loads((cdir / f).read_text(encoding="utf-8"))
                except json.JSONDecodeError as e:
                    errors.append(f"{lid}: quiz.json invalid JSON: {e}")

    for code, lang in reg.languages.items():
        available = lang.get("status") == "available"
        missing, incomplete = [], []
        for lid, _ in iter_language_lessons(reg, code):
            if not reg.has_lesson_text(lid):
                missing.append(lid)
                continue
            if reg.is_language_pool(lid):
                incomplete.extend(
                    f"{lid}: {p}" for p in lesson_completeness(reg, lid, code)
                )
                continue
            overlay = reg.overlay_dir(lid, code)
            if reg.is_authored(lid, code) or (overlay is not None and overlay.is_dir()):
                incomplete.extend(
                    f"{lid}: {p}" for p in lesson_completeness(reg, lid, code)
                )
            else:
                missing.append(f"{lid} (no {code} exercise)")
        bucket = errors if strict else warnings
        if missing and available:
            bucket.append(
                f"language {code}: {len(missing)} lesson(s) without content: "
                + ", ".join(missing[:10])
                + ("…" if len(missing) > 10 else "")
            )
        bucket.extend(f"language {code}: {msg}" for msg in incomplete)

    for w in warnings:
        print(f"warning: {w}")
    for e in errors:
        print(f"ERROR: {e}")
    print(f"validate: {len(errors)} error(s), {len(warnings)} warning(s)")
    return reg, errors


def cmd_validate(args):
    _, errors = run_validate(args.strict)
    sys.exit(1 if errors else 0)


def run_solution(reg: Registry, lid: str, code: str, vtype: str, soldir: Path):
    cmd = VERIFY_COMMANDS[vtype]
    ensure_toolchain(cmd)
    with tempfile.TemporaryDirectory() as tmp:
        work = Path(tmp) / "work"
        for rel, src in reg.exercise_files(lid, code).items():
            place(src, work / rel)
        shutil.copytree(
            soldir, work, dirs_exist_ok=True, ignore=shutil.ignore_patterns(*JUNK_NAMES)
        )
        return subprocess.run(
            cmd, cwd=work, capture_output=True, text=True, check=False
        )


def cmd_ci(args):
    reg, errors = run_validate(strict=True)
    if errors:
        sys.exit(1)
    if args.language and args.language not in reg.languages:
        die(f"unknown language '{args.language}'. Known: {', '.join(reg.languages)}")
    failures, ran, skipped, matched = [], 0, [], 0
    for lid in reg.ordered_lessons():
        if args.filter and args.filter not in lid:
            continue
        matched += 1
        for code in reg.languages_for(lid):
            if args.language and code != args.language:
                continue
            label = f"{lid} [{code}]"
            if not reg.has_lesson_text(lid):
                skipped.append(f"{label}: no LESSON.md")
                continue
            vtype = reg.resolve_verify(lid, code)
            if vtype == "discussion":
                continue
            if not reg.is_authored(lid, code):
                reason = (
                    "no exercise"
                    if reg.is_language_pool(lid)
                    else f"no exercise variant for {code}"
                )
                skipped.append(f"{label}: {reason}")
                continue
            soldir = reg.solution_dir(lid, code)
            if soldir is None:
                skipped.append(f"{label}: no solution")
                continue
            proc = run_solution(reg, lid, code, vtype, soldir)
            ran += 1
            if proc.returncode != 0:
                failures.append(label)
                print(f"FAIL {label}\n{(proc.stdout + proc.stderr)[-2000:]}")
            else:
                print(f"ok   {label}")
    if args.filter and matched == 0:
        print(f"ci: no lesson id contains '{args.filter}'")
        sys.exit(1)
    for s in skipped:
        print(f"skip {s}")
    print(
        f"ci: {ran} solution(s) tested, {len(failures)} failed, "
        f"{len(skipped)} skipped (no exercise/solution pair)"
    )
    sys.exit(1 if failures else 0)


# ---------------------------------------------------------------- main


def main():
    ap = argparse.ArgumentParser(
        prog="tutor.py",
        description=__doc__,
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    ap.add_argument(
        "--workspace", default=".", help="workspace directory (default: cwd)"
    )
    sub = ap.add_subparsers(dest="cmd", required=True)

    p = sub.add_parser("init", help="create/refresh a workspace")
    p.add_argument("language")
    p.add_argument("--focus", help="comma-separated focus packs")
    p.add_argument(
        "--carry-over",
        metavar="DIR",
        help="another language's workspace whose passed shared lessons are noted here",
    )
    p.set_defaults(fn=cmd_init)

    p = sub.add_parser("sync", help="re-scaffold + diff report")
    p.set_defaults(fn=cmd_sync)

    p = sub.add_parser("status", help="session-start briefing")
    p.add_argument("--json", action="store_true")
    p.set_defaults(fn=cmd_status)

    p = sub.add_parser("mark", help="set lesson status")
    p.add_argument("lesson")
    p.add_argument("status", help=f"one of {STATUSES} or 'resolved'")
    p.add_argument("--grade")
    p.add_argument("--note")
    p.set_defaults(fn=cmd_mark)

    p = sub.add_parser("verify", help="run a lesson's automated checks")
    p.add_argument("lesson")
    p.set_defaults(fn=cmd_verify)

    p = sub.add_parser("guidance", help="set guidance mode")
    p.add_argument("mode", choices=GUIDANCE_MODES)
    p.set_defaults(fn=cmd_guidance)

    p = sub.add_parser("custom", help="manage tutor-generated custom lessons")
    csub = p.add_subparsers(dest="custom_cmd", required=True)
    c = csub.add_parser("add")
    c.add_argument("slug")
    c.add_argument("--title", required=True)
    c.set_defaults(fn=cmd_custom)

    p = sub.add_parser("graph", help="show curriculum graph / language matrix")
    p.add_argument("--language")
    p.add_argument("--focus", help="comma-separated focus packs")
    p.add_argument("--format", choices=("tree", "mermaid", "json"), default="tree")
    p.set_defaults(fn=cmd_graph)

    p = sub.add_parser("validate", help="repo-level registry/content validation")
    p.add_argument("--strict", action="store_true")
    p.set_defaults(fn=cmd_validate)

    p = sub.add_parser("ci", help="validate + run all solution tests")
    p.add_argument("--filter", help="only lessons whose id contains this substring")
    p.add_argument("--language", help="only this language's exercises and solutions")
    p.set_defaults(fn=cmd_ci)

    args = ap.parse_args()
    args.fn(args)


if __name__ == "__main__":
    main()
