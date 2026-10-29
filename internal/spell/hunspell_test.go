package spell

import (
	"slices"
	"strings"
	"testing"
)

// testAff exercises the supported directives with long flags and aliases.
const testAff = `SET UTF-8
FLAG long
TRY esianrtolcdugmphbyfvkwz
ICONV 1
ICONV ’ '
NEEDAFFIX Na
FORBIDDENWORD Fb
ONLYINCOMPOUND Oc
COMPOUNDBEGIN Cb
COMPOUNDEND Ce
COMPOUNDPERMITFLAG Cp
COMPOUNDMIN 3
AF 2
AF SsPu
AF Sv
REP 1
REP ph f
COMPOUNDRULE 1
COMPOUNDRULE (Nn)*(Nt)

SFX Ss Y 2
SFX Ss 0 s [^s]
SFX Ss 0 es s

SFX Sv Y 1
SFX Sv 0 ed/Sl .

SFX Sl Y 1
SFX Sl 0 ly .

SFX Lk Y 1
SFX Lk 0 s/CbCpOc .

PFX Pu Y 1
PFX Pu 0 un .

SFX Ns N 1
SFX Ns 0 ness .
`

const testDic = `11
kind/1
bus/Ss
walk/2
house/CbCeSs
boat/CeSs
work/CbLk
place/Ce
happi/NaNs
teh/Fb
1/Nn
th/Nt
l'
`

func testHunspell(t *testing.T) *Hunspell {
	h, err := LoadHunspell(strings.NewReader(testAff), strings.NewReader(testDic))
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func TestHunspellCorrect(t *testing.T) {
	h := testHunspell(t)
	good := []string{
		"kind", "kinds", "unkind", "unkinds", "buses", // suffix, prefix, both
		"walked", "walkedly", // two suffixes
		"happiness",                            // NEEDAFFIX stem with an affix
		"houseboat", "houseboats", "workplace", // compounds, linking s
		"Kind", "UNKIND", "111th", "l'house", "l’house",
	}
	for _, w := range good {
		if !h.Correct(w) {
			t.Errorf("%q should be correct", w)
		}
	}
	bad := []string{"kindes", "buss", "unbus", "happi", "teh", "works", "boathouse", "workplac", "1th1"}
	for _, w := range bad {
		if h.Correct(w) {
			t.Errorf("%q should be wrong", w)
		}
	}
}

func TestHunspellSuggest(t *testing.T) {
	h := testHunspell(t)
	tests := map[string]string{"kidn": "kind", "hous": "house", "Wlaked": "Walked", "phlace": "place"}
	for word, want := range tests {
		if got := h.Suggest(word, 3); len(got) == 0 || got[0] != want {
			t.Errorf("Suggest(%q) = %q, want %q first", word, got, want)
		}
	}
}

func TestSet(t *testing.T) {
	s := Set{New("hallo", "fiets"), testHunspell(t)}
	if !s.Correct("fiets") || !s.Correct("kinds") || s.Correct("fietz") {
		t.Error("a set should accept the words of each speller and no others")
	}
	if got := s.Suggest("fietz", 3); !slices.Contains(got, "fiets") {
		t.Errorf("Suggest(fietz) = %q", got)
	}
}
