#!/usr/bin/env bash
# Referee for the Dockerizing Go exercise.
#
# Every graded check is STATIC: it reads the Dockerfile you wrote. No daemon,
# no network, no root. If a Docker daemon happens to be reachable, three extra
# live checks run as a bonus — when it is not, or when the build cannot reach a
# registry, they are reported SKIP, never FAIL, and your grade is unaffected.
#
# Run from this folder:  bash check.sh
set -u
cd "$(dirname "$0")" || exit 1

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
	printf 'SKIP  %s\n      why: %s\n' "$1" "$2"
}

note() {
	printf '      note: %s\n' "$1"
}

finish() {
	echo
	echo "$pass passed, $fail failed, $skip skipped for missing tooling or network."
	if [ "$fail" -eq 0 ]; then
		if [ "$skip" -gt 0 ]; then
			echo "Static grading is green. The skipped checks need a reachable Docker daemon and registry."
		else
			echo "Green, live build included. Your image is small, static and unprivileged."
		fi
		exit 0
	fi
	echo "Fix the FAIL lines above and re-run: bash check.sh"
	exit 1
}

# ------------------------------------------------------------------ parsing

if [ ! -f Dockerfile ]; then
	bad "Dockerfile exists" \
		"create a file named exactly 'Dockerfile' in this folder — read Dockerfile.naive first, then write the multi-stage version"
	finish
fi
ok "Dockerfile exists"

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
norm="$work/norm"

# Drop comments and CRs, join backslash continuations, drop blank lines — so
# every later check can assume one instruction per line.
sed -e 's/\r$//' Dockerfile | awk '
	/^[[:space:]]*#/ { next }
	{
		line = $0
		if (buf != "") { sub(/^[[:space:]]+/, "", line); line = buf " " line; buf = "" }
		if (line ~ /\\[[:space:]]*$/) { sub(/\\[[:space:]]*$/, "", line); buf = line; next }
		sub(/^[[:space:]]+/, "", line)
		if (line != "") print line
	}
	END { if (buf != "") print buf }
' >"$norm"

from_count=$(grep -cEi '^FROM[[:space:]]' "$norm" || true)
if [ "$from_count" -lt 2 ]; then
	bad "Dockerfile has two or more stages" \
		"a multi-stage build needs at least two FROM lines: 'FROM golang:1.22-bookworm AS build' to compile, then a small runtime FROM that receives only the binary (found $from_count)"
	finish
fi
ok "Dockerfile has two or more stages ($from_count FROM lines)"

last_from=$(grep -nEi '^FROM[[:space:]]' "$norm" | tail -n1 | cut -d: -f1)
sed -n "1,$((last_from - 1))p" "$norm" >"$work/build"
sed -n "${last_from},\$p" "$norm" >"$work/final"

image_of() { awk '{ for (i = 2; i <= NF; i++) { if ($i ~ /^--/) continue; print tolower($i); exit } }'; }
final_base=$(sed -n "${last_from}p" "$norm" | image_of)

# ------------------------------------------------------------- static checks

grep -Ei '^FROM[[:space:]]' "$norm" >"$work/froms"
unpinned=""
while IFS= read -r line; do
	base=$(printf '%s\n' "$line" | image_of)
	# Strip the registry before looking for a tag, or the port in
	# `localhost:5000/timesvc` reads as one and an untagged image passes.
	ref=${base##*/}
	case "$base" in
	scratch | *@sha256:*) continue ;;
	esac
	case "$ref" in
	*:latest | *:) unpinned="$base" ;;
	*:*) ;;
	*) unpinned="$base" ;;
	esac
done <"$work/froms"
if [ -z "$unpinned" ]; then
	ok "every base image is pinned to a real tag or digest"
else
	bad "every base image is pinned to a real tag or digest" \
		"'$unpinned' pins nothing — an untagged image means :latest, and latest is a moving pointer, not a version. Write golang:1.22-bookworm (or a digest) so today's build and next month's build are the same build"
fi

case "$final_base" in
*golang*)
	bad "final stage does not start from the Go toolchain" \
		"the last FROM is '$final_base' — that ships the compiler, the stdlib source and a full userland. Start the final stage from scratch, gcr.io/distroless/static:nonroot, or alpine:3.20"
	;;
*)
	ok "final stage starts from a runtime base ($final_base)"
	;;
esac

if grep -Eqi '(^|[[:space:];&|(])go[[:space:]]+(build|install|run|test|mod)([[:space:]]|$)' "$work/final"; then
	bad "final stage contains no toolchain commands" \
		"the final stage runs the go tool, which means the toolchain has to be in the shipped image. Compile in the build stage and move only the binary across with COPY --from"
else
	ok "final stage contains no toolchain commands"
fi

if grep -Eqi '^COPY[[:space:]].*--from=' "$work/final"; then
	ok "final stage copies the binary from an earlier stage"
else
	bad "final stage copies the binary from an earlier stage" \
		"add 'COPY --from=build /out/timesvc /usr/local/bin/timesvc' — without --from, the final stage never receives what you compiled"
fi

if grep -Eq 'CGO_ENABLED[[:space:]]*=[[:space:]]*0' "$work/build"; then
	ok "build stage sets CGO_ENABLED=0"
else
	bad "build stage sets CGO_ENABLED=0" \
		"set it on the build command (RUN CGO_ENABLED=0 GOOS=linux go build …) or with ENV. Without it a cgo-linked binary needs a dynamic loader and libc that your runtime base does not have"
fi

mod_copy=$(grep -nEi '^COPY[[:space:]].*go\.mod' "$work/build" | head -n1 | cut -d: -f1)
mod_dl=$(grep -nEi 'go[[:space:]]+mod[[:space:]]+download' "$work/build" | head -n1 | cut -d: -f1)
src_copy=$(grep -nEi '^COPY[[:space:]]' "$work/build" | grep -Eiv 'go\.(mod|sum)' | grep -Eiv -- '--from=' | head -n1 | cut -d: -f1)
if [ -z "$mod_copy" ] || [ -z "$mod_dl" ]; then
	bad "dependency layer is separate from the source layer" \
		"copy the manifests first and resolve them on their own line: 'COPY go.mod go.sum ./' then 'RUN go mod download', before you copy any source"
elif [ -z "$src_copy" ]; then
	bad "dependency layer is separate from the source layer" \
		"the build stage never copies your source — add 'COPY . .' after the go mod download step"
elif [ "$mod_copy" -lt "$mod_dl" ] && [ "$mod_dl" -lt "$src_copy" ]; then
	ok "dependency layer is cached separately from the source layer"
else
	bad "dependency layer is cached separately from the source layer" \
		"order is wrong: COPY go.mod (line $mod_copy), go mod download (line $mod_dl), COPY source (line $src_copy). A layer's cache dies when its inputs change, and every later layer dies with it — so manifests must come first"
fi

user_line=$(grep -Ei '^USER[[:space:]]' "$work/final" | tail -n1)
user_val=$(printf '%s' "$user_line" | awk '{print $2}')
if [ -z "$user_line" ]; then
	bad "final stage runs as a non-root user" \
		"add a USER instruction to the final stage, e.g. 'USER 65532:65532'. With no USER the container process is uid 0"
elif [ "$user_val" = "root" ] || [ "${user_val%%:*}" = "0" ]; then
	bad "final stage runs as a non-root user" \
		"USER $user_val is root. Pick an unprivileged uid, e.g. 'USER 65532:65532'"
elif [ "$final_base" = "scratch" ] && ! printf '%s' "$user_val" | grep -Eq '^[0-9]+(:[0-9]+)?$'; then
	bad "final stage runs as a non-root user" \
		"on scratch there is no /etc/passwd, so a name like '$user_val' cannot be resolved and the container fails to start. Use a numeric uid: 'USER 65532:65532'"
else
	ok "final stage runs as a non-root user (USER $user_val)"
fi

if grep -Eq '^EXPOSE[[:space:]]+8080([/[:space:]]|$)' "$work/final"; then
	ok "final stage declares EXPOSE 8080"
else
	bad "final stage declares EXPOSE 8080" \
		"timesvc listens on 8080 — add 'EXPOSE 8080' to the final stage. It publishes nothing by itself; it is the documented contract that -p, Compose and your reviewers read"
fi

ldflags_line=$(grep -Ei 'go[[:space:]]+build' "$work/build" | grep -Ei -- '-ldflags' | head -n1)
if [ -z "$ldflags_line" ]; then
	bad "build strips debug information" \
		"add -ldflags=\"-s -w\" to your go build command: -w drops DWARF debug info, -s drops the symbol table, together a few MB of an image nobody debugs with delve"
elif printf '%s' "$ldflags_line" | grep -Eq -- '(^|[^[:alnum:]_-])-s([^[:alnum:]_-]|$)' &&
	printf '%s' "$ldflags_line" | grep -Eq -- '(^|[^[:alnum:]_-])-w([^[:alnum:]_-]|$)'; then
	ok "build strips debug information (-ldflags with -s and -w)"
else
	bad "build strips debug information" \
		"your -ldflags value is missing -s or -w; write -ldflags=\"-s -w\""
fi
if grep -Eqi 'go[[:space:]]+build' "$work/build" && ! grep -Eqi -- '-trimpath' "$work/build"; then
	note "consider adding -trimpath to go build: it keeps your local filesystem paths out of the binary and makes builds reproducible (not graded)"
fi

case "$final_base" in
scratch)
	if grep -Eqi '^COPY[[:space:]].*--from=.*(ca-certificates\.crt|/etc/ssl/)' "$work/final"; then
		ok "TLS trust is handled for a scratch base (CA bundle copied in)"
	else
		bad "TLS trust is handled for a scratch base" \
			"scratch is empty, so there is no CA bundle and every HTTPS call fails with 'x509: certificate signed by unknown authority'. Copy it in: COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/"
	fi
	;;
*distroless* | *alpine*)
	ok "TLS trust is handled ($final_base ships a CA bundle)"
	;;
*debian* | *ubuntu*)
	if grep -Eqi 'ca-certificates' "$work/final"; then
		ok "TLS trust is handled (ca-certificates installed)"
	else
		bad "TLS trust is handled for a $final_base base" \
			"the slim Debian/Ubuntu images ship no CA bundle — either apt-get install -y ca-certificates in the final stage, or copy the bundle from the builder"
	fi
	;;
*)
	ok "TLS trust: unrecognised base '$final_base' — verify its CA bundle yourself"
	;;
esac

if grep -Eq '^(ENTRYPOINT|CMD)[[:space:]]*\[' "$work/final"; then
	ok "final stage starts the binary in exec form"
else
	bad "final stage starts the binary in exec form" \
		"use the JSON array form, e.g. ENTRYPOINT [\"/usr/local/bin/timesvc\"]. Shell form wraps your process in /bin/sh — which scratch and distroless do not have, and which swallows the signals your service must receive"
fi

if grep -q 'GET /healthz' main.go 2>/dev/null && grep -q 'GET /now' main.go 2>/dev/null &&
	grep -q 'time/tzdata' main.go 2>/dev/null; then
	ok "the provided service is unchanged"
else
	bad "the provided service is unchanged" \
		"main.go is given to you finished — restore its /healthz and /now routes and the time/tzdata import; this lesson grades the image, not the Go"
fi

if [ ! -f NOTES.md ]; then
	bad "NOTES.md answers all three questions" \
		"NOTES.md is missing — restore it from the exercise and answer the questions on the A: lines"
elif grep -q 'TODO' NOTES.md; then
	bad "NOTES.md answers all three questions" \
		"NOTES.md still contains a TODO placeholder — replace every one with your own answer"
else
	answered=$(grep -c '^A:.\{40,\}' NOTES.md || true)
	if [ "$answered" -ge 3 ]; then
		ok "NOTES.md answers all three questions"
	else
		bad "NOTES.md answers all three questions" \
			"only $answered of 3 A: lines carry a real answer (40+ characters). Write the sizes, the base-image trade-off and the CGO failure in your own words"
	fi
fi

# --------------------------------------------------------------- live checks

img=tutor-dockerizing-go:check
cname=tutor-dockerizing-go-check

# The budget follows the base you chose. Criterion 8 permits a slim
# Debian/Ubuntu runtime, and that userland alone weighs more than everything a
# scratch or distroless image is allowed to be — so a single ceiling would fail
# a legal choice. What is graded is that nothing came across that you did not
# ask for, not that everyone landed on the same base.
case "$final_base" in
*debian* | *ubuntu*) size_ceiling_mb=120 ;;
*alpine*) size_ceiling_mb=40 ;;
*) size_ceiling_mb=25 ;;
esac

if ! command -v docker >/dev/null 2>&1 || ! docker info >/dev/null 2>&1; then
	reason="no reachable Docker daemon — install Docker Desktop or the engine to run these"
	skipped "live: image builds" "$reason"
	skipped "live: image is under ${size_ceiling_mb} MB" "$reason"
	skipped "live: container serves GET /healthz" "$reason"
	finish
fi

docker rm -f "$cname" >/dev/null 2>&1 || true
if docker build -t "$img" . >"$work/build.log" 2>&1; then
	ok "live: image builds"
	built=yes
elif grep -qiE 'network|dial tcp|no such host|timeout|TLS handshake|proxyconnect|429 Too Many' "$work/build.log"; then
	skipped "live: image builds" \
		"the build could not reach a registry to pull its base images — that is the network's problem, not your Dockerfile's"
	tail -n 5 "$work/build.log" | sed 's/^/        /'
	built=no
else
	bad "live: image builds" \
		"docker build failed — the tail of its output is below. Read the first error, not the last"
	tail -n 15 "$work/build.log" | sed 's/^/        /'
	built=no
fi

if [ "$built" = yes ]; then
	bytes=$(docker image inspect -f '{{.Size}}' "$img" 2>/dev/null || echo 0)
	mb=$((bytes / 1000000))
	if [ "$bytes" -gt 0 ] && [ "$mb" -le "$size_ceiling_mb" ]; then
		ok "live: image is ${mb} MB (ceiling ${size_ceiling_mb} MB)"
	else
		bad "live: image is under ${size_ceiling_mb} MB" \
			"your image is ${mb} MB, and the budget for a '$final_base' base is ${size_ceiling_mb} MB. Something heavy came across on top of the base — check that the final stage copies only the binary, and compare with: docker image ls $img"
	fi

	if ! command -v python3 >/dev/null 2>&1; then
		skipped "live: container serves GET /healthz" "python3 not found, so there is no way to make the request"
	else
		docker run -d --rm --name "$cname" -p 127.0.0.1:0:8080 "$img" >/dev/null 2>&1
		hostport=$(docker port "$cname" 8080/tcp 2>/dev/null | head -n1 | sed 's/.*://')
		served=no
		if [ -n "$hostport" ]; then
			for _ in 1 2 3 4 5 6 7 8 9 10; do
				if python3 -c "import sys,urllib.request;sys.exit(0 if urllib.request.urlopen('http://127.0.0.1:$hostport/healthz',timeout=2).status==200 else 1)" >/dev/null 2>&1; then
					served=yes
					break
				fi
				sleep 1
			done
		fi
		if [ "$served" = yes ]; then
			ok "live: container serves GET /healthz"
		else
			bad "live: container serves GET /healthz" \
				"the image built but did not answer. Run 'docker logs $cname' after starting it yourself: 'no such file or directory' on a binary that exists means a dynamic loader is missing (CGO), and 'unable to find user' means USER is a name on a base with no /etc/passwd"
			docker logs "$cname" 2>&1 | tail -n 10 | sed 's/^/        /'
		fi
		docker rm -f "$cname" >/dev/null 2>&1 || true
	fi
	# Leave the learner's machine as we found it.
	docker image rm -f "$img" >/dev/null 2>&1 || true
else
	skipped "live: image is under ${size_ceiling_mb} MB" "the build did not complete, so there is no image to measure"
	skipped "live: container serves GET /healthz" "the build did not complete, so there is no image to run"
fi

finish
