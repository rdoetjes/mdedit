package ui

import (
	"bytes"
	"path"
	"slices"

	"gioui.org/layout"

	"mdedit/internal/spell"
	"mdedit/internal/vfs"
)

// languages returns the chosen spelling languages; English by default.
func (st settings) languages() []string {
	if len(st.Languages) == 0 {
		return []string{spell.BuiltIn}
	}
	return st.Languages
}

// dictArrival is a dictionary loaded (or not) in the background.
type dictArrival struct {
	code       string
	dict       spell.Speller
	err        error
	downloaded bool
}

// setLanguages switches to the given languages, loading the ones not
// loaded yet; announce shows in the status bar when they are ready.
func (s *spellChecker) setLanguages(codes []string, announce bool) {
	s.languages = codes
	for _, code := range codes {
		s.load(code, announce)
	}
	s.combine()
}

// load starts loading a language unless it is loaded or loading.
func (s *spellChecker) load(code string, announce bool) {
	switch {
	case s.loaded[code] != nil || s.loading[code]:
	case code == spell.BuiltIn:
		s.loaded[code] = spell.English()
	default:
		s.loading[code] = true
		if announce {
			s.a.setStatus("Loading the %s dictionary…", spell.LanguageName(code))
		}
		go loadDictionary(code, s.arrivals, s.a.invalidate)
	}
}

// combine makes the speller from the user's words and the chosen
// languages loaded so far.
func (s *spellChecker) combine() {
	s.dict = spell.Set{s.user}
	for _, code := range s.languages {
		if d := s.loaded[code]; d != nil {
			s.dict = append(s.dict, d)
		}
	}
}

// handleArrivals takes the dictionaries loaded in the background.
func (s *spellChecker) handleArrivals() {
	for {
		select {
		case d := <-s.arrivals:
			s.arrived(d)
		default:
			return
		}
	}
}

func (s *spellChecker) arrived(d dictArrival) {
	delete(s.loading, d.code)
	name := spell.LanguageName(d.code)
	if d.err != nil {
		s.a.setStatus("Could not load the %s dictionary: %v", name, d.err)
		return
	}
	s.loaded[d.code] = d.dict
	s.combine()
	s.recheck()
	if d.downloaded {
		s.a.setStatus("Downloaded the %s dictionary", name)
	}
}

// loadDictionary reads a language's dictionary from the dictionary folder,
// downloading it first if it isn't there, and sends the result. It runs
// on its own goroutine.
func loadDictionary(code string, out chan<- dictArrival, wake func()) {
	defer wake()
	aff, err1 := vfs.ReadFile(dictPath(code, "aff"))
	dic, err2 := vfs.ReadFile(dictPath(code, "dic"))
	downloaded := false
	if err1 != nil || err2 != nil {
		var err error
		if aff, dic, err = downloadDictionary(code); err != nil {
			out <- dictArrival{code: code, err: err}
			return
		}
		downloaded = true
	}
	h, err := spell.LoadHunspell(bytes.NewReader(aff), bytes.NewReader(dic))
	out <- dictArrival{code: code, dict: h, err: err, downloaded: downloaded}
}

// downloadDictionary downloads a dictionary and keeps it.
func downloadDictionary(code string) (aff, dic []byte, err error) {
	if aff, dic, err = spell.Download(code); err != nil {
		return nil, nil, err
	}
	if err = vfs.WriteFile(dictPath(code, "aff"), aff); err == nil {
		err = vfs.WriteFile(dictPath(code, "dic"), dic)
	}
	return aff, dic, err
}

func dictPath(code, ext string) string {
	return path.Join(dictionaryDir(), code+"."+ext)
}

// haveDictionary reports whether a language can be used without
// downloading it.
func (s *spellChecker) haveDictionary(code string) bool {
	if code == spell.BuiltIn || s.loaded[code] != nil {
		return true
	}
	_, err := vfs.ModTime(dictPath(code, "dic"))
	return err == nil
}

// cmdLanguages opens the spelling languages dialog.
func (a *App) cmdLanguages(layout.Context) {
	a.dialog = newLanguageDialog(a.spell.languages, a.spell.haveDictionary, a.languagesChosen)
}

// languagesChosen applies and saves the languages chosen in the dialog,
// or does nothing if it was cancelled (codes is nil).
func (a *App) languagesChosen(codes []string) {
	a.dialog = nil
	if codes == nil || slices.Equal(codes, a.spell.languages) {
		return
	}
	a.spell.setLanguages(codes, true)
	a.spell.recheck()
	st := loadSettings()
	st.Languages = codes
	if err := st.save(); err != nil {
		a.setStatus("Could not save the spelling languages: %v", err)
	}
}
