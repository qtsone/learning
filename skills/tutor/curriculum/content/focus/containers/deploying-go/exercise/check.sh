#!/usr/bin/env bash
# Referee for the Deploying Go on Kubernetes exercise.
#
# The grade comes from reading the files you wrote: the manifests, your Go
# source, and SHUTDOWN.md. No cluster, no kubectl, no Docker daemon, no
# network, no root. If a Go toolchain or a kubectl with a reachable cluster
# happens to be here, extra checks run as a bonus; when they are not, they are
# reported SKIPPED, never failed.
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
# not, so the grader carries a small YAML reader that covers the manifest
# subset this exercise asks for.

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


def strip_go_comments(src):
    # Every Go check below reads behaviour out of the source, and a comment is
    # not behaviour: "// deliberately does not read ready state" must not count
    # as reading it. String literals survive, so route paths still match.
    out, i, n = [], 0, len(src)
    while i < n:
        ch = src[i]
        if ch in "\"'`":
            out.append(ch)
            i += 1
            while i < n:
                c = src[i]
                out.append(c)
                i += 1
                if c == "\\" and ch != "`" and i < n:
                    out.append(src[i])
                    i += 1
                elif c == ch:
                    break
            continue
        if ch == "/" and src[i:i + 2] == "//":
            while i < n and src[i] != "\n":
                i += 1
            continue
        if ch == "/" and src[i:i + 2] == "/*":
            i += 2
            while i < n and src[i:i + 2] != "*/":
                if src[i] == "\n":
                    out.append("\n")
                i += 1
            i += 2
            continue
        out.append(ch)
        i += 1
    return "".join(out)


# ---------------------------------------------------------------- checks


def main():
    files = sorted(f for f in os.listdir(".") if f.endswith((".yaml", ".yml")))
    if not files:
        rec("FAIL", "the manifests exist",
            "there is no YAML next to check.sh. Keep configmap.yaml, deployment.yaml and "
            "service.yaml here (one multi-document file separated by --- also works)")
        return
    docs = []
    for name in files:
        text = open(name, encoding="utf-8", errors="replace").read()
        try:
            docs.extend(parse_documents(text))
        except YamlError as e:
            rec("FAIL", "the manifests parse as YAML",
                "%s, %s. Indent two spaces per level with spaces only, keep a space after every "
                "colon and after every '- ', and avoid anchors and flow style outside short lists"
                % (name, e))
            return
    rec("PASS", "the manifests parse as YAML (%d objects in %d file(s))" % (len(docs), len(files)))

    by_kind = {}
    for doc in docs:
        by_kind.setdefault(str(doc.get("kind") or "?"), []).append(doc)

    missing = [k for k in ("ConfigMap", "Deployment", "Service") if k not in by_kind]
    if missing:
        rec("FAIL", "the manifests declare a ConfigMap, a Deployment and a Service",
            "nothing here has kind: %s — every object needs apiVersion, kind and metadata.name"
            % ", ".join(missing))
        return
    dep, svc, cms = by_kind["Deployment"][0], by_kind["Service"][0], by_kind["ConfigMap"]

    api_problems = []
    if dep.get("apiVersion") != "apps/v1":
        api_problems.append("Deployment must be apps/v1, not %r" % dep.get("apiVersion"))
    if svc.get("apiVersion") != "v1":
        api_problems.append("Service must be v1, not %r" % svc.get("apiVersion"))
    for cm in cms:
        if cm.get("apiVersion") != "v1":
            api_problems.append("ConfigMap must be v1, not %r" % cm.get("apiVersion"))
    if api_problems:
        rec("FAIL", "the manifests declare a ConfigMap, a Deployment and a Service",
            "; ".join(api_problems))
    else:
        rec("PASS", "the manifests declare a ConfigMap, a Deployment and a Service")

    pod_labels = as_dict(dig(dep, "spec", "template", "metadata", "labels"))
    selector = as_dict(dig(svc, "spec", "selector"))
    if not selector:
        rec("FAIL", "the Service selector matches the Deployment's pod labels",
            "the Service has no spec.selector, so it selects no pods and its endpoint list stays empty")
    elif not pod_labels:
        rec("FAIL", "the Service selector matches the Deployment's pod labels",
            "the pod template has no metadata.labels for the Service to select")
    elif all(pod_labels.get(k) == v for k, v in selector.items()):
        rec("PASS", "the Service selector matches the Deployment's pod labels")
    else:
        rec("FAIL", "the Service selector matches the Deployment's pod labels",
            "selector %s does not match pod labels %s — the Service would have zero endpoints and "
            "every request would be refused, with both objects looking perfectly healthy"
            % (selector, pod_labels))

    containers = as_list(dig(dep, "spec", "template", "spec", "containers"))
    containers = [c for c in containers if isinstance(c, dict)]
    if not containers:
        rec("FAIL", "the pod template declares a container",
            "spec.template.spec.containers needs one entry with a name and an image")
        return
    c = containers[0]
    podspec = as_dict(dig(dep, "spec", "template", "spec"))

    # --- image ---
    image = str(c.get("image") or "")
    tag = image.rsplit(":", 1)[1] if ":" in image.rsplit("/", 1)[-1] else ""
    if not image:
        rec("FAIL", "the container image is pinned to a real tag",
            "the container has no image: — use the one you built in dockerizing-go, e.g. timesvc:1.4.0")
    elif "@sha256:" in image or (tag and tag != "latest"):
        rec("PASS", "the container image is pinned to a real tag (%s)" % image)
    else:
        rec("FAIL", "the container image is pinned to a real tag",
            "%r is a moving pointer: the pod that starts during tonight's incident may not run the code "
            "you tested. Pin a version tag or a digest — and remember a rollback is only meaningful when "
            "each revision names a different image" % image)

    # --- rolling update ---
    replicas = integer(dig(dep, "spec", "replicas"))
    strategy = as_dict(dig(dep, "spec", "strategy"))
    stype = strategy.get("type")
    surge = strategy.get("rollingUpdate", {})
    surge = as_dict(surge)
    max_surge, max_unavail = surge.get("maxSurge"), surge.get("maxUnavailable")
    if replicas is None or replicas < 2:
        rec("FAIL", "the Deployment can roll without losing capacity",
            "replicas is %s — a single replica has nowhere to send traffic while it is being replaced. "
            "Run at least 2" % replicas)
    elif stype != "RollingUpdate":
        rec("FAIL", "the Deployment can roll without losing capacity",
            "set spec.strategy.type: RollingUpdate explicitly (the alternative, Recreate, stops every old "
            "pod before starting any new one — a deliberate outage)")
    elif str(max_unavail) not in ("0", "0%"):
        rec("FAIL", "the Deployment can roll without losing capacity",
            "maxUnavailable is %r — with anything above 0 the rollout may take serving pods away before "
            "replacements are ready. Set maxUnavailable: 0 and let maxSurge pay for the extra pod"
            % max_unavail)
    elif integer(str(max_surge).rstrip("%")) is None or integer(str(max_surge).rstrip("%")) < 1:
        rec("FAIL", "the Deployment can roll without losing capacity",
            "maxSurge is %r — with maxUnavailable: 0 and maxSurge: 0 the rollout can never start, "
            "because it is allowed neither to remove a pod nor to add one" % max_surge)
    else:
        rec("PASS", "the Deployment can roll without losing capacity "
            "(replicas %d, maxSurge %s, maxUnavailable %s)" % (replicas, max_surge, max_unavail))

    # --- probes ---
    port_names = {}
    port_numbers = set()
    for p in as_list(c.get("ports")):
        p = as_dict(p)
        num = integer(p.get("containerPort"))
        if num is not None:
            port_numbers.add(num)
            if p.get("name"):
                port_names[str(p["name"])] = num

    def probe_target(probe):
        """(path, port-as-written, resolved-port-number) for an httpGet probe."""
        http_get = as_dict(as_dict(probe).get("httpGet"))
        if not http_get:
            return None, None, None
        port = http_get.get("port")
        num = integer(port)
        if num is None:
            num = port_names.get(str(port))
        return http_get.get("path"), port, num

    ready_path, ready_port, ready_num = probe_target(c.get("readinessProbe"))
    live_path, live_port, live_num = probe_target(c.get("livenessProbe"))
    start_probe = as_dict(c.get("startupProbe"))
    start_path, _, start_num = probe_target(start_probe)

    if ready_path is None:
        rec("FAIL", "the readiness probe asks the service whether it wants traffic",
            "add a readinessProbe with httpGet.path: /readyz and httpGet.port: http — that endpoint is "
            "the only thing that decides whether this pod is in the Service's endpoints")
    elif ready_path != "/readyz":
        rec("FAIL", "the readiness probe asks the service whether it wants traffic",
            "readinessProbe hits %r, but the service reports readiness on /readyz" % ready_path)
    elif ready_num not in port_numbers:
        rec("FAIL", "the readiness probe asks the service whether it wants traffic",
            "readinessProbe port %r does not resolve to a containerPort (%s). Use the port name you "
            "declared, or the number itself" % (ready_port, sorted(port_numbers)))
    else:
        rec("PASS", "the readiness probe asks the service whether it wants traffic (%s)" % ready_path)

    if live_path is None:
        rec("FAIL", "the liveness probe is not pointed at the readiness endpoint",
            "add a livenessProbe with httpGet.path: /healthz — the endpoint that answers 'this process "
            "still works', not 'this process wants traffic'")
    elif live_path == ready_path:
        rec("FAIL", "the liveness probe is not pointed at the readiness endpoint",
            "both probes hit %r. The moment the pod drains, readiness turns 503 — and a liveness probe "
            "reading the same endpoint reports the container dead and gets it restarted in the middle of "
            "the drain. Worse, any dependency you check in readiness now restarts your pods when *it* is "
            "down. Point liveness at /healthz" % live_path)
    elif live_path != "/healthz":
        rec("FAIL", "the liveness probe is not pointed at the readiness endpoint",
            "livenessProbe hits %r, but the service's liveness endpoint is /healthz" % live_path)
    elif live_num not in port_numbers:
        rec("FAIL", "the liveness probe is not pointed at the readiness endpoint",
            "livenessProbe port %r does not resolve to a containerPort (%s)" % (live_port, sorted(port_numbers)))
    else:
        rec("PASS", "the liveness probe is not pointed at the readiness endpoint (%s)" % live_path)

    if not start_probe:
        rec("FAIL", "a startup probe covers the slow first seconds",
            "add a startupProbe (httpGet /healthz). Until it succeeds the kubelet holds the liveness and "
            "readiness probes back, so you can keep liveness impatient without racing a slow start")
    elif start_path is None or start_num not in port_numbers:
        rec("FAIL", "a startup probe covers the slow first seconds",
            "the startupProbe needs an httpGet with a path and a port that resolves to a containerPort")
    else:
        budget = (integer(start_probe.get("failureThreshold")) or 3) * (
            integer(start_probe.get("periodSeconds")) or 10)
        if budget < 30:
            rec("FAIL", "a startup probe covers the slow first seconds",
                "failureThreshold x periodSeconds gives this container %ds to come up. Budget at least "
                "30s (e.g. failureThreshold: 30, periodSeconds: 1) — the point of the probe is to be "
                "generous once, not impatient forever" % budget)
        else:
            rec("PASS", "a startup probe covers the slow first seconds (%ds budget)" % budget)

    # --- termination ---
    tgps = integer(podspec.get("terminationGracePeriodSeconds"))
    if tgps is None:
        rec("FAIL", "terminationGracePeriodSeconds is set explicitly",
            "add terminationGracePeriodSeconds to the pod spec (a sibling of containers:). Leaving it out "
            "means 30 seconds by default — fine as a number, useless as a decision: nobody reading this "
            "file can tell whether 30 is enough for your shutdown path")
    elif tgps <= 0:
        rec("FAIL", "terminationGracePeriodSeconds is set explicitly",
            "terminationGracePeriodSeconds: %d means SIGKILL immediately — no drain at all" % tgps)
    else:
        rec("PASS", "terminationGracePeriodSeconds is set explicitly (%ds)" % tgps)

    lifecycle = as_dict(c.get("lifecycle"))
    pre_stop = as_dict(lifecycle.get("preStop"))
    sleep_seconds = None
    if "sleep" in pre_stop:
        sleep_seconds = integer(as_dict(pre_stop.get("sleep")).get("seconds"))
    if sleep_seconds is None and "exec" in pre_stop:
        cmd = " ".join(str(x) for x in as_list(as_dict(pre_stop["exec"]).get("command")))
        m = re.search(r"sleep\s+([0-9]+)", cmd)
        if m:
            sleep_seconds = int(m.group(1))
    if not pre_stop:
        rec("FAIL", "a preStop hook delays SIGTERM while the endpoints drain",
            "add lifecycle.preStop with sleep: seconds: 10. Removing this pod from the Service's "
            "endpoints and sending it SIGTERM happen in parallel and nothing synchronises them, so a "
            "server that stops accepting the instant SIGTERM lands refuses connections that kube-proxy "
            "is still sending. The sleep buys the dataplane time to catch up")
    elif sleep_seconds is None:
        rec("FAIL", "a preStop hook delays SIGTERM while the endpoints drain",
            "the preStop hook is there but this grader cannot find a sleep in it. Use the native handler "
            "(preStop: sleep: seconds: 10) or an exec whose command sleeps for a fixed number of seconds")
    elif sleep_seconds < 5:
        rec("FAIL", "a preStop hook delays SIGTERM while the endpoints drain",
            "%ds is not enough for every kube-proxy in the cluster to notice. 5-15s is the usual range"
            % sleep_seconds)
    else:
        rec("PASS", "a preStop hook delays SIGTERM while the endpoints drain (%ds)" % sleep_seconds)
        if "exec" in pre_stop:
            rec("NOTE", "an exec preStop hook runs a program inside your container",
                "the distroless image from dockerizing-go has no shell and no /bin/sleep, so an exec "
                "hook that works on your laptop can fail there. The native preStop: sleep: handler "
                "needs nothing in the image")

    # --- config from the ConfigMap ---
    env = {}
    for e in as_list(c.get("env")):
        e = as_dict(e)
        if e.get("name"):
            env[str(e["name"])] = e

    cm_data = {}
    cm_names = set()
    for cm in cms:
        name = str(dig(cm, "metadata", "name") or "")
        cm_names.add(name)
        cm_data[name] = as_dict(cm.get("data"))

    env_from_cms = [
        str(as_dict(as_dict(f).get("configMapRef")).get("name") or "")
        for f in as_list(c.get("envFrom"))
        if as_dict(as_dict(f).get("configMapRef")).get("name")
    ]

    def config_value(key):
        entry = env.get(key)
        if entry is not None:
            if entry.get("value") is not None:
                return str(entry["value"])
            ref = as_dict(dig(entry, "valueFrom", "configMapKeyRef"))
            if ref:
                return cm_data.get(str(ref.get("name")), {}).get(str(ref.get("key")))
            return None
        for name in env_from_cms:
            if key in cm_data.get(name, {}):
                return cm_data[name][key]
        return None

    unknown = [n for n in env_from_cms if n not in cm_names]
    wanted = {k: config_value(k) for k in ("PORT", "LOG_LEVEL", "SHUTDOWN_TIMEOUT")}
    absent = sorted(k for k, v in wanted.items() if v is None)
    shutdown_seconds = go_duration(wanted["SHUTDOWN_TIMEOUT"])
    if unknown:
        rec("FAIL", "the service's configuration comes from the ConfigMap",
            "envFrom references ConfigMap %r, which this file does not define — the pod would stay in "
            "CreateContainerConfigError" % unknown[0])
    elif absent:
        rec("FAIL", "the service's configuration comes from the ConfigMap",
            "the container never receives %s. timesvc reads PORT, LOG_LEVEL and SHUTDOWN_TIMEOUT from "
            "the environment; put all three in the ConfigMap and pull them into the container, either "
            "per key with configMapKeyRef or wholesale with envFrom.configMapRef" % ", ".join(absent))
    elif shutdown_seconds is None:
        rec("FAIL", "the service's configuration comes from the ConfigMap",
            "SHUTDOWN_TIMEOUT is %r, which time.ParseDuration rejects — the process would exit at "
            "startup. Write a Go duration such as 20s" % wanted["SHUTDOWN_TIMEOUT"])
    else:
        rec("PASS", "the service's configuration comes from the ConfigMap "
            "(PORT, LOG_LEVEL, SHUTDOWN_TIMEOUT=%s)" % wanted["SHUTDOWN_TIMEOUT"])

    # --- the shutdown budget adds up ---
    if tgps is None or sleep_seconds is None or shutdown_seconds is None:
        rec("FAIL", "the grace period covers preStop plus the server's own shutdown timeout",
            "this check needs all three numbers: terminationGracePeriodSeconds, the preStop sleep and "
            "SHUTDOWN_TIMEOUT. Fix the checks above first")
    elif tgps < sleep_seconds + shutdown_seconds + 5:
        rec("FAIL", "the grace period covers preStop plus the server's own shutdown timeout",
            "preStop %ds + SHUTDOWN_TIMEOUT %gs needs at least %gs of grace, but "
            "terminationGracePeriodSeconds is %ds. The preStop hook runs *inside* the grace period, so "
            "the kubelet would SIGKILL this container mid-drain — dropped connections, and a shutdown "
            "path that never gets to log why"
            % (sleep_seconds, shutdown_seconds, sleep_seconds + shutdown_seconds + 5, tgps))
    else:
        rec("PASS", "the grace period covers preStop plus the server's own shutdown timeout "
            "(%ds >= %ds + %gs + margin)" % (tgps, sleep_seconds, shutdown_seconds))

    # --- resources ---
    res = as_dict(c.get("resources"))
    req, lim = as_dict(res.get("requests")), as_dict(res.get("limits"))
    cpu_req, cpu_lim = quantity(req.get("cpu")), quantity(lim.get("cpu"))
    mem_req, mem_lim = quantity(req.get("memory")), quantity(lim.get("memory"))
    if None in (cpu_req, mem_req):
        rec("FAIL", "the container declares resource requests and limits",
            "resources.requests needs cpu and memory. Requests are what the scheduler reserves — without "
            "them the pod is BestEffort, lands anywhere, and is first in line for eviction")
    elif None in (cpu_lim, mem_lim):
        rec("FAIL", "the container declares resource requests and limits",
            "resources.limits needs cpu and memory. The memory limit is what turns a leak into one dead "
            "pod instead of a dead node, and the CPU limit is the number GOMAXPROCS has to agree with")
    elif cpu_lim < cpu_req or mem_lim < mem_req:
        rec("FAIL", "the container declares resource requests and limits",
            "a limit below its request is rejected by the API server: cpu %s/%s, memory %s/%s "
            "(request/limit)" % (req.get("cpu"), lim.get("cpu"), req.get("memory"), lim.get("memory")))
    else:
        rec("PASS", "the container declares resource requests and limits "
            "(cpu %s/%s, memory %s/%s)" % (req.get("cpu"), lim.get("cpu"), req.get("memory"), lim.get("memory")))

    # --- GOMAXPROCS ---
    gmp = env.get("GOMAXPROCS")
    ref = as_dict(dig(gmp or {}, "valueFrom", "resourceFieldRef"))
    if gmp is None:
        rec("FAIL", "GOMAXPROCS is derived from the CPU limit, not from the node",
            "set GOMAXPROCS on the container: either a literal matching the CPU limit, or the downward "
            "API (valueFrom.resourceFieldRef.resource: limits.cpu, which rounds up to a whole core). "
            "A Go runtime that sizes itself from the node builds a scheduler for cores this container is "
            "not allowed to use, and the cgroup throttles it for the rest of every period")
    elif ref:
        resource = str(ref.get("resource") or "")
        if resource == "limits.cpu":
            rec("PASS", "GOMAXPROCS is derived from the CPU limit, not from the node (downward API)")
        else:
            rec("FAIL", "GOMAXPROCS is derived from the CPU limit, not from the node",
                "resourceFieldRef reads %r. The request is what the scheduler reserved; the *limit* is "
                "the cgroup quota the runtime has to live inside. Use limits.cpu" % resource)
    else:
        want = integer(gmp.get("value"))
        ceiling = int(math.ceil(cpu_lim)) if cpu_lim else None
        if want is None or want < 1:
            rec("FAIL", "GOMAXPROCS is derived from the CPU limit, not from the node",
                "GOMAXPROCS is %r; it must be a whole number of cores" % gmp.get("value"))
        elif ceiling is not None and want != ceiling:
            rec("FAIL", "GOMAXPROCS is derived from the CPU limit, not from the node",
                "GOMAXPROCS is %d but the CPU limit is %s, which rounds up to %d core(s). Above the "
                "quota you buy throttling; below it you leave the quota unused"
                % (want, lim.get("cpu"), ceiling))
        else:
            rec("PASS", "GOMAXPROCS is derived from the CPU limit, not from the node (%d)" % want)

    # --- GOMEMLIMIT ---
    gml = env.get("GOMEMLIMIT")
    if gml is None:
        rec("FAIL", "GOMEMLIMIT leaves headroom under the memory limit",
            "set GOMEMLIMIT on the container, a bit under resources.limits.memory (e.g. 110MiB under a "
            "128Mi limit). The collector's default target is a ratio of the live heap and knows nothing "
            "about the cgroup: without this the heap grows past the limit and the kernel OOM-kills the "
            "process instead of the GC running sooner")
    elif gml.get("value") is None:
        rec("FAIL", "GOMEMLIMIT leaves headroom under the memory limit",
            "write GOMEMLIMIT as a literal value. Reading limits.memory through the downward API gives "
            "you exactly the limit in bytes, which leaves no room for the binary, stacks and everything "
            "else the runtime does not count")
    else:
        raw = str(gml["value"])
        val = gomemlimit_bytes(raw)
        if val is None:
            rec("FAIL", "GOMEMLIMIT leaves headroom under the memory limit",
                "%r is not a value the Go runtime accepts. It takes bytes with an optional B, KiB, MiB, "
                "GiB or TiB suffix — note that Kubernetes writes 512Mi and Go wants 512MiB, and a bad "
                "value makes the runtime refuse to start" % raw)
        elif mem_lim is None:
            rec("FAIL", "GOMEMLIMIT leaves headroom under the memory limit",
                "set resources.limits.memory first — GOMEMLIMIT only means something relative to it")
        elif val >= mem_lim:
            rec("FAIL", "GOMEMLIMIT leaves headroom under the memory limit",
                "GOMEMLIMIT %s is not below the %s memory limit. It is a soft limit the GC aims at; the "
                "cgroup limit is hard and enforced by the kernel, so the soft one has to be lower"
                % (human(val), human(mem_lim)))
        elif val < 0.5 * mem_lim:
            rec("FAIL", "GOMEMLIMIT leaves headroom under the memory limit",
                "GOMEMLIMIT %s is less than half of the %s limit — that much headroom means paying for "
                "memory you have told the runtime never to use. 80-90%% is the usual range"
                % (human(val), human(mem_lim)))
        else:
            rec("PASS", "GOMEMLIMIT leaves headroom under the memory limit (%s under %s)"
                % (human(val), human(mem_lim)))

    # ---------------------------------------------------------- the Go side
    sources = sorted(
        f for f in os.listdir(".") if f.endswith(".go") and not f.endswith("_test.go")
    )
    raw = "\n".join(open(f, encoding="utf-8", errors="replace").read() for f in sources)
    src = strip_go_comments(raw) + "\n"
    if not src.strip():
        rec("FAIL", "the service handles SIGTERM", "no non-test Go source found next to check.sh")
        return

    if re.search(r"signal\.NotifyContext\(", src) or re.search(r"signal\.Notify\(", src):
        if "SIGTERM" in src:
            rec("PASS", "the service handles SIGTERM")
        else:
            rec("FAIL", "the service handles SIGTERM",
                "you install a signal handler but never mention SIGTERM. SIGINT alone covers Ctrl-C on "
                "your laptop; Kubernetes sends SIGTERM, and an unhandled SIGTERM kills the process "
                "instantly — Go's default disposition for it, not a graceful stop")
    else:
        rec("FAIL", "the service handles SIGTERM",
            "nothing listens for signals. Use signal.NotifyContext(context.Background(), "
            "syscall.SIGINT, syscall.SIGTERM) in main and pass that context to run")

    shutdown_call = re.search(r"\.Shutdown\(", src)
    if not shutdown_call:
        rec("FAIL", "shutdown drains in-flight requests instead of severing them",
            "call srv.Shutdown(ctx). Close() slams every connection shut, including requests being "
            "served right now; Shutdown stops accepting, then waits for the ones already accepted")
    elif "context.WithTimeout(" not in src:
        rec("FAIL", "shutdown drains in-flight requests instead of severing them",
            "Shutdown needs a context with a deadline, built from SHUTDOWN_TIMEOUT. Without one a single "
            "stuck handler holds the drain open until the kubelet SIGKILLs you")
    else:
        rec("PASS", "shutdown drains in-flight requests instead of severing them")

    ready_false = re.search(r"ready\.Store\(\s*false\s*\)|[sS]etReady\(\s*false\s*\)", src)
    if not ready_false:
        rec("FAIL", "readiness is withdrawn before the server stops accepting",
            "nothing ever sets the readiness flag to false. The first thing the shutdown path does is "
            "start answering 503 on /readyz, so anything still polling readiness stops choosing this pod")
    elif not shutdown_call:
        rec("FAIL", "readiness is withdrawn before the server stops accepting",
            "there is no Shutdown call for the flip to come before")
    elif ready_false.start() > shutdown_call.start():
        rec("FAIL", "readiness is withdrawn before the server stops accepting",
            "the readiness flip appears after the Shutdown call in your source. Reverse them: withdraw "
            "readiness first, then stop accepting. The other order advertises a pod that no longer takes "
            "connections")
    else:
        rec("PASS", "readiness is withdrawn before the server stops accepting")

    body = re.search(r"func\s+(?:\([^()]*\)\s*)?healthz\s*\(.*?\n\}\n", src, re.S)
    if not body:
        rec("FAIL", "the liveness handler does not depend on readiness",
            "no healthz function found. This grader locates the liveness handler by name, so keep it "
            "called healthz — a method on your app type or a plain function, whichever you prefer")
    elif re.search(r"ready\b", body.group(0)) or "StatusServiceUnavailable" in body.group(0):
        rec("FAIL", "the liveness handler does not depend on readiness",
            "healthz still reads the readiness state. Liveness answers one question — is this process "
            "still working — and every extra thing it checks is another way to get your pods restarted "
            "for something a restart cannot fix")
    else:
        rec("PASS", "the liveness handler does not depend on readiness")

    contract = [r for r in ('"GET /healthz"', '"GET /readyz"', '"GET /now"') if r not in src]
    if contract:
        rec("FAIL", "the service still keeps its contract with the platform",
            "route(s) %s are no longer registered — the manifests probe those paths" % ", ".join(contract))
    elif not re.search(r"slog\.NewJSONHandler\(\s*os\.Stdout", src):
        rec("FAIL", "the service still keeps its contract with the platform",
            "logs must stay structured and go to stdout: that is the whole logging interface a container "
            "has. A file inside the container is invisible to kubectl logs and dies with the pod")
    else:
        rec("PASS", "the service still keeps its contract with the platform")

    # ---------------------------------------------------------- SHUTDOWN.md
    if not os.path.exists("SHUTDOWN.md"):
        rec("FAIL", "SHUTDOWN.md answers all four questions",
            "SHUTDOWN.md is missing — restore it from the exercise and answer on the A: lines")
        return
    notes = open("SHUTDOWN.md", encoding="utf-8", errors="replace").read()
    answers = [ln[2:].strip() for ln in notes.splitlines() if ln.startswith("A:")]
    real = [a for a in answers if len(a) >= 80 and "TODO" not in a]
    if len(real) < 4:
        rec("FAIL", "SHUTDOWN.md answers all four questions",
            "only %d of 4 A: lines carry a real answer (80+ characters, no TODO left)" % len(real))
    else:
        rec("PASS", "SHUTDOWN.md answers all four questions")

    low = "\n".join(answers).lower()
    terms = {
        "SIGTERM": "sigterm",
        "terminationGracePeriodSeconds": "terminationgraceperiodseconds",
        "SIGKILL": "sigkill",
        "preStop": "prestop",
        "endpoints": "endpoint",
    }
    absent_terms = [name for name, needle in terms.items() if needle not in low]
    if absent_terms:
        rec("FAIL", "SHUTDOWN.md names the whole termination sequence",
            "the write-up never mentions %s. Q1 asks for every step in order, by name — that vocabulary "
            "is what makes the next incident a conversation instead of a guess" % ", ".join(absent_terms))
    else:
        rec("PASS", "SHUTDOWN.md names the whole termination sequence")

    ordering = False
    for a in real:
        text_l = a.lower()
        has_subject = "readi" in text_l or "endpoint" in text_l
        has_order = any(w in text_l for w in ("before", "first", "prior to", "ahead of"))
        has_object = any(
            w in text_l for w in ("accept", "shutdown", "shut down", "sigterm", "stop serving", "close")
        )
        if has_subject and has_order and has_object:
            ordering = True
            break
    if ordering:
        rec("PASS", "SHUTDOWN.md states the ordering requirement explicitly")
    else:
        rec("FAIL", "SHUTDOWN.md states the ordering requirement explicitly",
            "no answer says, in one sentence, which of the two comes first. Q2 wants it spelled out: "
            "readiness fails (and the pod leaves the endpoints) BEFORE the server stops accepting new "
            "connections — otherwise traffic is still being routed to a socket that is already closed")


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

if command -v go >/dev/null 2>&1; then
	if out="$(go test ./... 2>&1)"; then
		record PASS "go: the shutdown tests pass"
	elif printf '%s' "$out" | grep -qiE 'GOCACHE|GOPATH|permission denied|cannot find GOROOT|toolchain not available'; then
		record SKIP "go: the shutdown tests pass" \
			"the Go toolchain here cannot build: $(flatten "$out")"
	else
		record FAIL "go: the shutdown tests pass" \
			"$(flatten "$(printf '%s' "$out" | grep -E '^(--- FAIL|    )' | head -n 4)") — run 'go test ./...' yourself for the full output"
	fi
else
	record SKIP "go: the shutdown tests pass" \
		"no Go toolchain on PATH, so shutdown_test.go could not be run"
fi

kube_reason="no kubectl on PATH. A local cluster is worth having: 'kind create cluster' is one command"
if command -v kubectl >/dev/null 2>&1; then
	if out="$(kubectl apply --dry-run=client -f . 2>&1)"; then
		record PASS "kubectl: the manifests survive a client-side dry run"
	elif printf '%s' "$out" | grep -qiE 'connection refused|unable to connect|no configuration has been provided|couldn.t get current server'; then
		record SKIP "kubectl: the manifests survive a client-side dry run" \
			"kubectl is installed but has no usable kubeconfig: $(flatten "$out")"
	else
		record FAIL "kubectl: the manifests survive a client-side dry run" \
			"kubectl rejected the manifests: $(flatten "$out")"
	fi

	if kubectl version --request-timeout=3s >/dev/null 2>&1; then
		if out="$(kubectl apply --dry-run=server -f . 2>&1)"; then
			record PASS "kubectl: the API server validates the manifests (server dry run)"
		elif printf '%s' "$out" | grep -qiE 'namespaces? "[^"]*" not found'; then
			record SKIP "kubectl: the API server validates the manifests (server dry run)" \
				"the namespace does not exist in this cluster yet: kubectl create namespace timesvc"
		else
			record FAIL "kubectl: the API server validates the manifests (server dry run)" \
				"the API server rejected the manifests (nothing was created): $(flatten "$out")"
		fi
	else
		record SKIP "kubectl: the API server validates the manifests (server dry run)" \
			"no reachable cluster. 'kind create cluster' gives you one in about a minute"
	fi
else
	record SKIP "kubectl: the manifests survive a client-side dry run" "$kube_reason"
	record SKIP "kubectl: the API server validates the manifests (server dry run)" "$kube_reason"
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
