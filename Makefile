.PHONY: all build build-cli build-tui build-desktop test vet clean

all: build

build: build-cli build-tui build-desktop

build-cli:
	go build -o bin/kumokura ./cmd/kumokura

build-tui:
	go build -o bin/kumokura-tui ./cmd/kumokura-tui

build-desktop:
	go build -tags wayland -o bin/kumokura-desktop ./cmd/kumokura-desktop
test:
	go test -tags wayland ./...

vet:
	go vet -tags wayland ./...

clean:
	rm -rf bin/kumokura bin/kumokura-tui bin/kumokura-desktop
