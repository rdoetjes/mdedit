// Command mdedit is a Markdown editor with a live preview and PDF export.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gioui.org/app"
	"gioui.org/unit"

	"mdedit/internal/mdmodel"
	"mdedit/internal/pdfout"
	"mdedit/internal/ui"
)

var version = "dev"

func init() { ui.Version = version }

func main() {
	pdf := flag.String("pdf", "", "export `file.md` to PDF without opening a window (output next to it, or -o)")
	out := flag.String("o", "", "output path for -pdf")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: mdedit [file.md]\n       mdedit -pdf file.md [-o out.pdf]\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("mdedit", version)
		return
	}
	if *pdf != "" {
		if err := exportPDF(*pdf, *out); err != nil {
			fmt.Fprintln(os.Stderr, "mdedit:", err)
			os.Exit(1)
		}
		return
	}
	if flag.NArg() > 1 {
		flag.Usage()
		os.Exit(2)
	}

	editor, err := ui.New(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdedit:", err)
		os.Exit(1)
	}
	go func() {
		w := new(app.Window)
		w.Option(app.Title("mdedit"), app.Size(unit.Dp(1280), unit.Dp(820)), app.MinSize(unit.Dp(480), unit.Dp(320)))
		if err := editor.Run(w); err != nil {
			fmt.Fprintln(os.Stderr, "mdedit:", err)
			os.Exit(1)
		}
		os.Exit(0)
	}()
	app.Main()
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
