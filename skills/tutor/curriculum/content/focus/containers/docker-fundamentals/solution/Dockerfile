# One stage, so the shipped image still carries the whole Go toolchain.
# That is a real cost; removing it is the next lesson.
FROM golang:1.22-bookworm

WORKDIR /src

# Dependencies change rarely, source changes constantly: copying the manifest
# on its own keeps the download layer cached across ordinary edits.
COPY go.mod ./
RUN go mod download

COPY . .
RUN go build -o /bin/clockwork .

# Documentation only — publishing the port still requires docker run -p.
EXPOSE 8080

CMD ["/bin/clockwork"]
