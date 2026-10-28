# mdedit — Markdown editor
#
# Works with GNU make and BSD make (FreeBSD's default make), so platform
# detection happens inside the recipes rather than with make conditionals.
#
#   make            build ./bin/mdedit for this machine
#   make test       run the tests
#   make install    install the binary, menu entry and icons under $(PREFIX)
#                   (default /usr/local; use sudo)
#   make desktop    write bin/mdedit.desktop (menu entry, Exec=$(PREFIX)/bin/mdedit)
#   make install-desktop   install the menu entry and icons under $(PREFIX)/share
#   make icons      redraw the PNG icons in assets/icons
#   make macos      build bin/MDEdit.app (macOS only); drag it into Applications
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

.PHONY: all build run test install uninstall clean deps desktop install-desktop uninstall-desktop icons macos

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

install: build install-desktop
	install -d $(DESTDIR)$(PREFIX)/bin
	install -m 755 $(BIN) $(DESTDIR)$(PREFIX)/bin/mdedit

uninstall: uninstall-desktop
	rm -f $(DESTDIR)$(PREFIX)/bin/mdedit

clean:
	rm -rf bin

# Desktop integration (freedesktop.org: Linux and the BSDs).
DESKTOP   = bin/mdedit.desktop
ICONSIZES = 16 24 32 48 64 128 256 512

desktop:
	@mkdir -p bin
	@printf '%s\n' \
		'[Desktop Entry]' \
		'Type=Application' \
		'Name=MDEdit' \
		'GenericName=Markdown Editor' \
		'Comment=Write Markdown with live preview, maths and PDF export' \
		'Exec=$(PREFIX)/bin/mdedit %f' \
		'TryExec=$(PREFIX)/bin/mdedit' \
		'Icon=mdedit' \
		'Terminal=false' \
		'Categories=Development;Office;Science;' \
		'MimeType=text/markdown;text/x-markdown;' \
		'Keywords=markdown;editor;latex;math;pdf;' \
		'StartupWMClass=mdedit' > $(DESKTOP)
	@echo "wrote $(DESKTOP)"

install-desktop: desktop
	install -d $(DESTDIR)$(PREFIX)/share/applications
	install -m 644 $(DESKTOP) $(DESTDIR)$(PREFIX)/share/applications/mdedit.desktop
	@for s in $(ICONSIZES); do \
		d=$(DESTDIR)$(PREFIX)/share/icons/hicolor/$${s}x$${s}/apps; \
		install -d $$d && install -m 644 assets/icons/$${s}x$${s}/apps/mdedit.png $$d/mdedit.png; \
	done
	@if command -v update-desktop-database >/dev/null 2>&1; then \
		update-desktop-database $(DESTDIR)$(PREFIX)/share/applications || true; fi
	@if command -v gtk-update-icon-cache >/dev/null 2>&1; then \
		gtk-update-icon-cache -q -t $(DESTDIR)$(PREFIX)/share/icons/hicolor || true; fi

uninstall-desktop:
	rm -f $(DESTDIR)$(PREFIX)/share/applications/mdedit.desktop
	@for s in $(ICONSIZES); do rm -f $(DESTDIR)$(PREFIX)/share/icons/hicolor/$${s}x$${s}/apps/mdedit.png; done

icons:
	$(GO) run ./tools/mkicon assets/icons

# macOS application bundle.
APP       = bin/MDEdit.app
BUNDLE_ID ?= org.mdedit.MDEdit

macos: build
	@if [ "$$(uname -s)" != Darwin ]; then echo "make macos needs macOS (iconutil, codesign)"; exit 1; fi
	rm -rf $(APP) bin/mdedit.iconset
	mkdir -p $(APP)/Contents/MacOS $(APP)/Contents/Resources
	cp $(BIN) $(APP)/Contents/MacOS/mdedit
	sed -e 's/@VERSION@/$(VERSION)/g' -e 's/@BUNDLE_ID@/$(BUNDLE_ID)/g' assets/macos/Info.plist > $(APP)/Contents/Info.plist
	printf 'APPL????' > $(APP)/Contents/PkgInfo
	$(GO) run ./tools/mkicon -macos bin/mdedit.iconset
	iconutil -c icns -o $(APP)/Contents/Resources/mdedit.icns bin/mdedit.iconset
	rm -rf bin/mdedit.iconset
	codesign --force --sign - $(APP)
	@echo "Built $(APP): drag it into /Applications"

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
