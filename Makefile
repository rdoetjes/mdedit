# mdedit — Markdown editor
#
# Works with GNU make and BSD make (FreeBSD's default make), so platform
# detection happens inside the recipes rather than with make conditionals.
#
#   make            build ./bin/mdedit for this machine
#   make test       run the tests
#   make install    install to $(PREFIX)/bin (default /usr/local)
#   make deps       show the system packages needed to build
#
# Gio, the UI toolkit, uses cgo on Linux, the BSDs and macOS. All Go code is
# linked into one self-contained binary; only the system's windowing and GPU
# libraries (X11/Wayland, EGL/GL, xkbcommon) are loaded at run time, as for
# every GPU-accelerated desktop application.

GO      ?= go
VERSION ?= 0.1.0
PREFIX  ?= /usr/local
BIN     ?= bin/mdedit
# Extra build tags, e.g. TAGS=nowayland or TAGS=nox11 to drop a backend.
TAGS    ?=

LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: all build run test install uninstall clean deps

all: build

build:
	@mkdir -p bin
	@cc="$${CC:-cc}"; \
	if [ "$$(uname -s)" = Darwin ]; then cc=clang; fi; \
	echo "CGO_ENABLED=1 CC=$$cc $(GO) build -trimpath -tags '$(TAGS)' -o $(BIN) ./cmd/mdedit"; \
	CGO_ENABLED=1 CC="$$cc" $(GO) build -trimpath -tags '$(TAGS)' -ldflags '$(LDFLAGS)' -o $(BIN) ./cmd/mdedit

run: build
	./$(BIN) $(FILE)

test:
	@cc="$${CC:-cc}"; \
	if [ "$$(uname -s)" = Darwin ]; then cc=clang; fi; \
	CGO_ENABLED=1 CC="$$cc" $(GO) test ./...

install: build
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BIN) $(DESTDIR)$(PREFIX)/bin/mdedit

uninstall:
	rm -f $(DESTDIR)$(PREFIX)/bin/mdedit

clean:
	rm -rf bin

deps:
	@echo "Build requirements: Go >= 1.25, a C compiler and pkg-config, plus:"
	@echo
	@echo "  Debian/Ubuntu: sudo apt install gcc pkg-config libwayland-dev libx11-dev libx11-xcb-dev \\"
	@echo "                   libxkbcommon-x11-dev libgles2-mesa-dev libegl1-mesa-dev libffi-dev \\"
	@echo "                   libxcursor-dev libxfixes-dev libvulkan-dev"
	@echo "  Fedora:        sudo dnf install gcc pkg-config wayland-devel libX11-devel libxkbcommon-x11-devel \\"
	@echo "                   mesa-libGLES-devel mesa-libEGL-devel libXcursor-devel libXfixes-devel vulkan-headers"
	@echo "  Arch:          sudo pacman -S gcc pkgconf wayland libx11 libxkbcommon-x11 libxcursor libxfixes mesa vulkan-headers"
	@echo "  FreeBSD:       pkg install go pkgconf libX11 libxkbcommon libXcursor libXfixes wayland mesa-libs"
	@echo "  macOS:         xcode-select --install"
