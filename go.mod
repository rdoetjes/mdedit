module mdedit

go 1.26.0

toolchain go1.27.3

require (
	gioui.org v0.10.3
	gioui.org/x v0.10.3
	github.com/alecthomas/chroma/v2 v2.27.0
	github.com/go-pdf/fpdf v0.9.0
	github.com/go-text/typesetting v0.3.5
	github.com/yuin/goldmark v1.8.7
	golang.org/x/exp/shiny v0.0.0-20261014143641-f0251eb4aa3e
	golang.org/x/image v0.47.0
)

require (
	gioui.org/shader v1.0.9 // indirect
	git.wow.st/gmp/jni v0.0.0-20210610011705-34026c7e22d0 // indirect
	github.com/dlclark/regexp2/v2 v2.12.0 // indirect
	github.com/godbus/dbus/v5 v5.2.4 // indirect
	golang.org/x/net v0.62.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/text v0.43.0 // indirect
)

replace gioui.org => ./third_party/gioui.org
