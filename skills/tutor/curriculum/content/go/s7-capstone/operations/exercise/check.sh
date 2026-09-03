#!/usr/bin/env bash
# Referee for Capstone: Operations. It reads YOUR project and checks the
# operational surface: container image, CI gates, runbook, telemetry,
# lifecycle and configuration.
#
# Everything here is static. No network, no docker daemon, no cluster, and
# nothing in your project is modified — run it as often as you like. Checks
# that need a tool this machine does not have are reported as "skip", never
# as failures, and bonus checks that run and complain only "warn".
#
# Run it from this folder:  bash check.sh
set -u
cd "$(dirname "$0")" || exit 1

pass=0
fail=0
skipped=0
warned=0

ok() {
	pass=$((pass + 1))
	printf 'ok    %s\n' "$1"
}

bad() {
	fail=$((fail + 1))
	printf 'FAIL  %s\n      fix: %s\n' "$1" "$2"
}

skip() {
	skipped=$((skipped + 1))
	printf 'skip  %s\n      why: %s\n' "$1" "$2"
}

warn() {
	warned=$((warned + 1))
	printf 'warn  %s\n      note: %s\n' "$1" "$2"
}

finish() {
	echo
	printf '%d passed, %d failed, %d skipped, %d warning(s)\n' \
		"$pass" "$fail" "$skipped" "$warned"
	if [ "$fail" -eq 0 ]; then
		echo "Operations checks green. Skips are checks this machine could not run;"
		echo "warnings never block. The rest is graded in review."
		exit 0
	fi
	echo "Fix the FAIL lines above and re-run: bash check.sh"
	exit 1
}

# ----------------------------------------------------------------- project

# candidate prints the absolute path of a usable project directory, or the
# reason it is not one, and returns non-zero in that case.
candidate() {
	local path="$1"
	if [ ! -d "$path" ]; then
		printf 'not a directory: %s' "$path"
		return 1
	fi
	if [ ! -f "$path/go.mod" ]; then
		printf 'no go.mod in: %s' "$path"
		return 1
	fi
	(cd "$path" && pwd)
}

first_line() {
	awk 'NF { gsub(/^[ \t]+|[ \t]+$/, ""); print; exit }' "$1"
}

# default_project prints the projects/capstone directory of the workspace this
# exercise was scaffolded into: the nearest ancestor holding
# projects/capstone/go.mod. How deep an exercise sits below the workspace root
# is decided by lesson_workspace_dir() in tutor.py, so it is searched for
# rather than spelled as a fixed number of "../".
default_project() {
	local dir up
	dir=$(pwd)
	up=0
	while [ "$up" -le 6 ]; do
		if [ -f "$dir/projects/capstone/go.mod" ]; then
			printf '%s' "$dir/projects/capstone"
			return 0
		fi
		[ "$dir" != "/" ] || break
		dir=$(dirname "$dir")
		up=$((up + 1))
	done
	return 1
}

project=""
tried=""
note_try() { tried="$tried
  $1"; }

if [ -n "${TUTOR_CAPSTONE_DIR:-}" ]; then
	if out=$(candidate "$TUTOR_CAPSTONE_DIR"); then
		project="$out"
	else
		note_try "TUTOR_CAPSTONE_DIR=$TUTOR_CAPSTONE_DIR — $out"
	fi
else
	note_try "TUTOR_CAPSTONE_DIR — not set"
fi

if [ -z "$project" ]; then
	if [ -f capstone.path ]; then
		line=$(first_line capstone.path)
		if [ -z "$line" ]; then
			note_try "capstone.path — present but empty"
		elif out=$(candidate "$line"); then
			project="$out"
		else
			note_try "capstone.path -> $line — $out"
		fi
	else
		note_try "capstone.path — not present"
	fi
fi

if [ -z "$project" ]; then
	if out=$(default_project); then
		project="$out"
	else
		note_try "../../../../projects/capstone — no projects/capstone containing a go.mod above $(pwd)"
	fi
fi

if [ -z "$project" ]; then
	bad "the capstone project is findable" \
		"point this script at your project — see the three ways below"
	cat <<'EOF'

No capstone project found. Tried:
EOF
	printf '%s\n' "$tried"
	cat <<'EOF'

Point the referee at your project in one of these ways:
  1. put its path on the first line of capstone.path, next to this script:
       echo ../../../../projects/capstone > capstone.path
     then check it: ls "$(cat capstone.path)/go.mod"
  2. export TUTOR_CAPSTONE_DIR=/absolute/path/to/your/project
  3. create the project where the planning lesson put it: projects/capstone at
     your workspace root — the directory holding lessons/ — and go mod init
     there. Do not create one under lessons/: that decoy grades instead of
     your capstone.

A capstone project is a directory containing a go.mod.
EOF
	finish
fi
ok "capstone project found: $project"

# ------------------------------------------------------------ file helpers

go_files=$(find "$project" -mindepth 1 \
	\( -name vendor -o -name testdata -o -name node_modules -o -name '.*' \) -prune -o \
	-type f -name '*.go' ! -name '*_test.go' -print 2>/dev/null)

# src_has reports whether any non-test Go file matches an extended regex.
src_has() {
	local pat="$1" f
	[ -n "$go_files" ] || return 1
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		if grep -Eq -- "$pat" "$f" 2>/dev/null; then
			return 0
		fi
	done <<EOF
$go_files
EOF
	return 1
}

# src_matches prints every match of an extended regex across non-test Go files.
src_matches() {
	local pat="$1" f
	[ -n "$go_files" ] || return 0
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		grep -Eho -- "$pat" "$f" 2>/dev/null
	done <<EOF
$go_files
EOF
}

# section prints the body of the first heading matching a lowercase regex,
# including any deeper subheadings, up to the next heading of the same or
# higher level.
section() {
	awk -v pat="$2" '
	/^#{1,6}[ \t]/ {
		lvl = 0
		while (substr($0, lvl + 1, 1) == "#") lvl++
		head = tolower($0)
		if (inside) {
			if (lvl <= curlvl) { inside = 0 } else { print; next }
		}
		if (!inside && head ~ pat) { inside = 1; curlvl = lvl }
		next
	}
	inside { print }
	' "$1" 2>/dev/null
}

has_heading() {
	grep -Ei "^#{1,6}[[:space:]].*($2)" "$1" >/dev/null 2>&1
}

find_first() {
	local rel
	for rel in "$@"; do
		if [ -f "$project/$rel" ]; then
			printf '%s' "$project/$rel"
			return 0
		fi
	done
	return 1
}

# ------------------------------------------------------------- 1. packaging

dockerfile=$(find_first Dockerfile Containerfile docker/Dockerfile \
	build/Dockerfile deploy/Dockerfile build/package/Dockerfile) || dockerfile=""

if [ -n "$dockerfile" ]; then
	ok "container build file found: ${dockerfile#"$project"/}"
else
	bad "the project has a Dockerfile" \
		"add a multi-stage Dockerfile at the project root — a toolchain stage that builds the binary, a minimal runtime stage that carries only it"
fi

if [ -z "$dockerfile" ]; then
	skip "the image build is multi-stage" "no Dockerfile to read"
	skip "every base image is pinned to a tag or digest" "no Dockerfile to read"
	skip "the runtime stage drops to a non-root user" "no Dockerfile to read"
else
	stages=$(grep -Eci '^[[:space:]]*FROM[[:space:]]' "$dockerfile")
	if [ "$stages" -ge 2 ]; then
		ok "the image build is multi-stage ($stages FROM stages)"
	else
		bad "the image build is multi-stage" \
			"one FROM means the compiler, the module cache and your source ship to production; build in one stage, COPY --from= the binary into a minimal second stage"
	fi

	unpinned=$(awk '
	{
		l = $0
		gsub(/\r/, "", l)
		sub(/^[ \t]+/, "", l)
		if (tolower(l) !~ /^from[ \t]/) next
		n = split(l, t, /[ \t]+/)
		ref = ""; alias = ""
		for (i = 2; i <= n; i++) {
			if (t[i] ~ /^--/) continue
			if (ref == "") { ref = t[i]; continue }
			if (tolower(t[i]) == "as" && i < n) { alias = tolower(t[i+1]); break }
		}
		low = tolower(ref)
		if (low == "scratch") { if (alias != "") stage[alias] = 1; next }
		if (low in stage) { if (alias != "") stage[alias] = 1; next }
		if (ref ~ /@sha256:/) { if (alias != "") stage[alias] = 1; next }
		p = 0
		for (i = length(ref); i > 0; i--) if (substr(ref, i, 1) == "/") { p = i; break }
		last = (p > 0) ? substr(ref, p + 1) : ref
		if (tolower(last) ~ /:latest$/) print "  " ref " — floating :latest tag"
		else if (index(last, ":") == 0) print "  " ref " — no tag at all (means :latest)"
		if (alias != "") stage[alias] = 1
	}
	' "$dockerfile")
	if [ -z "$unpinned" ]; then
		ok "every base image is pinned to a tag or digest"
	else
		bad "every base image is pinned to a tag or digest" \
			"pin these, so the image you debug is the image you shipped:
$unpinned"
	fi

	userstate=$(awk '
	{
		l = $0
		gsub(/\r/, "", l)
		sub(/^[ \t]+/, "", l)
		n = split(l, t, /[ \t]+/)
		if (n == 0) next
		kw = tolower(t[1])
		if (kw == "from") {
			lastfrom = NR
			base = ""
			for (i = 2; i <= n; i++) { if (t[i] !~ /^--/) { base = tolower(t[i]); break } }
			lastbase = base
		}
		if (kw == "user" && n >= 2) { userline = NR; user = tolower(t[2]) }
	}
	END {
		if (userline > lastfrom) {
			split(user, u, ":")
			if (u[1] == "root" || u[1] == "0") print "root"
			else print "ok"
		} else if (lastbase ~ /nonroot/) print "ok"
		else if (userline > 0) print "earlier-stage"
		else print "missing"
	}
	' "$dockerfile")
	case "$userstate" in
	ok)
		ok "the runtime stage drops to a non-root user"
		;;
	root)
		bad "the runtime stage drops to a non-root user" \
			"the final stage sets USER root (or 0) — a container escape then starts as root on the host; create an unprivileged user and switch to it"
		;;
	earlier-stage)
		bad "the runtime stage drops to a non-root user" \
			"the only USER instruction is in an earlier build stage; each stage starts fresh, so repeat USER after the final FROM"
		;;
	*)
		bad "the runtime stage drops to a non-root user" \
			"add a USER instruction after the final FROM (e.g. USER 65532:65532, or a nonroot base image) — containers run as root unless told otherwise"
		;;
	esac
fi

# --------------------------------------------------------- 2. deploy is code

deploy_evidence=""
for rel in Makefile makefile Taskfile.yml Taskfile.yaml justfile Justfile; do
	if [ -f "$project/$rel" ] &&
		grep -Eqi '^[[:space:]]*(deploy|release|publish|install|push)[a-z0-9_-]*:' "$project/$rel"; then
		deploy_evidence="$rel"
		break
	fi
done
if [ -z "$deploy_evidence" ]; then
	for rel in deploy.sh scripts/deploy.sh deploy/deploy.sh scripts/release.sh \
		deploy/release.sh compose.yaml compose.yml docker-compose.yml \
		docker-compose.yaml; do
		if [ -f "$project/$rel" ]; then
			deploy_evidence="$rel"
			break
		fi
	done
fi
if [ -z "$deploy_evidence" ]; then
	manifest=$(find "$project/deploy" "$project/k8s" "$project/manifests" \
		"$project/deployments" -maxdepth 2 -type f \( -name '*.yaml' -o -name '*.yml' \) \
		-print 2>/dev/null | head -n 1)
	if [ -n "$manifest" ]; then
		deploy_evidence="${manifest#"$project"/}"
	fi
fi
if [ -n "$deploy_evidence" ]; then
	ok "deployment is scripted, not remembered: $deploy_evidence"
else
	bad "deployment is scripted, not remembered" \
		"add the deploy steps as code — a Makefile deploy target, scripts/deploy.sh, a compose file or manifests under deploy/. Steps that live in your head are not repeatable and cannot be reviewed"
fi

# --------------------------------------------------------------- 3. CI gates

ci_files=$(find "$project" -mindepth 1 -maxdepth 4 -type f \
	\( -path '*/.github/workflows/*.yml' -o -path '*/.github/workflows/*.yaml' \
	-o -name '.gitlab-ci.yml' -o -path '*/.circleci/config.yml' \
	-o -name '.drone.yml' -o -name 'azure-pipelines.yml' \
	-o -path '*/.buildkite/*.yml' -o -name 'Jenkinsfile' \) -print 2>/dev/null)

gates=""
pipeline=""
if [ -n "$ci_files" ]; then
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		gates="$gates
$(cat "$f" 2>/dev/null)"
	done <<EOF
$ci_files
EOF
	# Triggers are declared in the pipeline file itself and nowhere else, so
	# keep a copy before the delegated build files are appended below —
	# otherwise a Makefile `push:` target reads as a push trigger.
	pipeline="$gates"
	# A workflow that calls make/task delegates its gates; read those too.
	if printf '%s' "$gates" | grep -Eq '(^|[^a-z])(make|task|just)[[:space:]]'; then
		for rel in Makefile makefile Taskfile.yml Taskfile.yaml justfile Justfile; do
			if [ -f "$project/$rel" ]; then
				gates="$gates
$(cat "$project/$rel")"
			fi
		done
	fi
fi

gate_has() { printf '%s' "$gates" | grep -Eq "$1"; }
pipeline_has() { printf '%s' "$pipeline" | grep -Eq "$1"; }

if [ -n "$ci_files" ]; then
	first_ci=$(printf '%s' "$ci_files" | head -n 1)
	ok "a CI pipeline is defined: ${first_ci#"$project"/}"
else
	bad "a CI pipeline is defined" \
		"add a pipeline definition (e.g. .github/workflows/ci.yml) that runs your gates on a clean machine — S4's ci-cd lesson has the anatomy"
fi

if [ -z "$ci_files" ]; then
	skip "CI runs on push and on proposed merges" "no pipeline file to read"
	skip "CI gate: go build" "no pipeline file to read"
	skip "CI gate: go test -race" "no pipeline file to read"
	skip "CI gate: go vet" "no pipeline file to read"
	skip "CI gate: govulncheck" "no pipeline file to read"
else
	if pipeline_has '(^|[^a-z_-])push([^a-z_-]|$)' &&
		pipeline_has '(pull_request|merge_request|pull-request|pullRequest)'; then
		ok "CI runs on push and on proposed merges"
	else
		bad "CI runs on push and on proposed merges" \
			"trigger on both push and pull_request (or your host's merge-request event) — a gate that only runs after the merge cannot block it"
	fi

	if gate_has 'go[[:space:]]+build'; then
		ok "CI gate: go build"
	else
		bad "CI gate: go build" \
			"add a build step (go build ./...); the machine that builds it must not be only yours"
	fi

	if gate_has 'go[[:space:]]+test[^#]*-race' || gate_has '-race[^#]*go[[:space:]]+test'; then
		ok "CI gate: go test -race"
	else
		bad "CI gate: go test -race" \
			"run the suite with the race detector in CI (go test -race ./...) — CI's slower, differently-scheduled machine finds races your laptop hides"
	fi

	if gate_has 'go[[:space:]]+vet'; then
		ok "CI gate: go vet"
	else
		bad "CI gate: go vet" \
			"add a vet step (go vet ./...) so the checks you run by hand also run when you forget"
	fi

	if gate_has 'govulncheck'; then
		ok "CI gate: govulncheck"
	else
		bad "CI gate: govulncheck" \
			"add a vulnerability gate (govulncheck ./...) — you ran it by hand in the hardening lesson; dependencies grow vulnerabilities while you are not looking"
	fi
fi

# ---------------------------------------------------------------- 4. runbook

runbook=$(find_first RUNBOOK.md runbook.md docs/RUNBOOK.md docs/runbook.md \
	docs/ops/RUNBOOK.md) || runbook=""

if [ -n "$runbook" ]; then
	ok "a runbook exists: ${runbook#"$project"/}"
else
	bad "a runbook exists" \
		"write RUNBOOK.md at the project root — the page you open at 3am, not the page you write for a portfolio"
fi

if [ -z "$runbook" ]; then
	skip "runbook covers deploy, rollback, triage, paging" "no runbook to read"
	skip "deploy and rollback are commands, not prose" "no runbook to read"
	skip "runbook names at least two failure modes" "no runbook to read"
else
	missing=""
	has_heading "$runbook" 'deploy|ship|release' || missing="$missing
  how to deploy"
	has_heading "$runbook" 'roll ?back|revert' || missing="$missing
  how to roll back"
	has_heading "$runbook" 'down|outage|broken|triage|diagnos|not responding|first response' ||
		missing="$missing
  what to check when it is down"
	has_heading "$runbook" 'page|escalat|on.?call|contact|owner' || missing="$missing
  who to page"
	if [ -z "$missing" ]; then
		ok "runbook covers deploy, rollback, triage, paging"
	else
		bad "runbook covers deploy, rollback, triage, paging" \
			"add headings for the missing sections:$missing"
	fi

	body_deploy=$(section "$runbook" 'deploy|ship|release')
	body_rollback=$(section "$runbook" 'roll ?back|revert')
	blocks_missing=""
	printf '%s' "$body_deploy" | grep -Eq '^[[:space:]]*(```|~~~|\$ )' ||
		blocks_missing="$blocks_missing deploy"
	printf '%s' "$body_rollback" | grep -Eq '^[[:space:]]*(```|~~~|\$ )' ||
		blocks_missing="$blocks_missing rollback"
	if [ -z "$blocks_missing" ]; then
		ok "deploy and rollback are commands, not prose"
	else
		bad "deploy and rollback are commands, not prose" \
			"put the literal commands in a fenced code block under:$blocks_missing — a step you cannot paste while half awake is not a step"
	fi

	body_modes=$(section "$runbook" 'failure mode|known failure|common failure|things that (go|have gone) wrong|when it breaks')
	modes=$(printf '%s' "$body_modes" | grep -Ec '^[[:space:]]*([-*+][[:space:]]|[0-9]+[.)][[:space:]]|#{2,6}[[:space:]])')
	if [ "$modes" -ge 2 ]; then
		ok "runbook names at least two failure modes ($modes)"
	else
		bad "runbook names at least two failure modes" \
			"add a 'Known failure modes' section with at least two entries — each one: the symptom you would see, the check that confirms it, and the action. Generic advice is not a failure mode"
	fi
fi

# ----------------------------------------------------------- 5. observability

if src_has '"log/slog"|"github.com/rs/zerolog|"go.uber.org/zap|"github.com/sirupsen/logrus|"github.com/go-logr/logr'; then
	if src_has '\.(Info|Warn|Error|Debug|InfoContext|ErrorContext|WarnContext)\('; then
		ok "logs are structured, with real call sites"
	else
		bad "logs are structured, with real call sites" \
			"a structured logger is imported but never used to log anything — replace the fmt.Print/log.Printf lines that carry operational meaning"
	fi
else
	bad "logs are structured, with real call sites" \
		"log with log/slog (S5's observability lesson) — key/value attributes are what makes a log searchable at 3am; free-text lines are not"
fi

health_by=""
if src_has '"[A-Z]* ?/(healthz|readyz|livez|health|ready|live)'; then
	health_by="an HTTP health/readiness route"
elif [ -n "$dockerfile" ] && grep -Eqi '^[[:space:]]*HEALTHCHECK' "$dockerfile"; then
	health_by="a HEALTHCHECK instruction in the Dockerfile"
elif [ -n "$runbook" ] && has_heading "$runbook" 'health|readiness|is it (up|healthy)|self.?check' &&
	printf '%s' "$(section "$runbook" 'health|readiness|is it (up|healthy)|self.?check')" |
	grep -Eq '^[[:space:]]*(```|~~~|\$ )'; then
	health_by="a documented health command in the runbook"
fi
if [ -n "$health_by" ]; then
	ok "there is a health signal: $health_by"
else
	bad "there is a health signal" \
		"a service: expose /healthz (is the process broken?) and /readyz (can it serve now?). A CLI or library: a self-check subcommand plus a runbook section showing the command and its expected output"
fi

readme=$(find_first README.md readme.md docs/README.md docs/readme.md) || readme=""

# counters_doc prints the doc file whose metrics/counters section names at
# least two things, for a CLI that reports counters at the end of a run
# instead of publishing them.
counters_doc() {
	local doc body named
	for doc in "$runbook" "$readme"; do
		[ -n "$doc" ] || continue
		has_heading "$doc" 'metric|counter|stats|tally' || continue
		body=$(section "$doc" 'metric|counter|stats|tally')
		named=$(printf '%s' "$body" |
			grep -Eo '`[A-Za-z_][A-Za-z0-9_.]+`|[A-Za-z][A-Za-z0-9]*_[A-Za-z0-9_]+' |
			sort -u | wc -l | tr -d ' ')
		if [ "$named" -ge 2 ]; then
			printf '%s' "$doc"
			return 0
		fi
	done
	return 1
}

# counters_computed reports whether the source actually keeps counts, rather
# than only documenting them.
counters_computed() {
	src_has '[A-Za-z0-9_.]*([Cc]ount|[Cc]ounter|[Ss]tat|[Tt]ally)[A-Za-z0-9_.]*[[:space:]]*(\+\+|\+=)' ||
		src_has 'type[[:space:]]+[A-Za-z0-9_]*([Cc]ount|[Ss]tats|[Tt]ally|[Mm]etrics)[A-Za-z0-9_]*[[:space:]]+struct'
}

metrics_by=""
if src_has '"expvar"|"runtime/metrics"|prometheus|go\.opentelemetry\.io|"/metrics"'; then
	metrics_by="counters the process publishes (expvar, /metrics or a client library)"
elif metrics_doc=$(counters_doc) && counters_computed; then
	metrics_by="end-of-run counters, documented in ${metrics_doc#"$project"/}"
fi

if [ -n "$metrics_by" ]; then
	ok "there is a metrics surface: $metrics_by"
elif ! src_has '^package[[:space:]]+main([[:space:]]|$)'; then
	skip "there is a metrics surface" \
		"no package main — a library hands its counters back to the caller instead of publishing them, so this one is reviewed in conversation rather than grepped"
else
	bad "there is a metrics surface" \
		"expose counters someone else can read. A service: publish them — stdlib expvar over /metrics is enough. A CLI or batch tool: count the work as you do it and print the totals at the end of a run, then list them under a Metrics heading in your README or runbook. Without numbers over time, 'is it healthy?' can only be answered by reading logs by hand"
fi

# ---------------------------------------------------------------- 6. lifecycle

if src_has 'signal\.NotifyContext' ||
	(src_has 'signal\.Notify' && src_has 'SIGTERM|os\.Interrupt'); then
	ok "the process handles termination signals"
else
	bad "the process handles termination signals" \
		"catch SIGTERM (signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)) — every rollout, scale-down and restart begins with SIGTERM, and an unhandled one kills you mid-request"
fi

if src_has 'http\.Server|ListenAndServe'; then
	if src_has '\.Shutdown\('; then
		ok "in-flight requests are drained on shutdown"
	else
		bad "in-flight requests are drained on shutdown" \
			"call srv.Shutdown(ctx) with a bounded timeout after the signal, so requests already in progress finish instead of being cut off"
	fi
else
	skip "in-flight requests are drained on shutdown" \
		"no http.Server found — for a CLI, cancellation of the signal context is the equivalent, and it is reviewed rather than grepped"
fi

# ----------------------------------------------------- 7. configuration surface

doc_files=$(find "$project" -mindepth 1 -maxdepth 3 \
	\( -name vendor -o -name node_modules -o -name '.*' \) -prune -o \
	-type f -name '*.md' -print 2>/dev/null)
doc_blob=""
if [ -n "$doc_files" ]; then
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		doc_blob="$doc_blob
$(cat "$f" 2>/dev/null)"
	done <<EOF
$doc_files
EOF
fi

config_doc=""
config_items=0
if [ -n "$doc_files" ]; then
	while IFS= read -r f; do
		[ -n "$f" ] || continue
		if has_heading "$f" 'config|environment|env var|settings|tunables'; then
			body=$(section "$f" 'config|environment|env var|settings|tunables')
			items=$(printf '%s' "$body" |
				grep -Eo '([A-Z][A-Z0-9_]{3,}|--[a-z][a-z0-9-]+)' | sort -u | wc -l | tr -d ' ')
			if [ "$items" -ge 2 ]; then
				config_doc="$f"
				config_items="$items"
				break
			fi
		fi
	done <<EOF
$doc_files
EOF
fi

if [ -n "$config_doc" ]; then
	ok "the configuration surface is documented: ${config_doc#"$project"/} ($config_items entries)"
else
	bad "the configuration surface is documented" \
		"add a Configuration section listing every knob — name, what it does, default, and whether it is a secret. Undocumented configuration is discovered by whoever is on call"
fi

env_names=$(src_matches 'os\.(Getenv|LookupEnv)\("[A-Za-z_][A-Za-z0-9_]*"' |
	sed -e 's/.*("//' -e 's/"$//' | sort -u)
undocumented=""
if [ -n "$env_names" ]; then
	while IFS= read -r name; do
		[ -n "$name" ] || continue
		if ! printf '%s' "$doc_blob" | grep -q -- "$name"; then
			undocumented="$undocumented
  $name"
		fi
	done <<EOF
$env_names
EOF
fi
if [ -z "$env_names" ]; then
	skip "every environment variable read by the code is documented" \
		"the code reads no environment variables through os.Getenv"
elif [ -z "$undocumented" ]; then
	ok "every environment variable read by the code is documented"
else
	bad "every environment variable read by the code is documented" \
		"these are read by the code but appear in no document:$undocumented"
fi

# ------------------------------------------------------------- bonus checks

if ! command -v docker >/dev/null 2>&1; then
	skip "bonus: the image actually builds" "docker is not installed"
elif [ -z "$dockerfile" ]; then
	skip "bonus: the image actually builds" "no Dockerfile to build"
elif [ "${TUTOR_OPS_ALLOW_NETWORK:-0}" != "1" ]; then
	skip "bonus: the image actually builds" \
		"building pulls base images; re-run with TUTOR_OPS_ALLOW_NETWORK=1 when you are online"
elif ! docker info >/dev/null 2>&1; then
	skip "bonus: the image actually builds" "the docker daemon is not responding"
else
	if docker build -f "$dockerfile" -t tutor-capstone-ops-check "$project" >/tmp/tutor-ops-docker.log 2>&1; then
		ok "bonus: the image actually builds"
	else
		warn "bonus: the image actually builds" \
			"docker build failed — see /tmp/tutor-ops-docker.log (this never blocks the lesson)"
	fi
fi

k8s_manifests=$(find "$project/deploy" "$project/k8s" "$project/manifests" \
	"$project/deployments" -maxdepth 2 -type f \( -name '*.yaml' -o -name '*.yml' \) \
	-print 2>/dev/null | head -n 20)
if ! command -v kubectl >/dev/null 2>&1; then
	skip "bonus: manifests parse (kubectl client dry-run)" "kubectl is not installed"
elif [ -z "$k8s_manifests" ]; then
	skip "bonus: manifests parse (kubectl client dry-run)" "no manifests under deploy/, k8s/ or manifests/"
elif [ "${TUTOR_OPS_ALLOW_NETWORK:-0}" != "1" ]; then
	skip "bonus: manifests parse (kubectl client dry-run)" \
		"this bonus check shells out to kubectl, which reads your kubeconfig and may reach for a cluster; re-run with TUTOR_OPS_ALLOW_NETWORK=1 when that is fine"
else
	dry_out=""
	dry_fail=0
	while IFS= read -r m; do
		[ -n "$m" ] || continue
		if ! out=$(kubectl apply --dry-run=client --validate=false -f "$m" 2>&1); then
			dry_fail=1
			dry_out="$dry_out
  ${m#"$project"/}: $(printf '%s' "$out" | head -n 1)"
		fi
	done <<EOF
$k8s_manifests
EOF
	if [ "$dry_fail" -eq 0 ]; then
		ok "bonus: manifests parse (kubectl client dry-run)"
	else
		warn "bonus: manifests parse (kubectl client dry-run)" \
			"kubectl could not parse:$dry_out"
	fi
fi

deploy_scripts=$(find "$project" -mindepth 1 -maxdepth 3 \
	\( -name vendor -o -name node_modules -o -name '.*' \) -prune -o \
	-type f -name '*.sh' -print 2>/dev/null | head -n 20)
if ! command -v shellcheck >/dev/null 2>&1; then
	skip "bonus: deploy scripts pass shellcheck" "shellcheck is not installed"
elif [ -z "$deploy_scripts" ]; then
	skip "bonus: deploy scripts pass shellcheck" "the project has no shell scripts"
else
	sc_fail=""
	while IFS= read -r s; do
		[ -n "$s" ] || continue
		if ! shellcheck -S warning "$s" >/dev/null 2>&1; then
			sc_fail="$sc_fail ${s#"$project"/}"
		fi
	done <<EOF
$deploy_scripts
EOF
	if [ -z "$sc_fail" ]; then
		ok "bonus: deploy scripts pass shellcheck"
	else
		warn "bonus: deploy scripts pass shellcheck" \
			"shellcheck has findings in:$sc_fail — run it yourself and read them; the script that deploys is the one you least want to be wrong"
	fi
fi

finish
