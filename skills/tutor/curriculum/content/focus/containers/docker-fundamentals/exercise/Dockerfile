# The previous owner wrote this in ten minutes and moved on. It builds an
# image that starts. It is also wrong in five ways that LESSON.md names.
# Rewrite it; check.sh grades the result.
FROM golang:latest
COPY . .
RUN go build -o /clockwork .
CMD /clockwork
