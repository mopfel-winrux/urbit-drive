# Drive client builds. Nothing to install first but git, curl and a Go
# toolchain (any recent one: go.mod's go line fetches the right version).
#
#   make                      bin/drive for this machine
#   make test
#   make dist-linux-amd64     a static binary for that target, packaged into dist/
#   make dist-linux           both Linux targets
#   make dist-darwin-arm64    (on macOS) Drive.app, dmg and zip
#
# libmesa is the git submodule third_party/libmesa (fetched on first use)
# and is built with zig 0.15.2: scripts/zig.sh uses $ZIG or zig on PATH
# when either is that version, and otherwise downloads it into build/.
# Cross builds compile libmesa for the target into build/libmesa/<target>
# (scripts/build.sh).

LIBMESA := $(abspath third_party/libmesa)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
export LIBMESA VERSION

TARGETS := linux-amd64 linux-arm64 windows-amd64 darwin-arm64 darwin-amd64

.PHONY: build test vet clean libmesa dist-linux $(addprefix dist-,$(TARGETS)) $(addprefix bin-,$(TARGETS))

build: libmesa
	go build -ldflags "-X main.version=$(VERSION)" -o bin/drive ./cmd/drive

test: libmesa
	go test ./...

vet: libmesa
	go vet ./...

libmesa:
	scripts/libmesa.sh

# bin-<os>-<arch>: just the binary, in build/out/<os>-<arch>/
$(addprefix bin-,$(TARGETS)): bin-%:
	ZIG=$$(scripts/zig.sh) scripts/build.sh $(word 1,$(subst -, ,$*)) $(word 2,$(subst -, ,$*))

# dist-<os>-<arch>: the binary, packaged into dist/
$(addprefix dist-,$(TARGETS)): dist-%: bin-%
	scripts/package.sh $(word 1,$(subst -, ,$*)) $(word 2,$(subst -, ,$*))

dist-linux: dist-linux-amd64 dist-linux-arm64

clean:
	rm -rf bin build dist
