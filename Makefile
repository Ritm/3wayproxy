.PHONY: all build test relay relay-docker clean dist-amd64 dist-arm64

all: build

build: build-client build-agg

export GOFLAGS ?= -buildvcs=false

build-client:
	cd client && CGO_ENABLED=0 go build -o ../bin/3wayproxy-client ./cmd/3wayproxy-client

build-agg:
	cd aggregator && CGO_ENABLED=0 go build -o ../bin/3wayproxy-agg ./cmd/3wayproxy-agg

dist-amd64:
	./scripts/build-release.sh amd64

dist-arm64:
	./scripts/build-release.sh arm64

test:
	cd shared && go test ./...

relay:
	cd relay && uvicorn app.main:app --host 127.0.0.1 --port 8000 --reload

relay-docker:
	docker compose -f deploy/docker-compose.dev.yml up --build

clean:
	rm -rf bin/
	find . -name '*.test' -delete
