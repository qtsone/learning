#!/usr/bin/env bash
# Referee for the Kubernetes Configuration exercise.
#
# The grade comes from reading the manifests you wrote: no cluster, no kubectl,
# no Docker, no network, no root. If a kubectl and a reachable cluster happen to
# be here, two dry-run checks are added as a bonus; when they are not, those are
# reported SKIPPED, never failed.
#
# Run from this folder:  bash check.sh
set -u
cd "$(dirname "$0")"

results="$(mktemp)"
trap 'rm -f "$results"' EXIT

record() { printf '%s|%s|%s\n' "$1" "$2" "${3:-}" >>"$results"; }

# One line, no pipes: the report loop splits records on '|'.
flatten() { printf '%s' "$1" | tr '\n|' '  ' | cut -c1-220; }

# ---------------------------------------------------------------- static
# python3 is guaranteed here (the tutor engine itself is Python). PyYAML is
# not, so the grader carries a small YAML reader that covers the manifest
# subset this exercise asks for: block mappings, block lists, quoted scalars,
# simple inline [a, b] / {k: v}, and | block scalars. No anchors, no aliases.

python3 - >>"$results" <<'PY'
import base64
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


def tokenize(lines, first):
    toks = []
    for offset, raw in enumerate(lines):
        n = first + offset
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
                if nxt is None:
                    out[key] = None
                elif nxt[0] > indent:
                    out[key] = self.block(nxt[0])
                elif nxt[0] == indent and (nxt[1] == "-" or nxt[1].startswith("- ")):
                    # A block list may sit at its key's own indentation.
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
            item = tok[1][1:].strip()
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
                inner = [(indent + 2, item, tok[2])] + sub
                out.append(Parser(inner).block(indent + 2))
            else:
                out.append(flow(item, tok[2]))
        return out


def parse_documents(text):
    """Every YAML document in one file, '---' separated. Empty ones are dropped."""
    docs, chunk, first = [], [], 1
    for n, raw in enumerate(text.splitlines(), 1):
        if raw.rstrip() == "---":
            docs.append((first, chunk))
            chunk, first = [], n + 1
        else:
            chunk.append(raw)
    docs.append((first, chunk))
    out = []
    for start, lines in docs:
        toks = tokenize(lines, start)
        if not toks:
            continue
        doc = Parser(toks).block(toks[0][0])
        if doc is None:
            continue
        if not isinstance(doc, dict):
            raise YamlError("line %d: a manifest must be a mapping (apiVersion, kind, ...)" % start)
        out.append(doc)
    return out


# ---------------------------------------------------------------- helpers


def as_dict(v):
    return v if isinstance(v, dict) else {}


def as_list(v):
    if v is None:
        return []
    return v if isinstance(v, list) else [v]


def dig(obj, *path):
    cur = obj
    for key in path:
        cur = as_dict(cur).get(key)
    return cur


def num(v):
    try:
        return float(str(v).strip())
    except (TypeError, ValueError):
        return None


SUFFIX = [
    ("Ki", 1024.0), ("Mi", 1024.0 ** 2), ("Gi", 1024.0 ** 3),
    ("Ti", 1024.0 ** 4), ("Pi", 1024.0 ** 5), ("Ei", 1024.0 ** 6),
    ("m", 1e-3), ("k", 1e3), ("M", 1e6), ("G", 1e9),
    ("T", 1e12), ("P", 1e15), ("E", 1e18),
]


def quantity(v):
    """A Kubernetes resource quantity as a float: 500m, 1, 128Mi, 1.5Gi, 2e3."""
    if v is None:
        return None
    s = str(v).strip()
    if not s:
        return None
    for suffix, factor in SUFFIX:
        if s.endswith(suffix) and len(s) > len(suffix):
            head = num(s[: -len(suffix)])
            return None if head is None else head * factor
    return num(s)


def manifest_files():
    out = []
    for root, dirs, files in os.walk("."):
        dirs[:] = sorted(d for d in dirs if not d.startswith(".") and d != "app")
        for name in sorted(files):
            if name.endswith((".yaml", ".yml")):
                out.append(os.path.join(root, name).lstrip("./"))
    return out


def container_ports(container):
    names, numbers = set(), set()
    for p in as_list(container.get("ports")):
        p = as_dict(p)
        if p.get("name"):
            names.add(str(p["name"]))
        if p.get("containerPort") is not None:
            numbers.add(str(p["containerPort"]))
    return names, numbers


def env_entries(container):
    return [as_dict(e) for e in as_list(container.get("env"))]


def probe_http(probe):
    return as_dict(as_dict(probe).get("httpGet"))


def probe_budget(probe):
    period = num(as_dict(probe).get("periodSeconds")) or 10.0
    threshold = num(as_dict(probe).get("failureThreshold")) or 3.0
    delay = num(as_dict(probe).get("initialDelaySeconds")) or 0.0
    return delay + period * threshold


LIVENESS_PATHS = ("/healthz", "/livez")
READINESS_PATH = "/readyz"


# ---------------------------------------------------------------- checks


def main():
    files = manifest_files()
    if not files:
        rec("FAIL", "the manifests parse as YAML",
            "no .yaml files found — write your manifests next to check.sh")
        return

    docs = []
    for path in files:
        text = open(path, encoding="utf-8", errors="replace").read()
        try:
            for doc in parse_documents(text):
                docs.append((path, doc))
        except YamlError as exc:
            rec("FAIL", "the manifests parse as YAML",
                "%s: %s. Indent two spaces per level, spaces only, and keep a space after "
                "every colon" % (path, exc))
            return
    rec("PASS", "the manifests parse as YAML")

    broken = []
    for path, doc in docs:
        missing = [k for k in ("apiVersion", "kind") if not doc.get(k)]
        if not dig(doc, "metadata", "name"):
            missing.append("metadata.name")
        if missing:
            broken.append("%s (%s)" % (path, doc.get("kind") or "no kind"))
    if broken:
        rec("FAIL", "every object declares apiVersion, kind and metadata.name",
            "incomplete objects: %s. The API server has no defaults for those three — an object "
            "without them is not addressable" % ", ".join(sorted(set(broken))))
    else:
        rec("PASS", "every object declares apiVersion, kind and metadata.name")

    spaces = {}
    for path, doc in docs:
        if doc.get("kind") == "Namespace":
            continue
        spaces.setdefault(str(dig(doc, "metadata", "namespace") or "(none)"), []).append(
            "%s/%s" % (doc.get("kind"), dig(doc, "metadata", "name")))
    if len(spaces) > 1:
        rec("FAIL", "every object lives in the same namespace",
            "%s. A pod can only reference a ConfigMap or Secret in its own namespace, so a "
            "Deployment in one and its config in another never starts — it waits in "
            "CreateContainerConfigError"
            % "; ".join("%s: %s" % (ns, ", ".join(sorted(objs))) for ns, objs in sorted(spaces.items())))
    else:
        rec("PASS", "every object lives in the same namespace")

    def of_kind(kind):
        return [d for _, d in docs if d.get("kind") == kind]

    deploys = of_kind("Deployment")
    if not deploys:
        rec("FAIL", "the Deployment is still here",
            "no Deployment object found — deployment.yaml is the file you extend in this lesson")
        return
    deploy = deploys[0]
    dname = dig(deploy, "metadata", "name")
    pod = as_dict(dig(deploy, "spec", "template", "spec"))
    containers = [as_dict(c) for c in as_list(pod.get("containers"))]
    if not containers:
        rec("FAIL", "the Deployment still declares its container",
            "spec.template.spec.containers is empty — restore the container block from the starter")
        return
    app = next((c for c in containers if c.get("name") == "timesvc"), containers[0])
    port_names, port_numbers = container_ports(app)

    # ------------------------------------------------------------ ConfigMap
    cms = of_kind("ConfigMap")
    cm = cms[0] if cms else None
    cm_name = dig(cm, "metadata", "name") if cm else None
    cm_data = as_dict(dig(cm, "data")) if cm else {}
    wanted_env = ["LOG_LEVEL", "NOW_CACHE_TTL"]
    if cm is None:
        rec("FAIL", "a ConfigMap carries the service's non-secret settings",
            "no ConfigMap object found — fill in configmap.yaml with a data: block")
    else:
        missing = [k for k in wanted_env if not str(cm_data.get(k) or "").strip()]
        if missing:
            rec("FAIL", "a ConfigMap carries the service's non-secret settings",
                "ConfigMap %s has no value for %s — those are the keys main.go reads from the "
                "environment" % (cm_name, " and ".join(missing)))
        else:
            rec("PASS", "a ConfigMap carries the service's non-secret settings")

    file_keys = [k for k, v in cm_data.items() if "." in k and len(str(v or "").strip()) >= 3]
    if cm is None:
        rec("FAIL", "the ConfigMap also carries a whole config file",
            "add a features.yaml key holding a few lines of YAML (features.yaml: | then an "
            "indented body) — a key whose name looks like a filename becomes a file when mounted")
    elif "features.yaml" not in file_keys:
        rec("FAIL", "the ConfigMap also carries a whole config file",
            "features.yaml is missing or empty in ConfigMap %s. Use a block scalar: "
            "'features.yaml: |' and then two-space-indented lines. Mounted, that key becomes the "
            "file main.go re-reads on every request" % cm_name)
    else:
        rec("PASS", "the ConfigMap also carries a whole config file")

    # ------------------------------------------------------- env injection
    from_cm = set()
    for src in as_list(app.get("envFrom")):
        ref = as_dict(as_dict(src).get("configMapRef"))
        if ref.get("name") == cm_name:
            from_cm.update(cm_data.keys())
    for entry in env_entries(app):
        ref = as_dict(dig(entry, "valueFrom", "configMapKeyRef"))
        if ref.get("name") == cm_name and entry.get("name"):
            from_cm.add(str(entry["name"]))
    if cm is None:
        pass
    elif all(k in from_cm for k in wanted_env):
        rec("PASS", "the container reads its settings from the ConfigMap")
    else:
        rec("FAIL", "the container reads its settings from the ConfigMap",
            "%s never reaches the container. Import the whole ConfigMap with "
            "envFrom: - configMapRef: name: %s, or name each key with "
            "env.valueFrom.configMapKeyRef. A value typed straight into the Deployment is not "
            "configuration, it is a redeploy"
            % (" and ".join(k for k in wanted_env if k not in from_cm), cm_name))

    # -------------------------------------------------------- ConfigMap file
    cm_volumes = set()
    for vol in as_list(pod.get("volumes")):
        vol = as_dict(vol)
        if as_dict(vol.get("configMap")).get("name") == cm_name and vol.get("name"):
            cm_volumes.add(str(vol["name"]))
    mounts = [as_dict(m) for m in as_list(app.get("volumeMounts"))]
    cm_mounts = [m for m in mounts if str(m.get("name")) in cm_volumes]
    good_paths = ("/etc/timesvc/config", "/etc/timesvc/config/features.yaml")
    if cm is None:
        pass
    elif not cm_volumes:
        rec("FAIL", "features.yaml is mounted into the container as a file",
            "no volume is backed by ConfigMap %s. Add one under spec.template.spec.volumes: "
            "- name: features / configMap: name: %s" % (cm_name, cm_name))
    elif not cm_mounts:
        rec("FAIL", "features.yaml is mounted into the container as a file",
            "the volume exists but no container mounts it — add a volumeMount with the same "
            "name and mountPath: /etc/timesvc/config")
    elif not any(str(m.get("mountPath", "")).rstrip("/") in good_paths for m in cm_mounts):
        rec("FAIL", "features.yaml is mounted into the container as a file",
            "mounted at %s; main.go reads $CONFIG_DIR/features.yaml, default "
            "/etc/timesvc/config/features.yaml. Either mount the ConfigMap at "
            "/etc/timesvc/config or set CONFIG_DIR to where you mounted it"
            % ", ".join(str(m.get("mountPath")) for m in cm_mounts))
    else:
        rec("PASS", "features.yaml is mounted into the container as a file")

    # --------------------------------------------------------------- Secret
    secrets = of_kind("Secret")
    secret = secrets[0] if secrets else None
    s_name = dig(secret, "metadata", "name") if secret else None
    s_string = as_dict(dig(secret, "stringData")) if secret else {}
    s_data = as_dict(dig(secret, "data")) if secret else {}
    api_key = str(s_string.get("API_KEY") or "").strip()
    encoded = str(s_data.get("API_KEY") or "").strip()
    undecodable = False
    if encoded and not api_key:
        try:
            api_key = base64.b64decode(encoded, validate=True).decode("utf-8").strip()
        except Exception:
            api_key, undecodable = "", True
    if secret is None:
        rec("FAIL", "a Secret carries the API key",
            "no Secret object found — secret.yaml needs an API_KEY under stringData")
    elif undecodable:
        rec("FAIL", "a Secret carries the API key",
            "data.API_KEY in Secret %s is not valid base64. data: wants base64; stringData: "
            "takes plain text and the API server encodes it for you — which is exactly how much "
            "protection that encoding is" % s_name)
    elif not api_key:
        rec("FAIL", "a Secret carries the API key",
            "Secret %s has no API_KEY. Put it under stringData: so you can still read what you "
            "wrote" % s_name)
    elif api_key.lower() in ("change-me", "changeme", "todo", "secret", "password"):
        rec("FAIL", "a Secret carries the API key",
            "API_KEY is still a placeholder (%r) — invent a value. Placeholders survive into "
            "production more often than anyone admits" % api_key)
    else:
        rec("PASS", "a Secret carries the API key")

    referenced, literal = False, []
    for src in as_list(app.get("envFrom")):
        if as_dict(as_dict(src).get("secretRef")).get("name") == s_name:
            referenced = True
    for entry in env_entries(app):
        ref = as_dict(dig(entry, "valueFrom", "secretKeyRef"))
        if entry.get("name") == "API_KEY" and ref.get("name") == s_name:
            referenced = True
        value = str(entry.get("value") or "")
        if entry.get("name") == "API_KEY" and value:
            literal.append("env API_KEY")
        elif api_key and value and value == api_key:
            literal.append("env %s" % entry.get("name"))
    for vol in as_list(pod.get("volumes")):
        if as_dict(as_dict(vol).get("secret")).get("secretName") == s_name:
            referenced = True
    if secret is None:
        pass
    elif literal:
        rec("FAIL", "the API key reaches the container by reference, not by value",
            "the Deployment contains the key itself (%s). deployment.yaml is the file everyone "
            "reads, diffs and pastes into a ticket — reference it with "
            "valueFrom.secretKeyRef: name: %s, key: API_KEY" % (", ".join(literal), s_name))
    elif referenced:
        rec("PASS", "the API key reaches the container by reference, not by value")
    else:
        rec("FAIL", "the API key reaches the container by reference, not by value",
            "nothing in the pod refers to Secret %s. Add env: - name: API_KEY / valueFrom: "
            "secretKeyRef: name: %s, key: API_KEY (or envFrom.secretRef, or a secret volume)"
            % (s_name, s_name))

    # --------------------------------------------------------------- probes
    ready = as_dict(app.get("readinessProbe"))
    live = as_dict(app.get("livenessProbe"))
    start = as_dict(app.get("startupProbe"))
    ready_path = str(probe_http(ready).get("path") or "")
    live_path = str(probe_http(live).get("path") or "")
    start_path = str(probe_http(start).get("path") or "")

    def port_ok(probe):
        port = str(probe_http(probe).get("port") or "")
        return port in port_names or port in port_numbers

    if not ready:
        rec("FAIL", "the readiness probe asks the dependency-aware endpoint",
            "no readinessProbe. Without one every pod is 'ready' the instant the process starts, "
            "so a rolling update sends traffic to a container that is still warming up. Add "
            "readinessProbe.httpGet with path: %s, port: http" % READINESS_PATH)
    elif ready_path != READINESS_PATH:
        rec("FAIL", "the readiness probe asks the dependency-aware endpoint",
            "readiness points at %r; main.go answers readiness on %s, the endpoint that knows "
            "about warm-up and about dependencies it cannot serve without"
            % (ready_path or "no httpGet path", READINESS_PATH))
    elif not port_ok(ready):
        rec("FAIL", "the readiness probe asks the dependency-aware endpoint",
            "readinessProbe.httpGet.port is %r, which is not a port this container declares. Use "
            "the port name 'http' (or 8080) — names survive a port change"
            % str(probe_http(ready).get("port") or ""))
    else:
        rec("PASS", "the readiness probe asks the dependency-aware endpoint")

    if not live:
        rec("FAIL", "the liveness probe asks a process-local endpoint",
            "no livenessProbe. Nothing then notices a wedged process that still holds its socket "
            "open. Add livenessProbe.httpGet with path: /healthz, port: http")
    elif live_path and live_path == ready_path:
        pass  # the dedicated check below owns this one
    elif live_path not in LIVENESS_PATHS:
        rec("FAIL", "the liveness probe asks a process-local endpoint",
            "liveness points at %r; main.go answers liveness on /healthz, which touches nothing "
            "outside the process. Liveness must only answer 'is this process still able to "
            "serve', because the only remedy it has is killing the container"
            % (live_path or "no httpGet path"))
    elif not port_ok(live):
        rec("FAIL", "the liveness probe asks a process-local endpoint",
            "livenessProbe.httpGet.port is %r, which is not a port this container declares"
            % str(probe_http(live).get("port") or ""))
    else:
        rec("PASS", "the liveness probe asks a process-local endpoint")

    if live and ready and live_path and live_path == ready_path:
        rec("FAIL", "the liveness probe is not pointed at the readiness endpoint",
            "both probes hit %s. When the database blips, every replica fails liveness at once "
            "and the kubelet restarts them all — losing warm caches and connection pools, and "
            "hammering the recovering dependency with reconnects while readiness would have just "
            "held traffic back until it came back. Liveness: /healthz. Readiness: /readyz"
            % live_path)
    elif live and ready:
        rec("PASS", "the liveness probe is not pointed at the readiness endpoint")

    budget = probe_budget(start)
    if not start:
        rec("FAIL", "a startup probe gives the process a boot budget",
            "no startupProbe. Without one you must slow the liveness probe down to survive the "
            "worst boot you ever have, which is the same as having no liveness probe for that "
            "long. Add startupProbe.httpGet path: /healthz with failureThreshold and "
            "periodSeconds multiplying out to at least 30s")
    elif start_path == READINESS_PATH:
        rec("FAIL", "a startup probe gives the process a boot budget",
            "the startup probe points at %s. It gates liveness, not traffic — point it at the "
            "same cheap endpoint as liveness (%s)" % (READINESS_PATH, LIVENESS_PATHS[0]))
    elif not start_path:
        rec("FAIL", "a startup probe gives the process a boot budget",
            "startupProbe has no httpGet.path — give it path: /healthz, port: http")
    elif budget < 30:
        rec("FAIL", "a startup probe gives the process a boot budget",
            "the startup budget is only %ds (initialDelaySeconds + periodSeconds x "
            "failureThreshold). Make it at least 30s: that number is your answer to 'how slow "
            "can this process legitimately be on its worst day'" % int(budget))
    else:
        rec("PASS", "a startup probe gives the process a boot budget")

    vague = []
    for label, probe in (("liveness", live), ("readiness", ready)):
        if not probe:
            continue
        for field in ("periodSeconds", "failureThreshold"):
            if probe.get(field) is None:
                vague.append("%s.%s" % (label, field))
    if not live or not ready:
        pass
    elif vague:
        rec("FAIL", "the probes state their own timings",
            "missing %s. The defaults (period 10s, failureThreshold 3, timeout 1s) are real "
            "numbers with real consequences — 30 seconds of downtime before a restart, and a 1s "
            "timeout that fails under load. Write them down so they are a decision"
            % ", ".join(vague))
    else:
        rec("PASS", "the probes state their own timings")

    # ------------------------------------------------------------ resources
    requests = as_dict(dig(app, "resources", "requests"))
    limits = as_dict(dig(app, "resources", "limits"))
    need = []
    if quantity(requests.get("cpu")) is None:
        need.append("requests.cpu")
    if quantity(requests.get("memory")) is None:
        need.append("requests.memory")
    if quantity(limits.get("cpu")) is None:
        need.append("limits.cpu")
    if quantity(limits.get("memory")) is None:
        need.append("limits.memory")
    if need:
        rec("FAIL", "the container declares what it needs and what it may take",
            "missing %s. Requests are what the scheduler subtracts from a node's capacity to "
            "decide where this pod fits; a pod with none is BestEffort and is evicted first. "
            "A memory limit is the line above which the kernel OOM-kills the container, and a "
            "CPU limit is the ceiling the CFS quota throttles you against — set it generously "
            "rather than leaving it out"
            % ", ".join(need))
    else:
        rec("PASS", "the container declares what it needs and what it may take")

    inverted = []
    for key in ("cpu", "memory"):
        req, lim = quantity(requests.get(key)), quantity(limits.get(key))
        if req is not None and lim is not None and lim < req:
            inverted.append("%s: limit %s < request %s" % (key, limits.get(key), requests.get(key)))
    if inverted:
        rec("FAIL", "every limit is at least its request",
            "%s. The API server rejects this outright: a request is a floor the scheduler "
            "guarantees, so a lower ceiling is a contradiction" % "; ".join(inverted))
    else:
        rec("PASS", "every limit is at least its request")

    # ----------------------------------------------------------------- HPA
    hpas = of_kind("HorizontalPodAutoscaler")
    hpa = hpas[0] if hpas else None
    target = as_dict(dig(hpa, "spec", "scaleTargetRef")) if hpa else {}
    api = str(hpa.get("apiVersion") or "") if hpa else ""
    minr, maxr = num(dig(hpa, "spec", "minReplicas")), num(dig(hpa, "spec", "maxReplicas"))
    if hpa is None:
        rec("FAIL", "an HPA targets this Deployment",
            "no HorizontalPodAutoscaler found — write hpa.yaml with apiVersion "
            "autoscaling/v2 and a scaleTargetRef pointing at Deployment %s" % dname)
    elif not api.startswith("autoscaling/v2"):
        rec("FAIL", "an HPA targets this Deployment",
            "apiVersion is %r. autoscaling/v1 only understands a CPU percentage field and the "
            "v2beta versions are gone — use autoscaling/v2" % api)
    elif target.get("kind") != "Deployment" or target.get("name") != dname:
        rec("FAIL", "an HPA targets this Deployment",
            "scaleTargetRef points at %s/%s; it must name kind: Deployment and name: %s exactly. "
            "An HPA that resolves nothing simply reports a scaling error nobody reads"
            % (target.get("kind"), target.get("name"), dname))
    elif minr is None or maxr is None or minr < 2 or maxr <= minr:
        rec("FAIL", "an HPA targets this Deployment",
            "minReplicas/maxReplicas are %s/%s. Use minReplicas of at least 2 (one replica means "
            "every node drain is an outage) and a maxReplicas above it that you would actually "
            "be willing to pay for" % (dig(hpa, "spec", "minReplicas"), dig(hpa, "spec", "maxReplicas")))
    else:
        rec("PASS", "an HPA targets this Deployment")

    cpu_metric = None
    for metric in as_list(dig(hpa, "spec", "metrics")) if hpa else []:
        metric = as_dict(metric)
        resource = as_dict(metric.get("resource"))
        if metric.get("type") == "Resource" and resource.get("name") == "cpu":
            cpu_metric = resource
    utilization = num(dig(cpu_metric, "target", "averageUtilization")) if cpu_metric else None
    if hpa is None:
        pass
    elif cpu_metric is None:
        rec("FAIL", "the HPA scales on CPU utilization of the request",
            "no metrics entry with type: Resource and resource.name: cpu. That is the metric "
            "every cluster can serve out of the box, via metrics-server")
    elif dig(cpu_metric, "target", "type") != "Utilization":
        rec("FAIL", "the HPA scales on CPU utilization of the request",
            "target.type is %r. AverageValue targets a raw core figure; Utilization is a "
            "percentage of the CPU *request*, which is what makes the target portable across "
            "pod sizes" % dig(cpu_metric, "target", "type"))
    elif utilization is None or not 1 <= utilization <= 100:
        rec("FAIL", "the HPA scales on CPU utilization of the request",
            "averageUtilization is %s; give it a percentage between 1 and 100 (70 is a common "
            "starting point: high enough to be efficient, low enough to leave room while new "
            "pods start)" % dig(cpu_metric, "target", "averageUtilization"))
    elif quantity(requests.get("cpu")) is None:
        rec("FAIL", "the HPA scales on CPU utilization of the request",
            "the HPA asks for %d%% of the CPU request, and the container has no CPU request to "
            "take a percentage of. The HPA would report targets as <unknown> and never scale"
            % int(utilization))
    else:
        rec("PASS", "the HPA scales on CPU utilization of the request")

    replicas = num(dig(deploy, "spec", "replicas"))
    if hpa is None or minr is None or maxr is None:
        pass
    elif replicas is None:
        rec("FAIL", "the Deployment's replica count sits inside the HPA's range",
            "spec.replicas is unset, so the first apply creates 1 pod, below your minReplicas "
            "of %d. Set it to a value in [%d, %d] and remember the HPA owns it from then on"
            % (int(minr), int(minr), int(maxr)))
    elif not minr <= replicas <= maxr:
        rec("FAIL", "the Deployment's replica count sits inside the HPA's range",
            "spec.replicas is %d but the HPA allows [%d, %d]. On apply the two controllers "
            "immediately disagree, and every re-apply of this file drags the count back"
            % (int(replicas), int(minr), int(maxr)))
    else:
        rec("PASS", "the Deployment's replica count sits inside the HPA's range")

    # ---------------------------------------------------------------- NOTES
    if not os.path.exists("NOTES.md"):
        rec("FAIL", "NOTES.md answers all three questions",
            "NOTES.md is missing — restore it from the exercise and answer on the A: lines")
    else:
        notes = open("NOTES.md", encoding="utf-8", errors="replace").read()
        answered = len(re.findall(r"(?m)^A:.{40,}$", notes))
        if "TODO" in notes:
            rec("FAIL", "NOTES.md answers all three questions",
                "NOTES.md still contains a TODO placeholder — replace every one with your answer")
        elif answered < 3:
            rec("FAIL", "NOTES.md answers all three questions",
                "only %d of 3 A: lines carry a real answer (40+ characters). Write the env/file "
                "trade-off, what a Secret really protects, and the liveness/readiness trace in "
                "your own words" % answered)
        else:
            rec("PASS", "NOTES.md answers all three questions")


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

# ---------------------------------------------------------------- live (bonus)
# Everything below is optional. Missing tooling is reported SKIPPED, never
# FAILED: the grade above already stands on its own.

client_check="live: kubectl accepts the manifests (client dry-run)"
server_check="live: the API server accepts the manifests (server dry-run)"

if ! command -v kubectl >/dev/null 2>&1; then
	reason="kubectl is not installed here. Install kubectl and a local cluster (kind is one command) to get these two"
	record SKIP "$client_check" "$reason"
	record SKIP "$server_check" "$reason"
elif ! kubectl cluster-info --request-timeout=5s >/dev/null 2>&1; then
	reason="kubectl is here but no cluster answered. Start one: kind create cluster --name tutor"
	record SKIP "$client_check" "$reason"
	record SKIP "$server_check" "$reason"
else
	if out="$(kubectl apply --dry-run=client -f . 2>&1)"; then
		record PASS "$client_check"
	else
		record FAIL "$client_check" "kubectl rejected a manifest: $(flatten "$out")"
	fi
	if out="$(kubectl apply --dry-run=server -f . 2>&1)"; then
		record PASS "$server_check"
	else
		record FAIL "$server_check" \
			"the API server rejected a manifest (nothing was created): $(flatten "$out")"
	fi
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
	esac
done <"$results"

echo
if [ "$fail" -eq 0 ]; then
	printf '%d checks passed, 0 failed, %d skipped for missing tooling.\n' "$pass" "$skip"
	if [ "$skip" -gt 0 ]; then
		echo "The skipped checks are a bonus, not part of the grade — your manifests pass."
	fi
	exit 0
fi
printf '%d passed, %d failed, %d skipped for missing tooling.\n' "$pass" "$fail" "$skip"
echo "Fix the FAIL lines above and re-run: bash check.sh"
exit 1
