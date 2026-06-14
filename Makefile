BINARY  := box
PKG     := github.com/shariqnaiyer/agentbox
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(PKG)/cmd/box.Version=$(VERSION)

GOOSARCH := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64

.PHONY: build install test vet cross clean fmt

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

install:
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" .

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

cross:
	@mkdir -p dist
	@for t in $(GOOSARCH); do \
		os=$${t%/*}; arch=$${t#*/}; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch \
			go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)_$${os}_$${arch} . ; \
	done

clean:
	rm -rf bin dist
