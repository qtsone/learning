"""End-to-end tests for skills/tutor/scripts/tutor.py.

The engine resolves its curriculum relative to its own file, so every test
copies tutor.py into a scratch skill directory next to a small synthetic
curriculum and drives it through subprocess, exactly as the tutor skill does.
Real toolchains (go, uv) are only needed by the tests that say so; everything
else runs on bash. Run with: python3 -m unittest discover -s tests -v
"""

import json
import os
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

REPO = Path(__file__).resolve().parents[1]
ENGINE = REPO / "skills" / "tutor" / "scripts" / "tutor.py"

CHECK_ANSWER = """#!/usr/bin/env bash
# passes once answer.txt holds 42
[ -f answer.txt ] && [ "$(cat answer.txt)" = "42" ] && { echo ok; exit 0; }
echo "FAIL: write 42 into answer.txt"; exit 1
"""
CHECK_DONE = """#!/usr/bin/env bash
[ -f done.txt ] && { echo ok; exit 0; }
echo "FAIL: create done.txt"; exit 1
"""
GO_MOD = "module tutor.local/x\n\ngo 1.22\n"
GO_SRC = "package x\n\nfunc Answer() int {\n\t// TODO: implement.\n\treturn 0\n}\n"
GO_SOL = "package x\n\nfunc Answer() int {\n\treturn 42\n}\n"
GO_TEST = (
    'package x\n\nimport "testing"\n\n'
    "func TestAnswer(t *testing.T) {\n\tif Answer() != 42 {\n"
    '\t\tt.Fatalf("got %d, want 42", Answer())\n\t}\n}\n'
)
PYPROJECT = """[project]
name = "x"
version = "0.0.0"
requires-python = ">=3.14"
dependencies = []

[dependency-groups]
dev = ["pytest==9.1.1"]

[tool.uv]
package = false
"""
PY_SRC = "def answer() -> int:\n    # TODO: implement.\n    return 0\n"
PY_SOL = "def answer() -> int:\n    return 42\n"
PY_TEST = "from x import answer\n\n\ndef test_answer():\n    assert answer() == 42\n"

MIXED_LESSON = """# Mixed

> `shared.s0.mixed` · ~1h · Stage: Shared

Neutral paragraph one.

<!-- lang: intro -->

Neutral paragraph two.

<!-- lang: extra -->

## Exercise

Build the thing; run the checks in `exercise/README.md`.
"""
GO_INTRO = "**In Go:** slices are views.\n\n```go\nxs := []int{1}\n```\n"
PY_INTRO = "**In Python:** lists copy on slice.\n\n```python\nxs = [1]\n```\n"
PY_EXTRA = "**In Python:** one more note.\n"
PY_ONLY = "**In Python:** only Python says this.\n"


def lesson_meta(title, vtype, **extra):
    return {
        "title": title,
        "duration": "1h",
        "objectives": ["Do the thing", "Explain the thing"],
        "verify": {"type": vtype},
        **extra,
    }


def base_registry():
    return {
        "schema": 1,
        "version": "2026.09.0",
        "languages": {
            "go": {
                "name": "Go",
                "status": "available",
                "verify": {"type": "gotest"},
                "workspace_ignore": ["*.test"],
                "path": ["s0", "g1"],
            },
            "py": {
                "name": "Python",
                "status": "available",
                "verify": {"type": "pytest"},
                "path": ["s0", "p1"],
            },
        },
        "stages": {
            "s0": {
                "title": "Shared",
                "slug": "shared",
                "pool": "shared",
                "dir": "shared/s0",
                "lessons": ["shared.s0.talk", "shared.s0.setup", "shared.s0.mixed"],
            },
            "g1": {
                "title": "Go basics",
                "slug": "go-basics",
                "pool": "go",
                "dir": "go/g1",
                "lessons": ["go.g1.hello", "go.g1.chat", "go.g1.tool"],
            },
            "p1": {
                "title": "Py basics",
                "slug": "py-basics",
                "pool": "py",
                "dir": "py/p1",
                "lessons": ["py.p1.hello", "py.p1.tool"],
            },
        },
        "packs": {
            "kit": {
                "title": "Kit",
                "dir": "focus/kit",
                "languages": ["go", "py"],
                "insertions": [
                    {
                        "after": "s0",
                        "slug": "kit",
                        "title": "Kit",
                        "lessons": ["focus.kit.one"],
                    }
                ],
            },
            "empty": {
                "title": "Empty",
                "dir": "focus/empty",
                "languages": ["go", "py"],
                "insertions": [],
            },
        },
        "lessons": {
            "shared.s0.talk": lesson_meta("Talk", "discussion"),
            "shared.s0.setup": lesson_meta("Setup", "script"),
            "shared.s0.mixed": lesson_meta("Mixed", "tests"),
            "go.g1.hello": lesson_meta("Hello Go", "gotest"),
            "go.g1.chat": lesson_meta("Chat", "discussion"),
            "go.g1.tool": lesson_meta("Tool", "script"),
            "py.p1.hello": lesson_meta("Hello Py", "pytest"),
            "py.p1.tool": lesson_meta("Py tool", "script"),
            "focus.kit.one": lesson_meta("Kit one", "script"),
        },
    }


class Scratch:
    """A scratch copy of the engine beside a synthetic curriculum."""

    def __init__(self, root: Path):
        self.root = root
        self.skill = root / "skill"
        self.content = self.skill / "curriculum" / "content"
        self.registry_path = self.skill / "curriculum" / "registry.json"
        (self.skill / "scripts").mkdir(parents=True)
        shutil.copy2(ENGINE, self.skill / "scripts" / "tutor.py")
        self.content.mkdir(parents=True)
        self.registry = base_registry()
        self.save_registry()
        self.seed_content()

    def save_registry(self):
        self.registry_path.write_text(json.dumps(self.registry, indent=2) + "\n")

    def write(self, rel: str, text: str):
        path = self.content / rel
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(text)
        return path

    def lesson(self, rel: str, lid: str, extra_files=None):
        self.write(f"{rel}/LESSON.md", f"# {lid}\n\n> `{lid}` · ~1h\n\nTheory.\n")
        self.write(f"{rel}/TUTOR.md", "# Tutor notes\n")
        self.write(
            f"{rel}/quiz.json", json.dumps({"pass_rule": "all", "questions": []})
        )
        for name, text in (extra_files or {}).items():
            self.write(f"{rel}/{name}", text)

    def seed_content(self):
        self.lesson("shared/s0/talk", "shared.s0.talk")
        self.lesson(
            "shared/s0/setup", "shared.s0.setup", {"exercise/check.sh": CHECK_DONE}
        )
        self.write("go/shared/s0/setup/solution/done.txt", "done\n")
        self.lesson("shared/s0/mixed", "shared.s0.mixed")
        self.write("shared/s0/mixed/LESSON.md", MIXED_LESSON)
        self.write("go/shared/s0/mixed/snippets/intro.md", GO_INTRO)
        self.write("go/shared/s0/mixed/exercise/README.md", "Run go test.\n")
        self.write("go/shared/s0/mixed/exercise/go.mod", GO_MOD)
        self.write("go/shared/s0/mixed/exercise/x.go", GO_SRC)
        self.write("go/shared/s0/mixed/exercise/x_test.go", GO_TEST)
        self.write("go/shared/s0/mixed/solution/x.go", GO_SOL)
        self.write("go/shared/s0/mixed/TUTOR.md", "# Go notes\n")
        self.write(
            "go/shared/s0/mixed/quiz.json",
            json.dumps({"questions": [{"id": "q1", "prompt": "go?"}]}),
        )
        self.write("py/shared/s0/mixed/snippets/intro.md", PY_INTRO)
        self.write("py/shared/s0/mixed/snippets/extra.md", PY_EXTRA)
        self.write("py/shared/s0/mixed/exercise/README.md", "Run pytest.\n")
        self.write_py_project("py/shared/s0/mixed/exercise")
        self.write("py/shared/s0/mixed/solution/x.py", PY_SOL)
        self.lesson(
            "go/g1/hello",
            "go.g1.hello",
            {
                "exercise/go.mod": GO_MOD,
                "exercise/x.go": GO_SRC,
                "exercise/x_test.go": GO_TEST,
                "solution/x.go": GO_SOL,
            },
        )
        self.lesson("go/g1/chat", "go.g1.chat")
        self.lesson(
            "go/g1/tool",
            "go.g1.tool",
            {"exercise/check.sh": CHECK_ANSWER, "solution/answer.txt": "42\n"},
        )
        self.lesson("py/p1/hello", "py.p1.hello", {"solution/x.py": PY_SOL})
        self.write_py_project("py/p1/hello/exercise")
        self.lesson(
            "py/p1/tool",
            "py.p1.tool",
            {"exercise/check.sh": CHECK_ANSWER, "solution/answer.txt": "42\n"},
        )
        self.lesson(
            "focus/kit/one",
            "focus.kit.one",
            {"exercise/check.sh": CHECK_ANSWER, "solution/answer.txt": "42\n"},
        )

    def write_py_project(self, rel: str, lock: str = "# placeholder lock\n"):
        self.write(f"{rel}/pyproject.toml", PYPROJECT)
        self.write(f"{rel}/.python-version", "3.14\n")
        self.write(f"{rel}/uv.lock", lock)
        self.write(f"{rel}/x.py", PY_SRC)
        self.write(f"{rel}/test_x.py", PY_TEST)

    def run(self, *args, workspace=None, check=True, env=None):
        cmd = [sys.executable, str(self.skill / "scripts" / "tutor.py")]
        if workspace is not None:
            cmd += ["--workspace", str(workspace)]
        cmd += list(args)
        proc = subprocess.run(cmd, capture_output=True, text=True, env=env, check=False)
        if check and proc.returncode != 0:
            raise AssertionError(
                f"{' '.join(args)} exited {proc.returncode}\n"
                f"stdout:\n{proc.stdout}\nstderr:\n{proc.stderr}"
            )
        return proc

    def run_json(self, *args, workspace=None):
        return json.loads(self.run(*args, workspace=workspace).stdout)

    def workspace(self, name: str) -> Path:
        ws = self.root / name
        ws.mkdir()
        return ws


class EngineTest(unittest.TestCase):
    def setUp(self):
        self._tmp = tempfile.TemporaryDirectory()
        self.sc = Scratch(Path(self._tmp.name))

    def tearDown(self):
        self._tmp.cleanup()

    def validate(self, *args, expect_ok=True):
        proc = self.sc.run("validate", *args, check=False)
        if expect_ok:
            self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        else:
            self.assertNotEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        return proc.stdout

    def test_validate_strict_passes_on_synthetic_curriculum(self):
        out = self.validate("--strict")
        self.assertIn("0 error(s)", out)

    def test_verify_type_invariants(self):
        reg = self.sc.registry
        reg["lessons"]["shared.s0.mixed"]["verify"]["type"] = "gotest"
        reg["lessons"]["go.g1.hello"]["verify"]["type"] = "tests"
        reg["lessons"]["py.p1.hello"]["verify"]["type"] = "gotest"
        reg["languages"]["py"]["verify"]["type"] = "script"
        self.sc.save_registry()
        out = self.validate(expect_ok=False)
        self.assertIn("shared.s0.mixed", out)
        self.assertIn("go.g1.hello", out)
        self.assertIn("py.p1.hello", out)
        self.assertIn("language py", out)

    def test_lesson_is_rendered_per_language(self):
        go_ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=go_ws)
        py_ws = self.sc.workspace("py")
        self.sc.run_json("init", "py", workspace=py_ws)
        go_text = (go_ws / "lessons/01-shared/03-mixed/LESSON.md").read_text()
        py_text = (py_ws / "lessons/01-shared/03-mixed/LESSON.md").read_text()
        self.assertEqual(
            go_text,
            MIXED_LESSON.replace("<!-- lang: intro -->\n", GO_INTRO).replace(
                "<!-- lang: extra -->\n\n", ""
            ),
        )
        self.assertEqual(
            py_text,
            MIXED_LESSON.replace("<!-- lang: intro -->\n", PY_INTRO).replace(
                "<!-- lang: extra -->\n", PY_EXTRA
            ),
        )
        self.assertNotIn("<!-- lang:", go_text)

    def test_python_only_edits_leave_go_workspace_untouched(self):
        go_ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=go_ws)
        before = {
            str(p.relative_to(go_ws)): p.read_bytes()
            for p in go_ws.rglob("*")
            if p.is_file()
        }
        self.sc.write(
            "py/shared/s0/mixed/snippets/extra.md", "**In Python:** changed.\n"
        )
        self.sc.write(
            "shared/s0/mixed/LESSON.md",
            MIXED_LESSON.replace(
                "Neutral paragraph two.\n",
                "Neutral paragraph two.\n\n<!-- lang: py-only -->\n",
            ),
        )
        self.sc.write("py/shared/s0/mixed/snippets/py-only.md", PY_ONLY)
        self.sc.write("py/shared/s0/setup/exercise/README.md", "py brief\n")
        self.sc.registry["lessons"]["py.p1.new"] = lesson_meta("New", "discussion")
        self.sc.registry["stages"]["p1"]["lessons"].append("py.p1.new")
        self.sc.save_registry()
        status = self.sc.run_json("status", "--json", workspace=go_ws)
        self.assertFalse(status["sync_needed"], status)
        report = self.sc.run_json("sync", workspace=go_ws)
        self.assertEqual(report["updated"], [])
        self.assertEqual(report["needs_review"], [])
        after = {
            str(p.relative_to(go_ws)): p.read_bytes()
            for p in go_ws.rglob("*")
            if p.is_file()
        }
        self.assertEqual(before, after)
        py_ws = self.sc.workspace("py")
        self.sc.run_json("init", "py", workspace=py_ws)
        py_text = (py_ws / "lessons/01-shared/03-mixed/LESSON.md").read_text()
        self.assertIn(
            "Neutral paragraph two.\n\n"
            + PY_ONLY
            + "\n**In Python:** changed.\n\n## Exercise",
            py_text,
        )
        self.sc.registry["lessons"]["go.g1.hello"]["title"] = "Hello again"
        self.sc.save_registry()
        status = self.sc.run_json("status", "--json", workspace=go_ws)
        self.assertTrue(status["sync_needed"], status)

    def test_overlay_exercise_scaffolds_and_tutor_files_never_do(self):
        ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=ws)
        mixed = ws / "lessons/01-shared/03-mixed"
        self.assertTrue((mixed / "exercise/x_test.go").exists())
        self.assertTrue((mixed / "exercise/README.md").exists())
        self.assertFalse((mixed / "snippets").exists())
        self.assertFalse((mixed / "solution").exists())
        self.assertFalse((mixed / "TUTOR.md").exists())
        self.assertFalse((mixed / "quiz.json").exists())
        setup = ws / "lessons/01-shared/02-setup"
        self.assertTrue((setup / "exercise/check.sh").exists())
        self.assertFalse((setup / "solution").exists())
        self.sc.write(
            "go/shared/s0/setup/exercise/check.sh", "#!/usr/bin/env bash\nexit 0\n"
        )
        self.sc.write("go/shared/s0/setup/exercise/hint.txt", "go only\n")
        report = self.sc.run_json("sync", workspace=ws)
        self.assertIn("shared.s0.setup:exercise/check.sh", report["updated"])
        self.assertEqual(
            (setup / "exercise/check.sh").read_text(), "#!/usr/bin/env bash\nexit 0\n"
        )
        self.assertTrue((setup / "exercise/hint.txt").exists())

    def test_shared_lesson_without_variant_is_content_pending(self):
        shutil.rmtree(self.sc.content / "py/shared/s0/mixed")
        out = self.validate()
        self.assertIn("warning", out)
        self.assertIn("shared.s0.mixed", out)
        self.validate("--strict", expect_ok=False)
        py_ws = self.sc.workspace("py")
        report = self.sc.run_json("init", "py", workspace=py_ws)
        self.assertIn("shared.s0.mixed", report["pending_content"])
        self.assertFalse((py_ws / "lessons/01-shared/03-mixed").exists())
        roadmap = (py_ws / "ROADMAP.md").read_text()
        self.assertIn("content pending", roadmap)
        status = self.sc.run_json("status", "--json", workspace=py_ws)
        self.assertEqual(
            status["progress"]["authored"], status["progress"]["composed"] - 1
        )
        for _ in range(3):
            nxt = status["next"]["lesson"]
            self.assertNotEqual(nxt, "shared.s0.mixed")
            self.sc.run_json("mark", nxt, "passed", workspace=py_ws)
            status = self.sc.run_json("status", "--json", workspace=py_ws)
        go_ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=go_ws)
        self.assertTrue((go_ws / "lessons/01-shared/03-mixed/exercise/x.go").exists())
        rows = self.sc.run_json("graph", "--format", "json")
        by = {r["language"]: r for r in rows}
        self.assertEqual(by["go"]["authored"], by["go"]["lessons"])
        self.assertEqual(by["py"]["authored"], by["py"]["lessons"] - 1)

    def test_graph_reports_resolved_verify_per_language(self):
        groups = self.sc.run_json("graph", "--language", "py", "--format", "json")
        mixed = next(
            lesson
            for g in groups
            for lesson in g["lessons"]
            if lesson["id"] == "shared.s0.mixed"
        )
        self.assertEqual(mixed["verify"], "pytest")
        groups = self.sc.run_json("graph", "--language", "go", "--format", "json")
        mixed = next(
            lesson
            for g in groups
            for lesson in g["lessons"]
            if lesson["id"] == "shared.s0.mixed"
        )
        self.assertEqual(mixed["verify"], "gotest")

    def test_verify_runs_script_and_counts_attempts(self):
        ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=ws)
        proc = self.sc.run("verify", "shared.s0.setup", workspace=ws, check=False)
        self.assertEqual(proc.returncode, 1)
        result = json.loads(proc.stdout)
        self.assertEqual(result["command"], "bash ./check.sh")
        self.assertFalse(result["passed"])
        (ws / "lessons/01-shared/02-setup/exercise/done.txt").write_text("x")
        result = self.sc.run_json("verify", "shared.s0.setup", workspace=ws)
        self.assertTrue(result["passed"])
        state = json.loads((ws / ".tutor/state.json").read_text())
        self.assertEqual(state["lessons"]["shared.s0.setup"]["attempts"], 2)
        result = self.sc.run_json("verify", "go.g1.chat", workspace=ws)
        self.assertEqual(result["verify"], "discussion")

    def test_verify_refuses_missing_exercise_and_missing_toolchain(self):
        shutil.rmtree(self.sc.content / "py/shared/s0/mixed")
        ws = self.sc.workspace("py")
        self.sc.run_json("init", "py", workspace=ws)
        proc = self.sc.run("verify", "shared.s0.mixed", workspace=ws, check=False)
        self.assertEqual(proc.returncode, 1)
        self.assertIn("content pending", proc.stderr)
        state = json.loads((ws / ".tutor/state.json").read_text())
        self.assertEqual(state["lessons"]["shared.s0.mixed"].get("attempts", 0), 0)
        env = dict(os.environ, PATH="/usr/bin:/bin")
        proc = self.sc.run("verify", "py.p1.hello", workspace=ws, check=False, env=env)
        self.assertEqual(proc.returncode, 1)
        self.assertIn("uv", proc.stderr)
        self.assertIn("astral.sh", proc.stderr)
        state = json.loads((ws / ".tutor/state.json").read_text())
        self.assertEqual(state["lessons"]["py.p1.hello"].get("attempts", 0), 0)

    def test_verify_resolves_runner_for_workspace_language(self):
        ws = self.sc.workspace("py")
        self.sc.run_json("init", "py", workspace=ws)
        proc = self.sc.run("verify", "shared.s0.mixed", workspace=ws, check=False)
        if proc.returncode == 1 and "not found on PATH" in proc.stderr:
            self.skipTest("uv not installed")
        result = json.loads(proc.stdout)
        self.assertTrue(result["command"].startswith("uv run"))
        self.assertEqual(result["verify"], "pytest")

    def test_ci_is_lesson_centric(self):
        proc = self.sc.run("ci", "--filter", "tool", check=False)
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("ok   go.g1.tool [go]", proc.stdout)
        self.assertIn("ok   py.p1.tool [py]", proc.stdout)
        self.assertIn("2 solution(s) tested, 0 failed", proc.stdout)
        proc = self.sc.run("ci", "--filter", "setup", check=False)
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("ok   shared.s0.setup [go]", proc.stdout)
        self.assertIn("shared.s0.setup [py]", proc.stdout)
        self.assertIn("no solution", proc.stdout)
        self.assertIn("1 solution(s) tested, 0 failed, 1 skipped", proc.stdout)
        proc = self.sc.run("ci", "--filter", "kit", "--language", "py", check=False)
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("ok   focus.kit.one [py]", proc.stdout)
        self.assertNotIn("[go]", proc.stdout)
        proc = self.sc.run("ci", "--filter", "nothing-matches", check=False)
        self.assertEqual(proc.returncode, 1)
        self.assertIn("nothing-matches", proc.stdout + proc.stderr)
        self.sc.write("go/g1/tool/solution/answer.txt", "41\n")
        proc = self.sc.run("ci", "--filter", "go.g1.tool", check=False)
        self.assertEqual(proc.returncode, 1)
        self.assertIn("FAIL go.g1.tool [go]", proc.stdout)

    def test_ci_reports_skips_with_reasons(self):
        self.sc.registry["languages"]["py"]["status"] = "stub"
        self.sc.registry["lessons"]["py.p1.new"] = lesson_meta("New", "pytest")
        self.sc.registry["stages"]["p1"]["lessons"].append("py.p1.new")
        self.sc.save_registry()
        shutil.rmtree(self.sc.content / "py/shared/s0/mixed")
        self.validate("--strict")
        proc = self.sc.run("ci", "--language", "py", "--filter", "mixed", check=False)
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn(
            "skip shared.s0.mixed [py]: no exercise variant for py", proc.stdout
        )
        self.assertIn("0 solution(s) tested, 0 failed, 1 skipped", proc.stdout)
        proc = self.sc.run(
            "ci", "--language", "py", "--filter", "py.p1.new", check=False
        )
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("skip py.p1.new [py]: no LESSON.md", proc.stdout)
        self.assertIn("0 solution(s) tested, 0 failed, 1 skipped", proc.stdout)

    def test_ci_tests_stub_language_content_on_disk(self):
        self.sc.registry["languages"]["py"]["status"] = "stub"
        self.sc.save_registry()
        self.validate("--strict")
        proc = self.sc.run("ci", "--language", "py", "--filter", "tool", check=False)
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("ok   py.p1.tool [py]", proc.stdout)
        proc = self.sc.run("init", "py", workspace=self.sc.workspace("py"), check=False)
        self.assertEqual(proc.returncode, 1)
        self.assertIn("stub", proc.stderr)

    @unittest.skipUnless(shutil.which("go"), "go toolchain not installed")
    def test_ci_runs_go_solutions(self):
        proc = self.sc.run("ci", "--language", "go", "--filter", "hello", check=False)
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("ok   go.g1.hello [go]", proc.stdout)
        self.assertIn("1 solution(s) tested, 0 failed", proc.stdout)

    @unittest.skipUnless(shutil.which("uv"), "uv not installed")
    def test_ci_runs_pytest_solutions_locked(self):
        exercise = self.sc.content / "py/p1/hello/exercise"
        (exercise / "uv.lock").unlink()
        lock = subprocess.run(
            ["uv", "lock", "-q"],
            cwd=exercise,
            capture_output=True,
            text=True,
            check=False,
        )
        if lock.returncode != 0:
            self.skipTest(f"uv lock failed: {lock.stderr[-300:]}")
        proc = self.sc.run(
            "ci", "--language", "py", "--filter", "py.p1.hello", check=False
        )
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("ok   py.p1.hello [py]", proc.stdout)
        self.assertIn("1 solution(s) tested, 0 failed", proc.stdout)
        self.assertFalse((exercise / ".venv").exists())
        ws = self.sc.workspace("py")
        self.sc.run_json("init", "py", workspace=ws)
        proc = self.sc.run("verify", "py.p1.hello", workspace=ws, check=False)
        self.assertEqual(proc.returncode, 1)
        self.assertFalse(json.loads(proc.stdout)["passed"])
        (ws / "lessons/02-py-basics/01-hello/exercise/x.py").write_text(PY_SOL)
        result = self.sc.run_json("verify", "py.p1.hello", workspace=ws)
        self.assertTrue(result["passed"])
        shared = self.sc.content / "py/shared/s0/mixed/exercise"
        (shared / "uv.lock").unlink()
        subprocess.run(["uv", "lock", "-q"], cwd=shared, check=True)
        self.sc.run_json("sync", workspace=ws)
        proc = self.sc.run("verify", "shared.s0.mixed", workspace=ws, check=False)
        self.assertEqual(proc.returncode, 1)
        self.assertEqual(json.loads(proc.stdout)["verify"], "pytest")
        (ws / "lessons/01-shared/03-mixed/exercise/x.py").write_text(PY_SOL)
        result = self.sc.run_json("verify", "shared.s0.mixed", workspace=ws)
        self.assertTrue(result["passed"])

    def test_validate_requires_pytest_project_files(self):
        (self.sc.content / "py/p1/hello/exercise/uv.lock").unlink()
        out = self.validate("--strict", expect_ok=False)
        self.assertIn("uv.lock", out)
        self.assertIn("py.p1.hello", out)

    def test_validate_anchor_and_layout_rules(self):
        self.sc.write(
            "shared/s0/mixed/LESSON.md",
            MIXED_LESSON + "\n<!-- lang: intro -->\n",
        )
        self.sc.write(
            "py/shared/s0/mixed/snippets/orphan.md", "**In Python:** nobody asked.\n"
        )
        self.sc.write("go/shared/s0/mixed/snippets/intro.md", GO_INTRO.rstrip("\n"))
        self.sc.write("shared/s0/talk/exercises/go/README.md", "old layout\n")
        self.sc.write(
            "py/shared/s0/nothing/snippets/intro.md", "**In Python:** ghost.\n"
        )
        out = self.validate(expect_ok=False)
        self.assertIn("duplicate anchor", out)
        self.assertIn("orphan", out)
        self.assertIn("trailing newline", out)
        self.assertIn("exercises/", out)
        self.assertIn("nothing", out)

    def test_validate_rejects_unexpected_overlay_entries(self):
        self.sc.write("py/shared/s0/mixed/LESSON.md", "# Python copy\n")
        self.sc.write("py/shared/s0/mixed/snippet/intro.md", PY_INTRO)
        self.sc.write("py/go/g1/hello/snippets/intro.md", PY_INTRO)
        self.sc.write("rust/shared/s0/mixed/snippets/intro.md", "**In Rust:** no.\n")
        out = self.validate(expect_ok=False)
        self.assertIn("py/shared/s0/mixed/: unexpected entry 'LESSON.md'", out)
        self.assertIn("py/shared/s0/mixed/: unexpected entry 'snippet'", out)
        self.assertIn("py/: unexpected entry 'go'", out)
        self.assertIn("content/: unexpected entry 'rust'", out)
        self.assertEqual(out.count("unexpected entry"), 4, out)

    def test_validate_soft_warnings_stay_warnings(self):
        (self.sc.content / "go/shared/s0/mixed/exercise/README.md").unlink()
        self.sc.write(
            "shared/s0/mixed/LESSON.md",
            MIXED_LESSON + "\nRun `go test ./...` from `exercise/`.\n",
        )
        out = self.validate("--strict")
        self.assertIn("README.md", out)
        self.assertIn("go test", out)
        self.assertIn("0 error(s)", out)

    def test_init_seeds_gitignore_and_reports_warnings(self):
        ws = self.sc.workspace("py")
        (ws.parent / "pyproject.toml").write_text("[project]\nname='outer'\n")
        report = self.sc.run_json("init", "py", "--focus", "empty", workspace=ws)
        self.assertTrue(any("empty" in w for w in report["warnings"]), report)
        self.assertTrue(any("pyproject.toml" in w for w in report["warnings"]), report)
        ignore = (ws / ".gitignore").read_text()
        self.assertIn(".venv/", ignore)
        self.assertIn("__pycache__/", ignore)
        (ws / ".gitignore").write_text("mine\n")
        self.sc.run_json("init", "py", workspace=ws)
        self.assertEqual((ws / ".gitignore").read_text(), "mine\n")
        go_ws = self.sc.workspace("go")
        report = self.sc.run_json("init", "go", workspace=go_ws)
        self.assertIn("*.test", (go_ws / ".gitignore").read_text())
        self.assertFalse(any("pyproject" in w for w in report.get("warnings", [])))
        status = self.sc.run_json("status", "--json", workspace=ws)
        self.assertTrue(any("pyproject.toml" in w for w in status["warnings"]))

    def test_carry_over_copies_grades_into_notes(self):
        go_ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=go_ws)
        self.sc.run_json(
            "mark", "shared.s0.talk", "passed", "--grade", "A", workspace=go_ws
        )
        self.sc.run_json("mark", "shared.s0.setup", "passed", workspace=go_ws)
        self.sc.run_json(
            "mark", "go.g1.hello", "passed", "--grade", "B", workspace=go_ws
        )
        py_ws = self.sc.workspace("py")
        report = self.sc.run_json(
            "init", "py", "--carry-over", str(go_ws), workspace=py_ws
        )
        self.assertEqual(
            sorted(report["carried_over"]), ["shared.s0.setup", "shared.s0.talk"]
        )
        state = json.loads((py_ws / ".tutor/state.json").read_text())
        talk = state["lessons"]["shared.s0.talk"]
        self.assertEqual(talk["status"], "todo")
        self.assertIn("carried over from go", talk["notes"])
        self.assertIn("(A)", talk["notes"])
        report = self.sc.run_json(
            "init", "py", "--carry-over", str(go_ws), workspace=py_ws
        )
        self.assertEqual(report["carried_over"], [])
        proc = self.sc.run(
            "init",
            "go",
            "--carry-over",
            str(go_ws),
            workspace=self.sc.workspace("go2"),
            check=False,
        )
        self.assertEqual(proc.returncode, 1)

    def test_old_manifest_nags_once(self):
        ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=ws)
        manifest_path = ws / ".tutor/manifest.json"
        manifest = json.loads(manifest_path.read_text())
        del manifest["composition_hash"]
        manifest["registry_hash"] = "deadbeef"
        manifest_path.write_text(json.dumps(manifest))
        status = self.sc.run_json("status", "--json", workspace=ws)
        self.assertTrue(status["sync_needed"], status)
        self.assertEqual([k for k, v in status["pending"].items() if v], [])
        report = self.sc.run_json("sync", workspace=ws)
        self.assertEqual(
            [k for k in report if isinstance(report[k], list) and report[k]], []
        )
        manifest = json.loads(manifest_path.read_text())
        self.assertIn("composition_hash", manifest)
        self.assertNotIn("registry_hash", manifest)
        status = self.sc.run_json("status", "--json", workspace=ws)
        self.assertFalse(status["sync_needed"], status)

    def test_snippet_change_triggers_needs_review(self):
        ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=ws)
        self.sc.run_json("mark", "shared.s0.mixed", "passed", workspace=ws)
        (self.sc.content / "go/shared/s0/mixed/snippets/intro.md").unlink()
        report = self.sc.run_json("sync", workspace=ws)
        self.assertEqual(report["updated"], ["shared.s0.mixed:LESSON.md"])
        self.assertEqual(report["needs_review"], ["shared.s0.mixed"])
        rendered = (ws / "lessons/01-shared/03-mixed/LESSON.md").read_text()
        self.assertNotIn("In Go:", rendered)
        self.assertIn("Neutral paragraph one.\n\nNeutral paragraph two.", rendered)

    def test_sync_quadrants_still_hold(self):
        ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=ws)
        self.sc.run_json("mark", "shared.s0.talk", "passed", workspace=ws)
        self.sc.run_json("mark", "go.g1.hello", "passed", workspace=ws)
        talk = ws / "lessons/01-shared/01-talk/LESSON.md"
        hello = ws / "lessons/02-go-basics/01-hello/exercise/x.go"
        hello.write_text("package x\n// learner edit\n")
        self.sc.write("shared/s0/talk/LESSON.md", "# Talk v2\n")
        self.sc.write("go/g1/hello/exercise/x.go", "package x\n// upstream edit\n")
        report = self.sc.run_json("sync", workspace=ws)
        self.assertIn("shared.s0.talk:LESSON.md", report["updated"])
        self.assertEqual(talk.read_text(), "# Talk v2\n")
        self.assertEqual(
            report["conflicts"],
            [
                {
                    "lesson": "go.g1.hello",
                    "file": "exercise/x.go",
                    "sidecar": "lessons/02-go-basics/01-hello/exercise/x.go.upstream",
                }
            ],
        )
        self.assertEqual(hello.read_text(), "package x\n// learner edit\n")
        self.assertEqual(
            sorted(report["needs_review"]), ["go.g1.hello", "shared.s0.talk"]
        )
        report = self.sc.run_json("sync", workspace=ws)
        self.assertEqual(
            [k for k in report if isinstance(report[k], list) and report[k]], []
        )
        self.sc.registry["stages"]["g1"]["lessons"].remove("go.g1.chat")
        self.sc.save_registry()
        report = self.sc.run_json("sync", workspace=ws)
        self.assertEqual(report["removed"], ["go.g1.chat"])
        self.assertTrue((ws / ".tutor/attic/02-chat").exists())
        self.assertEqual(report["renamed"][0]["lesson"], "go.g1.tool")

    def test_carry_over_is_validated_before_anything_is_scaffolded(self):
        ws = self.sc.workspace("py")
        proc = self.sc.run(
            "init",
            "py",
            "--carry-over",
            str(ws.parent / "nowhere"),
            workspace=ws,
            check=False,
        )
        self.assertEqual(proc.returncode, 1)
        self.assertIn("no tutor workspace", proc.stderr)
        self.assertFalse((ws / ".tutor").exists())
        self.assertFalse((ws / "lessons").exists())

    def test_verify_distinguishes_off_path_from_pending(self):
        ws = self.sc.workspace("go")
        self.sc.run_json("init", "go", workspace=ws)
        proc = self.sc.run("verify", "py.p1.hello", workspace=ws, check=False)
        self.assertEqual(proc.returncode, 1)
        self.assertIn("not on the go path", proc.stderr)
        self.assertNotIn("pending", proc.stderr)

    def test_validate_reports_unknown_stage_without_traceback(self):
        self.sc.registry["languages"]["py"]["path"].append("nope")
        self.sc.save_registry()
        proc = self.sc.run("validate", check=False)
        self.assertEqual(proc.returncode, 1)
        self.assertIn("unknown stage 'nope'", proc.stdout)
        self.assertNotIn("Traceback", proc.stderr)

    def test_validate_rejects_anchors_in_fences_and_malformed_anchors(self):
        self.sc.write(
            "shared/s0/mixed/LESSON.md",
            MIXED_LESSON
            + "\n```go\n<!-- lang: fenced -->\n```\n\n<!-- lang: loose -->  \n",
        )
        self.sc.write("shared/s0/mixed/TUTOR.md", "# notes\n<!-- lang: intro -->\n")
        out = self.validate(expect_ok=False)
        self.assertIn("fenced", out)
        self.assertIn("inside a fenced code block", out)
        self.assertIn("malformed anchor", out)
        self.assertIn("TUTOR.md carries a language anchor", out)

    def test_lesson_that_loses_its_variant_is_parked(self):
        ws = self.sc.workspace("py")
        self.sc.run_json("init", "py", workspace=ws)
        self.assertTrue((ws / "lessons/01-shared/03-mixed/exercise/x.py").exists())
        shutil.rmtree(self.sc.content / "py/shared/s0/mixed/exercise")
        report = self.sc.run_json("sync", workspace=ws)
        self.assertIn("shared.s0.mixed", report["pending_content"])
        self.assertIn("shared.s0.mixed", report["removed"])
        self.assertFalse((ws / "lessons/01-shared/03-mixed").exists())
        self.assertTrue((ws / ".tutor/attic/03-mixed/exercise/x.py").exists())

    def test_ci_ignores_junk_inside_solutions(self):
        self.sc.write("go/g1/tool/solution/__pycache__/x.pyc", "junk")
        self.sc.write("go/g1/tool/solution/.DS_Store", "junk")
        proc = self.sc.run("ci", "--filter", "go.g1.tool", check=False)
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        shutil.rmtree(self.sc.content / "go/g1/tool/solution")
        self.sc.write("go/g1/tool/solution/.DS_Store", "junk")
        proc = self.sc.run("ci", "--filter", "go.g1.tool", check=False)
        self.assertIn("no solution", proc.stdout)


if __name__ == "__main__":
    unittest.main()
