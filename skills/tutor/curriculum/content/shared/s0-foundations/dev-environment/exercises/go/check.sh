#!/usr/bin/env bash
# Verifies the dev-environment exercise (LESSON.md has the acceptance
# criteria). Read-only and safe to re-run; run it from a freshly opened
# terminal so your shell-profile changes are actually loaded.
set -u

pass=0
fail=0

ok() {
	printf 'ok   %s\n' "$1"
	pass=$((pass + 1))
}

bad() {
	printf 'FAIL %s\n     fix: %s\n' "$1" "$2"
	fail=$((fail + 1))
}

finish() {
	printf '\n%d check(s) passed, %d failed.\n' "$pass" "$fail"
	if [ "$fail" -eq 0 ]; then
		echo "All green — your dev environment works end to end."
		exit 0
	fi
	echo "Fix the FAIL lines above (top to bottom), open a NEW terminal, re-run."
	exit 1
}

# --- 1. toolchain installed and on PATH -------------------------------------

if command -v go >/dev/null 2>&1; then
	ok "the 'go' command is on PATH ($(command -v go))"
else
	bad "the 'go' command was not found" \
		"install Go (LESSON.md, 'Installing the toolchain'), then open a NEW terminal"
	echo
	echo "Nothing else can be checked until 'go' runs. Fix this first."
	finish
fi

if go version 2>/dev/null | grep -Eq 'go[0-9]+\.[0-9]+'; then
	ok "go version reports: $(go version)"
else
	bad "'go version' did not print a version line" \
		"the installation looks broken — reinstall from https://go.dev/dl/"
fi

# --- 2. Go's tool directory on PATH -----------------------------------------

gobin="$(go env GOBIN)"
if [ -z "$gobin" ]; then
	gobin="$(go env GOPATH)/bin"
fi
case ":$PATH:" in
*":$gobin:"*)
	ok "Go's tool directory is on PATH ($gobin)"
	;;
*)
	bad "Go's tool directory ($gobin) is not on PATH" \
		"add   export PATH=\"\$PATH:$gobin\"   to your shell profile, then open a new terminal"
	;;
esac

path_in_profile=""
for profile in .zshrc .zprofile .bashrc .bash_profile .profile; do
	if [ -f "$HOME/$profile" ] && grep -Eq "go/bin|GOPATH|GOBIN" "$HOME/$profile"; then
		path_in_profile="$HOME/$profile"
		break
	fi
done
if [ -n "$path_in_profile" ]; then
	ok "the PATH line lives in a shell profile ($path_in_profile)"
else
	bad "no shell profile mentions Go's tool directory" \
		"a PATH edit typed in the terminal dies with the window — put the export line in ~/.zshrc (zsh) or ~/.bashrc (bash), then open a new terminal"
fi

# --- 3. PROJECTS_DIR set persistently and existing --------------------------

if [ -z "${PROJECTS_DIR:-}" ]; then
	bad "the PROJECTS_DIR environment variable is not set" \
		"add   export PROJECTS_DIR=\"\$HOME/projects\"   (or your chosen folder) to your shell profile, then open a new terminal"
elif [ ! -d "$PROJECTS_DIR" ]; then
	bad "PROJECTS_DIR points at '$PROJECTS_DIR', which is not a directory" \
		"create it:   mkdir -p \"\$PROJECTS_DIR\""
else
	ok "PROJECTS_DIR is set and exists ($PROJECTS_DIR)"
fi

in_profile=""
for profile in .zshrc .zprofile .bashrc .bash_profile .profile; do
	if [ -f "$HOME/$profile" ] && grep -q "PROJECTS_DIR" "$HOME/$profile"; then
		in_profile="$HOME/$profile"
		break
	fi
done
if [ -n "$in_profile" ]; then
	ok "PROJECTS_DIR is set in a shell profile ($in_profile)"
else
	bad "PROJECTS_DIR was not found in any shell profile file" \
		"an export typed in the terminal dies with the window — put the line in ~/.zshrc (zsh) or ~/.bashrc (bash)"
fi

# --- 4-6. the hello project -------------------------------------------------

if [ -z "${PROJECTS_DIR:-}" ] || [ ! -d "${PROJECTS_DIR:-}" ]; then
	echo
	echo "Skipping the project checks until PROJECTS_DIR is set and exists."
	finish
fi

proj="$PROJECTS_DIR/hello"
if [ ! -d "$proj" ]; then
	bad "the project directory $proj does not exist" \
		"create it (LESSON.md, 'Exercise'):   mkdir -p \"\$PROJECTS_DIR/hello\""
	finish
fi
ok "project directory exists ($proj)"

for f in main.go go.mod README.md .gitignore; do
	if [ -f "$proj/$f" ]; then
		ok "$f is present"
	else
		bad "$proj/$f is missing" \
			"create it — LESSON.md lists every file the project needs"
	fi
done

if [ -f "$proj/.gitignore" ] && grep -Eq '^/?hello$' "$proj/.gitignore"; then
	ok ".gitignore ignores the built binary (hello)"
elif [ -f "$proj/.gitignore" ]; then
	bad ".gitignore does not mention the built binary" \
		"add a line containing just   hello   so the compiled program is never committed"
fi

if [ -d "$proj/.git" ]; then
	ok "the project is a git repository"
	if git -C "$proj" rev-parse HEAD >/dev/null 2>&1; then
		ok "the repository has at least one commit"
	else
		bad "the repository has no commits yet" \
			"inside $proj run:   git add .   then   git commit -m \"Initial commit\""
	fi
	if git -C "$proj" cat-file -e HEAD:main.go 2>/dev/null; then
		ok "main.go is part of a commit"
	else
		bad "main.go has not been committed (staging with 'git add' is not enough)" \
			"inside $proj run:   git add main.go   then   git commit -m \"Add main.go\""
	fi
else
	bad "$proj is not a git repository" \
		"inside $proj run:   git init"
fi

if [ ! -f "$proj/main.go" ] || [ ! -f "$proj/go.mod" ]; then
	echo
	echo "Skipping the build check until main.go and go.mod exist."
	finish
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
if (cd "$proj" && go build -o "$tmpdir/hello" .) >"$tmpdir/build.log" 2>&1; then
	ok "go build succeeds"
	got="$("$tmpdir/hello")"
	want="Hello from my dev environment!"
	if [ "$got" = "$want" ]; then
		ok "the program prints the expected line"
	else
		bad "the program printed '$got'" \
			"it must print exactly '$want' — compare your main.go with LESSON.md, character by character"
	fi
else
	bad "go build failed inside $proj" \
		"read the compiler output below and compare main.go with LESSON.md"
	sed 's/^/     | /' "$tmpdir/build.log"
fi

finish
