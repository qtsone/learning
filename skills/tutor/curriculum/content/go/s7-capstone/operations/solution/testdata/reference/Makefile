VERSION ?= dev
IMAGE ?= registry.example.internal/tinysvc:$(VERSION)

.PHONY: build test vet vuln image deploy rollback

build:
	go build ./...

test:
	go test -race ./...

vet:
	go vet ./...

vuln:
	govulncheck ./...

image:
	docker build -t $(IMAGE) .

deploy:
	./deploy/deploy.sh $(VERSION)

rollback:
	./deploy/rollback.sh
