package spell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRealDictionaries loads every <code>.aff/.dic pair in the folder
// named by MDEDIT_DICTS and checks that a sample of each one's own words
// is accepted, and some known words and compounds. It is skipped without
// MDEDIT_DICTS, so the tests don't need the network.
func TestRealDictionaries(t *testing.T) {
	dir := os.Getenv("MDEDIT_DICTS")
	if dir == "" {
		t.Skip("MDEDIT_DICTS not set")
	}
	affs, _ := filepath.Glob(filepath.Join(dir, "*.aff"))
	for _, aff := range affs {
		code := strings.TrimSuffix(filepath.Base(aff), ".aff")
		h := loadDictionary(t, dir, code)
		if h != nil {
			checkOwnWords(t, code, h)
			checkKnown(t, code, h)
		}
	}
}

func loadDictionary(t *testing.T, dir, code string) *Hunspell {
	aff, err1 := os.Open(filepath.Join(dir, code+".aff"))
	dic, err2 := os.Open(filepath.Join(dir, code+".dic"))
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	defer aff.Close()
	defer dic.Close()
	start := time.Now()
	h, err := LoadHunspell(aff, dic)
	if err != nil {
		t.Errorf("%s: %v", code, err)
		return nil
	}
	t.Logf("%-8s %7d stems, loaded in %v", code, len(h.words), time.Since(start).Round(time.Millisecond))
	return h
}

// checkOwnWords checks every 50th stem that is a word on its own.
func checkOwnWords(t *testing.T, code string, h *Hunspell) {
	bad, n := 0, 0
	for w := range h.stems.words {
		n++
		if n%50 == 0 && !h.Correct(w) {
			bad++
		}
	}
	if bad*100 > n/50 {
		t.Errorf("%s: %d of %d sampled dictionary words rejected", code, bad, n/50)
	}
}

// knownWords are correct words per language, including inflections,
// compounds and elisions; the words after "|" are wrong.
var knownWords = map[string]string{
	"nl":    "fietsen spellingcontrole ziekenhuisbed werkte kinderen auto's zo'n | fietz gemaakd",
	"de":    "Häuser Donaudampfschifffahrt Rechtschreibprüfung schönsten | Hauss gemachd",
	"fr":    "maisons l'homme aujourd'hui mangeaient qu'il | maisonn",
	"es":    "casas comiendo niños | cassa",
	"en-GB": "colour organisation running 21st | color runing",
	"sv":    "husen stavningskontroll barnen | huss",
	"pl":    "domu pisałem książka | domm",
	"it":    "case mangiavano l'anno | casaa",
}

func checkKnown(t *testing.T, code string, h *Hunspell) {
	good, bad, _ := strings.Cut(knownWords[code], "|")
	for _, w := range strings.Fields(good) {
		if !h.Correct(w) {
			t.Errorf("%s: %q should be correct", code, w)
		}
	}
	for _, w := range strings.Fields(bad) {
		if h.Correct(w) {
			t.Errorf("%s: %q should be wrong", code, w)
		}
	}
}
