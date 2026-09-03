# Reference solution. The image from dockerizing-go, plus the two things that
# make it traceable: the commit goes *into* the binary, and onto the image.
#
# Note the ARG on both sides of the FROM. A build argument is scoped to the
# stage that declares it, so the final stage has to declare it again — the
# classic "why is my label empty?" of Docker builds.

FROM golang:1.22-bookworm AS build
ARG REVISION=dev
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
# -X writes a value into a package-level string variable at link time, so the
# binary can report which commit it is without reading anything at runtime.
RUN CGO_ENABLED=0 GOOS=linux go build \
	-trimpath \
	-ldflags="-s -w -X main.buildRevision=${REVISION}" \
	-o /out/timesvc .

FROM gcr.io/distroless/static:nonroot
ARG REVISION=dev
# The OCI labels a registry, a scanner and a human all read. revision is the
# one that answers "what is in this image?" without running it.
LABEL org.opencontainers.image.title="timesvc" \
	org.opencontainers.image.revision="${REVISION}" \
	org.opencontainers.image.source="https://github.com/ada/timesvc"
COPY --from=build /out/timesvc /usr/local/bin/timesvc
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/timesvc"]
