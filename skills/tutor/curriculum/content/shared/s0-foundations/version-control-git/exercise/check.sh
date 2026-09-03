#!/usr/bin/env bash
# Referee for the "Version Control with Git" exercise.
# Run from this folder:  bash check.sh
# Read-only and safe to re-run as often as you like.
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
		echo "All $pass checks passed. Your history tells a story — well done."
		exit 0
	fi
	echo "$pass passed, $fail failed. Fix the FAIL lines above and re-run: bash check.sh"
	exit 1
}

if ! command -v git >/dev/null 2>&1; then
	bad "git is installed" \
		"install Git first — see the One-time setup section of LESSON.md, then re-run"
	finish
fi
ok "git is installed"

# journal/.git specifically, not rev-parse: rev-parse would wrongly succeed by
# finding an enclosing repo when journal/ was never initialized.
if [ ! -d journal/.git ]; then
	bad "journal/ is a Git repository" \
		"create journal/, cd into it, run 'git init' (expected if you haven't started yet)"
	finish
fi
ok "journal/ is a Git repository"

commit_count=$(git -C journal rev-list --count HEAD 2>/dev/null || echo 0)
if [ "$commit_count" -ge 3 ]; then
	ok "history has at least 3 commits (found $commit_count)"
else
	bad "history has at least 3 commits (found $commit_count)" \
		"work through exercise steps 2-4; each step ends in a commit"
fi

if [ "$commit_count" -gt 0 ]; then
	short_subjects=$(git -C journal log --pretty=%s | awk 'length($0) < 10')
	if [ -z "$short_subjects" ]; then
		ok "every commit message is at least 10 characters"
	else
		bad "every commit message is at least 10 characters (too short: $(echo "$short_subjects" | head -n 3 | tr '\n' ';'))" \
			"messages like 'wip' or 'fix' tell future-you nothing — say what changed and why"
	fi
fi

if git -C journal ls-files --error-unmatch notes.txt >/dev/null 2>&1; then
	ok "notes.txt is tracked"
	notes_commits=$(git -C journal rev-list --count HEAD -- notes.txt)
	if [ "$notes_commits" -ge 2 ]; then
		ok "notes.txt appears in at least 2 commits (found $notes_commits)"
	else
		bad "notes.txt appears in at least 2 commits (found $notes_commits)" \
			"edit notes.txt again, check 'git diff', then stage and commit the change (step 3)"
	fi
else
	bad "notes.txt is tracked" \
		"create journal/notes.txt, then 'git add notes.txt' and commit it (step 2)"
fi

if git -C journal ls-files --error-unmatch .gitignore >/dev/null 2>&1; then
	ok ".gitignore is committed"
else
	bad ".gitignore is committed" \
		"write journal/.gitignore (step 4), then stage and commit it like any other file"
fi

if [ -f journal/secret.txt ]; then
	ok "secret.txt exists on disk"
else
	bad "secret.txt exists on disk" \
		"create journal/secret.txt with a pretend password (step 4)"
fi

if git -C journal check-ignore -q secret.txt; then
	ok "secret.txt is ignored"
else
	bad "secret.txt is ignored" \
		"add a 'secret.txt' line to journal/.gitignore"
fi

if git -C journal ls-files --error-unmatch secret.txt >/dev/null 2>&1; then
	bad "secret.txt is not tracked" \
		"it is in the draft or a commit — .gitignore only affects untracked files; ask your tutor how to untrack it"
else
	ok "secret.txt is not tracked"
fi

if git -C journal log --all --name-only --pretty=format: 2>/dev/null | grep -qx "secret.txt"; then
	bad "secret.txt appears nowhere in history" \
		"an old commit contains it, and commits are forever — deleting it later is not enough; ask your tutor"
else
	ok "secret.txt appears nowhere in history"
fi

build_file=$(find journal/build -type f 2>/dev/null | head -n 1)
if [ -n "$build_file" ]; then
	ok "build/ exists and contains a file"
	if git -C journal check-ignore -q "${build_file#journal/}"; then
		ok "files under build/ are ignored"
	else
		bad "files under build/ are ignored" \
			"add a 'build/' line (trailing slash) to journal/.gitignore"
	fi
else
	bad "build/ exists and contains a file" \
		"create journal/build/output.txt to play the role of compiled output (step 4)"
fi

if git -C journal log --all --name-only --pretty=format: 2>/dev/null | grep -q "^build/"; then
	bad "nothing under build/ appears in history" \
		"a commit contains build output — artifacts are regenerated, never committed; ask your tutor"
else
	ok "nothing under build/ appears in history"
fi

if [ -z "$(git -C journal status --porcelain 2>/dev/null)" ]; then
	ok "working tree is clean"
else
	bad "working tree is clean" \
		"run 'git status' inside journal/ — commit what belongs in history, ignore what doesn't"
fi

finish
