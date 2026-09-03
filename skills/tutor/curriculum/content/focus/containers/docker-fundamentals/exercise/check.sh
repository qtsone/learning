#!/usr/bin/env bash
# Referee for the Docker Fundamentals exercise. It reads the files you wrote —
# Dockerfile, .dockerignore, NOTES.md — and grades them. That grading needs no
# Docker daemon, no network and no root. If Docker *is* installed and running,
# a few bonus checks actually build the image; when it isn't, those are
# reported SKIPPED, never failed.
# Run from this folder:  bash check.sh
set -u
cd "$(dirname "$0")" || exit 1

pass=0
fail=0
skip=0

ok() {
	pass=$((pass + 1))
	printf 'ok      %s\n' "$1"
}

bad() {
	fail=$((fail + 1))
	printf 'FAIL    %s\n        fix: %s\n' "$1" "$2"
}

skipped() {
	skip=$((skip + 1))
	printf 'SKIP    %s\n        why: %s\n' "$1" "$2"
}

finish() {
	echo
	if [ "$fail" -eq 0 ]; then
		echo "$pass checks passed, $skip skipped for missing tooling."
		if [ "$skip" -gt 0 ]; then
			echo "The skipped checks are a bonus: install Docker and re-run to have your image really built."
		fi
		exit 0
	fi
	echo "$pass passed, $fail failed, $skip skipped for missing tooling."
	echo "Fix the FAIL lines above and re-run: bash check.sh"
	exit 1
}

# Normalized Dockerfile: comments dropped, line continuations joined, so that
# "instruction N comes before instruction M" is a question about real order.
df_lines() {
	awk '
		{ sub(/\r$/, "") }
		/^[[:space:]]*#/ { next }
		{ line = line $0 }
		/\\[[:space:]]*$/ { sub(/\\[[:space:]]*$/, "", line); next }
		{
			gsub(/^[[:space:]]+|[[:space:]]+$/, "", line)
			if (line != "") print line
			line = ""
		}
	' Dockerfile
}

# True when .dockerignore has a pattern covering $1. Handles the shapes people
# actually write: a bare name, a trailing slash, a leading ./ or **/, a glob.
di_matches() {
	local target=$1 line pat
	while IFS= read -r line || [ -n "$line" ]; do
		line=${line%$'\r'}
		line="${line#"${line%%[![:space:]]*}"}"
		line="${line%"${line##*[![:space:]]}"}"
		case "$line" in '' | '#'* | '!'*) continue ;; esac
		pat=${line%/}
		pat=${pat#./}
		pat=${pat#/}
		[ -z "$pat" ] && continue
		# shellcheck disable=SC2053
		[[ $target == $pat ]] && return 0
		# shellcheck disable=SC2053
		[[ $target == ${pat#\*\*/} ]] && return 0
	done <.dockerignore
	return 1
}

# Prose the learner wrote under NOTES.md heading "## <n>. …", with the quoted
# prompt and blank lines removed.
section_answer() {
	awk -v want="$1" '
		/^##[[:space:]]/ {
			cur = 0
			n = $2
			sub(/\./, "", n)
			if (n == want) cur = 1
			next
		}
		cur { print }
	' NOTES.md | grep -v '^[[:space:]]*>' | grep -v '^[[:space:]]*$'
}

# ---------------------------------------------------------------- provided files

for f in main.go go.mod; do
	if [ -f "$f" ]; then
		ok "provided file $f is still here"
	else
		bad "provided file $f is still here" \
			"restore $f — the exercise is to containerize this program, not to change it"
	fi
done

if [ -f Dockerfile ]; then
	ok "Dockerfile exists"
else
	bad "Dockerfile exists" \
		"create a Dockerfile in this folder — LESSON.md lists what it must contain"
	finish
fi

norm=$(df_lines)

# ---------------------------------------------------------------- base image

base=$(printf '%s\n' "$norm" | grep -iE '^FROM[[:space:]]' | head -n 1 |
	awk '{ for (i = 2; i <= NF; i++) if ($i !~ /^--/) { print $i; exit } }')

if [ -n "$base" ]; then
	ok "Dockerfile starts from a base image ($base)"
else
	bad "Dockerfile starts from a base image" \
		"every Dockerfile needs a FROM line, e.g. FROM golang:1.22-bookworm"
	finish
fi

ref=${base##*/}
case "$ref" in
*@sha256:*)
	ok "base image is pinned by digest"
	;;
*:latest)
	bad "base image is pinned to a real version" \
		"latest is not a version, it is the default tag name — use a concrete tag such as golang:1.22-bookworm (Go 1.22 or newer, since go.mod says go 1.22)"
	;;
*:*)
	ok "base image is pinned to a concrete tag"
	;;
*)
	bad "base image is pinned to a real version" \
		"FROM $base has no tag, so Docker silently uses :latest — name a concrete tag such as golang:1.22-bookworm"
	;;
esac

# ---------------------------------------------------------------- workdir

workdir=$(printf '%s\n' "$norm" | grep -iE '^WORKDIR[[:space:]]' | head -n 1 | awk '{ print $2 }')
if [ -z "$workdir" ]; then
	bad "Dockerfile sets a WORKDIR" \
		"add e.g. WORKDIR /src before your COPY steps, so later instructions do not depend on whatever directory the base image happened to leave you in"
elif [ "${workdir#/}" = "$workdir" ]; then
	bad "WORKDIR is an absolute path" \
		"WORKDIR $workdir is relative to an unknown starting directory — use an absolute path like /src"
else
	ok "Dockerfile sets an absolute WORKDIR ($workdir)"
fi

# ---------------------------------------------------------------- layer ordering

copy_manifest=$(printf '%s\n' "$norm" | grep -niE '^COPY[[:space:]].*go\.mod' | head -n 1 | cut -d: -f1)
copy_source=$(printf '%s\n' "$norm" | grep -niE '^COPY[[:space:]]' |
	grep -viE 'go\.(mod|sum)' | head -n 1 | cut -d: -f1)
mod_download=$(printf '%s\n' "$norm" |
	grep -niE '^RUN[[:space:]].*go[[:space:]]+mod[[:space:]]+download' | head -n 1 | cut -d: -f1)

if [ -z "$copy_source" ]; then
	bad "Dockerfile copies the source into the image" \
		"add a COPY step that brings the .go files in, e.g. COPY . ."
elif [ -z "$copy_manifest" ]; then
	bad "Dockerfile copies go.mod before the source" \
		"add COPY go.mod ./ above the source COPY (this module has no third-party dependencies, so there is no go.sum yet)"
elif [ "$copy_manifest" -lt "$copy_source" ]; then
	ok "dependency manifest is copied before the source"
else
	bad "Dockerfile copies go.mod before the source" \
		"move the COPY go.mod ./ step above the source COPY — otherwise every source edit invalidates the dependency layer too"
fi

if [ -z "$mod_download" ]; then
	bad "Dockerfile downloads dependencies in their own layer" \
		"add RUN go mod download between the manifest COPY and the source COPY — that layer is what the cache is meant to keep"
elif [ -n "$copy_manifest" ] && [ -n "$copy_source" ] &&
	[ "$copy_manifest" -lt "$mod_download" ] && [ "$mod_download" -lt "$copy_source" ]; then
	ok "go mod download sits between the manifest COPY and the source COPY"
else
	bad "go mod download sits between the manifest COPY and the source COPY" \
		"order is COPY go.mod ./ then RUN go mod download then COPY . . — anywhere else and the layer is rebuilt on every edit"
fi

# ---------------------------------------------------------------- start command

if printf '%s\n' "$norm" | grep -qiE '^(CMD|ENTRYPOINT)[[:space:]]'; then
	ok "Dockerfile declares a start command (CMD or ENTRYPOINT)"
else
	bad "Dockerfile declares a start command (CMD or ENTRYPOINT)" \
		"an image with no default command cannot be started with a bare docker run — add CMD [\"/bin/clockwork\"] (or whatever path you built to)"
fi

if printf '%s\n' "$norm" | grep -qiE '^(CMD|ENTRYPOINT)[[:space:]]+[^[]'; then
	bad "start command uses exec form, not shell form" \
		"write it as a JSON array: CMD [\"/bin/clockwork\"]. Shell form wraps your binary in /bin/sh -c, so the shell becomes PID 1 and your program never sees the SIGTERM from docker stop"
else
	ok "start command uses exec form"
fi

# ---------------------------------------------------------------- .dockerignore

if [ ! -f .dockerignore ]; then
	bad ".dockerignore exists" \
		"create .dockerignore in this folder — without it the whole build context, junk included, is sent to the daemon and can land in a layer"
else
	effective=$(grep -vE '^[[:space:]]*(#|$)' .dockerignore | wc -l | tr -d ' ')
	if [ "$effective" -ge 1 ]; then
		ok ".dockerignore exists and has patterns"
	else
		bad ".dockerignore exists and has patterns" \
			"the file is empty or all comments — add the patterns listed in the acceptance criteria"
	fi

	if di_matches .git; then
		ok ".dockerignore excludes .git"
	else
		bad ".dockerignore excludes .git" \
			"add a .git line — repository history is often the largest thing in the context and none of it belongs in the image"
	fi

	if di_matches local.env; then
		ok ".dockerignore excludes local.env"
	else
		bad ".dockerignore excludes local.env" \
			"add local.env (or *.env) — it holds local settings and a token; anything in the context can be baked into a layer and shipped"
	fi

	if di_matches scratch; then
		ok ".dockerignore excludes scratch/"
	else
		bad ".dockerignore excludes scratch/" \
			"add scratch/ — personal notes are pure weight in the context and invalidate the cache when they change"
	fi
fi

# ---------------------------------------------------------------- NOTES.md

if [ ! -f NOTES.md ]; then
	bad "NOTES.md exists" \
		"restore NOTES.md and answer its four questions — the written half of this exercise is what your tutor grills you on"
else
	titles=(
		"1: container vs virtual machine"
		"2: the build cache and your edit"
		"3: tags, digests and latest"
		"4: where a container's writes go"
	)
	for i in 1 2 3 4; do
		title=${titles[$((i - 1))]}
		answer=$(section_answer "$i")
		words=$(printf '%s\n' "$answer" | wc -w | tr -d ' ')
		if printf '%s\n' "$answer" | grep -q '^[[:space:]]*TODO: your answer here\.'; then
			bad "NOTES.md answers question $title" \
				"the TODO placeholder is still there — replace it with your own answer"
		elif [ "$words" -lt 30 ]; then
			bad "NOTES.md answers question $title" \
				"only $words words of answer — explain it properly, in prose, as if to a colleague (about three sentences)"
		else
			ok "NOTES.md answers question $title"
		fi
	done
fi

# ---------------------------------------------------------------- optional live checks

if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1; then
	if docker image inspect "$base" >/dev/null 2>&1 || docker pull "$base" >/dev/null 2>&1; then
		ok "live: base image $base is available locally"
		if build_log=$(docker build -t tutor-clockwork:check . 2>&1); then
			ok "live: docker build succeeds (tagged tutor-clockwork:check)"
			config=$(docker image inspect \
				--format '{{json .Config.Cmd}} {{json .Config.Entrypoint}}' \
				tutor-clockwork:check 2>/dev/null)
			case "$config" in
			"null null" | "")
				bad "live: built image has a default command" \
					"docker image inspect shows neither Cmd nor Entrypoint — your CMD line did not take effect"
				;;
			*)
				ok "live: built image carries a default command"
				;;
			esac
			# Leave the learner's machine as we found it.
			docker image rm -f tutor-clockwork:check >/dev/null 2>&1 || true
		else
			bad "live: docker build succeeds" \
				"read the build output below, find the first error, fix the Dockerfile"
			printf '%s\n' "$build_log" | tail -n 15 | sed 's/^/        | /'
		fi
	else
		skipped "live: build the image" \
			"cannot obtain base image $base (no registry access) — static grading above is unaffected"
	fi
else
	skipped "live: build the image" \
		"no running Docker daemon here. Install Docker and start it to get this bonus check; your grade comes from the static checks above"
fi

finish
