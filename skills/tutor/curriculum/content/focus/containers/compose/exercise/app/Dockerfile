# syntax=docker/dockerfile:1
#
# Provided for you — the shape you built in the previous lesson, plus one extra
# stage. Three stages, three jobs:
#
#   build  compiles the static binary
#   dev    keeps the toolchain and compiles whatever is mounted at /src
#   final  the image you would actually ship (built by default: it is last)
#
# Compose picks a stage with `target:`. Your app service targets dev, so your
# edits on the host are compiled inside the container without rebuilding.

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/service .

FROM golang:1.22-alpine AS dev
WORKDIR /src
ENV CGO_ENABLED=0
COPY . .
EXPOSE 8080
CMD ["go", "run", "."]

FROM gcr.io/distroless/static:nonroot AS final
COPY --from=build /out/service /service
EXPOSE 8080
# numeric, so a runtime enforcing runAsNonRoot can verify it
USER 65532:65532
ENTRYPOINT ["/service"]
