package ui

import (
	"fmt"

	"gioui.org/font"
	"gioui.org/font/gofont"
	"gioui.org/font/opentype"
	"gioui.org/layout"
	"gioui.org/text"

	"mdedit/internal/sysfont"
)

// The document font: the font of the text and headings in the preview and
// the PDF. Code keeps Go Mono, and the editor too. The chosen family is
// kept in the settings by the files of its faces, so starting mdedit needs
// no scan of the installed fonts; that happens when the dialog opens. The
// same standalone TrueType files go to the preview and the PDF.

// defaultFontName is the dialog's entry for the built-in Go font.
const defaultFontName = "Go (built in)"

// bodyTypeface is the name the chosen font has in the preview's shaper.
const bodyTypeface = "MDEdit Document"

// SavedFont loads the document font chosen in the editor, for exporting a
// PDF without a window. It is empty if none is chosen.
func SavedFont() (sysfont.Set, error) {
	st := loadSettings()
	if st.Font == nil {
		return sysfont.Set{}, nil
	}
	return st.Font.Load()
}

// restoreFont applies the saved font at start-up. If its files are gone,
// the Go font is used and the choice is kept, in case they come back.
func (a *App) restoreFont(st settings) {
	if st.Font == nil {
		return
	}
	if err := a.useFont(*st.Font); err != nil {
		a.setStatus("Could not load the font %s, using the built-in font: %v", st.Font.Name, err)
	}
}

// useFont shows the preview in a family and uses it for PDFs.
func (a *App) useFont(f sysfont.Family) error {
	set, err := f.Load()
	if err != nil {
		return err
	}
	shaper, err := bodyShaper(set)
	if err != nil {
		return err
	}
	a.preview.setFont(shaper, bodyTypeface)
	a.bodyFont = set
	return nil
}

// useDefaultFont goes back to the built-in font.
func (a *App) useDefaultFont() {
	a.preview.setFont(a.th.Shaper, "Go")
	a.bodyFont = sysfont.Set{}
}

// bodyShaper makes a shaper that knows a family as bodyTypeface, besides
// the usual fonts for code and for characters the family lacks.
func bodyShaper(set sysfont.Set) (*text.Shaper, error) {
	faces := append(gofont.Collection(), platformFonts()...)
	for i, ttf := range set.TTF {
		if ttf == nil {
			continue
		}
		face, err := opentype.Parse(ttf)
		if err != nil {
			return nil, fmt.Errorf("font %s: %w", set.Name, err)
		}
		faces = append(faces, font.FontFace{Font: bodyStyle(i), Face: face})
	}
	return text.NewShaper(text.WithCollection(faces)), nil
}

// bodyStyle describes style i of a sysfont.Set.
func bodyStyle(i int) font.Font {
	f := font.Font{Typeface: bodyTypeface}
	if i == sysfont.Bold || i == sysfont.BoldItalic {
		f.Weight = font.Bold
	}
	if i == sysfont.Italic || i == sysfont.BoldItalic {
		f.Style = font.Italic
	}
	return f
}

// cmdFonts opens the font dialog, and looks for the installed fonts the
// first time.
func (a *App) cmdFonts(layout.Context) {
	d := newFontDialog(a.fontName(), a.fontChosen)
	d.setFamilies(a.families)
	a.dialog = d
	if a.families == nil && a.fontArrival == nil {
		a.fontArrival = make(chan []sysfont.Family, 1)
		go scanFonts(a.fontArrival, a.invalidate)
	}
}

// scanFonts runs on its own goroutine.
func scanFonts(out chan<- []sysfont.Family, wake func()) {
	defer wake()
	out <- sysfont.Scan()
}

// handleFontScan takes the result of a scan, and gives it to the dialog.
func (a *App) handleFontScan() {
	select {
	case fams := <-a.fontArrival:
		a.families, a.fontArrival = fams, nil
		if d, ok := a.dialog.(*FontDialog); ok {
			d.setFamilies(fams)
		}
	default:
	}
}

func (a *App) fontName() string {
	if a.bodyFont.Name == "" {
		return defaultFontName
	}
	return a.bodyFont.Name
}

// fontChosen applies and saves the font chosen in the dialog; name is
// empty if it was cancelled.
func (a *App) fontChosen(name string) {
	a.dialog = nil
	a.focusEditor = true
	if name == "" || name == a.fontName() {
		return
	}
	st := loadSettings()
	st.Font = a.applyFont(name)
	a.saveSettings(st, "the document font")
}

// applyFont switches to a family by name, and returns what to save: the
// family, or nil for the built-in font (also if the family can't be used).
func (a *App) applyFont(name string) *sysfont.Family {
	for i := range a.families {
		if a.families[i].Name != name {
			continue
		}
		if err := a.useFont(a.families[i]); err != nil {
			a.setStatus("Could not load the font %s: %v", name, err)
			break
		}
		a.setStatus("Document font: %s", name)
		return &a.families[i]
	}
	a.useDefaultFont()
	return nil
}
