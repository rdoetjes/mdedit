package edit

import (
	"slices"
	"testing"
)

func TestFindAll(t *testing.T) {
	text := []rune("Go go GO gopher; Ünïcode ünïcode")
	cases := []struct {
		q    Query
		want []int
	}{
		{Query{Text: "go"}, []int{0, 3, 6, 9}},
		{Query{Text: "go", MatchCase: true}, []int{3, 9}},
		{Query{Text: "ünï"}, []int{17, 25}},
		{Query{Text: "aa"}, nil},
		{Query{Text: ""}, nil},
	}
	for _, c := range cases {
		if got := FindAll(text, c.q, 0); !slices.Equal(got, c.want) {
			t.Errorf("%+v: got %v, want %v", c.q, got, c.want)
		}
	}
	if got := FindAll([]rune("aaaa"), Query{Text: "aa"}, 0); !slices.Equal(got, []int{0, 2}) {
		t.Errorf("overlapping: %v", got)
	}
	if got := FindAll([]rune("x x x"), Query{Text: "x"}, 2); len(got) != 2 {
		t.Errorf("limit: %v", got)
	}
}

func TestNextPrevMatch(t *testing.T) {
	m := []int{2, 10, 20}
	for _, c := range [][3]int{{0, 0, 2}, {2, 0, 2}, {3, 1, 0}, {21, 0, 2}} {
		if n := NextMatch(m, c[0]); n != c[1] {
			t.Errorf("next from %d: %d, want %d", c[0], n, c[1])
		}
		if p := PrevMatch(m, c[0]); p != c[2] {
			t.Errorf("prev from %d: %d, want %d", c[0], p, c[2])
		}
	}
	if NextMatch(nil, 0) != -1 || PrevMatch(nil, 0) != -1 {
		t.Error("no matches should give -1")
	}
}

func TestReplace(t *testing.T) {
	q := Query{Text: "cat"}
	s, ok := ReplaceSelection(st(t, "a [Cat] here"), q, "dog")
	if !ok {
		t.Fatal("selection should match case-insensitively")
	}
	check(t, "replace selection", s, "a [dog] here")
	if _, ok := ReplaceSelection(st(t, "a [Ca]t here"), q, "dog"); ok {
		t.Error("partial selection replaced")
	}
	all, n := ReplaceAll(st(t, "cat Cat cAt|"), q, "dog")
	check(t, "replace all", all, "|dog dog dog")
	if n != 3 {
		t.Errorf("replaced %d, want 3", n)
	}
	cs, n := ReplaceAll(st(t, "cat Cat|"), Query{Text: "cat", MatchCase: true}, "x")
	check(t, "match case", cs, "|x Cat")
	if n != 1 {
		t.Errorf("replaced %d, want 1", n)
	}
}
