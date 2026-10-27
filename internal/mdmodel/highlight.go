package mdmodel

import (
	"image/color"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// HighlightStyle is the chroma style used for code blocks.
var HighlightStyle = "github"

var (
	hlMu    sync.Mutex
	hlCache = map[[2]string][]Span{}
)

// Highlight splits code into coloured monospace spans. Code without a known
// language is returned as a single plain span.
func Highlight(lang, code string) []Span {
	key := [2]string{lang, code}
	hlMu.Lock()
	if s, ok := hlCache[key]; ok {
		hlMu.Unlock()
		return s
	}
	hlMu.Unlock()

	spans := highlight(lang, code)

	hlMu.Lock()
	if len(hlCache) > 512 {
		hlCache = map[[2]string][]Span{}
	}
	hlCache[key] = spans
	hlMu.Unlock()
	return spans
}

func highlight(lang, code string) []Span {
	plain := []Span{{Text: code, Mono: true}}
	if lang == "" {
		return plain
	}
	lexer := lexers.Get(lang)
	if lexer == nil {
		return plain
	}
	lexer = chroma.Coalesce(lexer)
	style := styles.Get(HighlightStyle)
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return plain
	}
	var out []Span
	for tok := it(); tok != chroma.EOF; tok = it() {
		e := style.Get(tok.Type)
		s := Span{Text: tok.Value, Mono: true, Bold: e.Bold == chroma.Yes, Italic: e.Italic == chroma.Yes}
		if e.Colour.IsSet() {
			s.Color = color.NRGBA{R: e.Colour.Red(), G: e.Colour.Green(), B: e.Colour.Blue(), A: 0xff}
		}
		out = append(out, s)
	}
	out = mergeSpans(out)
	// Lexers usually append a trailing newline; drop it.
	if n := len(out); n > 0 {
		t := out[n-1].Text
		for len(t) > 0 && t[len(t)-1] == '\n' {
			t = t[:len(t)-1]
		}
		if t == "" {
			out = out[:n-1]
		} else {
			out[n-1].Text = t
		}
	}
	return out
}
