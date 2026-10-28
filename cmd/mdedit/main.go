// Command mdedit is a Markdown editor with a live preview and PDF export.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"gioui.org/io/event"
	"gioui.org/unit"

	"mdedit/internal/mdmodel"
	"mdedit/internal/pdfout"
	"mdedit/internal/ui"
)

var version = "dev"

func init() { ui.Version = version }

var (
	pdfFlag     = flag.String("pdf", "", "export `file.md` to PDF without opening a window (output next to it, or -o)")
	outFlag     = flag.String("o", "", "output path for -pdf")
	versionFlag = flag.Bool("version", false, "print the version and exit")
)

func main() {
	flag.Usage = usage
	flag.Parse()
	switch {
	case *versionFlag:
		fmt.Println("mdedit", version)
	case *pdfFlag != "":
		check(exportPDF(*pdfFlag, *outFlag))
	case flag.NArg() > 1:
		flag.Usage()
		os.Exit(2)
	default:
		editor, err := ui.New(flag.Arg(0))
		check(err)
		go runWindow(editor)
		app.Events(opener{editor}.handle)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "usage: mdedit [file.md]\n       mdedit -pdf file.md [-o out.pdf]\n\n")
	flag.PrintDefaults()
}

// check exits with an error message if err is not nil.
func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdedit:", err)
		os.Exit(1)
	}
}

// runWindow opens the editor window and exits when it is closed.
func runWindow(editor *ui.App) {
	w := new(app.Window)
	w.Option(app.Title("mdedit"), app.Size(unit.Dp(1280), unit.Dp(820)), app.MinSize(unit.Dp(480), unit.Dp(320)))
	check(editor.Run(w))
	os.Exit(0)
}

// opener passes files the operating system asks to open (the macOS
// Finder's "Open With", or dropping a file on the Dock icon) to the editor.
type opener struct{ editor *ui.App }

func (o opener) handle(e event.Event) bool {
	if u, ok := e.(app.URLEvent); ok && u.URL.Scheme == "file" {
		o.editor.RequestOpen(u.URL.Path)
	}
	return true
}

func exportPDF(in, out string) error {
	src, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	if out == "" {
		out = strings.TrimSuffix(in, filepath.Ext(in)) + ".pdf"
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(in), filepath.Ext(in))
	err = pdfout.Write(f, mdmodel.Parse(src), pdfout.Options{BaseDir: filepath.Dir(in), Title: base})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}
