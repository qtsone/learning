#!/usr/bin/env bash
# Referee for the CI/CD exercise: inspects your workflow file and runs your
# gates locally, exactly as a pipeline runner would. No network, no git.
# Run from this folder:  bash check.sh
set -u
cd "$(dirname "$0")"

pass=0
fail=0

ok() {
	pass=$((pass + 1))
	printf 'ok    %s\n' "$1"
}

bad() {
	fail=$((fail + 1))
	printf 'FAIL  %s\n      fix: %s\n' "$1" "$2"
}

finish() {
	echo
	if [ "$fail" -eq 0 ]; then
		echo "All $pass checks passed. Green pipeline — this repo is safe to merge into."
		exit 0
	fi
	echo "$pass passed, $fail failed. Fix the FAIL lines above and re-run: bash check.sh"
	exit 1
}

if ! command -v go >/dev/null 2>&1; then
	bad "go toolchain is installed" \
		"install Go and ensure it is on PATH (see the S0 dev-environment lesson), then re-run"
	finish
fi
ok "go toolchain is installed"

wf=$(find .github/workflows -maxdepth 1 \( -name '*.yml' -o -name '*.yaml' \) 2>/dev/null | sort | head -n 1)
if [ -z "$wf" ]; then
	bad "a workflow file exists under .github/workflows/" \
		"create .github/workflows/ci.yml — the anatomy section of LESSON.md shows the shape"
	finish
fi
ok "workflow file found: $wf"

if grep -Eq '(^|[^[:alnum:]_])push([^[:alnum:]_-]|$)' "$wf"; then
	ok "workflow triggers on push"
else
	bad "workflow triggers on push" \
		"add push under the on: key so every push starts a run"
fi

if grep -Eq '(^|[^[:alnum:]_])pull_request([^[:alnum:]_-]|$)' "$wf"; then
	ok "workflow triggers on pull_request"
else
	bad "workflow triggers on pull_request" \
		"add pull_request under the on: key so proposed merges are checked before they land"
fi

if grep -Eq 'runs-on:' "$wf"; then
	ok "job declares a runner (runs-on:)"
else
	bad "job declares a runner (runs-on:)" \
		"give your job runs-on: ubuntu-latest — every job needs a machine to run on"
fi

if grep -q 'actions/checkout' "$wf"; then
	ok "workflow checks out the repository (actions/checkout)"
else
	bad "workflow checks out the repository (actions/checkout)" \
		"add a step 'uses: actions/checkout@v4' — the runner's machine starts empty"
fi

if grep -q 'actions/setup-go' "$wf"; then
	ok "workflow installs Go (actions/setup-go)"
else
	bad "workflow installs Go (actions/setup-go)" \
		"add a step 'uses: actions/setup-go@v5' — a fresh machine has no toolchain"
fi

if grep -q 'gofmt' "$wf"; then
	ok "workflow has a gofmt formatting gate"
else
	bad "workflow has a gofmt formatting gate" \
		"add a run step for formatting; remember gofmt -l always exits 0 — LESSON.md shows the test -z idiom"
fi

if grep -Eq 'go[[:space:]]+vet' "$wf"; then
	ok "workflow has a go vet gate"
else
	bad "workflow has a go vet gate" \
		"add a run step: go vet ./..."
fi

if grep -Eq 'go[[:space:]]+test' "$wf"; then
	ok "workflow runs the tests (go test)"
else
	bad "workflow runs the tests (go test)" \
		"add a run step: go test ./..."
fi

if grep -q 'func TestMean' stats_test.go 2>/dev/null; then
	ok "TestMean still exists"
else
	bad "TestMean still exists" \
		"the tests are the specification — restore stats_test.go and fix the code instead"
fi

if grep -q 'func Describe' stats.go 2>/dev/null; then
	ok "Describe still exists"
else
	bad "Describe still exists" \
		"deleting flagged code is appeasing the gate — restore Describe and fix what vet complained about"
fi

if [ -z "$(gofmt -l . 2>/dev/null)" ]; then
	ok "gate: gofmt reports nothing to fix"
else
	bad "gate: gofmt reports nothing to fix" \
		"run 'gofmt -l .' to list offending files, then 'gofmt -w .' to fix them"
fi

if go vet ./... >/dev/null 2>&1; then
	ok "gate: go vet ./... passes"
else
	bad "gate: go vet ./... passes" \
		"run 'go vet ./...' yourself and read its finding: which function, which verb, which argument?"
fi

if go test ./... >/dev/null 2>&1; then
	ok "gate: go test ./... passes"
else
	bad "gate: go test ./... passes" \
		"run 'go test ./...' and read the log like a CI log: which test failed, expected what, got what — then fix the code"
fi

finish
