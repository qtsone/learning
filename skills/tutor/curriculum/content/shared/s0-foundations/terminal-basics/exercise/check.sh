#!/usr/bin/env bash
# Verifies the terminal-basics exercise (LESSON.md, "Exercise" section).
# Read-only: running it never creates, changes, or deletes your files.
set -u
cd "$(dirname "$0")"

fails=0
ok() { printf 'ok    %s\n' "$1"; }
bad() {
	printf 'FAIL  %s\n      fix: %s\n' "$1" "$2"
	fails=$((fails + 1))
}

if [ -d trip/photos ]; then
	ok "trip/photos/ exists"
else
	bad "directory trip/photos is missing" "create it (mkdir can make both levels at once — see the -p flag)"
fi
if [ -d trip/notes ]; then
	ok "trip/notes/ exists"
else
	bad "directory trip/notes is missing" "create it with mkdir"
fi

for photo in photo-001.jpg photo-002.jpg; do
	if [ -f "trip/photos/$photo" ]; then
		if [ -e "mess/$photo" ]; then
			bad "$photo is in trip/photos/ but ALSO still in mess/" "it should be moved, not copied — remove the leftover: rm mess/$photo"
		else
			ok "$photo moved into trip/photos/"
		fi
	else
		bad "$photo is not in trip/photos/" "move it there with mv (man mv shows the two-argument form)"
	fi
done

if [ -f trip/notes/draft.txt ]; then
	if [ -e mess/draft.txt ]; then
		bad "draft.txt is in trip/notes/ but ALSO still in mess/" "move, don't copy — remove the leftover: rm mess/draft.txt"
	else
		ok "draft.txt moved into trip/notes/"
	fi
else
	bad "draft.txt is not in trip/notes/" "move it there with mv"
fi

if [ -f trip/notes/draft-backup.txt ]; then
	if cmp -s trip/notes/draft.txt trip/notes/draft-backup.txt 2>/dev/null; then
		ok "draft-backup.txt is an identical copy of draft.txt"
	else
		bad "draft-backup.txt exists but its content differs from draft.txt" "make the copy again with cp so both files match"
	fi
else
	bad "trip/notes/draft-backup.txt is missing" "copy the draft: cp takes a source and a destination"
fi

if [ -z "$(find . -name old-draft.txt -print -quit)" ]; then
	ok "old-draft.txt is gone"
else
	bad "old-draft.txt still exists" "delete it with rm — and read the command back before pressing Enter, rm has no undo"
fi

if [ -e mess ]; then
	bad "the mess/ directory still exists" "once it's empty, remove it: rmdir mess (rmdir refuses while anything is left inside — that's your safety net)"
else
	ok "mess/ removed"
fi

if [ -f trip/location.txt ]; then
	first_line=$(head -n 1 trip/location.txt)
	case "$first_line" in
	/*/trip)
		ok "trip/location.txt holds the absolute path of trip"
		;;
	*)
		bad "trip/location.txt exists but doesn't hold trip's absolute path" "run pwd while INSIDE trip and redirect it: cd trip, then pwd > location.txt"
		;;
	esac
else
	bad "trip/location.txt is missing" "cd into trip, then send pwd's output into the file with >"
fi

if [ -f trip/notes/hidden.txt ]; then
	if grep -qxF '.' trip/notes/hidden.txt && grep -qxF '..' trip/notes/hidden.txt; then
		ok "trip/notes/hidden.txt shows the hidden . and .. entries"
	elif grep -qF 'location.txt' trip/notes/hidden.txt; then
		bad "hidden.txt lists trip's files but not the . and .. entries" "close — your flag hides . and .. themselves; man ls documents one that shows absolutely everything"
	else
		bad "hidden.txt doesn't look like a listing of trip" "run ls with the hidden-entries flag while INSIDE trip, redirected: ls <flag> > notes/hidden.txt"
	fi
else
	bad "trip/notes/hidden.txt is missing" "use man ls to find the flag that shows hidden entries, run it inside trip, and redirect the output into notes/hidden.txt"
fi

echo
if [ "$fails" -eq 0 ]; then
	echo "All checks passed — the mess is a trip. Tell your tutor you're ready for the quiz."
else
	echo "$fails check(s) failing — fix the first FAIL above, then run: bash ./check.sh"
	exit 1
fi
