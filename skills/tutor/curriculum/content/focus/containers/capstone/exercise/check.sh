#!/usr/bin/env bash
# Referee for the Containers Capstone.
#
# Every graded check is STATIC: it reads the files you wrote — the Dockerfile,
# the workflow, the manifests and overlays, and RUNBOOK.md. No cluster, no
# kubectl, no helm, no kustomize, no Docker daemon, no network, no root.
#
# If any of that tooling does happen to be installed here, extra checks run as
# a bonus. When it is missing, or a cluster or registry is unreachable, those
# checks are reported SKIPPED and never failed: your grade already stands.
#
# Run from this folder:  bash check.sh
set -u
cd "$(dirname "$0")" || exit 1

results="$(mktemp)"
trap 'rm -f "$results"' EXIT

record() { printf '%s|%s|%s\n' "$1" "$2" "${3:-}" >>"$results"; }

# One line, no pipes: the report loop splits records on '|'.
flatten() { printf '%s' "$1" | tr '\n|' '  ' | cut -c1-220; }

# ---------------------------------------------------------------- static
# python3 is guaranteed here (the tutor engine itself is Python). PyYAML is
# not, so the grader carries the same small YAML reader the earlier lessons of
# this pack used, covering the subset these files need.

python3 - >>"$results" <<'PY'
import math
import os
import re
import sys

OUT = []


def rec(kind, msg, fix=""):
    OUT.append("%s|%s|%s" % (kind, msg, fix))


class YamlError(Exception):
    pass


def strip_comment(line):
    out, quote = [], None
    for ch in line:
        if quote:
            out.append(ch)
            if ch == quote:
                quote = None
        elif ch in "\"'":
            quote = ch
            out.append(ch)
        elif ch == "#" and (not out or out[-1] in " \t"):
            break
        else:
            out.append(ch)
    return "".join(out).rstrip()


def tokenize(text, first_line):
    toks = []
    for n, raw in enumerate(text.splitlines(), first_line):
        lead = raw[: len(raw) - len(raw.lstrip())]
        if "\t" in lead:
            raise YamlError("line %d uses a tab to indent; YAML allows spaces only" % n)
        line = strip_comment(raw)
        if not line.strip() or line.strip() == "...":
            continue
        toks.append((len(line) - len(line.lstrip(" ")), line.strip(), n))
    return toks


def unquote(s):
    s = s.strip()
    if len(s) >= 2 and s[0] == s[-1] and s[0] in "\"'":
        return s[1:-1]
    return s


def split_top(s, sep):
    parts, buf, depth, quote = [], [], 0, None
    for ch in s:
        if quote:
            buf.append(ch)
            if ch == quote:
                quote = None
            continue
        if ch in "\"'":
            quote = ch
            buf.append(ch)
        elif ch in "[{":
            depth += 1
            buf.append(ch)
        elif ch in "]}":
            depth = max(0, depth - 1)
            buf.append(ch)
        elif ch == sep and depth == 0:
            parts.append("".join(buf))
            buf = []
        else:
            buf.append(ch)
    parts.append("".join(buf))
    return parts


def flow(s, line):
    s = s.strip()
    if s.startswith("["):
        if not s.endswith("]"):
            raise YamlError("line %d: unterminated [ ... ] list" % line)
        inner = s[1:-1].strip()
        return [] if not inner else [flow(p, line) for p in split_top(inner, ",")]
    if s.startswith("{"):
        if not s.endswith("}"):
            raise YamlError("line %d: unterminated { ... } mapping" % line)
        inner = s[1:-1].strip()
        out = {}
        for part in [p for p in split_top(inner, ",") if p.strip()]:
            k, sep, v = part.partition(":")
            if not sep:
                raise YamlError("line %d: expected 'key: value' inside { }" % line)
            out[unquote(k)] = flow(v, line)
        return out
    if s.startswith("&") or s.startswith("*") or s.startswith("<<"):
        raise YamlError(
            "line %d: anchors, aliases and merge keys are outside the subset "
            "this grader reads — write the value out" % line
        )
    return unquote(s)


MAP_ENTRY = re.compile(r"^(\"[^\"]*\"|'[^']*'|[^:\s]+)\s*:(\s|$)")


class Parser:
    def __init__(self, toks):
        self.toks, self.i = toks, 0

    def peek(self):
        return self.toks[self.i] if self.i < len(self.toks) else None

    def block(self, indent):
        tok = self.peek()
        if tok is None or tok[0] < indent:
            return None
        if tok[1] == "-" or tok[1].startswith("- "):
            return self.sequence(tok[0])
        return self.mapping(tok[0])

    def mapping(self, indent):
        out = {}
        while True:
            tok = self.peek()
            if tok is None or tok[0] < indent:
                break
            if tok[0] > indent:
                raise YamlError("line %d: unexpected indentation" % tok[2])
            if not MAP_ENTRY.match(tok[1]):
                raise YamlError("line %d: expected 'key:' or 'key: value'" % tok[2])
            key, _, rest = tok[1].partition(":")
            key, rest = unquote(key), rest.strip()
            self.i += 1
            if rest.startswith("|") or rest.startswith(">"):
                lines = []
                while True:
                    nxt = self.peek()
                    if nxt is None or nxt[0] <= indent:
                        break
                    lines.append(nxt[1])
                    self.i += 1
                out[key] = "\n".join(lines)
            elif rest:
                out[key] = flow(rest, tok[2])
            else:
                nxt = self.peek()
                is_item = nxt is not None and (nxt[1] == "-" or nxt[1].startswith("- "))
                if nxt is not None and nxt[0] > indent:
                    out[key] = self.block(nxt[0])
                elif is_item and nxt[0] == indent:
                    # A sequence may sit at its key's own indent — the style
                    # every kubectl example uses.
                    out[key] = self.sequence(indent)
                else:
                    out[key] = None
        return out

    def sequence(self, indent):
        out = []
        while True:
            tok = self.peek()
            if tok is None or tok[0] < indent:
                break
            if not (tok[1] == "-" or tok[1].startswith("- ")):
                break
            if tok[0] > indent:
                raise YamlError("line %d: unexpected indentation" % tok[2])
            after = tok[1][1:]
            item = after.strip()
            item_col = tok[0] + 1 + (len(after) - len(after.lstrip(" ")))
            self.i += 1
            sub = []
            while True:
                nxt = self.peek()
                if nxt is None or nxt[0] <= indent:
                    break
                sub.append(nxt)
                self.i += 1
            if not item:
                out.append(Parser(sub).block(sub[0][0]) if sub else None)
            elif MAP_ENTRY.match(item):
                inner = [(item_col, item, tok[2])] + sub
                out.append(Parser(inner).block(item_col))
            else:
                out.append(flow(item, tok[2]))
        return out


def parse_documents(text):
    """Split on lines that are exactly '---' and parse each document."""
    docs, chunk, start, line_no = [], [], 1, 0
    for line_no, raw in enumerate(text.splitlines(), 1):
        if raw.strip() == "---":
            docs.append(("\n".join(chunk), start))
            chunk, start = [], line_no + 1
        else:
            chunk.append(raw)
    docs.append(("\n".join(chunk), start))

    out = []
    for body, first in docs:
        toks = tokenize(body, first)
        if not toks:
            continue
        doc = Parser(toks).block(toks[0][0])
        if not isinstance(doc, dict):
            raise YamlError("line %d: every document must be a mapping" % first)
        out.append(doc)
    return out


# ---------------------------------------------------------------- helpers


def as_dict(v):
    return v if isinstance(v, dict) else {}


def as_list(v):
    if v is None:
        return []
    return v if isinstance(v, list) else [v]


def dig(node, *path):
    cur = node
    for key in path:
        if not isinstance(cur, dict):
            return None
        cur = cur.get(key)
    return cur


def read(path):
    try:
        return open(path, encoding="utf-8", errors="replace").read()
    except OSError:
        return None


SUFFIX = {
    "": 1.0, "m": 0.001, "k": 1e3, "M": 1e6, "G": 1e9, "T": 1e12, "P": 1e15,
    "Ki": 1024.0, "Mi": 1024.0 ** 2, "Gi": 1024.0 ** 3, "Ti": 1024.0 ** 4,
}


def quantity(v):
    """Kubernetes resource quantity -> float (cores, or bytes)."""
    if v is None:
        return None
    m = re.match(r"^([0-9]*\.?[0-9]+)([A-Za-z]*)$", str(v).strip())
    if not m or m.group(2) not in SUFFIX:
        return None
    return float(m.group(1)) * SUFFIX[m.group(2)]


DUR_UNITS = {"ns": 1e-9, "us": 1e-6, "µs": 1e-6, "ms": 1e-3, "s": 1.0, "m": 60.0, "h": 3600.0}
DUR_PART = re.compile(r"^([0-9]+(?:\.[0-9]+)?)(ns|us|µs|ms|s|m|h)")


def go_duration(v):
    """Go duration string -> seconds, or None if it is not one."""
    s = str(v or "").strip()
    if not s:
        return None
    total, i = 0.0, 0
    while i < len(s):
        m = DUR_PART.match(s[i:])
        if not m:
            return None
        total += float(m.group(1)) * DUR_UNITS[m.group(2)]
        i += m.end()
    return total


def integer(v):
    try:
        return int(str(v).strip())
    except (TypeError, ValueError):
        return None


GOMEMLIMIT_UNITS = {"": 1, "B": 1, "KiB": 1024, "MiB": 1024 ** 2, "GiB": 1024 ** 3, "TiB": 1024 ** 4}


def gomemlimit_bytes(v):
    m = re.match(r"^([0-9]+)(B|KiB|MiB|GiB|TiB)?$", str(v or "").strip())
    if not m:
        return None
    return int(m.group(1)) * GOMEMLIMIT_UNITS[m.group(2) or ""]


def human(n):
    return "%.0f MiB" % (n / 1024.0 / 1024.0)


def image_tag(ref):
    """('repository', 'tag or digest') for an image reference."""
    ref = str(ref or "")
    if "@" in ref:
        repo, _, digest = ref.partition("@")
        return repo, digest
    last = ref.rsplit("/", 1)[-1]
    if ":" in last:
        return ref.rsplit(":", 1)[0], ref.rsplit(":", 1)[1]
    return ref, ""


def immutable_tag(tag):
    """A tag nobody can move under you: a digest, a sha- tag, or a version."""
    tag = str(tag or "")
    if tag.startswith("sha256:"):
        return True
    if re.match(r"^(sha-|git-|commit-)?[0-9a-f]{7,40}$", tag):
        return True
    if re.match(r"^v?[0-9]+\.[0-9]+\.[0-9]+", tag):
        return True
    return False


# ---------------------------------------------------------------- Dockerfile


def dockerfile_instructions(text):
    """One instruction per entry, continuations joined, comments dropped."""
    out, buf = [], ""
    for raw in text.splitlines():
        line = raw.rstrip("\r").rstrip()
        if not buf and re.match(r"^\s*#", line):
            continue
        line = line.strip()
        if buf:
            line = buf + " " + line
            buf = ""
        if line.endswith("\\"):
            buf = line[:-1].rstrip()
            continue
        if line:
            out.append(line)
    if buf:
        out.append(buf)
    return out


def base_of(from_line):
    for tok in from_line.split()[1:]:
        if tok.startswith("--"):
            continue
        return tok.lower()
    return ""


def check_dockerfile():
    """Returns the name of the build ARG that carries the commit, or ''."""
    text = read("Dockerfile")
    if text is None:
        rec("FAIL", "the image is multi-stage, pinned, and ships only the binary",
            "there is no Dockerfile next to check.sh — bring the one from dockerizing-go and add "
            "what a pipeline needs from it")
        return ""
    lines = dockerfile_instructions(text)
    froms = [(i, ln) for i, ln in enumerate(lines) if ln.upper().startswith("FROM ")]
    if len(froms) < 2:
        rec("FAIL", "the image is multi-stage, pinned, and ships only the binary",
            "only %d FROM line(s): compile in a build stage and copy the binary into a small runtime "
            "stage, or you ship the Go toolchain to production" % len(froms))
        return ""
    last = froms[-1][0]
    build, final = lines[:last], lines[last:]
    final_base = base_of(lines[last])

    problems = []
    for _, ln in froms:
        base = base_of(ln)
        tag = base.rsplit(":", 1)[1] if ":" in base.rsplit("/", 1)[-1] else ""
        if base == "scratch" or "@sha256:" in base:
            continue
        if not tag or tag == "latest":
            problems.append("base image %r pins nothing, and an unpinned base means a build you "
                            "cannot reproduce" % base)
    if "golang" in final_base:
        problems.append("the final stage starts from %r, which ships the compiler and stdlib "
                        "source to production" % final_base)
    if not any(re.match(r"(?i)^COPY\s.*--from=", ln) for ln in final):
        problems.append("the final stage never does COPY --from=, so it never receives the binary")
    if any(re.search(r"(?i)(^|[\s;&|(])go\s+(build|install|run|mod)(\s|$)", ln) for ln in final):
        problems.append("the final stage runs the go tool, which means the toolchain is in the image")
    if not any(re.search(r"CGO_ENABLED\s*=\s*0", ln) for ln in build):
        problems.append("the build stage does not set CGO_ENABLED=0, so the binary may need a "
                        "dynamic loader the runtime base does not have")
    if not any(re.match(r"(?i)^(ENTRYPOINT|CMD)\s*\[", ln) for ln in final):
        problems.append("the start command is not in exec form, so PID 1 is a shell that swallows "
                        "SIGTERM")
    if problems:
        rec("FAIL", "the image is multi-stage, pinned, and ships only the binary",
            "; ".join(problems))
    else:
        rec("PASS", "the image is multi-stage, pinned, and ships only the binary (%s)" % final_base)

    user = [ln for ln in final if re.match(r"(?i)^USER\s", ln)]
    val = user[-1].split()[1] if user else ""
    if not user:
        rec("FAIL", "the final stage runs as a non-root user",
            "add a USER instruction to the final stage, e.g. USER 65532:65532. With no USER the "
            "process is uid 0 inside the container, and a container escape starts as root")
    elif val == "root" or val.split(":")[0] == "0":
        rec("FAIL", "the final stage runs as a non-root user",
            "USER %s is root. Pick an unprivileged uid, e.g. USER 65532:65532" % val)
    elif final_base == "scratch" and not re.match(r"^[0-9]+(:[0-9]+)?$", val):
        rec("FAIL", "the final stage runs as a non-root user",
            "on scratch there is no /etc/passwd, so the name %r cannot be resolved and the container "
            "fails to start. Use a numeric uid" % val)
    else:
        rec("PASS", "the final stage runs as a non-root user (USER %s)" % val)

    args = []
    for ln in lines:
        m = re.match(r"(?i)^ARG\s+([A-Za-z_][A-Za-z0-9_]*)", ln)
        if m:
            args.append(m.group(1))
    rev_args = [a for a in args if re.search(r"(?i)rev|version|commit|sha", a)]
    build_cmds = [ln for ln in build if re.search(r"(?i)go\s+build", ln)]
    ldflags = " ".join(ln for ln in build_cmds if "-ldflags" in ln)
    stamped = re.search(r"-X\s*['\"]?\s*main\.buildRevision=", ldflags)
    arg_name = rev_args[0] if rev_args else ""
    if not rev_args:
        rec("FAIL", "the build stamps the commit into the binary",
            "no ARG carries the commit. Declare one (ARG REVISION) and pass it at build time; a "
            "binary that cannot say which commit it is leaves you diffing images to find out")
    elif not build_cmds:
        rec("FAIL", "the build stamps the commit into the binary",
            "no go build command found in the build stage")
    elif not stamped:
        rec("FAIL", "the build stamps the commit into the binary",
            "go build never sets main.buildRevision. Add it to the linker flags, next to -s -w: "
            "-ldflags=\"-s -w -X main.buildRevision=$%s\" — main.go reports it on GET /version"
            % arg_name)
    elif not re.search(r"\$\{?%s\b" % re.escape(arg_name), ldflags):
        rec("FAIL", "the build stamps the commit into the binary",
            "main.buildRevision is set to a fixed string instead of the %s build argument, so every "
            "image claims the same revision" % arg_name)
    else:
        rec("PASS", "the build stamps the commit into the binary (-X main.buildRevision=$%s)" % arg_name)

    labels = " ".join(ln for ln in final if re.match(r"(?i)^LABEL\s", ln))
    if "org.opencontainers.image.revision" not in labels:
        rec("FAIL", "the image is labelled with the commit it was built from",
            "add LABEL org.opencontainers.image.revision=$%s to the final stage — remember ARG does "
            "not cross a FROM, so the final stage has to declare the ARG again. The label is how "
            "docker inspect answers 'what is this image?' without running it"
            % (arg_name or "REVISION"))
    elif arg_name and not re.search(r"\$\{?%s\b" % re.escape(arg_name), labels):
        rec("FAIL", "the image is labelled with the commit it was built from",
            "the revision label is a fixed string, not the %s build argument" % arg_name)
    else:
        rec("PASS", "the image is labelled with the commit it was built from")
        if "org.opencontainers.image.source" not in labels:
            rec("NOTE", "consider LABEL org.opencontainers.image.source too",
                "it points at the repository the image was built from, which is what makes an image "
                "in a registry traceable by someone who did not build it (not graded)")
    return arg_name


# ---------------------------------------------------------------- workflow

WORKFLOW_DIR = os.path.join(".github", "workflows")


def step_text(step):
    """Everything a step says, flattened — uses, run, with values, if."""
    step = as_dict(step)
    bits = [str(step.get("uses") or ""), str(step.get("run") or ""), str(step.get("if") or "")]
    for k, v in as_dict(step.get("with")).items():
        bits.append("%s=%s" % (k, v))
    for k, v in as_dict(step.get("env")).items():
        bits.append("%s=%s" % (k, v))
    return "\n".join(b for b in bits if b)


def job_text(job):
    return "\n".join(step_text(s) for s in as_list(as_dict(job).get("steps")))


def needs_of(job):
    return [str(n) for n in as_list(as_dict(job).get("needs")) if n]


def reaches(jobs, start, target, seen=None):
    """Does job `start` depend, directly or transitively, on `target`?"""
    if seen is None:
        seen = set()
    for dep in needs_of(jobs.get(start, {})):
        if dep == target:
            return True
        if dep not in seen:
            seen.add(dep)
            if reaches(jobs, dep, target, seen):
                return True
    return False


PUSH_RE = re.compile(r"docker/build-push-action|docker\s+push|buildx\s+build[^\n]*--push")
NO_PUSH_JOB = (
    "no job in this workflow publishes an image. The build job has to push what it built — "
    "docker/build-push-action with push: true, or docker push — or nothing downstream has an "
    "artifact to deploy"
)
LOGIN_RE = re.compile(
    r"docker/login-action|docker\s+login|aws-actions/configure-aws-credentials|"
    r"google-github-actions/auth|azure/login"
)


def check_workflow(arg_name):
    if not os.path.isdir(WORKFLOW_DIR):
        rec("FAIL", "the release workflow parses",
            "there is no .github/workflows/ directory here — the pipeline is a file in the "
            "repository, not a dashboard setting")
        return
    files = sorted(
        os.path.join(WORKFLOW_DIR, f)
        for f in os.listdir(WORKFLOW_DIR)
        if f.endswith((".yml", ".yaml"))
    )
    workflows = []
    for path in files:
        try:
            docs = parse_documents(read(path))
        except YamlError as e:
            rec("FAIL", "the release workflow parses",
                "%s, %s. Indent with spaces only (never a tab, not even inside a run: block), two "
                "per level, a space after every colon and after every '- '" % (path, e))
            return
        for doc in docs:
            workflows.append((path, doc))
    candidates = [(p, w) for p, w in workflows if PUSH_RE.search(str(read(p)))]
    if not workflows:
        rec("FAIL", "the release workflow parses",
            "no workflow file under .github/workflows/ — write release.yml")
        return
    path, wf = (candidates or workflows)[0]
    rec("PASS", "the release workflow parses (%s)" % path)

    jobs = as_dict(wf.get("jobs"))
    if not jobs:
        rec("FAIL", "the workflow runs on merges to main and on demand",
            "the workflow declares no jobs")
        return

    on = wf.get("on")
    on_map = as_dict(on)
    on_keys = set(on_map) if on_map else set(str(x) for x in as_list(on))
    branches = [str(b) for b in as_list(dig(on_map, "push", "branches"))]
    tags = as_list(dig(on_map, "push", "tags"))
    problems = []
    if "push" not in on_keys:
        problems.append("nothing triggers on push, so merging to main releases nothing")
    elif branches and "main" not in branches and not tags:
        problems.append("push triggers on %s, not on main" % ", ".join(branches))
    if "workflow_dispatch" not in on_keys:
        problems.append("no workflow_dispatch trigger: during an incident you want to re-run this "
                        "release by hand, from the last good commit, without pushing anything")
    if problems:
        rec("FAIL", "the workflow runs on merges to main and on demand", "; ".join(problems))
    else:
        rec("PASS", "the workflow runs on merges to main and on demand")

    texts = {name: job_text(job) for name, job in jobs.items()}
    build_name = next((n for n, t in texts.items() if PUSH_RE.search(t)), "")
    test_name = next((n for n, t in texts.items() if re.search(r"go\s+test", t)), "")
    dev_name = next((n for n, t in texts.items() if "overlays/dev" in t), "")
    prod_name = next((n for n, t in texts.items() if "overlays/prod" in t), "")
    build = as_dict(jobs.get(build_name))

    if not test_name:
        rec("FAIL", "the tests gate everything downstream",
            "no job runs go test. A pipeline that publishes without testing is an automated way to "
            "ship a bad build faster than you could by hand")
    elif not build_name:
        rec("FAIL", "the tests gate everything downstream", NO_PUSH_JOB)
    elif not reaches(jobs, build_name, test_name):
        rec("FAIL", "the tests gate everything downstream",
            "job %r does not need %r, so both start at once and an image is published while the "
            "tests are still running. Jobs run in parallel unless needs: says otherwise"
            % (build_name, test_name))
    else:
        rec("PASS", "the tests gate everything downstream (%s needs %s)" % (build_name, test_name))

    perms = as_dict(build.get("permissions"))
    perm_text = " ".join("%s:%s" % (k, v) for k, v in perms.items())
    login = LOGIN_RE.search(texts.get(build_name, ""))
    if not build_name:
        rec("FAIL", "the publishing job asks for exactly the credentials it needs", NO_PUSH_JOB)
    elif not login:
        rec("FAIL", "the publishing job asks for exactly the credentials it needs",
            "job %r pushes but never logs in. Use docker/login-action against ghcr.io with "
            "${{ github.token }} — the token GitHub mints for this run and expires when it ends"
            % build_name)
    elif not perms:
        rec("FAIL", "the publishing job asks for exactly the credentials it needs",
            "job %r declares no permissions:, so it gets the workflow default — read-only here, and "
            "the push will fail with a 403. Grant it packages: write (plus contents: read), or "
            "id-token: write if you authenticate to a cloud registry with OIDC" % build_name)
    elif "packages:write" not in perm_text.replace(" ", "") and \
            "id-token:write" not in perm_text.replace(" ", ""):
        rec("FAIL", "the publishing job asks for exactly the credentials it needs",
            "job %r has permissions %s — publishing to GHCR needs packages: write, and OIDC to a "
            "cloud registry needs id-token: write" % (build_name, perms))
    else:
        rec("PASS", "the publishing job asks for exactly the credentials it needs (%s)" % perms)
        if as_dict(wf.get("permissions")).get("packages") == "write":
            rec("NOTE", "the workflow-level permissions also grant packages: write",
                "every job in the file then gets a token that can publish. Grant it on the one job "
                "that publishes instead (not graded)")

    build_bits = texts.get(build_name, "")
    tag_values = []
    for step in as_list(build.get("steps")):
        with_ = as_dict(as_dict(step).get("with"))
        if with_.get("tags"):
            tag_values.append(str(with_["tags"]))
        run = str(as_dict(step).get("run") or "")
        tag_values.extend(re.findall(r"-t\s+(\S+)", run))
        tag_values.extend(re.findall(r"docker\s+push\s+(\S+)", run))
    tag_blob = "\n".join(tag_values)
    metadata_sha = "docker/metadata-action" in build_bits and "type=sha" in build_bits
    floating = [t for t in tag_blob.split() if t.endswith(":latest")]
    if not build_name:
        rec("FAIL", "the image is tagged with the commit, not with a moving name", NO_PUSH_JOB)
    elif not tag_blob and not metadata_sha:
        rec("FAIL", "the image is tagged with the commit, not with a moving name",
            "the build step names no tags. Tag the image ${{ env.IMAGE }}:sha-${{ github.sha }} so "
            "the artifact and the commit are the same fact")
    elif not ("github.sha" in tag_blob or metadata_sha):
        rec("FAIL", "the image is tagged with the commit, not with a moving name",
            "the tags (%s) contain nothing derived from github.sha. A tag that can be moved makes "
            "every rollback a guess: the manifest still says the same string, so nothing changes "
            "when you re-apply it" % flat(tag_blob))
    elif floating and len(set(tag_blob.split())) == len(floating):
        rec("FAIL", "the image is tagged with the commit, not with a moving name",
            "%s is the only tag pushed. Push the immutable one too, and deploy that one" % floating[0])
    else:
        rec("PASS", "the image is tagged with the commit, not with a moving name")

    if not arg_name:
        rec("FAIL", "the build passes the commit into the image",
            "the Dockerfile declares no build argument for the commit yet, so there is nothing for "
            "the workflow to pass")
    elif re.search(r"%s\s*[:=]\s*[^\n]{0,40}github\.sha" % re.escape(arg_name), build_bits):
        rec("PASS", "the build passes the commit into the image (%s from github.sha)" % arg_name)
    else:
        rec("FAIL", "the build passes the commit into the image",
            "the build step never sets %s from github.sha. With docker/build-push-action that is "
            "build-args: %s=${{ github.sha }}; with the CLI it is --build-arg %s=${{ github.sha }}"
            % (arg_name, arg_name, arg_name))

    deploys = [("dev", dev_name), ("prod", prod_name)]
    missing = [env for env, name in deploys if not name]
    if missing:
        rec("FAIL", "each environment is deployed by naming the artifact this run produced",
            "no job deploys the %s overlay. Both environments are promoted through "
            "deploy/overlays/<env>, so each needs a job that applies one" % ", ".join(missing))
    else:
        problems = []
        for env, name in deploys:
            setter = [
                ln
                for ln in texts[name].splitlines()
                if re.search(r"kustomize\s+edit\s+set\s+image|kubectl\s+set\s+image|helm\s+upgrade", ln)
            ]
            if not setter:
                problems.append("job %r applies the %s overlay but never points it at this run's "
                                "image, so it deploys whatever tag is committed" % (name, env))
                continue
            line = " ".join(setter)
            if ":latest" in line:
                problems.append("job %r deploys :latest — the one tag that makes a rollback a no-op"
                                % name)
            elif not re.search(r"needs\.[A-Za-z0-9_-]+\.outputs|github\.sha", line):
                problems.append("job %r sets an image that is not this run's artifact: name it with "
                                "github.sha, or better with the digest the build job output" % name)
        if problems:
            rec("FAIL", "each environment is deployed by naming the artifact this run produced",
                "; ".join(problems))
        else:
            rec("PASS", "each environment is deployed by naming the artifact this run produced")

    if not (dev_name and prod_name and build_name):
        rec("FAIL", "dev is deployed before prod, and prod is gated",
            "this check needs a build job and a deploy job per environment")
    elif not reaches(jobs, dev_name, build_name):
        rec("FAIL", "dev is deployed before prod, and prod is gated",
            "job %r does not need %r, so dev can be deployed before the image it deploys exists"
            % (dev_name, build_name))
    elif not reaches(jobs, prod_name, dev_name):
        rec("FAIL", "dev is deployed before prod, and prod is gated",
            "job %r does not need %r. Promotion means the same artifact passes dev first; without "
            "the dependency both environments get it at the same moment and dev tests nothing"
            % (prod_name, dev_name))
    elif not as_dict(jobs.get(prod_name)).get("environment"):
        rec("FAIL", "dev is deployed before prod, and prod is gated",
            "job %r names no environment:. A GitHub environment is where a required reviewer and "
            "the production secrets live — it is the difference between a pipeline that can deploy "
            "to prod and one that may" % prod_name)
    else:
        rec("PASS", "dev is deployed before prod, and prod is gated (%s -> %s)" % (dev_name, prod_name))

    if not (dev_name and prod_name):
        rec("FAIL", "every deploy waits for the rollout and has an undo",
            "this check needs a deploy job per environment")
    else:
        problems = []
        for env, name in deploys:
            steps = as_list(as_dict(jobs.get(name)).get("steps"))
            text = texts[name]
            waited = re.search(r"rollout\s+status", text) or re.search(r"helm\s+upgrade[^\n]*--wait", text)
            undo = False
            for step in steps:
                cond = str(as_dict(step).get("if") or "")
                run = str(as_dict(step).get("run") or "")
                if "failure()" in cond and re.search(r"rollout\s+undo|helm\s+rollback", run):
                    undo = True
            if re.search(r"helm\s+upgrade[^\n]*--atomic", text):
                undo = True
            if not waited:
                problems.append("job %r never waits: add kubectl rollout status with a --timeout. "
                                "Without it the job goes green the moment the API server accepts the "
                                "object, which says nothing about whether a pod ever became ready"
                                % name)
            if not undo:
                problems.append("job %r has no undo: add a step with if: failure() that runs "
                                "kubectl rollout undo (helm upgrade --atomic counts too). Rollback "
                                "is a pipeline step, not an act of improvisation at 3 a.m." % name)
        if problems:
            rec("FAIL", "every deploy waits for the rollout and has an undo", "; ".join(problems))
        else:
            rec("PASS", "every deploy waits for the rollout and has an undo")


def flat(s):
    return " ".join(str(s).split())[:120]


SECRET_KEY = re.compile(
    r"(?i)\b(password|passwd|token|secret|api[_-]?key|access[_-]?key|private[_-]?key)\b\s*[:=]\s*(\S.*)$"
)
ALLOWED_VALUE = re.compile(
    r"^(\$|\"?\$\{\{|write$|read$|none$|inherit$|true$|false$|<|\"\"$|''$|\|$|>$)"
)
HIGH_ENTROPY = [
    (re.compile(r"ghp_[A-Za-z0-9]{16,}"), "a GitHub personal access token"),
    (re.compile(r"github_pat_[A-Za-z0-9_]{20,}"), "a GitHub fine-grained token"),
    (re.compile(r"AKIA[0-9A-Z]{16}"), "an AWS access key id"),
    (re.compile(r"-----BEGIN [A-Z ]*PRIVATE KEY-----"), "a private key"),
    (re.compile(r"\beyJ[A-Za-z0-9_-]{20,}\."), "a JWT"),
]


def check_no_plaintext_secrets():
    files = []
    if os.path.isdir(WORKFLOW_DIR):
        for n in sorted(os.listdir(WORKFLOW_DIR)):
            if n.endswith((".yml", ".yaml")):
                p = os.path.join(WORKFLOW_DIR, n)
                files.append((p, read(p)))
    for root, _, names in os.walk("deploy"):
        for n in sorted(names):
            if n.endswith((".yaml", ".yml")):
                p = os.path.join(root, n)
                files.append((p, read(p)))
    dockerfile = read("Dockerfile")
    if dockerfile is not None:
        files.append(("Dockerfile", dockerfile))
    runbook = read("RUNBOOK.md")

    findings = []
    for path, text in files:
        for n, line in enumerate((text or "").splitlines(), 1):
            bare = strip_comment(line).strip()
            m = SECRET_KEY.search(bare)
            if m and not ALLOWED_VALUE.match(m.group(2).strip()):
                findings.append("%s line %d assigns %s a literal value" % (path, n, m.group(1)))
            for rx, what in HIGH_ENTROPY:
                if rx.search(line):
                    findings.append("%s line %d looks like %s" % (path, n, what))
    for n, line in enumerate((runbook or "").splitlines(), 1):
        for rx, what in HIGH_ENTROPY:
            if rx.search(line):
                findings.append("RUNBOOK.md line %d looks like %s" % (n, what))
    for path, text in files:
        if path.startswith("deploy") and re.search(r"(?m)^kind:\s*Secret\s*$", text or ""):
            findings.append("%s commits a Secret object; its value is in your git history forever, "
                            "base64 is not encryption" % path)
    if findings:
        rec("FAIL", "nothing in the repository is a credential in plaintext",
            "%s. Credentials belong in repository or environment secrets and reach the runner as "
            "${{ secrets.NAME }}, or are minted per run by OIDC. A value committed once is leaked "
            "until it is rotated, not until it is deleted" % "; ".join(findings[:3]))
    else:
        rec("PASS", "nothing in the repository is a credential in plaintext")


# ---------------------------------------------------------------- manifests

BASE_DIR = os.path.join("deploy", "base")
OVERLAYS = {"dev": os.path.join("deploy", "overlays", "dev"),
            "prod": os.path.join("deploy", "overlays", "prod")}


def load_dir(path):
    """Every YAML document in a directory, as (filename, doc) pairs."""
    docs = []
    if not os.path.isdir(path):
        return docs
    for name in sorted(os.listdir(path)):
        if not name.endswith((".yaml", ".yml")):
            continue
        full = os.path.join(path, name)
        for doc in parse_documents(read(full)):
            docs.append((full, doc))
    return docs


def container_of(dep):
    containers = [c for c in as_list(dig(dep, "spec", "template", "spec", "containers"))
                  if isinstance(c, dict)]
    return containers[0] if containers else {}


def env_map(container):
    out = {}
    for e in as_list(container.get("env")):
        e = as_dict(e)
        if e.get("name"):
            out[str(e["name"])] = e
    return out


def check_manifests():
    """Returns the base image repository, or ''."""
    try:
        base_docs = load_dir(BASE_DIR)
    except YamlError as e:
        rec("FAIL", "the base Deployment still keeps its lifecycle contract",
            "%s does not parse: %s" % (BASE_DIR, e))
        return ""
    deps = [d for _, d in base_docs if d.get("kind") == "Deployment"]
    kusts = [d for _, d in base_docs if d.get("kind") == "Kustomization"]
    if not deps or not kusts:
        rec("FAIL", "the base Deployment still keeps its lifecycle contract",
            "deploy/base needs the Deployment, the Service and kustomization.yaml as they were "
            "given to you — this lesson grades the pipeline around them, not a rewrite of them")
        return ""
    dep, kust = deps[0], kusts[0]
    c = container_of(dep)
    podspec = as_dict(dig(dep, "spec", "template", "spec"))
    env = env_map(c)

    config = {}
    for gen in as_list(kust.get("configMapGenerator")):
        for lit in as_list(as_dict(gen).get("literals")):
            k, _, v = str(lit).partition("=")
            config[k.strip()] = v.strip()
    for _, doc in base_docs:
        if doc.get("kind") == "ConfigMap":
            config.update({str(k): str(v) for k, v in as_dict(doc.get("data")).items()})

    problems = []
    ready = dig(c, "readinessProbe", "httpGet", "path")
    live = dig(c, "livenessProbe", "httpGet", "path")
    if ready != "/readyz":
        problems.append("the readiness probe no longer hits /readyz")
    if live != "/healthz":
        problems.append("the liveness probe no longer hits /healthz")
    if live == ready:
        problems.append("both probes hit the same path, so the drain restarts the container")
    sleep_seconds = integer(dig(c, "lifecycle", "preStop", "sleep", "seconds"))
    if sleep_seconds is None:
        cmd = " ".join(str(x) for x in as_list(dig(c, "lifecycle", "preStop", "exec", "command")))
        m = re.search(r"sleep\s+([0-9]+)", cmd)
        sleep_seconds = int(m.group(1)) if m else None
    tgps = integer(podspec.get("terminationGracePeriodSeconds"))
    drain = go_duration(config.get("SHUTDOWN_TIMEOUT"))
    if sleep_seconds is None or sleep_seconds < 5:
        problems.append("the preStop sleep is gone or under 5s, so endpoint removal loses its race "
                        "with SIGTERM again")
    if tgps is None:
        problems.append("terminationGracePeriodSeconds is not set")
    if drain is None:
        problems.append("SHUTDOWN_TIMEOUT is missing from the configuration, or is not a Go duration")
    if None not in (tgps, sleep_seconds, drain) and tgps < sleep_seconds + drain + 5:
        problems.append("the budget no longer adds up: preStop %ds + SHUTDOWN_TIMEOUT %gs needs "
                        "more than the %ds grace period" % (sleep_seconds, drain, tgps))
    strategy = as_dict(dig(dep, "spec", "strategy", "rollingUpdate"))
    if str(strategy.get("maxUnavailable")) not in ("0", "0%"):
        problems.append("maxUnavailable is no longer 0, so a rollout can remove capacity before a "
                        "replacement is ready")
    if problems:
        rec("FAIL", "the base Deployment still keeps its lifecycle contract", "; ".join(problems))
    else:
        rec("PASS", "the base Deployment still keeps its lifecycle contract")

    res = as_dict(c.get("resources"))
    req, lim = as_dict(res.get("requests")), as_dict(res.get("limits"))
    cpu_lim, mem_lim = quantity(lim.get("cpu")), quantity(lim.get("memory"))
    problems = []
    if None in (quantity(req.get("cpu")), quantity(req.get("memory")), cpu_lim, mem_lim):
        problems.append("the container no longer declares both requests and limits for cpu and memory")
    gmp = env.get("GOMAXPROCS")
    ref = as_dict(dig(gmp or {}, "valueFrom", "resourceFieldRef"))
    if gmp is None:
        problems.append("GOMAXPROCS is gone, so the runtime sizes its scheduler from the node")
    elif not ref and integer(gmp.get("value")) != (int(math.ceil(cpu_lim)) if cpu_lim else None):
        problems.append("GOMAXPROCS no longer agrees with the CPU limit")
    elif ref and ref.get("resource") != "limits.cpu":
        problems.append("GOMAXPROCS reads %r rather than limits.cpu" % ref.get("resource"))
    gml = gomemlimit_bytes(as_dict(env.get("GOMEMLIMIT")).get("value"))
    if gml is None:
        problems.append("GOMEMLIMIT is missing or not in the syntax the Go runtime accepts")
    elif mem_lim and not (0.5 * mem_lim <= gml < mem_lim):
        problems.append("GOMEMLIMIT %s is not sensibly under the %s memory limit"
                        % (human(gml), human(mem_lim)))
    if problems:
        rec("FAIL", "the base still keeps the Go runtime inside its limits", "; ".join(problems))
    else:
        rec("PASS", "the base still keeps the Go runtime inside its limits")

    repo, tag = image_tag(c.get("image"))
    if not repo:
        rec("NOTE", "the base container declares no image", "the overlays cannot retag what is not there")
    elif tag == "latest" or not tag:
        rec("NOTE", "the base image is not pinned",
            "%r is a moving pointer; the overlays are where the real tag is chosen anyway" % c.get("image"))
    return repo


def check_overlays(base_repo):
    parsed, settings = {}, {}
    for envname, path in OVERLAYS.items():
        if not os.path.isdir(path):
            rec("FAIL", "both overlays are built on the base rather than copied from it",
                "%s does not exist. Both environments are promoted through deploy/overlays/<env>"
                % path)
            return
        try:
            docs = load_dir(path)
        except YamlError as e:
            rec("FAIL", "both overlays are built on the base rather than copied from it",
                "%s does not parse: %s" % (path, e))
            return
        kusts = [(f, d) for f, d in docs if str(d.get("kind")) == "Kustomization"]
        if not kusts:
            rec("FAIL", "both overlays are built on the base rather than copied from it",
                "%s/kustomization.yaml declares no Kustomization — it needs at least apiVersion: "
                "kustomize.config.k8s.io/v1beta1, kind: Kustomization and a resources list"
                % path)
            return
        parsed[envname] = (path, kusts[0][1], docs)

    problems = []
    for envname, (path, kust, _) in parsed.items():
        resources = [str(r) for r in as_list(kust.get("resources"))]
        if not resources:
            problems.append("%s lists no resources" % path)
            continue
        if not any(r.rstrip("/").endswith("base") for r in resources):
            problems.append("%s does not include the base (resources: - ../../base)" % path)
        strays = [r for r in resources if r.endswith((".yaml", ".yml"))]
        if strays:
            problems.append("%s lists %s as a resource: an overlay that re-declares an object owns "
                            "a second copy of it, and the two drift" % (path, strays[0]))
    if problems:
        rec("FAIL", "both overlays are built on the base rather than copied from it",
            "; ".join(problems))
    else:
        rec("PASS", "both overlays are built on the base rather than copied from it")

    def replica_count(kust, docs):
        for r in as_list(kust.get("replicas")):
            r = as_dict(r)
            if str(r.get("name")) == "timesvc":
                return integer(r.get("count"))
        for _, doc in docs:
            if doc.get("kind") == "Deployment" and integer(dig(doc, "spec", "replicas")) is not None:
                return integer(dig(doc, "spec", "replicas"))
        return None

    for envname, (path, kust, docs) in parsed.items():
        settings[envname] = {
            "namespace": kust.get("namespace"),
            "replicas": replica_count(kust, docs),
        }
    dev, prod = settings["dev"], settings["prod"]
    problems = []
    if not dev["namespace"] or not prod["namespace"]:
        problems.append("both overlays must set a namespace, so applying the wrong one cannot "
                        "quietly overwrite the other environment")
    elif dev["namespace"] == prod["namespace"]:
        problems.append("both overlays deploy into namespace %r — dev would overwrite prod"
                        % dev["namespace"])
    if dev["replicas"] is None or prod["replicas"] is None:
        problems.append("each overlay must choose its own replica count (the replicas: field, or a "
                        "patch that sets spec.replicas)")
    elif prod["replicas"] < 2:
        problems.append("prod runs %d replica(s): with maxUnavailable: 0 there is nothing to surge "
                        "over, so every rollout is a gap in service" % prod["replicas"])
    elif prod["replicas"] == dev["replicas"]:
        problems.append("dev and prod both run %d replicas — if the environments really are "
                        "identical, one of them is sized wrong" % dev["replicas"])
    if problems:
        rec("FAIL", "dev and prod differ where environments should differ", "; ".join(problems))
    else:
        rec("PASS", "dev and prod differ where environments should differ (namespaces %s/%s, "
            "replicas %d/%d)" % (dev["namespace"], prod["namespace"], dev["replicas"], prod["replicas"]))

    problems = []
    for envname, (path, kust, _) in parsed.items():
        images = [as_dict(i) for i in as_list(kust.get("images"))]
        entry = None
        for img in images:
            if not base_repo or str(img.get("name")) in (base_repo, base_repo.rsplit("/", 1)[-1]):
                entry = img
                break
        if entry is None:
            problems.append("%s sets no images: entry for %s, so the pipeline has nothing to "
                            "rewrite and the overlay deploys whatever the base happens to say"
                            % (path, base_repo or "the service image"))
            continue
        tag = str(entry.get("newTag") or entry.get("digest") or "")
        if not tag:
            problems.append("%s names the image but no newTag or digest" % path)
        elif tag == "latest":
            problems.append("%s pins the image to latest, which is not a version: the tag can move "
                            "under a Deployment that never changes, so rolling back the manifest "
                            "rolls back nothing" % path)
        elif not immutable_tag(tag):
            problems.append("%s uses tag %r, which nothing stops a human from moving. Commit a "
                            "sha- tag or a digest as the bootstrap value" % (path, tag))
    if problems:
        rec("FAIL", "every overlay pins the image to an immutable reference", "; ".join(problems))
    else:
        rec("PASS", "every overlay pins the image to an immutable reference")

    problems = []
    for envname, (path, kust, _) in parsed.items():
        patch_docs = []
        for p in as_list(kust.get("patches")):
            rel = as_dict(p).get("path")
            if not rel:
                problems.append("%s uses an inline patch: this grader (and every reviewer) reads "
                                "patches better as files — use patches: - path: something.yaml" % path)
                continue
            full = os.path.join(path, str(rel))
            if not os.path.exists(full):
                problems.append("%s references patch %s, which does not exist" % (path, rel))
                continue
            try:
                patch_docs.extend(parse_documents(read(full)))
            except YamlError as e:
                problems.append("%s does not parse: %s" % (full, e))
        touched = False
        for doc in patch_docs:
            c = container_of(doc)
            mem = quantity(dig(c, "resources", "limits", "memory"))
            if mem is None:
                continue
            touched = True
            gml = gomemlimit_bytes(as_dict(env_map(c).get("GOMEMLIMIT")).get("value"))
            if gml is None:
                problems.append("the %s patch changes limits.memory but leaves GOMEMLIMIT at the "
                                "base's value. GOMAXPROCS follows limits.cpu through the downward "
                                "API; GOMEMLIMIT is a literal and follows nothing, so the runtime "
                                "would aim at a heap this container is not allowed to have"
                                % envname)
            elif not (0.5 * mem <= gml < mem):
                problems.append("the %s patch sets GOMEMLIMIT %s against a %s memory limit — it has "
                                "to be under the limit, with headroom for what the runtime does not "
                                "count" % (envname, human(gml), human(mem)))
        if not touched:
            problems.append("the %s overlay never sizes its own resources; requests and limits are "
                            "exactly where environments differ" % envname)
    if problems:
        rec("FAIL", "an overlay that changes the memory limit changes GOMEMLIMIT with it",
            "; ".join(problems))
    else:
        rec("PASS", "an overlay that changes the memory limit changes GOMEMLIMIT with it")


# ---------------------------------------------------------------- runbook

SECTIONS = [
    "What this service is",
    "How a change reaches production",
    "Verify a release",
    "Roll back",
    "From a clean state",
    "Secrets and access",
]


def check_runbook():
    text = read("RUNBOOK.md")
    if text is None:
        rec("FAIL", "RUNBOOK.md keeps its six sections and every one is written",
            "RUNBOOK.md is missing — restore it from the exercise and answer under every heading")
        return
    bodies, current = {}, None
    for line in text.splitlines():
        m = re.match(r"^##\s+(.*?)\s*$", line)
        if m:
            current = m.group(1)
            bodies[current] = []
        elif current:
            bodies[current].append(line)
    written, thin, absent = {}, [], []
    for name in SECTIONS:
        if name not in bodies:
            absent.append(name)
            continue
        body = "\n".join(bodies[name]).strip()
        if "TODO" in body or len(body) < 150:
            thin.append(name)
            continue  # an unanswered section has nothing for the checks below
        written[name] = body
    if absent:
        rec("FAIL", "RUNBOOK.md keeps its six sections and every one is written",
            "missing heading(s): %s. The headings are the contract: someone paged at 3 a.m. scans "
            "for them" % ", ".join("## " + a for a in absent))
    elif thin:
        rec("FAIL", "RUNBOOK.md keeps its six sections and every one is written",
            "section(s) %s still hold a TODO or under 150 characters of prose"
            % ", ".join(thin))
    else:
        rec("PASS", "RUNBOOK.md keeps its six sections and every one is written")

    path = written.get("How a change reaches production", "")
    if re.search(r"(?i)\bsha\b|digest|immutable", path):
        rec("PASS", "the runbook says what identifies a release")
    else:
        rec("FAIL", "the runbook says what identifies a release",
            "'How a change reaches production' never names what a release *is* — the commit SHA "
            "tag, or the digest. That name is what the next person types into a rollback")

    verify = written.get("Verify a release", "")
    if not re.search(r"rollout\s+status|helm\s+status", verify):
        rec("FAIL", "the runbook verifies a release with commands",
            "'Verify a release' has no kubectl rollout status (or helm status). A green workflow "
            "means the commands exited 0, not that pods are serving")
    elif not re.search(r"(?i)/version|revision|/readyz|get pods", verify):
        rec("FAIL", "the runbook verifies a release with commands",
            "'Verify a release' waits for the rollout but never checks *what* is running. Curl "
            "/version and compare the revision with the commit you shipped")
    else:
        rec("PASS", "the runbook verifies a release with commands")

    undo = written.get("Roll back", "")
    if not re.search(r"rollout\s+undo|helm\s+rollback|kustomize\s+edit\s+set\s+image|kubectl\s+set\s+image", undo):
        rec("FAIL", "the runbook's rollback is something you can paste",
            "'Roll back' has no concrete command. kubectl rollout undo deployment/timesvc, or "
            "re-deploying the previous immutable tag — prose alone is not a rollback")
    elif not re.search(r"rollout\s+status|/version|revision|verify|confirm", undo):
        rec("FAIL", "the runbook's rollback is something you can paste",
            "'Roll back' never verifies the rollback. The undo is a deploy like any other and can "
            "fail the same ways")
    else:
        rec("PASS", "the runbook's rollback is something you can paste")

    clean = written.get("From a clean state", "")
    stages = {
        "namespace": r"(?i)namespace",
        "apply": r"(?i)kubectl\s+apply|kustomize\s+build|helm\s+install",
        "image": r"(?i)image|registry|ghcr|pull",
    }
    absent = [k for k, rx in stages.items() if not re.search(rx, clean)]
    if absent:
        rec("FAIL", "the runbook can rebuild the service from nothing",
            "'From a clean state' does not mention %s. An empty cluster needs the namespace, the "
            "apply, and access to the image — this is the section you will be glad of exactly once"
            % ", ".join(absent))
    else:
        rec("PASS", "the runbook can rebuild the service from nothing")

    creds = written.get("Secrets and access", "")
    if not re.search(r"(?i)secret|oidc|credential", creds):
        rec("FAIL", "the runbook accounts for the pipeline's credentials",
            "'Secrets and access' names no credential. List every one the pipeline uses, where it "
            "lives, and what it can do")
    elif not re.search(r"(?i)never|not\s+.*(commit|bake|image|manifest)|rotate", creds):
        rec("FAIL", "the runbook accounts for the pipeline's credentials",
            "'Secrets and access' lists credentials but never says what must not happen to them: "
            "no secret in an image layer, none in a manifest, and each one rotatable")
    else:
        rec("PASS", "the runbook accounts for the pipeline's credentials")


def main():
    arg_name = check_dockerfile()
    check_workflow(arg_name)
    check_no_plaintext_secrets()
    base_repo = check_manifests()
    check_overlays(base_repo)
    check_runbook()


try:
    main()
except Exception as exc:  # a crash here is an exercise bug, not a learner bug
    sys.stderr.write("static grader crashed: %r\n" % (exc,))
    sys.exit(2)
sys.stdout.write("\n".join(OUT) + "\n")
PY

if [ $? -ne 0 ]; then
	echo "check.sh: the static grader crashed. That is a bug in the exercise, not in your files."
	exit 2
fi

# ---------------------------------------------------------------- tooling (bonus)
# Everything below is optional. Missing tooling, an unreachable cluster or no
# network are reported SKIPPED, never FAILED: the grade above stands on its own.

if command -v go >/dev/null 2>&1; then
	if out="$(go build ./... 2>&1)" && [ -z "$(gofmt -l . 2>/dev/null)" ]; then
		record PASS "go: the service still builds and is formatted"
	elif printf '%s' "$out" | grep -qiE 'GOCACHE|GOPATH|permission denied|cannot find GOROOT|toolchain not available'; then
		record SKIP "go: the service still builds and is formatted" \
			"the Go toolchain here cannot build: $(flatten "$out")"
	else
		record FAIL "go: the service still builds and is formatted" \
			"$(flatten "${out:-gofmt -l reported: $(gofmt -l . 2>/dev/null)}")"
	fi
else
	record SKIP "go: the service still builds and is formatted" "no Go toolchain on PATH"
fi

img=tutor-capstone:check
cname=tutor-capstone-check
size_ceiling_mb=30
work=$(mktemp -d)
trap 'rm -f "$results"; rm -rf "$work"' EXIT

# Build with whatever the Dockerfile called its revision argument, so this
# check grades the chain you wired rather than the name you happened to pick.
argname=$(grep -Ei '^[[:space:]]*ARG[[:space:]]+' Dockerfile 2>/dev/null |
	awk '{print $2}' | cut -d= -f1 | grep -Ei 'rev|version|commit|sha' | head -n1)
argname=${argname:-REVISION}

if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
	docker rm -f "$cname" >/dev/null 2>&1 || true
	if docker build --build-arg "$argname=checksha" -t "$img" . >"$work/build.log" 2>&1; then
		record PASS "docker: the image builds"
		built=yes
	elif grep -qiE 'network|dial tcp|no such host|timeout|TLS handshake|proxyconnect|429 Too Many' "$work/build.log"; then
		record SKIP "docker: the image builds" \
			"the build could not reach a registry: $(flatten "$(tail -n 3 "$work/build.log")")"
		built=no
	else
		record FAIL "docker: the image builds" \
			"$(flatten "$(tail -n 6 "$work/build.log")")"
		built=no
	fi

	if [ "$built" = yes ]; then
		user=$(docker image inspect -f '{{.Config.User}}' "$img" 2>/dev/null)
		label=$(docker image inspect -f '{{index .Config.Labels "org.opencontainers.image.revision"}}' "$img" 2>/dev/null)
		bytes=$(docker image inspect -f '{{.Size}}' "$img" 2>/dev/null || echo 0)
		mb=$((bytes / 1000000))
		if [ -z "$user" ] || [ "$user" = "root" ] || [ "${user%%:*}" = "0" ]; then
			record FAIL "docker: the built image is unprivileged, labelled and small" \
				"the image runs as ${user:-root}"
		elif [ "$label" != "checksha" ]; then
			record FAIL "docker: the built image is unprivileged, labelled and small" \
				"org.opencontainers.image.revision is '${label:-unset}', not the $argname build argument this check passed in"
		elif [ "$mb" -gt "$size_ceiling_mb" ]; then
			record FAIL "docker: the built image is unprivileged, labelled and small" \
				"the image is ${mb} MB (ceiling ${size_ceiling_mb} MB) — something heavy came across from the build stage"
		else
			record PASS "docker: the built image is unprivileged, labelled and small (${mb} MB)"
		fi

		docker run -d --rm --name "$cname" -p 127.0.0.1:0:8080 "$img" >/dev/null 2>&1
		hostport=$(docker port "$cname" 8080/tcp 2>/dev/null | head -n1 | sed 's/.*://')
		seen=""
		if [ -n "$hostport" ]; then
			for _ in 1 2 3 4 5 6 7 8 9 10; do
				seen=$(python3 -c "
import sys,urllib.request
try:
    print(urllib.request.urlopen('http://127.0.0.1:$hostport/version',timeout=2).read().decode())
except Exception:
    sys.exit(1)
" 2>/dev/null) && break
				sleep 1
			done
		fi
		if printf '%s' "$seen" | grep -q 'checksha'; then
			record PASS "docker: the running container reports the revision it was built from"
		elif [ -n "$seen" ]; then
			record FAIL "docker: the running container reports the revision it was built from" \
				"GET /version answered $(flatten "$seen") — the linker flag never reached main.buildRevision"
		else
			record FAIL "docker: the running container reports the revision it was built from" \
				"the container did not answer GET /version. Start it yourself and read docker logs $cname"
		fi
		docker rm -f "$cname" >/dev/null 2>&1 || true
	else
		record SKIP "docker: the built image is unprivileged, labelled and small" "no image was built"
		record SKIP "docker: the running container reports the revision it was built from" "no image was built"
	fi
else
	reason="no reachable Docker daemon, so the image could not be built here"
	record SKIP "docker: the image builds" "$reason"
	record SKIP "docker: the built image is unprivileged, labelled and small" "$reason"
	record SKIP "docker: the running container reports the revision it was built from" "$reason"
fi

render=""
if command -v kustomize >/dev/null 2>&1; then
	render="kustomize build"
elif command -v kubectl >/dev/null 2>&1; then
	render="kubectl kustomize"
fi
if [ -n "$render" ]; then
	rendered=yes
	for envdir in deploy/overlays/dev deploy/overlays/prod; do
		if out="$($render "$envdir" 2>&1)"; then
			printf '%s' "$out" >"$work/$(basename "$envdir").yaml"
		else
			record FAIL "kustomize: both overlays render" \
				"$envdir: $(flatten "$out")"
			rendered=no
			break
		fi
	done
	if [ "$rendered" = yes ]; then
		record PASS "kustomize: both overlays render"
		missing=""
		for needle in /readyz /healthz preStop terminationGracePeriodSeconds GOMEMLIMIT; do
			grep -q -- "$needle" "$work/prod.yaml" || missing="$missing $needle"
		done
		if [ -n "$missing" ]; then
			record FAIL "kustomize: the rendered prod manifests keep the base's contract" \
				"the rendered output has lost:$missing — an overlay is replacing the base instead of patching it"
		else
			record PASS "kustomize: the rendered prod manifests keep the base's contract"
		fi
		if command -v kubectl >/dev/null 2>&1; then
			if out="$(kubectl apply --dry-run=client -f "$work/prod.yaml" 2>&1)"; then
				record PASS "kubectl: the rendered prod manifests survive a client-side dry run"
			elif printf '%s' "$out" | grep -qiE 'connection refused|unable to connect|no configuration has been provided|couldn.t get current server'; then
				record SKIP "kubectl: the rendered prod manifests survive a client-side dry run" \
					"kubectl has no usable kubeconfig here: $(flatten "$out")"
			else
				record FAIL "kubectl: the rendered prod manifests survive a client-side dry run" \
					"$(flatten "$out")"
			fi
		else
			record SKIP "kubectl: the rendered prod manifests survive a client-side dry run" \
				"no kubectl on PATH"
		fi
	else
		record SKIP "kustomize: the rendered prod manifests keep the base's contract" \
			"an overlay did not render"
		record SKIP "kubectl: the rendered prod manifests survive a client-side dry run" \
			"an overlay did not render"
	fi
else
	reason="neither kustomize nor kubectl is on PATH, so the overlays could not be rendered"
	record SKIP "kustomize: both overlays render" "$reason"
	record SKIP "kustomize: the rendered prod manifests keep the base's contract" "$reason"
	record SKIP "kubectl: the rendered prod manifests survive a client-side dry run" "$reason"
fi

wf=$(find .github/workflows -maxdepth 1 \( -name '*.yml' -o -name '*.yaml' \) 2>/dev/null | sort | head -n 1)
if [ -z "$wf" ]; then
	record SKIP "actionlint: the workflow is valid GitHub Actions" "no workflow file to lint"
elif command -v actionlint >/dev/null 2>&1; then
	if out="$(actionlint -shellcheck= -pyflakes= "$wf" 2>&1)"; then
		record PASS "actionlint: the workflow is valid GitHub Actions"
	else
		record FAIL "actionlint: the workflow is valid GitHub Actions" "$(flatten "$out")"
	fi
else
	record SKIP "actionlint: the workflow is valid GitHub Actions" \
		"actionlint is not installed — it is the one tool that reads a workflow the way GitHub does"
fi

chart=$(find . -name Chart.yaml -not -path './.git/*' 2>/dev/null | sort | head -n 1)
if [ -z "$chart" ]; then
	record SKIP "helm: the chart lints" \
		"no Chart.yaml here — this capstone promotes with overlays. Bring the chart you wrote in helm-kustomize if you want it linted too"
elif command -v helm >/dev/null 2>&1; then
	if out="$(helm lint "$(dirname "$chart")" 2>&1)"; then
		record PASS "helm: the chart lints"
	else
		record FAIL "helm: the chart lints" "$(flatten "$out")"
	fi
else
	record SKIP "helm: the chart lints" "helm is not installed"
fi

# ---------------------------------------------------------------- report

pass=0
fail=0
skip=0
while IFS='|' read -r kind msg fix; do
	case "$kind" in
	PASS)
		pass=$((pass + 1))
		printf 'ok    %s\n' "$msg"
		;;
	FAIL)
		fail=$((fail + 1))
		printf 'FAIL  %s\n      fix: %s\n' "$msg" "$fix"
		;;
	SKIP)
		skip=$((skip + 1))
		printf 'skip  %s\n      why: %s\n' "$msg" "$fix"
		;;
	NOTE)
		printf 'note  %s\n      %s\n' "$msg" "$fix"
		;;
	esac
done <"$results"

echo
if [ "$fail" -eq 0 ]; then
	printf '%d checks passed, 0 failed, %d skipped for missing tooling.\n' "$pass" "$skip"
	if [ "$skip" -gt 0 ]; then
		echo "The skipped checks are a bonus, not part of the grade — your files pass."
	fi
	exit 0
fi
printf '%d passed, %d failed, %d skipped for missing tooling.\n' "$pass" "$fail" "$skip"
echo "Fix the FAIL lines above and re-run: bash check.sh"
exit 1
