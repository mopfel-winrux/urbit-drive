# Drive client builds.
#
#   make                      the dev build (bin/drive), against libmesa's own zig-out
#   make dist-linux-amd64     a static binary for that target, packaged into dist/
#   make dist-linux           both Linux targets
#   make dist-darwin-arm64    (on macOS) Drive.app, dmg and zip
#   make test
#
# Cross builds compile libmesa for the target into build/libmesa/<target>
# first (scripts/build.sh). Set ZIG to zig 0.15.2 and LIBMESA to the
# libmesa checkout if they aren't on PATH / next to this repo.

ZIG     ?= zig
LIBMESA ?= $(abspath ../../libmesa/libmesa)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
export ZIG LIBMESA VERSION

TARGETS := linux-amd64 linux-arm64 windows-amd64 darwin-arm64 darwin-amd64

.PHONY: build test vet clean dist-linux $(addprefix dist-,$(TARGETS)) $(addprefix bin-,$(TARGETS))

build:
	go build -ldflags "-X main.version=$(VERSION)" -o bin/drive ./cmd/drive

test:
	go test ./...

vet:
	go vet ./...

# bin-<os>-<arch>: just the binary, in build/out/<os>-<arch>/
$(addprefix bin-,$(TARGETS)): bin-%:
	scripts/build.sh $(word 1,$(subst -, ,$*)) $(word 2,$(subst -, ,$*))

# dist-<os>-<arch>: the binary, packaged into dist/
$(addprefix dist-,$(TARGETS)): dist-%: bin-%
	scripts/package.sh $(word 1,$(subst -, ,$*)) $(word 2,$(subst -, ,$*))

dist-linux: dist-linux-amd64 dist-linux-arm64

clean:
	rm -rf bin build dist
