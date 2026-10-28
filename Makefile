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
#   make wasm       build the browser version into bin/web, plus its web
#                   server bin/mdedit-web
#   make serve      run bin/mdedit-web on http://localhost:8080
#   make install-jail   (sudo; FreeBSD) install the browser version and its
#                   service into the jail at $(JAIL_ROOT)
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

.PHONY: all build run test install uninstall clean deps desktop install-desktop uninstall-desktop icons macos \
	wasm serve install-jail

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
	GOOS=js GOARCH=wasm $(GO) vet ./...

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

# Browser version (WebAssembly). The editor is the same program; the
# platform_js.go files replace the file system with the browser's file
# picker, downloads and localStorage. bin/web is everything a web server
# needs; bin/mdedit-web is a small server for it (see cmd/mdedit-web).
WEB  = bin/web
PORT ?= 8080
# The server is built for this machine; to build it for the jail elsewhere:
# make wasm SERVER_GOOS=freebsd SERVER_GOARCH=amd64
SERVER_GOOS   ?=
SERVER_GOARCH ?=

wasm:
	@mkdir -p $(WEB)
	rm -f $(WEB)/*.gz $(WEB)/*.br
	GOOS=js GOARCH=wasm $(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(WEB)/mdedit.wasm ./cmd/mdedit
	@root=$$($(GO) env GOROOT); js=$$root/lib/wasm/wasm_exec.js; \
	if [ ! -f $$js ]; then js=$$root/misc/wasm/wasm_exec.js; fi; \
	echo "cp $$js $(WEB)/wasm_exec.js"; cp $$js $(WEB)/wasm_exec.js
	cp web/index.html $(WEB)/index.html
	cp assets/icons/64x64/apps/mdedit.png $(WEB)/icon.png
	gzip -9 -k -f $(WEB)/mdedit.wasm $(WEB)/wasm_exec.js $(WEB)/index.html
	@if command -v brotli >/dev/null 2>&1; then \
		echo "brotli -q 11 $(WEB)/mdedit.wasm"; brotli -f -k -q 11 $(WEB)/mdedit.wasm; \
	fi
	CGO_ENABLED=0 GOOS=$(SERVER_GOOS) GOARCH=$(SERVER_GOARCH) $(GO) build -trimpath -ldflags '-s -w' -o bin/mdedit-web ./cmd/mdedit-web
	@echo "Built $(WEB) and bin/mdedit-web; try: make serve"

serve:
	@if [ ! -f $(WEB)/mdedit.wasm ] || [ ! -x bin/mdedit-web ]; then echo "run make wasm first"; exit 1; fi
	./bin/mdedit-web -dir $(WEB) -addr localhost:$(PORT)

# Install the browser version into a FreeBSD jail, from the host: the web
# files into WEBROOT, the server and its rc.d service into the jail, the
# service enabled. Build first as yourself (make wasm), then: sudo make
# install-jail. If the jail is running, the service is (re)started.
JAIL_ROOT ?= /zroot/sylve/jails/100
WEBROOT   ?= /usr/local/www/content_renderer

install-jail:
	@if [ "$$(id -u)" != 0 ]; then echo "install-jail must run as root: sudo make install-jail"; exit 1; fi
	@if [ ! -f $(WEB)/mdedit.wasm ] || [ ! -x bin/mdedit-web ]; then \
		echo "nothing to install: run make wasm first (as yourself, not with sudo)"; exit 1; fi
	@if [ ! -d $(JAIL_ROOT)/usr/local ]; then echo "no jail at $(JAIL_ROOT) (set JAIL_ROOT=…)"; exit 1; fi
	install -d -m 755 $(JAIL_ROOT)$(WEBROOT)
	rm -f $(JAIL_ROOT)$(WEBROOT)/*.gz $(JAIL_ROOT)$(WEBROOT)/*.br
	install -m 644 $(WEB)/* $(JAIL_ROOT)$(WEBROOT)/
	install -d -m 755 $(JAIL_ROOT)/usr/local/bin $(JAIL_ROOT)/usr/local/etc/rc.d
	install -m 755 bin/mdedit-web $(JAIL_ROOT)/usr/local/bin/mdedit-web
	install -m 755 assets/freebsd/mdedit_web $(JAIL_ROOT)/usr/local/etc/rc.d/mdedit_web
	sysrc -R $(JAIL_ROOT) mdedit_web_enable=YES mdedit_web_dir=$(WEBROOT)
	@jid=$$(jls jid path 2>/dev/null | awk -v p=$(JAIL_ROOT) '$$2 == p { print $$1 }'); \
	if [ -n "$$jid" ]; then \
		echo "jexec $$jid service mdedit_web restart"; jexec $$jid service mdedit_web restart; \
	else \
		echo "The jail isn't running; the service starts with it."; \
	fi
	@echo "Installed. Port and HTTPS: sysrc -R $(JAIL_ROOT) mdedit_web_addr=:8080 mdedit_web_flags=…"
