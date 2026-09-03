# syntax=docker/dockerfile:1
#
# The build for release 1.1.0 — the image the manifests in this folder deploy.
# It is the Dockerfile you wrote two lessons ago, kept here so `timesvc:1.1.0`
# exists without leaving the lesson. Nothing in check.sh grades it; the two
# lines that matter for *this* lesson are the static build (no shell in the
# final image, so the preStop hook has to be the native `sleep:` handler) and
# the exec-form ENTRYPOINT (so PID 1 is your process and SIGTERM reaches it).

FROM golang:1.22-bookworm AS build

WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
	-trimpath \
	-ldflags="-s -w" \
	-o /out/timesvc .

FROM gcr.io/distroless/static:nonroot

COPY --from=build /out/timesvc /usr/local/bin/timesvc

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/timesvc"]
