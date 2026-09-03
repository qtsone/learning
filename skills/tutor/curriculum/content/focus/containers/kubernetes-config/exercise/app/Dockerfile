# syntax=docker/dockerfile:1
# ^ a parser directive, so it has to be the very first line, before this
# comment. The rest is the multi-stage distroless build from Dockerizing Go,
# unchanged. It is here because this lesson ships a different service under a
# new tag, not because there is anything to do to it — you are not graded on
# this file.

FROM golang:1.22-bookworm AS build

WORKDIR /src

# Manifests first: this layer's cache survives every source edit, so the
# module download is paid once per dependency change, not once per keystroke.
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 forces the pure-Go net and os/user implementations, so the
# result is a static binary with no dynamic loader and no libc dependency —
# the reason a Go service can run on an otherwise empty image.
RUN CGO_ENABLED=0 GOOS=linux go build \
	-trimpath \
	-ldflags="-s -w" \
	-o /out/timesvc .

FROM gcr.io/distroless/static:nonroot

COPY --from=build /out/timesvc /usr/local/bin/timesvc

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/timesvc"]
