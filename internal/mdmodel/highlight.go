package mdmodel

import (
	"strings"
	"sync"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
)

var (
	hlMu    sync.Mutex
	hlCache = map[[2]string][]Span{}
)

// Highlight splits code into monospace spans tagged with their token type
// (keyword, string, comment …); the renderer colours them for its theme.
// Code without a known language is returned as a single plain span.
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

// langAliases maps fence names chroma doesn't know to a lexer it does.
var langAliases = map[string]string{
	"f#":     "fsharp",
	"x86asm": "nasm",
	"pascal": "objectpascal",
	"delphi": "objectpascal",
	"plsql":  "sql",
	"less":   "scss",
}

// normalizeLang turns a fence info word into a lexer name. It accepts
// Pandoc-style "{.python}" and is case-insensitive for aliases.
func normalizeLang(lang string) string {
	lang = strings.Trim(lang, "{}")
	lang = strings.TrimPrefix(lang, ".")
	if a, ok := langAliases[strings.ToLower(lang)]; ok {
		return a
	}
	return lang
}

func highlight(lang, code string) []Span {
	plain := []Span{{Text: code, Mono: true}}
	lexer := lexerFor(lang)
	if lexer == nil {
		return plain
	}
	it, err := lexer.Tokenise(nil, code)
	if err != nil {
		return plain
	}
	var out []Span
	for tok := it(); tok != chroma.EOF; tok = it() {
		out = append(out, Span{Text: tok.Value, Mono: true, Highlighted: true, Token: tok.Type})
	}
	return trimTrailingNewlines(mergeSpans(out))
}

func lexerFor(lang string) chroma.Lexer {
	if lang == "" {
		return nil
	}
	lexer := lexers.Get(normalizeLang(lang))
	if lexer == nil {
		return nil
	}
	return chroma.Coalesce(lexer)
}

// trimTrailingNewlines drops the newline lexers usually append.
func trimTrailingNewlines(out []Span) []Span {
	n := len(out)
	if n == 0 {
		return out
	}
	t := strings.TrimRight(out[n-1].Text, "\n")
	if t == "" {
		return out[:n-1]
	}
	out[n-1].Text = t
	return out
}
