#!/usr/bin/env bash
# Verifies the Branches & Remotes exercise. Run from this folder, as often
# as you like:  bash check.sh
set -u
cd "$(dirname "$0")"

pass=0
fail=0

ok() {
    pass=$((pass + 1))
    printf 'ok   %s\n' "$1"
}

bad() {
    fail=$((fail + 1))
    printf 'FAIL %s\n' "$1"
    printf '     fix: %s\n' "$2"
}

finish() {
    printf '\n%d passed, %d failed\n' "$pass" "$fail"
    if [ "$fail" -eq 0 ]; then
        echo 'All green — tell your tutor you are ready for the quiz.'
        exit 0
    fi
    exit 1
}

g() { git -C repo "$@"; }

# repo/.git specifically, not rev-parse: rev-parse would wrongly succeed by
# finding an enclosing repo when repo/ was never initialized.
if [ ! -d repo/.git ]; then
    bad 'repo/ is a git repository' \
        'step 1 — from this folder: mkdir repo && cd repo && git init -b main'
    finish
fi
ok 'repo/ is a git repository'

if g rev-parse --verify -q refs/heads/main >/dev/null; then
    ok 'a branch named main exists'
else
    bad 'a branch named main exists' \
        'rename your current branch: git branch -M main'
    finish
fi

current=$(g symbolic-ref --short HEAD 2>/dev/null || echo '(detached)')
if [ "$current" = main ]; then
    ok 'you finished on the main branch'
else
    bad "you finished on the main branch (currently on: $current)" \
        'switch back: git switch main'
fi

if [ -z "$(g status --porcelain)" ]; then
    ok 'working tree is clean (everything committed)'
else
    bad 'working tree is clean' \
        'uncommitted changes remain — run git status, then git add and git commit them'
fi

if g cat-file -e main:list.md 2>/dev/null; then
    ok 'list.md is committed on main'
else
    bad 'list.md is committed on main' \
        'create repo/list.md (step 1), then: git add list.md && git commit'
    finish
fi

for b in add-milk weekend-plans; do
    if g rev-parse --verify -q "refs/heads/$b" >/dev/null; then
        ok "branch $b exists"
        if g merge-base --is-ancestor "$b" main; then
            ok "branch $b is merged into main"
        else
            bad "branch $b is merged into main" \
                "on main, run: git merge $b"
        fi
    else
        bad "branch $b exists" \
            "create it with git switch -c $b (see the exercise steps)"
    fi
done

if [ "$(g rev-list --merges --count main)" -ge 1 ]; then
    ok 'main has a merge commit (the conflict merge)'
else
    bad 'main has a merge commit' \
        'the weekend-plans merge must be a real merge — commit on BOTH main and weekend-plans before merging (steps 5-6)'
fi

content=$(g show main:list.md 2>/dev/null || true)
if printf '%s\n' "$content" | grep -qE '^(<<<<<<<|=======|>>>>>>>)'; then
    bad 'list.md has no leftover conflict markers' \
        'open list.md, delete the <<<<<<< / ======= / >>>>>>> lines, keep the final text, then git add list.md && git commit'
else
    ok 'list.md has no leftover conflict markers'
fi

first=$(printf '%s\n' "$content" | head -n 1)
if [ "$first" = '# Family weekend shopping list' ]; then
    ok 'conflict resolved: first line of list.md is correct'
else
    bad "conflict resolved (first line is: \"$first\")" \
        'the resolved first line must be exactly: # Family weekend shopping list'
fi

for item in bread milk; do
    if printf '%s\n' "$content" | grep -q -- "- $item"; then
        ok "list still contains \"- $item\""
    else
        bad "list still contains \"- $item\"" \
            "the merged list.md must keep its \"- $item\" line"
    fi
done

if [ -d remote.git ] && [ "$(git -C remote.git rev-parse --is-bare-repository 2>/dev/null)" = true ]; then
    ok 'remote.git is a bare repository (your stand-in GitHub)'
else
    bad 'remote.git is a bare repository' \
        'step 2 — from inside repo/: git init --bare ../remote.git'
    finish
fi

if g remote get-url origin >/dev/null 2>&1; then
    ok 'repo has a remote named origin'
else
    bad 'repo has a remote named origin' \
        'from inside repo/: git remote add origin ../remote.git'
    finish
fi

if [ "$(git -C remote.git rev-parse --verify -q refs/heads/main 2>/dev/null || true)" = "$(g rev-parse main)" ]; then
    ok 'the remote has your latest main (pushed)'
else
    bad 'the remote has your latest main' \
        'push after the final merge: git push origin main'
fi

local_branch=$(g rev-parse --verify -q refs/heads/add-milk 2>/dev/null || echo missing)
tracking=$(g rev-parse --verify -q refs/remotes/origin/add-milk 2>/dev/null || true)
if [ "$tracking" = "$local_branch" ]; then
    ok 'the add-milk branch was pushed (origin/add-milk matches it)'
else
    bad 'the add-milk branch was pushed' \
        'push it like a pull-request branch: git push -u origin add-milk'
fi

if [ "$(g rev-parse --abbrev-ref 'main@{upstream}' 2>/dev/null || true)" = origin/main ]; then
    ok 'local main tracks origin/main (upstream set)'
else
    bad 'local main tracks origin/main' \
        'set it once with git push -u origin main — then plain git push / git pull just work'
fi

finish
