# The image, roughly as you left it in dockerizing-go — minus the two things a
# pipeline needs from it.
#
# TODO 1: accept the commit as a build argument (ARG) and stamp it into the
#         binary with -ldflags "-X main.buildRevision=...", so GET /version in
#         a running pod names the commit it was built from.
# TODO 2: label the image with the same commit
#         (org.opencontainers.image.revision) and with its source repository,
#         so `docker inspect` answers the question without running anything.
# TODO 3: the final stage still runs as root. Fix that.

FROM golang:1.22-bookworm AS build
WORKDIR /src

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/timesvc .

FROM gcr.io/distroless/static:nonroot
COPY --from=build /out/timesvc /usr/local/bin/timesvc
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/timesvc"]
