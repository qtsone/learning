#!/usr/bin/env bash
# Referee for the Kubernetes Core exercise.
#
# The grade comes from reading the manifests you wrote: no cluster, no
# kubectl, no Docker daemon, no network, no root. If a kubectl and a reachable
# cluster happen to be here, two live checks are added as a bonus; when they
# are not, they are reported SKIPPED, never failed.
#
# Only one of those two creates anything, and it refuses to touch a cluster it
# does not recognise as a local throwaway one (set TUTOR_LIVE_APPLY=1 to
# override). It never deletes the Namespace, whatever the context.
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
# not, so the grader carries a small YAML reader covering the manifest subset
# the lesson asks you to stay inside.

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


def tokenize(text):
    toks = []
    for n, raw in enumerate(text.splitlines(), 1):
        lead = raw[: len(raw) - len(raw.lstrip())]
        if "\t" in lead:
            raise YamlError("line %d uses a tab to indent; YAML allows spaces only" % n)
        line = strip_comment(raw)
        if not line.strip() or line.strip() in ("---", "..."):
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


def is_item(tok):
    return tok[1] == "-" or tok[1].startswith("- ")


class Parser:
    def __init__(self, toks):
        self.toks, self.i = toks, 0

    def peek(self):
        return self.toks[self.i] if self.i < len(self.toks) else None

    def block(self, indent):
        tok = self.peek()
        if tok is None or tok[0] < indent:
            return None
        if is_item(tok):
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
                if nxt and nxt[0] > indent:
                    out[key] = self.block(nxt[0])
                elif nxt and nxt[0] == indent and is_item(nxt):
                    # a block sequence may sit at its key's own column
                    out[key] = self.sequence(indent)
                else:
                    out[key] = None
        return out

    def sequence(self, indent):
        out = []
        while True:
            tok = self.peek()
            if tok is None or tok[0] < indent or not is_item(tok):
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


def split_docs(text):
    docs, cur = [], []
    for line in text.splitlines():
        if line.rstrip() == "---":
            docs.append("\n".join(cur))
            cur = []
        else:
            cur.append(line)
    docs.append("\n".join(cur))
    return docs


def parse_file(path):
    """Every YAML document in one file, as (doc, path) pairs."""
    text = open(path, encoding="utf-8", errors="replace").read()
    out = []
    for chunk in split_docs(text):
        toks = tokenize(chunk)
        if not toks:
            continue
        doc = Parser(toks).block(toks[0][0])
        if not isinstance(doc, dict):
            raise YamlError("every document must be a mapping (apiVersion:, kind:, ...)")
        out.append((doc, path))
    return out


# ---------------------------------------------------------------- helpers


def as_dict(v):
    return v if isinstance(v, dict) else {}


def as_list(v):
    if v is None:
        return []
    return v if isinstance(v, list) else [v]


def dig(obj, *keys):
    cur = obj
    for k in keys:
        cur = as_dict(cur).get(k)
    return cur


def scalar(v):
    return "" if v is None else str(v).strip()


def as_int(v):
    try:
        return int(scalar(v))
    except (TypeError, ValueError):
        return None


def labels_of(v):
    return {
        str(k): scalar(x)
        for k, x in as_dict(v).items()
        if not isinstance(x, (dict, list))
    }


def show(pairs):
    return ", ".join("%s=%s" % (k, v) for k, v in sorted(pairs.items())) or "(none)"


def image_parts(image):
    """(repository, tag) — tag empty when the reference carries none."""
    ref = image.partition("@")[0]
    repo, sep, tag = ref.rpartition(":")
    if not sep or "/" in tag:  # no tag, or we split a registry:port
        return ref, ""
    return repo, tag


NS = "timesvc"
NAME = "timesvc"
PORT = "8080"


# ---------------------------------------------------------------- checks


def main():
    files = sorted(
        f
        for f in os.listdir(".")
        if f.endswith((".yaml", ".yml")) and os.path.isfile(f)
    )
    if not files:
        rec("FAIL", "the manifests parse as YAML",
            "no .yaml files here — write deployment.yaml and service.yaml next to check.sh")
        return

    docs = []
    for f in files:
        try:
            docs.extend(parse_file(f))
        except YamlError as e:
            rec("FAIL", "the manifests parse as YAML",
                "%s: %s. Indent with two spaces per level, keep a space after every colon, "
                "and stay inside the subset LESSON.md describes" % (f, e))
            return
    rec("PASS", "the manifests parse as YAML")

    def of_kind(kind):
        return [(d, f) for d, f in docs if scalar(d.get("kind")) == kind]

    deps, svcs = of_kind("Deployment"), of_kind("Service")
    if len(deps) != 1:
        rec("FAIL", "exactly one Deployment is defined",
            "found %d objects with kind: Deployment in %s — this exercise wants exactly one, "
            "in deployment.yaml. A bare kind: Pod is not a substitute: nothing would recreate it"
            % (len(deps), ", ".join(files)))
        return
    rec("PASS", "exactly one Deployment is defined")
    dep, dep_file = deps[0]

    # --- Deployment identity ---
    api = scalar(dep.get("apiVersion"))
    if api == "apps/v1":
        rec("PASS", "the Deployment declares apiVersion: apps/v1")
    else:
        rec("FAIL", "the Deployment declares apiVersion: apps/v1",
            "%s says apiVersion: %r. Deployments live in the apps group; only core objects "
            "(Service, Pod, ConfigMap, Namespace) use a bare v1"
            % (dep_file, api or "(missing)"))

    dmeta = as_dict(dep.get("metadata"))
    dname, dns_ = scalar(dmeta.get("name")), scalar(dmeta.get("namespace"))
    if dname == NAME and dns_ == NS:
        rec("PASS", "the Deployment is named %s in the %s namespace" % (NAME, NS))
    elif dname != NAME:
        rec("FAIL", "the Deployment is named %s in the %s namespace" % (NAME, NS),
            "metadata.name is %r; the rest of the pack refers to this workload as %s"
            % (dname or "(missing)", NAME))
    else:
        rec("FAIL", "the Deployment is named %s in the %s namespace" % (NAME, NS),
            "metadata.namespace is %r — set it to %s, the namespace namespace.yaml creates, so "
            "the object cannot land in default by accident" % (dns_ or "(missing)", NS))

    spec = as_dict(dep.get("spec"))

    # --- replicas ---
    replicas = as_int(spec.get("replicas"))
    if replicas is None:
        rec("FAIL", "the Deployment asks for at least 2 replicas",
            "spec.replicas is missing. Omitting it defaults to 1, and one pod means every "
            "rollout, node drain and crash is a full outage — say the number you want")
    elif replicas < 2:
        rec("FAIL", "the Deployment asks for at least 2 replicas",
            "spec.replicas is %d — with maxUnavailable: 0 a single replica cannot even roll "
            "out without extra capacity, and a node drain takes the service down" % replicas)
    else:
        rec("PASS", "the Deployment asks for at least 2 replicas")

    # --- selector vs template labels ---
    tmpl = as_dict(spec.get("template"))
    pod_labels = labels_of(dig(tmpl, "metadata", "labels"))
    sel_raw = as_dict(spec.get("selector"))
    match_labels = labels_of(sel_raw.get("matchLabels"))
    if not sel_raw:
        rec("FAIL", "the Deployment selector matches its own pod template",
            "spec.selector is missing. A Deployment owns pods by label, not by name — add "
            "selector: matchLabels: {app: %s} and stamp the same label on the template" % NAME)
    elif not match_labels:
        rec("FAIL", "the Deployment selector matches its own pod template",
            "spec.selector must contain matchLabels (a flat map here is a Service's syntax, "
            "not a Deployment's) — the API server rejects it otherwise")
    elif not pod_labels:
        rec("FAIL", "the Deployment selector matches its own pod template",
            "spec.template.metadata.labels is empty, so the selector matches nothing and the "
            "API server refuses the object. Stamp the pods with the labels you select on")
    else:
        bad = {k: v for k, v in match_labels.items() if pod_labels.get(k) != v}
        if bad:
            rec("FAIL", "the Deployment selector matches its own pod template",
                "selector wants %s but the template stamps %s — the API server rejects a "
                "Deployment whose selector does not match its own template"
                % (show(bad), show(pod_labels)))
        else:
            rec("PASS", "the Deployment selector matches its own pod template")

    if pod_labels.get("app") == NAME:
        rec("PASS", "the pod template carries the label app=%s" % NAME)
    else:
        rec("FAIL", "the pod template carries the label app=%s" % NAME,
            "spec.template.metadata.labels has %s — this exercise expects app: %s so the "
            "Service, the selector and every kubectl -l flag agree" % (show(pod_labels), NAME))

    # --- container ---
    containers = [c for c in as_list(dig(tmpl, "spec", "containers")) if isinstance(c, dict)]
    image = scalar(containers[0].get("image")) if containers else ""
    repo, tag = image_parts(image)
    if len(containers) != 1:
        rec("FAIL", "one container runs a pinned %s image" % NAME,
            "found %d containers under spec.template.spec.containers; this pod runs one "
            "process. If you could run them on different machines, they are different pods"
            % len(containers))
    elif not scalar(containers[0].get("name")):
        rec("FAIL", "one container runs a pinned %s image" % NAME,
            "the container has no name — kubectl logs and exec need it once a pod holds more "
            "than one container, and describe output is unreadable without it")
    elif not image:
        rec("FAIL", "one container runs a pinned %s image" % NAME,
            "the container has no image: field. Use the image you built in lesson two, "
            "image: %s:1.0.0" % NAME)
    elif "@sha256:" in image:
        rec("PASS", "one container runs a pinned %s image" % NAME)
    elif tag in ("", "latest"):
        rec("FAIL", "one container runs a pinned %s image" % NAME,
            "image: %r is untagged or :latest. A Deployment only rolls out when the pod "
            "template changes, and :latest never changes — so your new build is never "
            "deployed, while the default pull policy for it is Always. Pin %s:1.0.0"
            % (image, NAME))
    elif repo.rsplit("/", 1)[-1] != NAME:
        rec("FAIL", "one container runs a pinned %s image" % NAME,
            "image: %r does not point at the %s image you built in lesson two "
            "(a registry prefix is fine: ghcr.io/you/%s:1.0.0)" % (image, NAME, NAME))
    else:
        rec("PASS", "one container runs a pinned %s image" % NAME)

    cports = [p for p in as_list(containers[0].get("ports")) if isinstance(p, dict)] if containers else []
    numbers = [scalar(p.get("containerPort")) for p in cports]
    if PORT in numbers:
        rec("PASS", "the container declares containerPort %s" % PORT)
    elif not cports:
        rec("FAIL", "the container declares containerPort %s" % PORT,
            "no ports: on the container. It is documentation the Service and your teammates "
            "read — add ports: - name: http, containerPort: %s" % PORT)
    else:
        rec("FAIL", "the container declares containerPort %s" % PORT,
            "the container declares %s; timesvc listens on %s, because a non-root process "
            "cannot bind below 1024" % (", ".join(numbers) or "(nothing)", PORT))

    # --- rollout strategy ---
    strat = as_dict(spec.get("strategy"))
    stype = scalar(strat.get("type"))
    ru = as_dict(strat.get("rollingUpdate"))
    unavail, surge = scalar(ru.get("maxUnavailable")), scalar(ru.get("maxSurge"))
    surge_n = as_int(surge.rstrip("%"))
    if not strat:
        rec("FAIL", "the rollout never drops below the declared capacity",
            "spec.strategy is missing. The default is RollingUpdate with 25% each way, so a "
            "rollout may run at 75% capacity — state the pair you want: type: RollingUpdate "
            "with rollingUpdate: maxUnavailable: 0 and maxSurge: 1")
    elif stype != "RollingUpdate":
        rec("FAIL", "the rollout never drops below the declared capacity",
            "strategy.type is %r. Recreate kills every pod before starting the new ones — "
            "that is a deliberate outage, right for a singleton with a locked resource and "
            "wrong for an HTTP service" % (stype or "(missing)"))
    elif unavail not in ("0", "0%"):
        rec("FAIL", "the rollout never drops below the declared capacity",
            "rollingUpdate.maxUnavailable is %r; set it to 0 so Kubernetes must create the "
            "replacement pod before it may take an old one away" % (unavail or "(missing)"))
    elif surge_n is None or surge_n < 1:
        rec("FAIL", "the rollout never drops below the declared capacity",
            "rollingUpdate.maxSurge is %r. With maxUnavailable: 0 and maxSurge: 0 the rollout "
            "can never start — it is allowed neither to remove a pod nor to add one"
            % (surge or "(missing)"))
    else:
        rec("PASS", "the rollout never drops below the declared capacity")

    # --- Service ---
    if len(svcs) != 1:
        rec("FAIL", "exactly one Service is defined",
            "found %d objects with kind: Service — this exercise wants exactly one, in "
            "service.yaml, with apiVersion: v1" % len(svcs))
        notes()
        return
    rec("PASS", "exactly one Service is defined")
    svc, svc_file = svcs[0]

    smeta = as_dict(svc.get("metadata"))
    sapi = scalar(svc.get("apiVersion"))
    sname, sns = scalar(smeta.get("name")), scalar(smeta.get("namespace"))
    if sapi != "v1":
        rec("FAIL", "the Service is v1, named %s, in the %s namespace" % (NAME, NS),
            "%s says apiVersion: %r. A Service is in the core group, which has no group name: "
            "plain v1, not apps/v1" % (svc_file, sapi or "(missing)"))
    elif sname != NAME:
        rec("FAIL", "the Service is v1, named %s, in the %s namespace" % (NAME, NS),
            "metadata.name is %r. The Service name is the in-cluster DNS name other pods dial, "
            "so call it %s" % (sname or "(missing)", NAME))
    elif sns != NS:
        rec("FAIL", "the Service is v1, named %s, in the %s namespace" % (NAME, NS),
            "metadata.namespace is %r — a Service can only select pods in its own namespace, "
            "so it must be %s, like the Deployment" % (sns or "(missing)", NS))
    else:
        rec("PASS", "the Service is v1, named %s, in the %s namespace" % (NAME, NS))

    sspec = as_dict(svc.get("spec"))
    stype2 = scalar(sspec.get("type"))
    if stype2 == "ClusterIP":
        rec("PASS", "the Service type is ClusterIP")
    elif not stype2:
        rec("FAIL", "the Service type is ClusterIP",
            "spec.type is missing. ClusterIP is the default, but write it: the next reader "
            "should not have to know the default to know what is exposed")
    else:
        rec("FAIL", "the Service type is ClusterIP",
            "spec.type is %r. NodePort opens a port on every node and LoadBalancer bills you "
            "for real infrastructure; an internal service needs neither. Use kubectl "
            "port-forward to reach a ClusterIP from your laptop" % stype2)

    # --- the selector, the classic bug ---
    sel = sspec.get("selector")
    ssel = labels_of(sel)
    if sel is None:
        rec("FAIL", "the Service selector matches the pod template labels",
            "spec.selector is missing, so the Service selects no pods and every request to it "
            "is refused while everything reports healthy. Add selector: app: %s" % NAME)
    elif "matchLabels" in as_dict(sel):
        rec("FAIL", "the Service selector matches the pod template labels",
            "a Service selector is a flat map — matchLabels is Deployment syntax. Written this "
            "way the Service looks for a label literally called matchLabels and finds nothing")
    elif not ssel:
        rec("FAIL", "the Service selector matches the pod template labels",
            "spec.selector is empty. An empty selector does not mean 'all pods' here — it "
            "means the Service has no endpoints unless you manage them by hand")
    elif not pod_labels:
        rec("FAIL", "the Service selector matches the pod template labels",
            "the pod template has no labels at all, so nothing can select it")
    else:
        miss = {k: v for k, v in ssel.items() if pod_labels.get(k) != v}
        dep_labels = labels_of(dmeta.get("labels"))
        if miss and all(dep_labels.get(k) == v for k, v in miss.items()):
            rec("FAIL", "the Service selector matches the pod template labels",
                "the selector (%s) matches the Deployment's own metadata.labels, not the pod "
                "template's (%s). A Service selects pods; the Deployment object is not one"
                % (show(ssel), show(pod_labels)))
        elif miss:
            rec("FAIL", "the Service selector matches the pod template labels",
                "the selector wants %s and the pods are stamped %s, so this Service matches "
                "zero pods. Nothing errors when that happens: kubectl reports everything "
                "healthy and traffic goes nowhere. Check with kubectl get endpointslices"
                % (show(miss), show(pod_labels)))
        else:
            rec("PASS", "the Service selector matches the pod template labels")

    # --- ports ---
    sports = [p for p in as_list(sspec.get("ports")) if isinstance(p, dict)]
    entry = next((p for p in sports if scalar(p.get("port")) == "80"), None)
    named = {scalar(p.get("name")): scalar(p.get("containerPort")) for p in cports}
    if not sports:
        rec("FAIL", "the Service forwards port 80 to the container's %s" % PORT,
            "spec.ports is missing — add - port: 80 with targetPort: %s" % PORT)
    elif entry is None:
        rec("FAIL", "the Service forwards port 80 to the container's %s" % PORT,
            "no entry with port: 80. port is what callers dial on the Service; this exercise "
            "wants the boring 80 in front of the container's %s" % PORT)
    else:
        target = scalar(entry.get("targetPort"))
        if not target:
            rec("FAIL", "the Service forwards port 80 to the container's %s" % PORT,
                "targetPort is missing, and omitting it does not mean 'the container port' — "
                "it defaults to the value of port, so this Service would forward to 80 on the "
                "pod, where nothing is listening")
        elif target == PORT:
            rec("PASS", "the Service forwards port 80 to the container's %s" % PORT)
        elif target in named:
            if named[target] == PORT:
                rec("PASS", "the Service forwards port 80 to the container's %s" % PORT)
            else:
                rec("FAIL", "the Service forwards port 80 to the container's %s" % PORT,
                    "targetPort: %s names a container port declared as %s, not %s"
                    % (target, named[target] or "(no number)", PORT))
        elif as_int(target) is None:
            rec("FAIL", "the Service forwards port 80 to the container's %s" % PORT,
                "targetPort: %s is a name, but the container declares no port with that name. "
                "Either name the container port the same way, or use the number %s"
                % (target, PORT))
        else:
            rec("FAIL", "the Service forwards port 80 to the container's %s" % PORT,
                "targetPort is %s while the container listens on %s. This mismatch is silent: "
                "the Service gets endpoints, and every connection is refused" % (target, PORT))

    notes()


def notes():
    if not os.path.exists("NOTES.md"):
        rec("FAIL", "NOTES.md answers all three questions",
            "NOTES.md is missing — restore it from the exercise and answer on the A: lines")
        return
    text = open("NOTES.md", encoding="utf-8", errors="replace").read()
    if "TODO" in text:
        rec("FAIL", "NOTES.md answers all three questions",
            "NOTES.md still contains a TODO placeholder — replace every one with your answer")
        return
    answered = len(re.findall(r"(?m)^A:.{40,}", text))
    if answered >= 3:
        rec("PASS", "NOTES.md answers all three questions")
    else:
        rec("FAIL", "NOTES.md answers all three questions",
            "only %d of 3 A: lines carry a real answer (40+ characters). Write the "
            "reconciliation walk, the two backoff states and the empty-endpoints diagnosis "
            "in your own words" % answered)


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

dryrun="live: the API server accepts the manifests (server-side dry run)"
rollout="live: the Deployment rolls out and the Service finds its pods"

if ! command -v kubectl >/dev/null 2>&1; then
	why="kubectl is not installed. Install it plus kind/minikube/k3d to get these two"
	record SKIP "$dryrun" "$why"
	record SKIP "$rollout" "$why"
elif ! kubectl cluster-info --request-timeout=5s >/dev/null 2>&1; then
	why="kubectl is here but no cluster answers. Try: kind create cluster --name tutor"
	record SKIP "$dryrun" "$why"
	record SKIP "$rollout" "$why"
else
	if out="$(kubectl apply --dry-run=server -f . 2>&1)"; then
		record PASS "$dryrun"
	else
		record FAIL "$dryrun" \
			"the API server itself rejected a manifest: $(flatten "$out")"
	fi

	# The dry run above created nothing. This one really does, so it only runs
	# against a cluster you can afford to lose: a laptop kubeconfig pointing at
	# a shared cluster is the normal case, not the exception.
	ctx="$(kubectl config current-context 2>/dev/null)"
	throwaway=no
	case "$ctx" in
	kind-* | k3d-* | minikube | docker-desktop | rancher-desktop | orbstack)
		throwaway=yes
		;;
	esac

	# Everything except the Namespace: this script may remove the workload it
	# created, but deleting a namespace would cascade to whatever else lives
	# in it — including things that were there before you.
	workload=()
	for f in *.yaml *.yml; do
		[ -f "$f" ] || continue
		case "$f" in namespace.yaml | namespace.yml) continue ;; esac
		workload+=(-f "$f")
	done

	if [ "$throwaway" = no ] && [ "${TUTOR_LIVE_APPLY:-0}" != 1 ]; then
		record SKIP "$rollout" \
			"nothing was created: kubectl points at '${ctx:-no current context}', which is not a recognised local throwaway cluster. Switch to kind/k3d/minikube/Docker Desktop, or re-run with TUTOR_LIVE_APPLY=1 if you mean this one"
	elif [ "${#workload[@]}" -eq 0 ]; then
		record SKIP "$rollout" \
			"no manifest here besides namespace.yaml, so there is nothing to roll out"
	else
		[ -f namespace.yaml ] && kubectl apply -f namespace.yaml >/dev/null 2>&1
		if out="$(kubectl apply "${workload[@]}" 2>&1)"; then
			if kubectl rollout status deployment/timesvc -n timesvc --timeout=45s >/dev/null 2>&1; then
				eps="$(kubectl get endpointslices -n timesvc \
					-l kubernetes.io/service-name=timesvc \
					-o jsonpath='{.items[*].endpoints[*].addresses[*]}' 2>/dev/null)"
				if [ -n "$eps" ]; then
					record PASS "$rollout"
				else
					record FAIL "$rollout" \
						"pods are ready but the Service has no endpoints — its selector matches none of them"
				fi
			else
				record SKIP "$rollout" \
					"the pods did not become ready in 45s, which is expected unless you built the image and loaded it (kind load docker-image timesvc:1.0.0). Run: kubectl describe pod -n timesvc"
			fi
			kubectl delete "${workload[@]}" --ignore-not-found --wait=false >/dev/null 2>&1
		else
			record SKIP "$rollout" "kubectl apply did not run here: $(flatten "$out")"
		fi
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
