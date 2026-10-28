// Package lint enforces the code style of this repository:
//
//   - no anonymous functions (function literals);
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
	maxIndent = 4
	maxLines  = 40
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

func skipDir(name string) bool {
	return name == "bin" || strings.HasPrefix(name, ".")
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
		v.report(n.Pos(), "anonymous function")
	case *ast.FuncDecl:
		v.checkLength(n)
	}
	return v
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
