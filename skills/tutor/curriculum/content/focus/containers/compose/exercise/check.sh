#!/usr/bin/env bash
# Referee for the Compose exercise.
#
# The grade comes from reading the files you wrote: no Docker daemon, no
# network, no root. If a Docker daemon happens to be running, two live checks
# are added as a bonus; when it is not, they are reported SKIPPED, never failed.
#
# Run from this folder:  bash check.sh
set -u
cd "$(dirname "$0")" || exit 1

results="$(mktemp)"
trap 'rm -f "$results"' EXIT

record() { printf '%s|%s|%s\n' "$1" "$2" "${3:-}" >>"$results"; }

# One line, no pipes: the report loop splits records on '|'.
flatten() { printf '%s' "$1" | tr '\n|' '  ' | cut -c1-220; }

# `up --wait` returns as soon as app is *running*, but the dev stage compiles
# (`go run .`) before it listens, so the published port stays dark for a while
# and a probe fired straight away is refused or reset by the port forwarder.
# Returns 0 once /healthz answers, 1 if the container exits, 2 on timeout.
wait_for_app() {
	local port="$1" deadline=$(($(date +%s) + 90))
	while [ "$(date +%s)" -lt "$deadline" ]; do
		if curl -fsS --max-time 5 "http://127.0.0.1:$port/healthz" >/dev/null 2>&1; then
			return 0
		fi
		if [ "$(docker compose ps --format '{{.State}}' app 2>/dev/null)" != running ]; then
			return 1
		fi
		sleep 2
	done
	return 2
}

# ---------------------------------------------------------------- static
# python3 is guaranteed here (the tutor engine itself is Python). PyYAML is
# not, so the grader carries a small YAML reader that covers the compose subset.

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


def parse(text):
    toks = tokenize(text)
    if not toks:
        raise YamlError("the file is empty")
    doc = Parser(toks).block(toks[0][0])
    if not isinstance(doc, dict):
        raise YamlError("the top level must be a mapping (services:, volumes:, ...)")
    return doc


# ---------------------------------------------------------------- helpers


def as_dict(v):
    return v if isinstance(v, dict) else {}


def as_list(v):
    if v is None:
        return []
    return v if isinstance(v, list) else [v]


def dotenv(path):
    out = {}
    if not os.path.exists(path):
        return out
    for raw in open(path, encoding="utf-8", errors="replace"):
        line = raw.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        k, _, v = line.partition("=")
        out[k.strip()] = unquote(v.strip())
    return out


def service_env(svc, base):
    """(merged env, keys written literally in compose.yaml)"""
    env, literal = {}, set()
    for entry in as_list(svc.get("env_file")):
        path = entry.get("path") if isinstance(entry, dict) else entry
        if isinstance(path, str):
            env.update(dotenv(os.path.join(base, path)))
    raw = svc.get("environment")
    if isinstance(raw, dict):
        for k, v in raw.items():
            env[k] = "" if v is None else str(v)
            literal.add(k)
    else:
        for item in as_list(raw):
            if isinstance(item, str):
                k, _, v = item.partition("=")
                env[k.strip()] = v.strip()
                literal.add(k.strip())
    return env, literal


def url_host(raw):
    rest = raw.split("://", 1)[1] if "://" in raw else raw
    rest = rest.split("/", 1)[0]
    if "@" in rest:
        rest = rest.rsplit("@", 1)[1]
    return rest.rsplit(":", 1)[0] if ":" in rest else rest


def mounts(svc):
    out = []
    for m in as_list(svc.get("volumes")):
        if isinstance(m, dict):
            out.append((str(m.get("source") or ""), str(m.get("target") or "")))
        elif isinstance(m, str):
            parts = split_top(m, ":")
            if len(parts) == 1:
                out.append(("", parts[0]))
            else:
                out.append((parts[0], parts[1]))
    return out


def port_pairs(svc):
    out = []
    for p in as_list(svc.get("ports")):
        if isinstance(p, dict):
            out.append((str(p.get("published") or ""), str(p.get("target") or "")))
        elif isinstance(p, str):
            parts = split_top(p, ":")
            pair = ("", parts[0]) if len(parts) == 1 else (parts[-2], parts[-1])
            out.append((pair[0].strip(), pair[1].split("/")[0].strip()))
    return out


def is_bind(source):
    return source.startswith((".", "/", "~"))


# ---------------------------------------------------------------- checks


def main():
    path = None
    for cand in ("compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"):
        if os.path.exists(cand):
            path = cand
            break
    if path is None:
        rec("FAIL", "a compose file exists", "create compose.yaml next to check.sh")
        return
    text = open(path, encoding="utf-8", errors="replace").read()
    try:
        doc = parse(text)
    except YamlError as e:
        rec("FAIL", "%s parses as YAML" % path,
            "%s. Indent with two spaces per level and keep a space after every colon" % e)
        return
    rec("PASS", "%s parses as YAML" % path)

    services = as_dict(doc.get("services"))
    app, db = services.get("app"), services.get("db")
    if not isinstance(app, dict) or not isinstance(db, dict):
        rec("FAIL", "services app and db are both defined",
            "the file needs a top-level services: block containing an app service and a db service — "
            "those names are also the DNS names they reach each other by")
        return
    rec("PASS", "services app and db are both defined")

    # --- build ---
    build = app.get("build")
    context = build if isinstance(build, str) else as_dict(build).get("context")
    if isinstance(context, str) and context.strip("./") == "app":
        rec("PASS", "app builds from the ./app context")
    else:
        rec("FAIL", "app builds from the ./app context",
            "give app a build: block with context: ./app so Compose builds the provided Dockerfile")

    target = as_dict(build).get("target") if isinstance(build, dict) else None
    if target == "dev":
        rec("PASS", "app builds the dev stage of the Dockerfile")
    else:
        rec("FAIL", "app builds the dev stage of the Dockerfile",
            "add target: dev under build: — without it Compose builds the last stage, which has no "
            "toolchain and ignores your bind-mounted source")

    # --- image pinning ---
    image = db.get("image")
    if not isinstance(image, str) or not image.strip():
        rec("FAIL", "db runs a pinned Postgres image",
            "set image: postgres:<version>-alpine on the db service")
    elif "postgres" not in image:
        rec("FAIL", "db runs a pinned Postgres image",
            "the healthcheck and the credentials in this exercise assume the official postgres image")
    elif "@sha256:" in image:
        rec("PASS", "db runs a pinned Postgres image")
    elif ":" not in image or image.rsplit(":", 1)[1] in ("latest", ""):
        rec("FAIL", "db runs a pinned Postgres image",
            "pin a real version tag such as postgres:16.4-alpine — latest (or no tag at all) means "
            "your machine and your teammate's can run different databases")
    else:
        rec("PASS", "db runs a pinned Postgres image")

    app_env, app_literal = service_env(app, ".")
    db_env, db_literal = service_env(db, ".")

    # --- service discovery ---
    dburl = app_env.get("DATABASE_URL", "")
    host = url_host(dburl) if dburl else ""
    if not dburl:
        rec("FAIL", "app reaches the database by service name",
            "give app a DATABASE_URL (environment: or env_file:) of the form "
            "postgres://app:${POSTGRES_PASSWORD}@db:5432/appdb?sslmode=disable")
    elif host in ("localhost", "127.0.0.1", "0.0.0.0", "::1"):
        rec("FAIL", "app reaches the database by service name",
            "localhost inside a container is that container itself — use the service name db as the host")
    elif re.match(r"^\d+\.\d+\.\d+\.\d+$", host):
        rec("FAIL", "app reaches the database by service name",
            "container IPs change on every up — use the service name db as the host")
    elif host != "db":
        rec("FAIL", "app reaches the database by service name",
            "DATABASE_URL points at %r; it must be the db service name so Compose DNS resolves it" % host)
    else:
        rec("PASS", "app reaches the database by service name")

    # --- credentials ---
    if "POSTGRES_PASSWORD" not in db_env:
        rec("FAIL", "the database password is configured and not hard-coded",
            "the postgres image refuses to start without POSTGRES_PASSWORD — set it on db via "
            "${POSTGRES_PASSWORD} or an env_file")
    else:
        leaks = []
        if "POSTGRES_PASSWORD" in db_literal and "${" not in db_env["POSTGRES_PASSWORD"]:
            leaks.append("db POSTGRES_PASSWORD")
        userinfo = dburl.split("://", 1)[1].split("/", 1)[0] if "://" in dburl else ""
        if "DATABASE_URL" in app_literal and "@" in userinfo:
            creds = userinfo.rsplit("@", 1)[0]
            if ":" in creds and "${" not in creds.split(":", 1)[1]:
                leaks.append("app DATABASE_URL")
        if leaks:
            rec("FAIL", "the database password is configured and not hard-coded",
                "a literal password sits in compose.yaml (%s) — compose.yaml is committed, so "
                "interpolate ${POSTGRES_PASSWORD} and keep the value in .env" % ", ".join(leaks))
        else:
            rec("PASS", "the database password is configured and not hard-coded")

    # --- interpolation is satisfied ---
    needed = set()
    for name, rest in re.findall(r"\$\{([A-Za-z_][A-Za-z0-9_]*)([^}]*)\}", text):
        if not rest.startswith((":-", "-")):
            needed.add(name)
    local = dotenv(".env")
    if not os.path.exists(".env"):
        rec("FAIL", "a .env file supplies the values compose.yaml interpolates",
            "run: cp .env.example .env — Compose loads .env automatically and .env stays out of git")
    else:
        missing = sorted(n for n in needed if n not in local and n not in os.environ)
        if missing:
            rec("FAIL", "a .env file supplies the values compose.yaml interpolates",
                "compose.yaml interpolates %s with no default and .env does not define them" % ", ".join(missing))
        elif local.get("POSTGRES_PASSWORD", "") in ("", "change-me"):
            rec("FAIL", "a .env file supplies the values compose.yaml interpolates",
                "POSTGRES_PASSWORD in .env is still the placeholder — set a real local value")
        else:
            rec("PASS", "a .env file supplies the values compose.yaml interpolates")

    # --- healthcheck ---
    hc = as_dict(db.get("healthcheck"))
    test = hc.get("test")
    probe = " ".join(str(x) for x in test) if isinstance(test, list) else str(test or "")
    if "pg_isready" not in probe:
        rec("FAIL", "db has a healthcheck that proves Postgres accepts connections",
            "add healthcheck.test on db, e.g. "
            '["CMD-SHELL", "pg_isready -U $$POSTGRES_USER -d $$POSTGRES_DB"] — '
            "a running process is not a ready database")
    elif isinstance(test, list) and test and test[0] not in ("CMD", "CMD-SHELL"):
        rec("FAIL", "db has a healthcheck that proves Postgres accepts connections",
            "the list form must start with CMD (exec) or CMD-SHELL (shell), otherwise Docker rejects it")
    else:
        rec("PASS", "db has a healthcheck that proves Postgres accepts connections")

    tuning = [k for k in ("interval", "retries") if k not in hc]
    if tuning:
        rec("FAIL", "the healthcheck states how often and how many times it probes",
            "add %s to db.healthcheck (interval: 5s, retries: 5, and start_period for slow first boots)"
            % " and ".join(tuning))
    else:
        rec("PASS", "the healthcheck states how often and how many times it probes")

    # --- ordering ---
    dep = app.get("depends_on")
    if isinstance(dep, list):
        rec("FAIL", "app waits for db to be healthy, not merely started",
            "the list form of depends_on only waits for the container to start — use the long form: "
            "depends_on: db: condition: service_healthy")
    elif not isinstance(dep, dict) or "db" not in dep:
        rec("FAIL", "app waits for db to be healthy, not merely started",
            "add depends_on with db: condition: service_healthy to the app service")
    elif as_dict(dep.get("db")).get("condition") != "service_healthy":
        rec("FAIL", "app waits for db to be healthy, not merely started",
            "condition must be service_healthy (service_started only waits for the process)")
    else:
        rec("PASS", "app waits for db to be healthy, not merely started")

    # --- volumes ---
    declared = as_dict(doc.get("volumes"))
    data = [m for m in mounts(db) if m[1].rstrip("/") == "/var/lib/postgresql/data"]
    if not data:
        rec("FAIL", "database data lives on a declared named volume",
            "mount a named volume at /var/lib/postgresql/data, e.g. dbdata:/var/lib/postgresql/data")
    else:
        source = data[0][0]
        if not source:
            rec("FAIL", "database data lives on a declared named volume",
                "that is an anonymous volume — name it so docker compose down does not orphan your data")
        elif is_bind(source):
            rec("FAIL", "database data lives on a declared named volume",
                "%s is a bind mount; Postgres data belongs in a named volume Docker manages, not in a "
                "host directory with host uids and host filesystem semantics" % source)
        elif source not in declared:
            rec("FAIL", "database data lives on a declared named volume",
                "declare %s under the top-level volumes: key so Compose creates it" % source)
        else:
            rec("PASS", "database data lives on a declared named volume")

    binds = [m for m in mounts(app) if is_bind(m[0])]
    source_binds = [m for m in binds if os.path.basename(m[0].rstrip("/")) == "app"]
    if not binds:
        rec("FAIL", "app bind-mounts ./app for live iteration",
            "add a bind mount such as ./app:/src so an edit on the host is what the dev stage compiles")
    elif not source_binds:
        rec("FAIL", "app bind-mounts ./app for live iteration",
            "the bind mount must expose ./app (your source), not some other host directory")
    elif source_binds[0][1].rstrip("/") != "/src":
        rec("FAIL", "app bind-mounts ./app for live iteration",
            "mount ./app at /src — that is the dev stage's WORKDIR, the directory "
            "`go run .` actually compiles. Mounted anywhere else the container "
            "keeps compiling the copy baked in at build time, so it starts, serves, "
            "and silently ignores every edit you make")
    else:
        rec("PASS", "app bind-mounts ./app for live iteration")

    # --- published ports ---
    app_ports = port_pairs(app)
    if not app_ports:
        rec("FAIL", "app publishes an overridable host port",
            'add ports: - "${APP_PORT:-8080}:8080" so you can reach the app from the host')
    else:
        good = [p for p in app_ports if p[1] == "8080" and "${" in p[0]]
        if good:
            rec("PASS", "app publishes an overridable host port")
        elif not [p for p in app_ports if p[1] == "8080"]:
            rec("FAIL", "app publishes an overridable host port",
                "the container side must be 8080 — that is the port the app listens on")
        else:
            rec("FAIL", "app publishes an overridable host port",
                'the host side is hard-coded; use "${APP_PORT:-8080}:8080" so a teammate whose 8080 is '
                "taken changes .env instead of editing the committed file")

    db_ports = port_pairs(db)
    if [p for p in db_ports if p[0] == "5432"]:
        rec("FAIL", "db does not claim the host's 5432",
            "app talks to db over the compose network, so db needs no published port at all — remove "
            "ports: from db (or publish an overridable non-default host port) instead of colliding with "
            "any Postgres already on this machine")
    else:
        rec("PASS", "db does not claim the host's 5432")


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

live_reason="no Docker daemon is reachable here. Install Docker and re-run to get this check"
if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
	if out="$(docker compose config -q 2>&1)"; then
		record PASS "live: docker compose config accepts the file"
	else
		record FAIL "live: docker compose config accepts the file" \
			"Compose itself rejected the file: $(flatten "$out")"
	fi

	docker compose down -v >/dev/null 2>&1
	if out="$(docker compose up -d --build --wait --wait-timeout 180 2>&1)"; then
		addr="$(docker compose port app 8080 2>/dev/null)"
		port="${addr##*:}"
		reaches="live: the running app reaches the running database"
		if [ -z "$port" ]; then
			record SKIP "$reaches" "the app service published no host port to probe"
		elif ! command -v curl >/dev/null 2>&1; then
			record SKIP "$reaches" "curl is not installed; try it yourself with http://localhost:$port/dbping"
		else
			wait_for_app "$port"
			case $? in
			0)
				if body="$(curl -fsS --max-time 15 "http://127.0.0.1:$port/dbping" 2>&1)"; then
					record PASS "$reaches ($(flatten "$body"))"
				else
					record FAIL "$reaches" \
						"/dbping failed: $(flatten "$body"). Check DATABASE_URL's host and the db healthcheck"
				fi
				;;
			1)
				record FAIL "$reaches" \
					"the app container exited before it listened: $(flatten "$(docker compose logs --no-log-prefix --tail 5 app 2>&1)")"
				;;
			*)
				record SKIP "$reaches" \
					"the app was still not listening after 90s, usually a slow first compile of go run — re-run check.sh"
				;;
			esac
		fi
		docker compose down -v >/dev/null 2>&1
	else
		docker compose down -v >/dev/null 2>&1
		record SKIP "live: the running app reaches the running database" \
			"the stack did not come up in time, usually image pulls or a first build: $(flatten "$out")"
	fi
else
	record SKIP "live: docker compose config accepts the file" "$live_reason"
	record SKIP "live: the running app reaches the running database" "$live_reason"
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
		echo "The skipped checks are a bonus, not part of the grade — your files pass."
	fi
	exit 0
fi
printf '%d passed, %d failed, %d skipped for missing tooling.\n' "$pass" "$fail" "$skip"
echo "Fix the FAIL lines above and re-run: bash check.sh"
exit 1
