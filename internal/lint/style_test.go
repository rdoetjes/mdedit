// Package lint enforces the code style of this repository:
//
//   - anonymous functions (closures) only when short: at most
//     maxClosureBody lines between their braces, and none inside another;
//   - no line indented more than maxIndent tabs;
//   - no function longer than maxLines lines.
package lint

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	maxIndent      = 4
	maxLines       = 40
	maxClosureBody = 5
)

func TestStyle(t *testing.T) {
	files := goFiles(t, "../..")
	if len(files) == 0 {
		t.Fatal("no Go files found")
	}
	var problems []string
	for _, f := range files {
		problems = append(problems, checkFile(t, f)...)
	}
	for _, p := range problems {
		t.Error(p)
	}
	if len(problems) > 0 {
		t.Logf("%d style problems", len(problems))
	}
}

func goFiles(t *testing.T, root string) []string {
	c := &collector{root: root}
	if err := filepath.WalkDir(root, c.visit); err != nil {
		t.Fatal(err)
	}
	return c.files
}

// collector gathers the Go files of the repository.
type collector struct {
	root  string
	files []string
}

func (c *collector) visit(path string, d os.DirEntry, err error) error {
	if err != nil {
		return err
	}
	if d.IsDir() && path != c.root && skipDir(d.Name()) {
		return filepath.SkipDir
	}
	if !d.IsDir() && strings.HasSuffix(path, ".go") {
		c.files = append(c.files, path)
	}
	return nil
}

// skipDir skips build output, hidden folders and third_party, which holds
// other people's code (a patched copy of Gio).
func skipDir(name string) bool {
	return name == "bin" || name == "third_party" || strings.HasPrefix(name, ".")
}

func checkFile(t *testing.T, path string) []string {
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	rel := strings.TrimPrefix(path, "../../")
	var out []string
	out = append(out, indentProblems(rel, string(src))...)
	out = append(out, funcProblems(rel, fset, file)...)
	return out
}

func indentProblems(path, src string) []string {
	var out []string
	for i, line := range strings.Split(src, "\n") {
		n := len(line) - len(strings.TrimLeft(line, "\t"))
		if n > maxIndent {
			out = append(out, fmt.Sprintf("%s:%d: indented %d levels (max %d)", path, i+1, n, maxIndent))
		}
	}
	return out
}

func funcProblems(path string, fset *token.FileSet, file *ast.File) []string {
	v := &funcVisitor{path: path, fset: fset}
	ast.Walk(v, file)
	return v.problems
}

type funcVisitor struct {
	path     string
	fset     *token.FileSet
	problems []string
}

func (v *funcVisitor) Visit(n ast.Node) ast.Visitor {
	switch n := n.(type) {
	case *ast.FuncLit:
		v.checkClosure(n)
	case *ast.FuncDecl:
		v.checkLength(n)
	}
	return v
}

// checkClosure allows a short closure; anything longer, or with a closure
// inside it, belongs in a named function that can be tested on its own.
func (v *funcVisitor) checkClosure(fn *ast.FuncLit) {
	lines := v.fset.Position(fn.End()).Line - v.fset.Position(fn.Pos()).Line - 1
	if lines > maxClosureBody {
		v.report(fn.Pos(), fmt.Sprintf("closure of %d lines (max %d): make it a named function", lines, maxClosureBody))
	}
	if hasClosure(fn.Body) {
		v.report(fn.Pos(), "nested closure: make it a named function")
	}
}

func hasClosure(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			found = true
		}
		return !found
	})
	return found
}

func (v *funcVisitor) checkLength(fn *ast.FuncDecl) {
	if fn.Body == nil {
		return
	}
	lines := v.fset.Position(fn.End()).Line - v.fset.Position(fn.Pos()).Line + 1
	if lines > maxLines {
		v.report(fn.Pos(), fmt.Sprintf("function %s is %d lines (max %d)", fn.Name.Name, lines, maxLines))
	}
}

func (v *funcVisitor) report(pos token.Pos, msg string) {
	p := v.fset.Position(pos)
	v.problems = append(v.problems, fmt.Sprintf("%s:%d: %s", v.path, p.Line, msg))
}

func TestClosureRules(t *testing.T) {
	long := "func f() {\n\tg(func() {\n" + strings.Repeat("\t\tx()\n", maxClosureBody+1) + "\t})\n}\n"
	tests := []struct {
		name, src string
		problems  int
	}{
		{"one line", "func f() { g(func() { size += 0.5 }) }\n", 0},
		{"at the limit", strings.Replace(long, "\t\tx()\n", "", 1), 0},
		{"too long", long, 1},
		{"nested", "func f() {\n\tg(func() {\n\t\th(func() { x() })\n\t})\n}\n", 1},
	}
	for _, tt := range tests {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "x.go", "package p\n"+tt.src, 0)
		if err != nil {
			t.Fatal(tt.name, err)
		}
		if got := funcProblems("x.go", fset, file); len(got) != tt.problems {
			t.Errorf("%s: problems %q, want %d", tt.name, got, tt.problems)
		}
	}
}
