#!/usr/bin/env python3
"""tutor.py — deterministic curriculum engine for the tutor skill.

Owns all learner-workspace state. The tutoring LLM never edits state files by
hand; it calls these subcommands:

    init <language> [--focus a,b]   create/refresh a workspace in CWD (idempotent)
    sync                            re-scaffold + JSON diff report
    status [--json]                 session-start briefing
    mark <lesson-id> <status> [--grade A..F] [--note ...]
    verify <lesson-id>              run the lesson's automated checks
    guidance <guided|standard|spartan>
    custom add <slug> --title T     register a tutor-generated custom lesson
    graph [--language X] [--focus ..] [--format tree|mermaid|json]
    validate [--strict]             repo-level registry/content validation
    ci [--filter substr]            validate + run all solution tests

Python 3 stdlib only. Curriculum is resolved relative to this file; the
workspace is the current directory (or --workspace).
"""

import argparse
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
TUTOR_ONLY_DIRS = {"solution", "solutions"}
VERIFY_COMMANDS = {
    "gotest": ["go", "test", "-race", "./..."],
    "pytest": ["python3", "-m", "pytest"],
    "script": ["bash", "./check.sh"],
}
TEST_FILE_GLOBS = {"gotest": "*_test.go", "pytest": "test_*.py"}
# The capstone harnesses tell learners how to reach projects/capstone at the
# workspace root from a scaffolded exercise directory. That depth belongs to
# lesson_workspace_dir(), so validate re-derives it and rejects stale text.
CAPSTONE_REL_RE = re.compile(r"(?:\.\./)+projects/capstone")
CAPSTONE_TEXT_SUFFIXES = (".md", ".go", ".sh", ".json")


def now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds")


def sha256_bytes(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def sha256_file(path: Path) -> str:
    return sha256_bytes(path.read_bytes())


def die(msg: str, code: int = 1) -> NoReturn:
    print(f"error: {msg}", file=sys.stderr)
    sys.exit(code)


def write_json_atomic(path: Path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    tmp = tempfile.NamedTemporaryFile(
        "w", dir=path.parent, delete=False, suffix=".tmp", encoding="utf-8"
    )
    try:
        json.dump(data, tmp, indent=2, ensure_ascii=False)
        tmp.write("\n")
        tmp.close()
        Path(tmp.name).replace(path)
    except BaseException:
        Path(tmp.name).unlink(missing_ok=True)
        raise


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

    def hash(self) -> str:
        return sha256_file(REGISTRY_PATH)[:12]

    def slug(self, lid: str) -> str:
        return lid.rsplit(".", 1)[-1]

    def content_dir(self, lid: str) -> Path:
        kind, key = self.owner.get(lid, (None, None))
        if kind == "stage":
            base = self.stages[key]["dir"]
        elif kind == "pack":
            base = self.packs[key]["dir"]
        else:
            die(f"lesson {lid} not referenced by any stage or pack")
        return CONTENT_DIR / base / self.slug(lid)

    def pool(self, lid: str) -> str:
        kind, key = self.owner[lid]
        return self.stages[key]["pool"] if kind == "stage" else "focus"

    def is_authored(self, lid: str) -> bool:
        return (self.content_dir(lid) / "LESSON.md").exists()

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

    def scaffold_map(self, lid: str, language: str) -> dict:
        """relative dest path -> source Path, for every learner-facing file."""
        src_root = self.content_dir(lid)
        if not src_root.exists():
            return {}
        out = {}
        for src in sorted(src_root.rglob("*")):
            if not src.is_file():
                continue
            rel = src.relative_to(src_root)
            parts = rel.parts
            if parts[0] in TUTOR_ONLY_DIRS or rel.name in TUTOR_ONLY_FILES:
                continue
            if parts[0] == "exercises":
                if len(parts) < 3 or parts[1] != language:
                    continue
                rel = Path("exercise").joinpath(*parts[2:])
            out[str(rel)] = src
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
        return self

    def save(self):
        self.state["updated"] = now()
        write_json_atomic(self.state_path, self.state)
        write_json_atomic(self.manifest_path, self.manifest)

    def lesson_state(self, lid: str) -> dict:
        return self.state["lessons"].setdefault(lid, {"status": "todo"})


def find_workspace(args) -> Workspace:
    return Workspace(Path(args.workspace).resolve())


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


def sync_workspace(ws: Workspace, reg: Registry) -> dict:
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
            seen.add(lid)
            ws.lesson_state(lid)
            if not reg.is_authored(lid):
                report["pending_content"].append(lid)
                continue
            expected_dir = lesson_workspace_dir(gi, group["slug"], li, reg.slug(lid))
            entry = man_lessons.get(lid)
            changed = False

            if entry is None:
                entry = {"dir": expected_dir, "files": {}}
                man_lessons[lid] = entry
                report["added"].append(lid)
            elif entry["dir"] != expected_dir:
                old, new = ws.root / entry["dir"], ws.root / expected_dir
                if old.exists():
                    new.parent.mkdir(parents=True, exist_ok=True)
                    old.rename(new)
                report["renamed"].append(
                    {"lesson": lid, "from": entry["dir"], "to": expected_dir}
                )
                entry["dir"] = expected_dir

            dest_root = ws.root / entry["dir"]
            smap = reg.scaffold_map(lid, language)

            for rel, src in smap.items():
                src_hash = sha256_file(src)
                man_hash = entry["files"].get(rel)
                dest = dest_root / rel
                ws_hash = sha256_file(dest) if dest.exists() else None
                if src_hash == man_hash:
                    continue
                if ws_hash is None or ws_hash == man_hash:
                    dest.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(src, dest)
                    if man_hash is not None:
                        report["updated"].append(f"{lid}:{rel}")
                elif ws_hash == src_hash:
                    pass
                else:
                    sidecar = dest.with_name(dest.name + ".upstream")
                    shutil.copy2(src, sidecar)
                    report["conflicts"].append(
                        {
                            "lesson": lid,
                            "file": rel,
                            "sidecar": str(sidecar.relative_to(ws.root)),
                        }
                    )
                entry["files"][rel] = src_hash
                changed = changed or man_hash is not None

            for rel in [r for r in entry["files"] if r not in smap]:
                dest = dest_root / rel
                if dest.exists() and sha256_file(dest) == entry["files"][rel]:
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
        if src.exists():
            attic = ws.tutor_dir / "attic" / Path(entry["dir"]).name
            attic.parent.mkdir(parents=True, exist_ok=True)
            if attic.exists():
                shutil.rmtree(attic)
            src.rename(attic)
        report["removed"].append(lid)

    prune_empty_dirs(ws.root / "lessons")
    ws.manifest["curriculum_version"] = reg.version
    ws.manifest["registry_hash"] = reg.hash()
    write_roadmap(ws, reg)
    ws.save()
    return report


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
    groups = reg.compose(st["language"], st["focuses"])
    counts = {s: 0 for s in STATUSES}
    total = 0
    for g in groups:
        for lid in g["lessons"]:
            total += 1
            counts[ws.lesson_state(lid)["status"]] += 1
    lang_name = reg.languages[st["language"]]["name"]
    focuses = ", ".join(st["focuses"]) or "none"
    lines = [
        f"# Roadmap — {lang_name}",
        "",
        f"> Generated by tutor.py — do not edit. Focuses: {focuses} · "
        f"Guidance: {st['guidance']} · Curriculum {reg.version}",
        ">",
        f"> Progress: **{counts['passed']}/{total} passed** · "
        f"{counts['in_progress']} in progress · {counts['needs_review']} need review · "
        f"{counts['skipped']} skipped",
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
            if not reg.is_authored(lid):
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
            "registry_hash": reg.hash(),
            "lessons": {},
        }
        report = sync_workspace(ws, reg)
        report["init"] = "new workspace created"
    print(json.dumps(report, indent=2))


def cmd_sync(args):
    reg = Registry.load()
    ws = find_workspace(args).load()
    print(json.dumps(sync_workspace(ws, reg), indent=2))


def compute_next(ws: Workspace, reg: Registry):
    groups = reg.compose(ws.state["language"], ws.state["focuses"])
    ordered = [lid for g in groups for lid in g["lessons"]]
    for bucket in ("needs_review", "in_progress"):
        for lid in ordered:
            if ws.lesson_state(lid)["status"] == bucket:
                return lid, bucket
    for lid in ordered:
        if ws.lesson_state(lid)["status"] == "todo" and reg.is_authored(lid):
            return lid, "todo"
    return None, None


def cmd_status(args):
    reg = Registry.load()
    ws = find_workspace(args).load()
    st = ws.state
    groups = reg.compose(st["language"], st["focuses"])
    ordered = [lid for g in groups for lid in g["lessons"]]
    counts = {s: 0 for s in STATUSES}
    for lid in ordered:
        counts[ws.lesson_state(lid)["status"]] += 1
    authored = sum(1 for lid in ordered if reg.is_authored(lid))
    next_id, next_reason = compute_next(ws, reg)
    conflicts = (
        sorted(
            str(p.relative_to(ws.root))
            for p in (ws.root / "lessons").rglob("*.upstream")
        )
        if (ws.root / "lessons").exists()
        else []
    )
    stale = ws.manifest.get("registry_hash") != reg.hash()

    data = {
        "workspace": str(ws.root),
        "language": st["language"],
        "focuses": st["focuses"],
        "guidance": st["guidance"],
        "curriculum_version": reg.version,
        "sync_needed": stale,
        "progress": {**counts, "composed": len(ordered), "authored": authored},
        "needs_review": [
            x for x in ordered if ws.lesson_state(x)["status"] == "needs_review"
        ],
        "in_progress": [
            x for x in ordered if ws.lesson_state(x)["status"] == "in_progress"
        ],
        "conflicts": conflicts,
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
        f"language   {st['language']}   "
        f"focuses: {', '.join(st['focuses']) or 'none'}   guidance: {st['guidance']}"
    )
    print(
        f"curriculum {reg.version}"
        + (
            "   ⚠ SYNC NEEDED (curriculum changed — run: tutor.py sync)"
            if stale
            else ""
        )
    )
    print(
        f"progress   {p['passed']}/{p['composed']} passed · "
        f"{p['in_progress']} in progress · {p['needs_review']} need review · "
        f"{p['skipped']} skipped · authored {p['authored']}/{p['composed']}"
    )
    if data["needs_review"]:
        print(f"⚠ review   {', '.join(data['needs_review'])}")
    if conflicts:
        print(f"⚠ conflict {', '.join(conflicts)}")
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
        lstate["note"] = args.note
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
    vtype = meta["verify"]["type"]
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
    entry = ws.manifest["lessons"].get(lid)
    if entry is None:
        die(f"'{lid}' is not scaffolded in this workspace")
    lesson_dir = ws.root / entry["dir"]
    cwd = lesson_dir / "exercise" if (lesson_dir / "exercise").exists() else lesson_dir
    cmd = VERIFY_COMMANDS[vtype]
    proc = subprocess.run(cmd, cwd=cwd, capture_output=True, text=True)
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
                    "authored": sum(1 for lid in lessons if reg.is_authored(lid)),
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

    focuses = [f.strip() for f in (args.focus or "").split(",") if f.strip()]
    groups = reg.compose(args.language, focuses)
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
                        "verify": reg.lessons[lid]["verify"]["type"],
                        "authored": reg.is_authored(lid),
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
                pending = "" if reg.is_authored(lid) else "  [content pending]"
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


def run_validate(strict: bool):
    errors, warnings = [], []
    reg = Registry.load()

    for key, stage in reg.stages.items():
        for field in ("title", "slug", "pool", "dir", "lessons"):
            if field not in stage:
                errors.append(f"stage {key}: missing field '{field}'")
    for code, lang in reg.languages.items():
        for k in lang.get("path", []):
            if k not in reg.stages:
                errors.append(f"language {code}: path references unknown stage '{k}'")
    for pname, pack in reg.packs.items():
        for ins in pack.get("insertions", []):
            if ins.get("after") not in reg.stages:
                errors.append(
                    f"pack {pname}: insertion after unknown stage '{ins.get('after')}'"
                )

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
        if vtype not in (*VERIFY_COMMANDS, "discussion"):
            errors.append(f"lesson {lid}: unknown verify type '{vtype}'")

    capstone_rel = capstone_rel_from_exercise()
    for path in CONTENT_DIR.rglob("*"):
        if not path.is_file():
            continue
        rel = path.relative_to(CONTENT_DIR)
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

    for code, lang in reg.languages.items():
        if lang.get("status") != "available":
            continue
        missing, incomplete = [], []
        for lid, _ in iter_language_lessons(reg, code):
            if not reg.is_authored(lid):
                missing.append(lid)
                continue
            meta = reg.lessons[lid]
            vtype = meta["verify"]["type"]
            cdir = reg.content_dir(lid)
            if vtype in TEST_FILE_GLOBS:
                exdirs = [cdir / "exercise", cdir / "exercises" / code]
                exdir = next((d for d in exdirs if d.is_dir()), None)
                if exdir is None or not list(exdir.rglob(TEST_FILE_GLOBS[vtype])):
                    incomplete.append(
                        f"{lid}: no {TEST_FILE_GLOBS[vtype]} under exercise dir"
                    )
                soldirs = [cdir / "solution", cdir / "solutions" / code]
                if not any(d.is_dir() and any(d.iterdir()) for d in soldirs):
                    incomplete.append(f"{lid}: no solution dir")
            for f in ("TUTOR.md", "quiz.json"):
                if not (cdir / f).exists():
                    incomplete.append(f"{lid}: missing {f}")
                elif f == "quiz.json":
                    try:
                        json.loads((cdir / f).read_text(encoding="utf-8"))
                    except json.JSONDecodeError as e:
                        errors.append(f"{lid}: quiz.json invalid JSON: {e}")
        bucket = errors if strict else warnings
        if missing:
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


def cmd_ci(args):
    reg, errors = run_validate(strict=False)
    if errors:
        sys.exit(1)
    failures, ran, skipped = [], 0, 0
    for code, lang in reg.languages.items():
        if lang.get("status") != "available":
            continue
        for lid, _ in iter_language_lessons(reg, code):
            if args.filter and args.filter not in lid:
                continue
            meta = reg.lessons.get(lid)
            if meta is None or not reg.is_authored(lid):
                continue
            vtype = meta["verify"]["type"]
            if vtype not in VERIFY_COMMANDS:
                continue
            cdir = reg.content_dir(lid)
            exdir = next(
                (
                    d
                    for d in (cdir / "exercise", cdir / "exercises" / code)
                    if d.is_dir()
                ),
                None,
            )
            soldir = next(
                (
                    d
                    for d in (cdir / "solution", cdir / "solutions" / code)
                    if d.is_dir()
                ),
                None,
            )
            if exdir is None or soldir is None:
                skipped += 1
                continue
            with tempfile.TemporaryDirectory() as tmp:
                work = Path(tmp) / "work"
                shutil.copytree(exdir, work)
                shutil.copytree(soldir, work, dirs_exist_ok=True)
                proc = subprocess.run(
                    VERIFY_COMMANDS[vtype], cwd=work, capture_output=True, text=True
                )
            ran += 1
            if proc.returncode != 0:
                failures.append(lid)
                print(f"FAIL {lid}\n{(proc.stdout + proc.stderr)[-2000:]}")
            else:
                print(f"ok   {lid}")
    print(
        f"ci: {ran} solution(s) tested, {len(failures)} failed, "
        f"{skipped} skipped (missing exercise/solution)"
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
    p.set_defaults(fn=cmd_ci)

    args = ap.parse_args()
    args.fn(args)


if __name__ == "__main__":
    main()
