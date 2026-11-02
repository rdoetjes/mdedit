#!/bin/sh
# Writes THIRD_PARTY_LICENSES.md: the licence of everything compiled into
# mdedit, whose terms require it to go along with the program. Run it after
# changing dependencies (make licenses).
set -e
out=THIRD_PARTY_LICENSES.md
modules="gioui.org/shader gioui.org/x github.com/alecthomas/chroma/v2
github.com/dlclark/regexp2/v2 github.com/go-pdf/fpdf github.com/go-text/typesetting
github.com/godbus/dbus/v5 github.com/yuin/goldmark golang.org/x/exp/shiny
golang.org/x/image golang.org/x/net golang.org/x/sys golang.org/x/text"

section() { # title, file
	printf '\n## %s\n\n```text\n' "$1" >>"$out"
	cat "$2" >>"$out"
	printf '```\n' >>"$out"
}

cat >"$out" <<'HEAD'
# Third-party licences

mdedit itself is under the terms in [LICENSE](LICENSE). The components below
stay under their own licences, which are not changed or restricted by it.
HEAD
root=$(go env GOROOT)
golicense=$root/LICENSE
[ -f "$golicense" ] || golicense=$root/../LICENSE # Homebrew keeps it there
[ -f "$golicense" ] || golicense=$(go list -m -f '{{.Dir}}' golang.org/x/sys)/LICENSE # the same text
section "The Go programming language (runtime and standard library)" "$golicense"
section "Gio, gioui.org (patched copy in third_party/gioui.org)" third_party/gioui.org/LICENSE
for m in $modules; do
	dir=$(go list -m -f '{{.Dir}}' "$m")
	file=$(ls "$dir" | grep -i -E '^(licen[cs]e|copying)' | head -1)
	section "$m" "$dir/$file"
done
section "STIX Two Math font (embedded)" internal/tex/fonts/OFL.txt
section "SCOWL / ESDB English word list (embedded)" internal/spell/SCOWL-LICENSE.txt
