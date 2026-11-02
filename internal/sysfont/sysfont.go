// Package sysfont finds the fonts installed on the computer and turns them
// into standalone TrueType files, which the preview (Gio) and the PDF (fpdf)
// can both use, so the two show the very same font.
//
// Only fonts both can use are listed: TrueType outlines (fpdf can't embed
// PostScript outlines, so most .otf files are out), with the letters and
// accents of Western languages, whose licence allows embedding in documents. Collections (.ttc) are fine:
// a face is cut out of them (Load).
package sysfont

import (
	"bytes"
	"encoding/binary"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"

	"github.com/go-text/typesetting/font"
	ot "github.com/go-text/typesetting/font/opentype"
)

// The four styles of a family, as indices into Family.Faces and Set.TTF.
const (
	Regular = iota
	Bold
	Italic
	BoldItalic
)

// Face is one font in a file; Index counts the fonts of a collection.
type Face struct {
	Path  string
	Index int
}

// Family is an installed font family. Faces[Regular] is always set; the
// other styles are nil if the family lacks them.
type Family struct {
	Name  string
	Faces [4]*Face
}

// Set holds the styles of a family as standalone TrueType files. A style
// the family lacks is nil.
type Set struct {
	Name string
	TTF  [4][]byte
}

// dirs are the folders fonts are installed in.
func dirs() []string {
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return []string{"/System/Library/Fonts", "/Library/Fonts", filepath.Join(home, "Library/Fonts")}
	case "windows":
		return []string{filepath.Join(os.Getenv("WINDIR"), "Fonts"),
			filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft/Windows/Fonts")}
	}
	return []string{"/usr/share/fonts", "/usr/local/share/fonts", "/usr/X11R6/lib/X11/fonts",
		filepath.Join(home, ".fonts"), filepath.Join(home, ".local/share/fonts")}
}

// scanner gathers the usable faces, by family.
type scanner struct {
	byName map[string][]candidate
}

// candidate is a face with what is needed to pick the family's styles.
type candidate struct {
	face   Face
	aspect font.Aspect
}

// Scan returns the usable font families, sorted by name. It reads the
// headers of all installed fonts, so it can take a while the first time.
func Scan() []Family {
	s := &scanner{byName: make(map[string][]candidate)}
	for _, d := range dirs() {
		filepath.WalkDir(d, s.visit) // a missing folder is simply skipped
	}
	out := []Family{} // empty, not nil: a scan found nothing
	for name, cs := range s.byName {
		if f, ok := pickStyles(name, cs); ok {
			out = append(out, f)
		}
	}
	slices.SortFunc(out, byName)
	return out
}

func byName(a, b Family) int {
	return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
}

func (s *scanner) visit(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return nil // keep going past unreadable folders
	}
	if !d.IsDir() && isFontFile(path) {
		s.addFile(path)
	}
	return nil
}

func isFontFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".ttf", ".ttc":
		return true
	}
	return false
}

func (s *scanner) addFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	lds, err := ot.NewLoaders(f)
	if err != nil {
		return
	}
	for i, ld := range lds {
		s.addFace(path, i, ld)
	}
}

func (s *scanner) addFace(path string, i int, ld *ot.Loader) {
	desc, _ := font.Describe(ld, nil)
	if desc.Family == "" || strings.HasPrefix(desc.Family, ".") || !usable(ld) {
		return
	}
	c := candidate{face: Face{Path: path, Index: i}, aspect: desc.Aspect}
	s.byName[desc.Family] = append(s.byName[desc.Family], c)
}

// usable reports whether both the preview and the PDF can use a face.
func usable(ld *ot.Loader) bool {
	return ld.HasTable(ot.MustNewTag("glyf")) && ld.HasTable(ot.MustNewTag("hmtx")) &&
		embeddable(ld) && hasUnicodeMap(ld) && hasLatin(ld)
}

// hasUnicodeMap reports whether the font has the one kind of character map
// fpdf reads: format 4 for Unicode. Without it fpdf doesn't fail, it just
// prints a message and writes text that shows wrongly, so it must be
// checked here. Symbol fonts (platform 3, encoding 0) don't have one.
func hasUnicodeMap(ld *ot.Loader) bool {
	cmap, err := ld.RawTable(ot.MustNewTag("cmap"))
	if err != nil || len(cmap) < 4 {
		return false
	}
	n := int(binary.BigEndian.Uint16(cmap[2:]))
	for i := 0; i < n && 4+8*i+8 <= len(cmap); i++ {
		if unicodeFormat4(cmap, cmap[4+8*i:]) {
			return true
		}
	}
	return false
}

// unicodeFormat4 reads a cmap subtable record as fpdf does.
func unicodeFormat4(cmap, rec []byte) bool {
	platform := binary.BigEndian.Uint16(rec)
	encoding := binary.BigEndian.Uint16(rec[2:])
	offset := int(binary.BigEndian.Uint32(rec[4:]))
	if platform != 0 && !(platform == 3 && encoding == 1) {
		return false
	}
	return offset+2 <= len(cmap) && binary.BigEndian.Uint16(cmap[offset:]) == 4
}

// embeddable reads the licence bits in the OS/2 table: bit 1 alone means
// the font may not be embedded in documents.
func embeddable(ld *ot.Loader) bool {
	os2, err := ld.RawTable(ot.MustNewTag("OS/2"))
	if err != nil || len(os2) < 10 {
		return true // no statement, as in many old fonts
	}
	return (int(os2[8])<<8|int(os2[9]))&0x000f != 2
}

func hasLatin(ld *ot.Loader) bool {
	f, err := font.NewFont(ld)
	if err != nil {
		return false
	}
	for _, r := range latinSample {
		if g, ok := f.Cmap.Lookup(r); !ok || g == 0 {
			return false
		}
	}
	return true
}

// latinSample are characters a font must have to be offered: without the
// accents and typographic punctuation of Western languages a document
// would show boxes in the PDF. Many fonts for other scripts, and novelty
// fonts, have plain a-z only.
const latinSample = "aezAEZ09ÄÖÜäöüéèêëïçß“”‘’—"

// pickStyles chooses a family's faces; a family needs an upright, normal
// width face to count.
func pickStyles(name string, cs []candidate) (Family, bool) {
	f := Family{Name: name}
	f.Faces[Regular] = best(cs, font.StyleNormal, font.WeightNormal, 0)
	f.Faces[Bold] = best(cs, font.StyleNormal, font.WeightBold, font.WeightSemibold)
	f.Faces[Italic] = best(cs, font.StyleItalic, font.WeightNormal, 0)
	f.Faces[BoldItalic] = best(cs, font.StyleItalic, font.WeightBold, font.WeightSemibold)
	return f, f.Faces[Regular] != nil
}

// best returns the face of a style nearest to a weight, at least minWeight
// heavy and, for lighter targets, not bold.
func best(cs []candidate, style font.Style, weight, minWeight font.Weight) *Face {
	var found *Face
	bestDist := float32(1 << 20)
	for i := range cs {
		a := cs[i].aspect
		if a.Style != style || a.Stretch != font.StretchNormal || a.Weight < minWeight {
			continue
		}
		if d := abs(float32(a.Weight - weight)); d < bestDist && d < 200 {
			found, bestDist = &cs[i].face, d
		}
	}
	return found
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// Load reads a family's faces into standalone TrueType files.
func (f Family) Load() (Set, error) {
	s := Set{Name: f.Name}
	for i, face := range f.Faces {
		if face == nil {
			continue
		}
		data, err := face.load()
		if err != nil {
			return Set{}, err
		}
		s.TTF[i] = data
	}
	return s, nil
}

func (fc Face) load() ([]byte, error) {
	data, err := os.ReadFile(fc.Path)
	if err != nil {
		return nil, err
	}
	lds, err := ot.NewLoaders(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if fc.Index >= len(lds) {
		return nil, fs.ErrNotExist
	}
	return standalone(lds[fc.Index])
}
