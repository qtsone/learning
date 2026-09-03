#!/usr/bin/env bash
# Referee for the distribution exercise: builds your tool the way a release
# pipeline would and inspects your goreleaser config. Nothing here touches the
# network or git, and goreleaser itself is optional — checks that need a tool
# you do not have are reported SKIPPED, never FAILED.
# Run from this folder:  bash check.sh
set -u
cd "$(dirname "$0")"

VERSION=1.4.2
COMMIT=9f8e7d6c
DATE=2024-05-01T10:00:00Z
PKG=tutor.local/digest/internal/buildinfo
LDFLAGS="-s -w -X $PKG.version=$VERSION -X $PKG.commit=$COMMIT -X $PKG.date=$DATE"
HELLO_SHA=5891b5b522d5df086d0ff0b110fbd9d21bb4fc7163af34d08286a2e846f6be03

tmp=$(mktemp -d) || exit 1
trap 'rm -rf "$tmp"' EXIT

pass=0
fail=0
skip=0

ok() {
	pass=$((pass + 1))
	printf 'ok    %s\n' "$1"
}

bad() {
	fail=$((fail + 1))
	printf 'FAIL  %s\n      fix: %s\n' "$1" "$2"
}

skipped() {
	skip=$((skip + 1))
	printf 'SKIP  %s\n      %s\n' "$1" "$2"
}

finish() {
	echo
	if [ "$fail" -eq 0 ]; then
		echo "All $pass checks passed ($skip skipped). This tool is shippable."
		exit 0
	fi
	echo "$pass passed, $fail failed, $skip skipped. Fix the FAIL lines above and re-run: bash check.sh"
	exit 1
}

# ---------------------------------------------------------------- toolchain

if ! command -v go >/dev/null 2>&1; then
	bad "go toolchain is installed" \
		"install Go and put it on PATH (S0 dev-environment), then re-run"
	finish
fi
ok "go toolchain is installed"

if [ -z "$(gofmt -l . 2>/dev/null)" ]; then
	ok "gofmt reports nothing to fix"
else
	bad "gofmt reports nothing to fix" \
		"run 'gofmt -l .' to see the files, then 'gofmt -w .'"
fi

if go vet ./... >"$tmp/vet.log" 2>&1; then
	ok "go vet ./... passes"
else
	bad "go vet ./... passes" \
		"run 'go vet ./...' and read the finding: $(head -n 3 "$tmp/vet.log" | tr '\n' ' ')"
fi

# ---------------------------------------------------------------- the binary

if ! go build -o "$tmp/plain" . >"$tmp/build.log" 2>&1; then
	bad "the module builds (go build .)" \
		"read the compiler output: $(head -n 3 "$tmp/build.log" | tr '\n' ' ')"
	finish
fi
ok "the module builds (go build .)"

printf 'hello\n' >"$tmp/hello.txt"
if "$tmp/plain" "$tmp/hello.txt" 2>/dev/null | grep -q "^$HELLO_SHA  "; then
	ok "digest still hashes files correctly"
else
	bad "digest still hashes files correctly" \
		"the tool's own job must keep working — restore main.go's behaviour"
fi

"$tmp/plain" --version >"$tmp/v.out" 2>"$tmp/v.err"
rc=$?
if [ "$rc" -eq 0 ]; then
	ok "digest --version exits 0"
else
	bad "digest --version exits 0" \
		"--version is a successful run, not an error; got exit code $rc"
fi

if [ -s "$tmp/v.out" ] && [ ! -s "$tmp/v.err" ]; then
	ok "digest --version writes to stdout and leaves stderr empty"
else
	bad "digest --version writes to stdout and leaves stderr empty" \
		"version information is data a script may read: stdout only (terminal-ux lesson)"
fi

if head -n 1 "$tmp/v.out" | grep -Eq '^digest[[:space:]]+dev([[:space:]]|$)'; then
	ok "an unstamped build reports the placeholder version 'dev'"
else
	bad "an unstamped build reports the placeholder version 'dev'" \
		"with no -X, String() must print 'digest dev (…)'; '(devel)' from ReadBuildInfo is not a real version. Got: $(head -n 1 "$tmp/v.out")"
fi

# ---------------------------------------------------------------- stamping

if ! go build -trimpath -ldflags "$LDFLAGS" -o "$tmp/rel1" . >"$tmp/rel.log" 2>&1; then
	bad "the tool builds with the documented -ldflags" \
		"go build -trimpath -ldflags \"$LDFLAGS\" -o digest . said: $(head -n 3 "$tmp/rel.log" | tr '\n' ' ')"
	finish
fi
ok "the tool builds with the documented -ldflags"

"$tmp/rel1" --version >"$tmp/rv.out" 2>/dev/null
stamped=$(head -n 1 "$tmp/rv.out")

if printf '%s\n' "$stamped" | grep -Eq "^digest[[:space:]]+v?$VERSION([[:space:]]|$)"; then
	ok "--version reports the injected version ($VERSION), not a placeholder"
else
	bad "--version reports the injected version ($VERSION), not a placeholder" \
		"-X silently does nothing when the symbol path is wrong: it must be $PKG.version, and version must be a package-level string var. Got: $stamped"
fi

if printf '%s\n' "$stamped" | grep -Fq "$COMMIT"; then
	ok "--version reports the injected commit"
else
	bad "--version reports the injected commit" \
		"stamp and print commit the same way as version ($PKG.commit). Got: $stamped"
fi

if printf '%s\n' "$stamped" | grep -Fq "$DATE"; then
	ok "--version reports the injected build date"
else
	bad "--version reports the injected build date" \
		"stamp and print date the same way as version ($PKG.date). Got: $stamped"
fi

if printf '%s\n' "$stamped" | grep -Eq 'go1\.[0-9]+' &&
	printf '%s\n' "$stamped" | grep -Fq "$(go env GOOS)/$(go env GOARCH)"; then
	ok "--version reports the go version and the target platform"
else
	bad "--version reports the go version and the target platform" \
		"runtime.Version(), runtime.GOOS and runtime.GOARCH belong in the line — no linker flag needed. Got: $stamped"
fi

go build -trimpath -ldflags "$LDFLAGS" -o "$tmp/rel2" . >/dev/null 2>&1
if cmp -s "$tmp/rel1" "$tmp/rel2"; then
	ok "the build is reproducible: same inputs, byte-identical binary"
else
	bad "the build is reproducible: same inputs, byte-identical binary" \
		"something varying got baked in — build with -trimpath and never stamp a wall-clock timestamp"
fi

# ---------------------------------------------------------------- cross-compilation

for target in linux/amd64 darwin/arm64 windows/amd64; do
	goos=${target%/*}
	goarch=${target#*/}
	out="$tmp/digest-$goos-$goarch"
	[ "$goos" = windows ] && out="$out.exe"
	if CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -ldflags "$LDFLAGS" -o "$out" . >"$tmp/x.log" 2>&1; then
		ok "cross-compiles for $target with CGO_ENABLED=0"
	else
		bad "cross-compiles for $target with CGO_ENABLED=0" \
			"pure-Go code needs no cross toolchain: $(head -n 3 "$tmp/x.log" | tr '\n' ' ')"
	fi
done

# ---------------------------------------------------------------- goreleaser config

cfg=""
for candidate in .goreleaser.yaml .goreleaser.yml goreleaser.yaml goreleaser.yml; do
	if [ -f "$candidate" ]; then
		cfg=$candidate
		break
	fi
done

if [ -z "$cfg" ]; then
	bad "a goreleaser config exists (.goreleaser.yaml)" \
		"write .goreleaser.yaml in this folder — the walkthrough in LESSON.md shows every key the checks below want"
	finish
fi
ok "goreleaser config found: $cfg"

facts="$tmp/facts"
: >"$facts"

to_json() {
	if python3 -c 'import yaml' >/dev/null 2>&1; then
		python3 -c 'import json,sys,yaml; json.dump(yaml.safe_load(open(sys.argv[1])), sys.stdout)' \
			"$cfg" >"$tmp/cfg.json" 2>"$tmp/yaml.err" && return 0
		return 1
	fi
	if command -v yq >/dev/null 2>&1; then
		yq -o=json '.' "$cfg" >"$tmp/cfg.json" 2>"$tmp/yaml.err" && return 0
		return 1
	fi
	return 2
}

facts_from_grep() {
	skipped "$cfg parses as YAML" \
		"no YAML parser available here (python3 with PyYAML, or yq) — falling back to text matching, so the key checks below are less precise"
	has() { grep -Eq "$1" "$cfg" && echo "$2=1" || echo "$2=0"; }
	{
		has '^version:[[:space:]]*2[[:space:]]*$' cfgversion2
		has '^builds:' builds
		has 'CGO_ENABLED=0' cgo0
		has '\-trimpath' trimpath
		has 'internal/buildinfo\.version=' xversion
		has 'internal/buildinfo\.commit=' xcommit
		has 'internal/buildinfo\.date=' xdate
		has '\{\{' template
		has '^archives:' archives
		has '^checksum:' checksum
		has '^changelog:' changelog
		has '^(brews|homebrew_casks):' brew
		has 'name:[[:space:]]*homebrew-' tap
	} >"$facts"
	if [ "$(grep -Ec '^[[:space:]]*-[[:space:]]*(linux|darwin|windows|freebsd)[[:space:]]*$' "$cfg")" -ge 2 ] ||
		grep -Eq 'goos:[[:space:]]*\[[^]]*,' "$cfg"; then
		echo "goosmulti=1" >>"$facts"
	else
		echo "goosmulti=0" >>"$facts"
	fi
	if [ "$(grep -Ec '^[[:space:]]*-[[:space:]]*(amd64|arm64|386|arm)[[:space:]]*$' "$cfg")" -ge 2 ] ||
		grep -Eq 'goarch:[[:space:]]*\[[^]]*,' "$cfg"; then
		echo "goarchmulti=1" >>"$facts"
	else
		echo "goarchmulti=0" >>"$facts"
	fi
}

to_json
parse_rc=$?

if [ "$parse_rc" -eq 1 ]; then
	bad "$cfg parses as YAML" \
		"YAML is indentation-sensitive and never accepts tabs: $(tail -n 3 "$tmp/yaml.err" | tr '\n' ' ')"
	finish
fi

if [ "$parse_rc" -eq 0 ]; then
	python3 - "$tmp/cfg.json" >"$facts" 2>/dev/null <<'PY'
import json, sys

cfg = json.load(open(sys.argv[1]))
if not isinstance(cfg, dict):
    cfg = {}


def aslist(v):
    if v is None:
        return []
    if isinstance(v, list):
        return [str(x) for x in v]
    return [str(v)]


builds = [b for b in (cfg.get("builds") or []) if isinstance(b, dict)]
env = " ".join(x for b in builds for x in aslist(b.get("env")))
flags = " ".join(x for b in builds for x in aslist(b.get("flags")))
ld = " ".join(x for b in builds for x in aslist(b.get("ldflags")))
goos = max([len(aslist(b.get("goos"))) for b in builds] or [0])
goarch = max([len(aslist(b.get("goarch"))) for b in builds] or [0])

brews = [b for b in (cfg.get("brews") or cfg.get("homebrew_casks") or []) if isinstance(b, dict)]
tap = 0
for b in brews:
    repo = b.get("repository") or b.get("tap") or {}
    if isinstance(repo, dict) and str(repo.get("name", "")).startswith("homebrew-"):
        tap = 1

facts = {
    "cfgversion2": str(cfg.get("version")) == "2",
    "builds": bool(builds),
    "cgo0": "CGO_ENABLED=0" in env.replace(" ", ""),
    "trimpath": "-trimpath" in flags,
    "xversion": "internal/buildinfo.version=" in ld,
    "xcommit": "internal/buildinfo.commit=" in ld,
    "xdate": "internal/buildinfo.date=" in ld,
    "template": "{{" in ld,
    "goosmulti": goos >= 2,
    "goarchmulti": goarch >= 2,
    "archives": bool(cfg.get("archives")),
    "checksum": bool(cfg.get("checksum")),
    "changelog": "changelog" in cfg,
    "brew": bool(brews),
    "tap": bool(tap),
}
for k, v in facts.items():
    print("%s=%d" % (k, 1 if v else 0))
PY
	if [ -s "$facts" ]; then
		ok "$cfg parses as YAML"
	else
		facts_from_grep
	fi
else
	facts_from_grep
fi

check_fact() {
	if grep -qx "$1=1" "$facts"; then
		ok "$2"
	else
		bad "$2" "$3"
	fi
}

check_fact cfgversion2 "config declares 'version: 2'" \
	"goreleaser v2 refuses a config without a top-level 'version: 2'"
check_fact builds "config has a builds section" \
	"add a builds: list with one entry for this tool"
check_fact cgo0 "the build sets CGO_ENABLED=0" \
	"add env: [CGO_ENABLED=0] to the build — cgo would need a C cross-toolchain per target"
check_fact trimpath "the build passes -trimpath" \
	"add flags: [-trimpath] so absolute paths from your machine stay out of the binary"
check_fact xversion "ldflags stamp the version symbol" \
	"add -X $PKG.version=… to the build's ldflags"
check_fact xcommit "ldflags stamp the commit symbol" \
	"add -X $PKG.commit=… to the build's ldflags"
check_fact xdate "ldflags stamp the date symbol" \
	"add -X $PKG.date=… to the build's ldflags"
check_fact template "ldflags use goreleaser template values" \
	"hard-coded values would be wrong on the next tag — use {{ .Version }}, {{ .FullCommit }}, {{ .CommitDate }}"
check_fact goosmulti "the build targets at least two operating systems" \
	"list goos: linux, darwin, windows"
check_fact goarchmulti "the build targets at least two architectures" \
	"list goarch: amd64, arm64"
check_fact archives "config has an archives section" \
	"users download an archive, not a bare binary — add archives:"
check_fact checksum "config has a checksum section" \
	"add checksum: so the release carries checksums.txt"
check_fact changelog "config has a changelog section" \
	"add changelog: — the release notes come from the commits since the last tag"
check_fact brew "config publishes a Homebrew formula" \
	"add a brews: entry (or homebrew_casks: on newer goreleaser)"
check_fact tap "the Homebrew repository is a tap (name starts with homebrew-)" \
	"brew only recognises taps whose repository is named homebrew-<something>, e.g. homebrew-tap"

if command -v goreleaser >/dev/null 2>&1; then
	if goreleaser check >"$tmp/gr.log" 2>&1; then
		ok "goreleaser check passes"
	else
		bad "goreleaser check passes" \
			"read its output: $(tail -n 5 "$tmp/gr.log" | tr '\n' ' ')"
	fi
else
	skipped "goreleaser check passes" \
		"goreleaser is not installed — install it later and run 'goreleaser check' plus 'goreleaser release --snapshot --clean'"
fi

finish
