package spell

import (
	"slices"
	"testing"

	"mdedit/internal/mdmodel"
)

func TestCorrect(t *testing.T) {
	d := English()
	for _, w := range []string{"the", "The", "THE", "colour", "color", "organise", "organize",
		"don't", "don’t", "London", "LONDON", "student's", "students'", "Raymond's", "running"} {
		if !d.Correct(w) {
			t.Errorf("%q should be correct", w)
		}
	}
	for _, w := range []string{"teh", "recieve", "london", "speling", "Wrold"} {
		if d.Correct(w) {
			t.Errorf("%q should be misspelt", w)
		}
	}
}

func TestSuggest(t *testing.T) {
	d := English()
	tests := []struct{ word, want string }{
		{"recieve", "receive"},
		{"teh", "the"},
		{"fwe", "few"},
		{"thier", "their"},
		{"speling", "spelling"},
		{"occured", "occurred"},
		{"wrod", "word"},
		{"Helo", "Hello"},
		{"WROLD", "WORLD"},
		{"definately", "definitely"},
		{"seperate", "separate"},
		{"becuase", "because"},
		{"mispelled", "misspelled"},
	}
	for _, tt := range tests {
		got := d.Suggest(tt.word, 5)
		if len(got) == 0 || got[0] != tt.want {
			t.Errorf("Suggest(%q) = %q, want %q first", tt.word, got, tt.want)
		}
	}
}

func TestAdd(t *testing.T) {
	d := New("hello")
	if d.Correct("mdedit") {
		t.Fatal("mdedit is not a word yet")
	}
	d.Add("mdedit")
	if !d.Correct("mdedit") || !d.Correct("Mdedit") {
		t.Error("added word not accepted")
	}
	if got := d.Suggest("mdedti", 3); !slices.Contains(got, "mdedit") {
		t.Errorf("Suggest(mdedti) = %q, want mdedit", got)
	}
}

const checkSample = "# A tset\n\n" +
	"Some `codde` and $x + yy$ maths, a [lnik](http://exampel.com/pathh) and\n" +
	"<span class=\"foo\">tagg</span> plus https://www.wrongg.org/x &nbsp; HTTP\n" +
	"camelCaseWordz, snake_cse, file.goo, v2beta, e.g. and **bolld** _itallic_.\n" +
	"\n```go\nfunc mian() {}\n```\n\n$$\n\\fracc{a}{b}\n$$\n\n" +
	"    indentedd code\n\n" +
	"Last ware-hous wrdo.\n"

func TestCheck(t *testing.T) {
	text := []rune(checkSample)
	misses := Check(English(), text, mdmodel.Parse([]byte(checkSample)))
	var got []string
	for _, m := range misses {
		if w := string(text[m.Start:m.End]); w != m.Word {
			t.Errorf("miss %q at %d-%d covers %q", m.Word, m.Start, m.End, w)
		}
		got = append(got, m.Word)
	}
	want := []string{"tset", "lnik", "tagg", "bolld", "itallic", "hous", "wrdo"}
	if !slices.Equal(got, want) {
		t.Errorf("misses %q, want %q", got, want)
	}
}
