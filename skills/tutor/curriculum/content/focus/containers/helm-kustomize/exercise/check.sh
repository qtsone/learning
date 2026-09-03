#!/usr/bin/env bash
# Referee for the Helm & Kustomize exercise.
#
# The grade comes from reading the files you wrote: the chart, the base, the two
# overlays and NOTES.md. No cluster, no kubectl, no helm, no kustomize, no
# Docker daemon, no network, no root. If helm, kustomize or a kubectl with a
# reachable cluster happens to be here, extra checks run as a bonus; when they
# are not, they are reported SKIPPED, never failed.
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
# not, so the grader carries the same small YAML reader the earlier lessons in
# this pack used, plus a stripper that removes template actions from the chart's
# templates so what is left can be read as YAML.

python3 - >>"$results" <<'PY'
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


def read_docs(path):
    return parse_documents(open(path, encoding="utf-8", errors="replace").read())


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


def leaves(node):
    if isinstance(node, dict):
        return sum(leaves(v) for v in node.values())
    if isinstance(node, list):
        return sum(leaves(v) for v in node)
    return 1


def has_key(node, name):
    if isinstance(node, dict):
        if name in node:
            return True
        return any(has_key(v, name) for v in node.values())
    if isinstance(node, list):
        return any(has_key(v, name) for v in node)
    return False


def container_of(doc):
    """First container of a Deployment, and the pod spec around it."""
    podspec = as_dict(dig(doc, "spec", "template", "spec"))
    containers = [c for c in as_list(podspec.get("containers")) if isinstance(c, dict)]
    return (containers[0] if containers else {}), podspec


def env_of(container):
    out = {}
    for e in as_list(container.get("env")):
        e = as_dict(e)
        if e.get("name"):
            out[str(e["name"])] = e
    return out


def probe_paths(container):
    def path(name):
        return dig(as_dict(container.get(name)), "httpGet", "path")

    return path("readinessProbe"), path("livenessProbe")


def prestop_seconds(container):
    pre = as_dict(dig(container, "lifecycle", "preStop"))
    if "sleep" in pre:
        return integer(as_dict(pre.get("sleep")).get("seconds"))
    if "exec" in pre:
        cmd = " ".join(str(x) for x in as_list(as_dict(pre["exec"]).get("command")))
        m = re.search(r"sleep\s+([0-9]+)", cmd)
        if m:
            return int(m.group(1))
    return None


# ------------------------------------------------------- template stripping
# Helm templates are not YAML: they are text that produces YAML. Rather than
# guess at Go template semantics, the grader removes the actions — control
# lines disappear, value actions become a placeholder — and reads what is left.
# Placeholders that turn out to be a single .Values lookup are then resolved
# against values.yaml, which is as close to a render as static reading gets.

ACTION_RE = re.compile(r"\{\{-?(.*?)-?\}\}")
CONTROL_RE = re.compile(r"^\s*(if|else|end|range|with|define|block|template\s|/\*)")
TPL_RE = re.compile(r"__TPL([0-9]+)__")


def strip_actions(text):
    exprs, out = [], []
    for raw in text.splitlines():
        found = ACTION_RE.findall(raw)
        if not found:
            out.append(raw)
            continue
        residue = ACTION_RE.sub("", raw).strip()
        values = [a for a in found if not CONTROL_RE.match(a)]
        if residue == "":
            if not values:
                continue
            # A line that is nothing but a value action belongs to the key above
            # it — that is what "resources:" followed by a toYaml/nindent line
            # means, whatever column the action was written in.
            for j in range(len(out) - 1, -1, -1):
                prev = out[j]
                if not prev.strip():
                    continue
                if prev.rstrip().endswith(":"):
                    exprs.append(values[0].strip())
                    out[j] = "%s __TPL%d__" % (prev.rstrip(), len(exprs) - 1)
                break
            continue

        def repl(m):
            exprs.append(m.group(1).strip())
            return "__TPL%d__" % (len(exprs) - 1)

        out.append(ACTION_RE.sub(repl, raw))
    return "\n".join(out), exprs


def restore(v, exprs):
    """Put the template actions back, for messages and for text matching."""
    if not isinstance(v, str):
        return v
    return TPL_RE.sub(lambda m: "{{ %s }}" % exprs[int(m.group(1))], v)


def templated(v):
    return isinstance(v, str) and TPL_RE.search(v) is not None


VALUES_PATH_RE = re.compile(r"\.Values((?:\.[A-Za-z0-9_]+)+)")
SKIP_EXISTENCE = re.compile(r"\b(default|hasKey|empty|dig|if|with|range|or|and|ternary)\b")


def lookup(values, dotted):
    cur = values
    for part in dotted.strip(".").split("."):
        if not isinstance(cur, dict) or part not in cur:
            return False, None
        cur = cur[part]
    return True, cur


def resolve(v, exprs, values):
    """A scalar that is exactly one .Values lookup -> its value from values.yaml."""
    if not isinstance(v, str):
        return v
    m = TPL_RE.fullmatch(v.strip())
    if not m:
        return None if templated(v) else v
    expr = exprs[int(m.group(1))].strip()
    head = expr.split("|")[0].strip()
    head = re.sub(r"^(toYaml|quote|int|int64|toString|float64)\s+", "", head).strip()
    pm = re.fullmatch(r"\.Values((?:\.[A-Za-z0-9_]+)+)", head)
    if not pm:
        return None
    ok, val = lookup(values, pm.group(1))
    return val if ok else None


# ------------------------------------------------------------ kustomize build
# A deliberately small kustomize: resources (files and directories), strategic
# merge patches, the replicas and images fields, and the generators. Enough to
# render this exercise's overlays and say what the cluster would receive.


class BuildError(Exception):
    pass


def load_kustomization(dirpath):
    for name in ("kustomization.yaml", "kustomization.yml", "Kustomization"):
        path = os.path.join(dirpath, name)
        if os.path.isfile(path):
            docs = read_docs(path)
            return (docs[0] if docs else {}), path
    raise BuildError("%s has no kustomization.yaml" % dirpath)


def merge(base, patch):
    """Strategic-merge-ish: maps merge, lists of named items merge by name."""
    if isinstance(base, dict) and isinstance(patch, dict):
        out = dict(base)
        for k, v in patch.items():
            out[k] = merge(base[k], v) if k in base else v
        return out
    if isinstance(base, list) and isinstance(patch, list):
        named = [x for x in base + patch if isinstance(x, dict) and "name" in x]
        if named and len(named) == len(base) + len(patch):
            out = [dict(x) for x in base]
            index = {str(x["name"]): i for i, x in enumerate(out)}
            for item in patch:
                key = str(item["name"])
                if key in index:
                    out[index[key]] = merge(out[index[key]], item)
                else:
                    out.append(item)
            return out
        return patch
    return patch


def generated_configmaps(kust):
    out = []
    for gen in as_list(kust.get("configMapGenerator")):
        gen = as_dict(gen)
        data = {}
        for literal in as_list(gen.get("literals")):
            key, sep, value = str(literal).partition("=")
            if sep:
                data[key.strip()] = value
        out.append({
            "apiVersion": "v1",
            "kind": "ConfigMap",
            "metadata": {"name": str(gen.get("name") or "")},
            "data": data,
            "_behavior": str(gen.get("behavior") or "create"),
        })
    return out


def build(dirpath, depth=0):
    if depth > 4:
        raise BuildError("resources point at each other in a loop")
    kust, _ = load_kustomization(dirpath)
    docs = []
    for entry in as_list(kust.get("resources")):
        target = os.path.normpath(os.path.join(dirpath, str(entry)))
        if os.path.isdir(target):
            docs.extend(build(target, depth + 1))
        elif os.path.isfile(target):
            docs.extend(read_docs(target))
        else:
            raise BuildError("%s: resources entry %r is neither a file nor a directory here"
                             % (dirpath, entry))

    for cm in generated_configmaps(kust):
        existing = next(
            (d for d in docs
             if d.get("kind") == "ConfigMap"
             and str(dig(d, "metadata", "name") or "").startswith(str(dig(cm, "metadata", "name")))),
            None,
        )
        if existing is not None and cm["_behavior"] in ("merge", "replace"):
            if cm["_behavior"] == "merge":
                existing["data"] = dict(as_dict(existing.get("data")), **as_dict(cm.get("data")))
            else:
                existing["data"] = as_dict(cm.get("data"))
        else:
            docs.append(cm)

    patches = []
    for p in as_list(kust.get("patches")):
        p = as_dict(p)
        if p.get("path"):
            patches.append(str(p["path"]))
        elif p.get("patch"):
            raise BuildError("%s: inline 'patch:' blocks are outside what this grader reads — "
                             "put the patch in a file and reference it with path:" % dirpath)
    patches.extend(str(p) for p in as_list(kust.get("patchesStrategicMerge")))

    for rel in patches:
        path = os.path.normpath(os.path.join(dirpath, rel))
        if not os.path.isfile(path):
            raise BuildError("%s: patch file %r does not exist" % (dirpath, rel))
        for patch in read_docs(path):
            name = str(dig(patch, "metadata", "name") or "")
            kind = patch.get("kind")
            hit = False
            for i, doc in enumerate(docs):
                if doc.get("kind") == kind and str(dig(doc, "metadata", "name") or "") == name:
                    docs[i] = merge(doc, patch)
                    hit = True
            if not hit:
                raise BuildError("%s: patch %s targets %s/%s, which nothing in the base declares"
                                 % (dirpath, rel, kind, name))

    for r in as_list(kust.get("replicas")):
        r = as_dict(r)
        for doc in docs:
            if str(dig(doc, "metadata", "name") or "") == str(r.get("name") or ""):
                doc.setdefault("spec", {})["replicas"] = r.get("count")

    for img in as_list(kust.get("images")):
        img = as_dict(img)
        for doc in docs:
            container, _ = container_of(doc)
            if not container:
                continue
            current = str(container.get("image") or "")
            repo = current.split(":")[0]
            if repo == str(img.get("name") or ""):
                container["image"] = "%s:%s" % (
                    img.get("newName") or repo, img.get("newTag") or current.partition(":")[2])

    prefix, suffix = str(kust.get("namePrefix") or ""), str(kust.get("nameSuffix") or "")
    for doc in docs:
        meta = as_dict(doc.get("metadata"))
        if kust.get("namespace"):
            meta["namespace"] = kust["namespace"]
        if prefix or suffix:
            meta["name"] = "%s%s%s" % (prefix, meta.get("name", ""), suffix)
        doc["metadata"] = meta
    return docs


# ---------------------------------------------------------------- chart checks


CHART = "chart"
TEMPLATES = os.path.join(CHART, "templates")


def check_chart():
    """Returns the parsed values, or {} when the chart cannot be read."""
    if not os.path.isdir(TEMPLATES):
        rec("FAIL", "the chart is laid out the way Helm expects",
            "expected chart/Chart.yaml, chart/values.yaml and chart/templates/ next to check.sh. "
            "Helm finds templates by directory name, not by configuration")
        return {}

    try:
        chart_docs = read_docs(os.path.join(CHART, "Chart.yaml"))
        values_docs = read_docs(os.path.join(CHART, "values.yaml"))
    except YamlError as e:
        rec("FAIL", "the chart is laid out the way Helm expects",
            "Chart.yaml or values.yaml does not parse: %s. Two spaces per level, spaces only, "
            "a space after every colon" % e)
        return {}
    except IOError:
        rec("FAIL", "the chart is laid out the way Helm expects",
            "chart/Chart.yaml and chart/values.yaml both have to exist — Chart.yaml is what makes "
            "a directory a chart")
        return {}

    chart = chart_docs[0] if chart_docs else {}
    values = values_docs[0] if values_docs else {}

    version = str(chart.get("version") or "")
    app_version = str(chart.get("appVersion") or "")
    if chart.get("apiVersion") != "v2":
        rec("FAIL", "Chart.yaml names and versions the chart",
            "apiVersion is %r. Helm 3 charts are apiVersion: v2 — v1 is the Helm 2 layout, where "
            "dependencies live in a separate requirements.yaml" % chart.get("apiVersion"))
    elif not chart.get("name"):
        rec("FAIL", "Chart.yaml names and versions the chart",
            "add name: timesvc. The chart's name is part of every default resource name it renders")
    elif not re.match(r"^[0-9]+\.[0-9]+\.[0-9]+", version):
        rec("FAIL", "Chart.yaml names and versions the chart",
            "version is %r, which is not a semantic version. It is what `helm install --version` "
            "asks for and what an installer reads to decide whether an upgrade is safe" % version)
    elif not app_version:
        rec("FAIL", "Chart.yaml names and versions the chart",
            "add appVersion — the version of timesvc this chart installs by default. It is a "
            "different number from version: a fix to your templates bumps version and leaves "
            "appVersion alone. Quote it, or 1.10 becomes the number 1.1")
    elif not chart.get("description"):
        rec("FAIL", "Chart.yaml names and versions the chart",
            "add a one-line description — it is what somebody searching a chart repository sees")
    else:
        rec("PASS", "Chart.yaml names and versions the chart (version %s, appVersion %s)"
            % (version, app_version))

    required = [
        ("replicaCount", "how many pods this chart installs by default"),
        ("image.repository", "the image this chart runs"),
        ("image.tag", "the image tag, so an installer can name the version they want"),
        ("image.pullPolicy", "whether the kubelet re-pulls the tag"),
        ("config.port", "the port the service listens on"),
        ("config.logLevel", "the log level"),
        ("config.shutdownTimeout", "how long the drain may take"),
        ("resources.requests.cpu", "the CPU the scheduler reserves"),
        ("resources.requests.memory", "the memory the scheduler reserves"),
        ("resources.limits.cpu", "the CPU ceiling"),
        ("resources.limits.memory", "the memory ceiling"),
    ]
    missing = [(k, why) for k, why in required if not lookup(values, k)[0]]
    tag = str(lookup(values, "image.tag")[1] or "")
    if missing:
        rec("FAIL", "values.yaml is the chart's API",
            "values.yaml has no %s (%s). Anything an installer must be able to change without "
            "editing your templates belongs here — and only that; %d key(s) still missing"
            % (missing[0][0], missing[0][1], len(missing)))
    elif tag in ("", "latest"):
        rec("FAIL", "values.yaml is the chart's API",
            "image.tag is %r. A chart that defaults to a moving tag installs something nobody can "
            "name afterwards, and `helm rollback` would put back the same words and a different "
            "image" % tag)
    else:
        rec("PASS", "values.yaml is the chart's API (%d keys, image tag %s)"
            % (len(required), tag))
        if app_version and tag and app_version != tag:
            rec("NOTE", "appVersion (%s) and image.tag (%s) disagree" % (app_version, tag),
                "not wrong — appVersion is documentation and image.tag is what runs — but if they "
                "are meant to be the same number, an installer reading `helm list` will believe the "
                "wrong one")

    names = [f for f in sorted(os.listdir(TEMPLATES)) if f.endswith((".yaml", ".yml"))]
    stripped, problems = {}, []
    for name in names:
        text = open(os.path.join(TEMPLATES, name), encoding="utf-8", errors="replace").read()
        body, exprs = strip_actions(text)
        try:
            stripped[name] = (parse_documents(body), exprs, text)
        except YamlError as e:
            problems.append("templates/%s: %s" % (name, e))
    if problems:
        rec("FAIL", "the templates read as YAML once the template actions are removed",
            "%s. This is the whole Helm failure mode: the templating layer has no idea it is "
            "producing YAML. Run `helm template chart` to see what it actually generates, and keep "
            "each {{ ... }} on one line — either as the value after a key:, or alone on a line "
            "directly under its key:" % problems[0])
        return values
    rec("PASS", "the templates read as YAML once the template actions are removed (%d file(s))"
        % len(stripped))

    unknown = []
    for name, (_, exprs, _) in stripped.items():
        for expr in exprs:
            if SKIP_EXISTENCE.search(expr):
                continue
            for path in VALUES_PATH_RE.findall(expr):
                if not lookup(values, path)[0]:
                    unknown.append("templates/%s reads .Values%s" % (name, path))
    if unknown:
        rec("FAIL", "every value the templates read exists in values.yaml",
            "%s, and values.yaml does not define it. Nothing type-checks a chart: Helm renders the "
            "missing value as empty and hands the API server a manifest with a hole in it%s"
            % (unknown[0], "" if len(unknown) == 1 else " (%d in total)" % len(unknown)))
    else:
        rec("PASS", "every value the templates read exists in values.yaml")

    deployment, exprs = None, []
    for name, (docs, ex, _) in stripped.items():
        for doc in docs:
            if doc.get("kind") == "Deployment":
                deployment, exprs = doc, ex
    if deployment is None:
        rec("FAIL", "the Deployment template drives replicas, image and resources from values",
            "no template renders a Deployment. Copy the base Deployment into chart/templates/ and "
            "replace the values an installer must be able to change")
        return values

    container, podspec = container_of(deployment)
    image_text = restore(container.get("image"), exprs)
    replicas = resolve(dig(deployment, "spec", "replicas"), exprs, values)
    resources = resolve(container.get("resources"), exprs, values)
    hard_coded = []
    if not templated(dig(deployment, "spec", "replicas")):
        hard_coded.append("replicas")
    if not (".Values.image.repository" in str(image_text) and ".Values.image.tag" in str(image_text)):
        hard_coded.append("image")
    if not templated(container.get("resources")):
        hard_coded.append("resources")
    if hard_coded:
        rec("FAIL", "the Deployment template drives replicas, image and resources from values",
            "%s still hard-coded in the template. That is a chart with no API: `helm install "
            "--set replicaCount=5` would render the same manifest, and the only way to change it "
            "would be to fork the chart. resources is one line: "
            "{{- toYaml .Values.resources | nindent 12 }}" % ", ".join(hard_coded))
    elif integer(replicas) is None or integer(replicas) < 1:
        rec("FAIL", "the Deployment template drives replicas, image and resources from values",
            "replicas comes from values but resolves to %r — give replicaCount a whole number "
            "at least 1" % replicas)
    elif not isinstance(resources, dict):
        rec("FAIL", "the Deployment template drives replicas, image and resources from values",
            "the resources block reads from values, but values.yaml does not hold a resources "
            "mapping with requests and limits under it")
    else:
        rec("PASS", "the Deployment template drives replicas, image and resources from values "
            "(replicas %s)" % replicas)

    ready_path, live_path = probe_paths(container)
    if ready_path != "/readyz" or live_path != "/healthz":
        rec("FAIL", "the chart still renders the probes the platform depends on",
            "the rendered readiness probe hits %r and liveness %r; they must stay /readyz and "
            "/healthz. Packaging is not the moment to lose the two endpoints that decide whether "
            "this pod gets traffic and whether it gets restarted — and they must not be the same "
            "path, or every drain restarts the container" % (ready_path, live_path))
    else:
        rec("PASS", "the chart still renders the probes the platform depends on")

    tgps = integer(resolve(podspec.get("terminationGracePeriodSeconds"), exprs, values))
    sleep = integer(resolve(dig(container, "lifecycle", "preStop", "sleep", "seconds"), exprs, values))
    if sleep is None:
        sleep = prestop_seconds(container)
    drain = go_duration(lookup(values, "config.shutdownTimeout")[1])
    if tgps is None or sleep is None:
        rec("FAIL", "the chart still renders the shutdown contract",
            "the rendered pod spec needs terminationGracePeriodSeconds and a preStop hook that "
            "sleeps (this grader resolved %r and %r). Without them the chart ships a service that "
            "drops requests on every rollout, however tidy the packaging is"
            % (tgps, sleep))
    elif drain is None:
        rec("FAIL", "the chart still renders the shutdown contract",
            "config.shutdownTimeout is %r, which time.ParseDuration rejects — the service would "
            "exit at startup. Write a Go duration such as 20s"
            % lookup(values, "config.shutdownTimeout")[1])
    elif tgps < sleep + drain + 5:
        rec("FAIL", "the chart still renders the shutdown contract",
            "preStop %ds + shutdownTimeout %gs needs at least %gs of grace, but the chart renders "
            "terminationGracePeriodSeconds: %d. The hook runs inside the grace period, so these "
            "defaults would get the container SIGKILLed mid-drain — and a bad default in a chart "
            "is a bad default in every installation of it"
            % (sleep, drain, sleep + drain + 5, tgps))
    else:
        rec("PASS", "the chart still renders the shutdown contract "
            "(%ds grace covers %ds preStop + %gs drain)" % (tgps, sleep, drain))

    env = env_of(container)
    gml = resolve(as_dict(env.get("GOMEMLIMIT")).get("value"), exprs, values)
    mem_limit = quantity(dig(resources if isinstance(resources, dict) else {}, "limits", "memory"))
    gml_bytes = gomemlimit_bytes(gml)
    if gml_bytes is None:
        rec("FAIL", "the chart's GOMEMLIMIT still fits under the rendered memory limit",
            "GOMEMLIMIT renders as %r. It has to be a value the Go runtime parses — bytes with an "
            "optional B/KiB/MiB/GiB/TiB suffix, so 110MiB and not Kubernetes' 110Mi — or the "
            "runtime refuses to start" % gml)
    elif mem_limit is None:
        rec("FAIL", "the chart's GOMEMLIMIT still fits under the rendered memory limit",
            "resources.limits.memory did not resolve, so there is nothing for GOMEMLIMIT to sit "
            "under")
    elif not 0.5 * mem_limit <= gml_bytes < mem_limit:
        rec("FAIL", "the chart's GOMEMLIMIT still fits under the rendered memory limit",
            "GOMEMLIMIT %s against a %s limit. The soft limit belongs below the hard one, and not "
            "so far below that you pay for memory the runtime is told never to use — 80-90%% is "
            "the usual range. Two values, one decision: whoever raises the limit has to move both"
            % (human(gml_bytes), human(mem_limit)))
    else:
        rec("PASS", "the chart's GOMEMLIMIT still fits under the rendered memory limit (%s under %s)"
            % (human(gml_bytes), human(mem_limit)))

    cm_doc, cm_exprs = None, []
    for name, (docs, ex, _) in stripped.items():
        for doc in docs:
            if doc.get("kind") == "ConfigMap":
                cm_doc, cm_exprs = doc, ex
    data = as_dict(as_dict(cm_doc).get("data")) if cm_doc else {}
    absent = [k for k in ("PORT", "LOG_LEVEL", "SHUTDOWN_TIMEOUT") if k not in data]
    from_values = [k for k, v in data.items() if templated(v)]
    refs = [restore(dig(as_dict(f), "configMapRef", "name"), exprs)
            for f in as_list(container.get("envFrom"))]
    refs += [restore(dig(e, "valueFrom", "configMapKeyRef", "name"), exprs)
             for e in env_of(container).values()]
    refs = [str(r) for r in refs if r]
    cm_name = restore(dig(cm_doc or {}, "metadata", "name"), cm_exprs)
    defined_in_cm = set(re.findall(r'include\s+"([^"]+)"', str(cm_name)))
    includes = set()
    for ref in refs:
        includes |= set(re.findall(r'include\s+"([^"]+)"', ref)) & defined_in_cm
    if cm_doc is None or absent:
        rec("FAIL", "the chart renders the ConfigMap the service reads",
            "the chart's ConfigMap template is missing %s. timesvc reads PORT, LOG_LEVEL and "
            "SHUTDOWN_TIMEOUT from its environment and refuses to start without them"
            % (", ".join(absent) if absent else "entirely"))
    elif len(from_values) < 3:
        rec("FAIL", "the chart renders the ConfigMap the service reads",
            "only %d of the three ConfigMap values come from .Values.config — the rest are frozen "
            "into the template where no installer can reach them" % len(from_values))
    elif not refs:
        rec("FAIL", "the chart renders the ConfigMap the service reads",
            "the chart renders a ConfigMap that no container reads. Pull it in with envFrom: "
            "- configMapRef: name: … (or per key with configMapKeyRef)")
    elif not includes:
        rec("FAIL", "the chart renders the ConfigMap the service reads",
            "the container names ConfigMap %r while the ConfigMap template renders as %r. Build "
            "both from the same helper: a hard-coded ConfigMap name means two releases of this "
            "chart in one namespace silently share — and overwrite — one object"
            % (refs[0], cm_name))
    else:
        rec("PASS", "the chart renders the ConfigMap the service reads (via %s)"
            % ", ".join(sorted(includes)))

    annotations = as_dict(dig(deployment, "spec", "template", "metadata", "annotations"))
    checksum = [k for k, v in annotations.items() if "sha256sum" in str(restore(v, exprs))]
    if not checksum:
        rec("FAIL", "changing only configuration still rolls the pods",
            "no pod-template annotation hashes the rendered ConfigMap. envFrom values are read "
            "once, at container creation: `helm upgrade` with a new log level would patch the "
            "ConfigMap, report success, leave the pod template byte-identical and therefore change "
            "nothing that is running. Add, under spec.template.metadata.annotations: "
            "checksum/config: {{ include (print $.Template.BasePath \"/configmap.yaml\") . | "
            "sha256sum }}")
    else:
        rec("PASS", "changing only configuration still rolls the pods (annotation %s)" % checksum[0])

    defined = set()
    helpers = os.path.join(TEMPLATES, "_helpers.tpl")
    if os.path.isfile(helpers):
        defined = set(re.findall(
            r'define\s+"([^"]+)"', open(helpers, encoding="utf-8", errors="replace").read()))
    users = [name for name, (_, _, text) in stripped.items()
             if defined & set(re.findall(r'include\s+"([^"]+)"', text))]
    if len(users) < 2:
        rec("FAIL", "names and labels come from the helpers, not from copies",
            "only %d template file(s) include a helper defined in _helpers.tpl. A name repeated in "
            "three templates is a name that will disagree in three templates — and a Deployment's "
            "selector is immutable, so the disagreement is not always fixable in place"
            % len(users))
    else:
        rec("PASS", "names and labels come from the helpers, not from copies (%d template files)"
            % len(users))
    return values


# ------------------------------------------------------------ kustomize checks

KROOT = "kustomize"
BASE = os.path.join(KROOT, "base")
OVERLAYS = {env: os.path.join(KROOT, "overlays", env) for env in ("dev", "prod")}


def check_kustomize():
    if not os.path.isdir(BASE) or not all(os.path.isdir(d) for d in OVERLAYS.values()):
        rec("FAIL", "the kustomization is laid out as a base plus overlays",
            "expected kustomize/base/ and kustomize/overlays/dev|prod next to check.sh. The layout "
            "is the documentation: a reader sees at a glance what is shared and what varies")
        return

    try:
        base_kust, _ = load_kustomization(BASE)
        base_docs = build(BASE)
    except (BuildError, YamlError) as e:
        rec("FAIL", "the base builds on its own", "%s" % e)
        return

    listed = {str(r) for r in as_list(base_kust.get("resources"))}
    on_disk = {f for f in os.listdir(BASE)
               if f.endswith((".yaml", ".yml")) and not f.startswith("kustomization")}
    forgotten = sorted(on_disk - listed)
    kinds = {d.get("kind") for d in base_docs}
    if base_kust.get("kind") != "Kustomization":
        rec("FAIL", "the base builds on its own",
            "kustomization.yaml needs apiVersion: kustomize.config.k8s.io/v1beta1 and "
            "kind: Kustomization. Older kustomize accepted the file without them; every tool that "
            "reads your repo now expects them")
    elif forgotten:
        rec("FAIL", "the base builds on its own",
            "%s is in kustomize/base/ but not in resources:. Kustomize builds what it is told "
            "about and silently ignores the rest — a manifest nobody listed is a manifest nobody "
            "deploys, and nothing warns you" % ", ".join(forgotten))
    elif not {"Deployment", "Service"} <= kinds:
        rec("FAIL", "the base builds on its own",
            "the base builds %s — it needs at least the Deployment and the Service"
            % (", ".join(sorted(str(k) for k in kinds)) or "nothing"))
    else:
        rec("PASS", "the base builds on its own (%d object(s))" % len(base_docs))

    gens = as_list(base_kust.get("configMapGenerator"))
    gen_name = str(dig(as_dict(gens[0] if gens else {}), "name") or "")
    gen_data = as_dict(next((d for d in generated_configmaps(base_kust)), {}).get("data"))
    disabled = str(dig(base_kust, "generatorOptions", "disableNameSuffixHash") or "").lower()
    base_container, _ = container_of(next((d for d in base_docs if d.get("kind") == "Deployment"), {}))
    referenced = {str(dig(as_dict(f), "configMapRef", "name") or "")
                  for f in as_list(base_container.get("envFrom"))}
    missing_keys = [k for k in ("PORT", "LOG_LEVEL", "SHUTDOWN_TIMEOUT") if k not in gen_data]
    if not gens:
        rec("FAIL", "the base generates its ConfigMap and keeps the name hash",
            "there is no configMapGenerator. Generate the ConfigMap instead of checking one in: "
            "kustomize appends a hash of the content to the name and rewrites every reference, so "
            "a changed value produces a new name, a new pod template and a rollout. A checked-in "
            "ConfigMap changes under pods that read their environment once, at startup")
    elif missing_keys:
        rec("FAIL", "the base generates its ConfigMap and keeps the name hash",
            "the generator does not produce %s. Write the three literals as KEY=value entries"
            % ", ".join(missing_keys))
    elif disabled in ("true", "yes"):
        rec("FAIL", "the base generates its ConfigMap and keeps the name hash",
            "generatorOptions.disableNameSuffixHash is true, which buys a stable name and gives up "
            "the only mechanism that makes a config change reach running pods. If you need the "
            "stable name for something else, you now owe the cluster a restart on every config "
            "change — and nobody remembers")
    elif gen_name not in referenced:
        rec("FAIL", "the base generates its ConfigMap and keeps the name hash",
            "the Deployment references ConfigMap %s but the generator is named %r. Kustomize "
            "rewrites references by name; a name it does not recognise is left alone, and the pod "
            "would point at a ConfigMap that does not exist"
            % (sorted(referenced) or "nothing", gen_name))
    else:
        rec("PASS", "the base generates its ConfigMap and keeps the name hash (%s)" % gen_name)

    rendered, kusts = {}, {}
    broken = []
    for env, path in sorted(OVERLAYS.items()):
        try:
            kusts[env], _ = load_kustomization(path)
            rendered[env] = build(path)
        except (BuildError, YamlError) as e:
            broken.append("%s: %s" % (env, e))
    if broken:
        rec("FAIL", "both overlays build on the base",
            "%s. Each overlay's kustomization.yaml lists the base *directory* under resources — "
            "and only strategic-merge patch files referenced with `patches: - path:` are read "
            "here, not inline patch: blocks or JSON 6902 operations" % broken[0])
        return

    bad_ref = []
    for env, path in sorted(OVERLAYS.items()):
        entries = [str(r) for r in as_list(kusts[env].get("resources"))]
        dirs = [e for e in entries
                if os.path.isdir(os.path.normpath(os.path.join(path, e)))]
        files = [e for e in entries if e not in dirs]
        if not dirs:
            bad_ref.append("%s lists no directory under resources" % env)
        elif files:
            bad_ref.append("%s lists %s directly" % (env, files[0]))
    if bad_ref:
        rec("FAIL", "both overlays build on the base",
            "%s. An overlay points at the base directory (../../base) so it inherits the base's "
            "kustomization too — its generators, its namespace, everything it adds later. Listing "
            "the base's manifest files one by one takes the YAML and leaves the reasoning behind"
            % bad_ref[0])
    else:
        rec("PASS", "both overlays build on the base")

    base_deployment = next((d for d in base_docs if d.get("kind") == "Deployment"), {})
    base_size = leaves(base_deployment)
    copies = []
    for env, path in sorted(OVERLAYS.items()):
        for name in sorted(os.listdir(path)):
            if not name.endswith((".yaml", ".yml")) or name.startswith("kustomization"):
                continue
            try:
                docs = read_docs(os.path.join(path, name))
            except YamlError as e:
                copies.append("%s/%s does not parse: %s" % (env, name, e))
                continue
            for doc in docs:
                if has_key(doc, "readinessProbe") or has_key(doc, "livenessProbe") \
                        or has_key(doc, "lifecycle"):
                    copies.append("%s/%s re-declares the probes or the lifecycle hook" % (env, name))
                elif base_size and leaves(doc) > 0.4 * base_size:
                    copies.append("%s/%s carries %d of the base's %d fields"
                                  % (env, name, leaves(doc), base_size))
    if copies:
        rec("FAIL", "the overlays patch the base instead of copying it",
            "%s. A patch carries identity — apiVersion, kind, metadata.name, the container's name "
            "— plus only the fields that differ. Copy the whole Deployment in and the copy is a "
            "fork: the next probe fix lands in the base and reaches neither environment, and "
            "nothing tells you" % copies[0])
    else:
        rec("PASS", "the overlays patch the base instead of copying it")

    ids = {}
    for env in sorted(OVERLAYS):
        k = kusts[env]
        ids[env] = (str(k.get("namespace") or ""), str(k.get("namePrefix") or ""),
                    str(k.get("nameSuffix") or ""))
    if any(i == ("", "", "") for i in ids.values()):
        rec("FAIL", "dev and prod cannot land on top of each other",
            "an overlay sets neither namespace nor namePrefix/nameSuffix, so it builds objects "
            "with the base's names. Apply both to one cluster and the second one silently "
            "overwrites the first")
    elif ids["dev"] == ids["prod"]:
        rec("FAIL", "dev and prod cannot land on top of each other",
            "both overlays build into %s. Give each environment its own namespace (or its own name "
            "prefix): the isolation is the point of having two overlays" % (ids["dev"],))
    else:
        rec("PASS", "dev and prod cannot land on top of each other (%s vs %s)"
            % (ids["dev"][0] or ids["dev"][1], ids["prod"][0] or ids["prod"][1]))

    deployments = {env: next((d for d in docs if d.get("kind") == "Deployment"), {})
                   for env, docs in rendered.items()}
    empty = sorted(env for env, doc in deployments.items() if not doc)
    if empty:
        why = ("the %s overlay builds no Deployment at all. Its resources: list has to reach the "
               "base — everything else in an overlay only makes sense once the base is in it"
               % empty[0])
        rec("FAIL", "the two overlays render different environments", why)
        rec("FAIL", "both rendered overlays keep everything the last lesson established", why)
        return

    facts = {}
    for env, doc in deployments.items():
        container, podspec = container_of(doc)
        res = as_dict(container.get("resources"))
        facts[env] = {
            "replicas": integer(dig(doc, "spec", "replicas")),
            "cpu": dig(res, "limits", "cpu"),
            "memory": dig(res, "limits", "memory"),
            "container": container,
            "podspec": podspec,
        }
    if any(f["replicas"] is None for f in facts.values()):
        rec("FAIL", "the two overlays render different environments",
            "one of the overlays renders a Deployment with no replicas. Use the typed field — "
            "replicas: [{name: timesvc, count: 1}] — or a patch that sets spec.replicas")
    elif facts["dev"]["replicas"] == facts["prod"]["replicas"]:
        rec("FAIL", "the two overlays render different environments",
            "both environments render replicas: %s. Dev does not need production's capacity and "
            "production does not survive on dev's" % facts["dev"]["replicas"])
    elif (facts["dev"]["cpu"], facts["dev"]["memory"]) == (facts["prod"]["cpu"], facts["prod"]["memory"]):
        rec("FAIL", "the two overlays render different environments",
            "both environments render the same resource limits (cpu %s, memory %s). The limits are "
            "the other half of what an environment *is*: a dev pod that reserves production's "
            "memory keeps everyone else off the node"
            % (facts["dev"]["cpu"], facts["dev"]["memory"]))
    else:
        rec("PASS", "the two overlays render different environments (replicas %s/%s, cpu limit %s/%s)"
            % (facts["dev"]["replicas"], facts["prod"]["replicas"],
               facts["dev"]["cpu"], facts["prod"]["cpu"]))

    lost = []
    for env in sorted(facts):
        container, podspec = facts[env]["container"], facts[env]["podspec"]
        ready_path, live_path = probe_paths(container)
        tgps = integer(podspec.get("terminationGracePeriodSeconds"))
        sleep = prestop_seconds(container)
        image = str(container.get("image") or "")
        tail = image.rsplit("/", 1)[-1]
        tag = tail.rsplit(":", 1)[1] if ":" in tail else ""
        gml = gomemlimit_bytes(as_dict(env_of(container).get("GOMEMLIMIT")).get("value"))
        mem = quantity(dig(as_dict(container.get("resources")), "limits", "memory"))
        if ready_path != "/readyz" or live_path != "/healthz":
            lost.append("%s renders readiness %r and liveness %r" % (env, ready_path, live_path))
        elif tgps is None or sleep is None:
            lost.append("%s renders terminationGracePeriodSeconds %r and a preStop sleep of %r"
                        % (env, tgps, sleep))
        elif "@sha256:" not in image and tag in ("", "latest"):
            lost.append("%s renders image %r" % (env, image))
        elif gml is None or mem is None or not 0.5 * mem <= gml < mem:
            lost.append("%s renders GOMEMLIMIT %s against a memory limit of %s"
                        % (env, human(gml) if gml else "nothing", human(mem) if mem else "nothing"))
    if lost:
        rec("FAIL", "both rendered overlays keep everything the last lesson established",
            "%s. An overlay is allowed to change how much of something an environment gets — never "
            "whether the service can drain, be probed, or stay inside its memory limit. Raising "
            "limits.memory in a patch without moving GOMEMLIMIT with it is the classic one: the "
            "manifest looks generous and the pod still gets OOM-killed at the old number" % lost[0])
    else:
        rec("PASS", "both rendered overlays keep everything the last lesson established")


# ---------------------------------------------------------------- NOTES.md


def check_notes():
    if not os.path.exists("NOTES.md"):
        rec("FAIL", "NOTES.md answers all three questions",
            "NOTES.md is missing — restore it from the exercise and answer on the A: lines")
        return
    notes = open("NOTES.md", encoding="utf-8", errors="replace").read()
    answers, current = [], None
    for line in notes.splitlines():
        if line.startswith("A:"):
            current = [line[2:].strip()]
            answers.append(current)
        elif current is not None and (line.startswith("#") or line.startswith("A:")):
            current = None
        elif current is not None:
            current.append(line.strip())
    answers = [" ".join(part for part in a if part).strip() for a in answers]
    real = [a for a in answers if len(a) >= 80 and "TODO" not in a]
    if len(real) < 3:
        rec("FAIL", "NOTES.md answers all three questions",
            "only %d of 3 A: lines carry a real answer (80+ characters, no TODO left). Q1 is the "
            "one that matters most: the choice, argued from what each tool does" % len(real))
    else:
        rec("PASS", "NOTES.md answers all three questions")

    low = "\n".join(answers).lower()
    terms = {
        "templating or templates": ("templat",),
        "patching or overlays": ("patch", "overlay"),
        "values.yaml as the chart's API": ("values",),
        "rendering before applying (helm template / kubectl kustomize)": (
            "helm template", "kubectl kustomize", "kustomize build", "render"),
        "the release lifecycle (install/upgrade/rollback)": ("rollback", "release", "revision"),
        "the config-change mechanism (hash suffix or checksum annotation)": (
            "hash", "checksum", "suffix"),
    }
    absent = [name for name, needles in terms.items() if not any(n in low for n in needles)]
    if absent:
        rec("FAIL", "NOTES.md argues the trade-off in both directions",
            "the write-up never mentions %s. A preference you cannot state in the other tool's "
            "vocabulary is a habit, not a decision — and the next team you join will have picked "
            "the other one" % absent[0])
    else:
        rec("PASS", "NOTES.md argues the trade-off in both directions")


def main():
    check_chart()
    check_kustomize()
    check_notes()


try:
    main()
except Exception as exc:  # a crash here is an exercise bug, not a learner bug
    sys.stderr.write("static grader crashed: %r\n" % (exc,))
    sys.exit(2)
sys.stdout.write("\n".join(OUT) + "\n")
PY

if [ $? -ne 0 ]; then
	echo "check.sh: the static grader crashed. That is a bug in the exercise, not in your file."
	exit 2
fi

# ---------------------------------------------------------------- tooling (bonus)
# Everything below is optional. Missing tooling is reported SKIPPED, never
# FAILED: the grade above already stands on its own.

if command -v helm >/dev/null 2>&1; then
	if out="$(helm lint chart 2>&1)"; then
		record PASS "helm: helm lint accepts the chart"
	else
		record FAIL "helm: helm lint accepts the chart" \
			"$(flatten "$out")"
	fi
	if out="$(helm template timesvc chart 2>&1)"; then
		record PASS "helm: the chart renders (helm template)"
	else
		record FAIL "helm: the chart renders (helm template)" \
			"Helm could not render the templates — this is the error message to read line by \
line, it names the file and the line: $(flatten "$out")"
	fi
else
	record SKIP "helm: helm lint accepts the chart" \
		"no helm on PATH. It is a single binary and worth having: helm.sh/docs/intro/install"
	record SKIP "helm: the chart renders (helm template)" \
		"no helm on PATH, so nothing rendered the chart for real"
fi

kbuild=""
if command -v kustomize >/dev/null 2>&1; then
	kbuild="kustomize build"
elif command -v kubectl >/dev/null 2>&1; then
	kbuild="kubectl kustomize"
fi
if [ -n "$kbuild" ]; then
	built=1
	for env in dev prod; do
		if out="$($kbuild "kustomize/overlays/$env" 2>&1)"; then
			record PASS "kustomize: the $env overlay builds ($kbuild)"
		else
			built=0
			record FAIL "kustomize: the $env overlay builds ($kbuild)" \
				"$(flatten "$out")"
		fi
	done
	if [ "$built" -eq 1 ] && command -v kubectl >/dev/null 2>&1; then
		if out="$($kbuild kustomize/overlays/prod | kubectl apply --dry-run=client -f - 2>&1)"; then
			record PASS "kubectl: the built prod overlay survives a client-side dry run"
		elif printf '%s' "$out" | grep -qiE 'connection refused|unable to connect|no configuration has been provided|couldn.t get current server'; then
			record SKIP "kubectl: the built prod overlay survives a client-side dry run" \
				"kubectl is installed but has no usable kubeconfig: $(flatten "$out")"
		else
			record FAIL "kubectl: the built prod overlay survives a client-side dry run" \
				"kubectl rejected the built manifests: $(flatten "$out")"
		fi
	else
		record SKIP "kubectl: the built prod overlay survives a client-side dry run" \
			"$(command -v kubectl >/dev/null 2>&1 && echo 'the overlays did not build, so there is nothing to validate' || echo 'no kubectl on PATH')"
	fi
else
	record SKIP "kustomize: the dev overlay builds" \
		"neither kustomize nor kubectl is on PATH (kubectl has kustomize built in: kubectl kustomize)"
	record SKIP "kustomize: the prod overlay builds" "same reason"
	record SKIP "kubectl: the built prod overlay survives a client-side dry run" "no kubectl on PATH"
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
