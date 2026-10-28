package mdmodel

import (
	"image/color"
	"testing"
)

func TestHighlightLanguages(t *testing.T) {
	samples := map[string]string{
		"bash":       "for f in *.md; do echo \"$f\"; done",
		"sh":         "export PATH=$HOME/bin:$PATH",
		"go":         "func main() { fmt.Println(42) }",
		"C#":         "public class Foo { int x = 1; }",
		"csharp":     "using System;",
		"c++":        "int main() { return 0; }",
		"C":          "#include <stdio.h>\nint main(void) { return 0; }",
		"python":     "def f(x):\n    return x + 1",
		"js":         "const x = () => 1;",
		"ts":         "let x: number = 1;",
		"rust":       "fn main() { let x = 1; }",
		"java":       "public static void main(String[] a) {}",
		"yaml":       "key: value\nlist:\n  - 1",
		"json":       `{"a": 1, "b": true}`,
		"sql":        "SELECT * FROM t WHERE id = 1;",
		"html":       "<div class=\"x\">hi</div>",
		"dockerfile": "FROM golang:1.25\nRUN go build",
		"F#":         "let x = 1",
		"pascal":     "begin writeln('hi'); end.",
		"{.python}":  "import os",
	}
	for lang, code := range samples {
		spans := Highlight(lang, code)
		if !anyColored(spans) {
			t.Errorf("%s: no highlighting", lang)
		}
		if text := joinText(spans); text != code {
			t.Errorf("%s: text changed: %q", lang, text)
		}
	}
	if s := Highlight("no-such-language", "x"); len(s) != 1 || s[0].Color != (color.NRGBA{}) {
		t.Errorf("unknown language should be plain, got %+v", s)
	}
}

func anyColored(spans []Span) bool {
	for _, s := range spans {
		if s.Color != (color.NRGBA{}) {
			return true
		}
	}
	return false
}

func joinText(spans []Span) string {
	var text string
	for _, s := range spans {
		text += s.Text
	}
	return text
}
