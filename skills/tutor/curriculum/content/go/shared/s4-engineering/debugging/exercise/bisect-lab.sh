#!/usr/bin/env bash
# Optional lab for the "Bisecting over history" section of LESSON.md.
#
# Builds a throwaway git repo in a temp directory: 12 innocent-looking
# commits, one of which quietly broke the test suite. Your job is to find
# the culprit with git bisect. Nothing here touches this exercise, your
# tutor workspace, or any repo of yours.
#
# Run it with:  bash bisect-lab.sh
set -euo pipefail

lab=$(mktemp -d "${TMPDIR:-/tmp}/bisect-lab.XXXXXX")
cd "$lab"

git init -q
git config user.name "Bisect Lab"
git config user.email "lab@example.invalid"
git config commit.gpgsign false

cat > go.mod <<'EOF'
module tutor.local/bisect-lab

go 1.22
EOF

cat > pricing.go <<'EOF'
package pricing

// Amount returns the price for qty units, with a 10% bulk discount
// from 10 units up.
func Amount(unit float64, qty int) float64 {
	total := unit * float64(qty)
	if qty >= 10 {
		total *= 0.9
	}
	return total
}
EOF

cat > pricing_test.go <<'EOF'
package pricing

import "testing"

func TestAmount(t *testing.T) {
	cases := []struct {
		name string
		unit float64
		qty  int
		want float64
	}{
		{"no discount below ten", 2, 9, 18},
		{"discount starts at ten", 2, 10, 18},
		{"discount above ten", 2, 20, 36},
	}
	for _, c := range cases {
		if got := Amount(c.unit, c.qty); got != c.want {
			t.Errorf("%s: Amount(%v, %d) = %v, want %v", c.name, c.unit, c.qty, got, c.want)
		}
	}
}
EOF

git add .
git commit -qm "v1: pricing engine with bulk discount"

for i in $(seq 2 12); do
	msg="v$i: routine maintenance"
	echo "$msg" >> CHANGELOG.md
	if [ "$i" -eq 8 ]; then
		sed 's/qty >= 10/qty > 10/' pricing.go > pricing.tmp && mv pricing.tmp pricing.go
		msg="v$i: tidy the discount check"
	fi
	git add .
	git commit -qm "$msg"
done

first=$(git rev-list --max-parents=0 HEAD)

cat <<EOF

Lab repo ready in: $lab

Start here:

  cd $lab
  go test ./...        # fails — somewhere in these 12 commits a regression slipped in
  git log --oneline    # 12 commits; the messages will not confess

Hunt it down in ~3-4 verdicts instead of reading 12 diffs:

  git bisect start
  git bisect bad                     # HEAD is broken
  git bisect good $first
  go test ./...                      # then: git bisect good  (or)  git bisect bad
                                     # repeat until git names the first bad commit

Or let git drive the whole hunt on test exit codes:

  git bisect reset
  git bisect start HEAD $first
  git bisect run go test ./...

When you have the culprit: read its diff with 'git show <sha>', check it
matches what the test says, then 'git bisect reset'.

Clean up when done:  rm -rf $lab
EOF
