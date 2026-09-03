# Stage 1: the toolchain. Pinned to an exact patch tag, so a rebuild next year
# produces the same binary rather than whatever "latest" means by then.
FROM golang:1.22.12-bookworm AS build

WORKDIR /src
# go.sum travels with go.mod so `go mod download` verifies against the hashes
# this project committed, and so changing one busts this layer's cache.
COPY go.mod go.sum ./
RUN go mod download
COPY . .

# A static binary has no libc to match at runtime, which is what lets the
# runtime stage be almost empty.
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags="-s -w" -o /out/tinysvc ./cmd/tinysvc

# Stage 2: the runtime. It carries the binary and nothing else — no compiler,
# no shell, no package manager, so there is nothing for an attacker to use.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

COPY --from=build /out/tinysvc /usr/local/bin/tinysvc

# Containers run as root unless told otherwise. 65532 is the nonroot user that
# ships in the base image.
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/tinysvc"]
