// Command sbom writes a software bill of materials for mdedit in CycloneDX
// JSON, as Annex I, Part II, point 1 of the EU Cyber Resilience Act asks.
// Run it from the repository root, or with make sbom:
//
//	go run ./tools/sbom -version 0.1.0 -o sbom.cdx.json
//
// It lists what the program is built from: the Go modules compiled in for
// each supported platform (not everything go.mod mentions), the Go toolchain,
// and the embedded data. The patched Gio in third_party is marked as modified.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const (
	product  = "mdedit"
	supplier = "Phonax-Soft"
	license  = "PolyForm-Noncommercial-1.0.0"
	website  = "https://github.com/rdoetjes/mdedit"
)

// targets are the supported platforms, as GOOS/GOARCH.
var targets = []string{"linux/amd64", "linux/arm64", "freebsd/amd64", "darwin/arm64", "darwin/amd64", "windows/amd64", "js/wasm"}

// embedded lists third-party data compiled into the program, which no Go
// module declares.
var embedded = []Component{
	{Type: "data", Name: "STIX Two Math", Version: "2.12",
		Description: "Maths font, embedded in internal/tex/fonts",
		Licenses:    []License{{License: LicenseID{ID: "OFL-1.1"}}}},
	{Type: "data", Name: "SCOWL/ESDB English word list", Version: "1e5b7d3",
		Description: "Built-in spelling dictionary (size 60), embedded in internal/spell",
		Licenses:    []License{{License: LicenseID{Name: "SCOWL licence, see internal/spell/SCOWL-LICENSE.txt"}}}},
}

// module is what go list reports about one module.
type module struct {
	Path, Version, Dir string
	Replace            *module
}

func main() {
	version := flag.String("version", "dev", "version of mdedit")
	out := flag.String("o", "sbom.cdx.json", "output file")
	flag.Parse()
	if err := run(*version, *out); err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}
}

func run(version, out string) error {
	mods, err := compiledModules()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(build(version, mods, moduleGraph()), "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(out, append(data, '\n'), 0o644)
}

// compiledModules returns the modules compiled into any target, sorted.
func compiledModules() ([]module, error) {
	seen := map[string]module{}
	for _, t := range targets {
		if err := addTarget(seen, t); err != nil {
			return nil, err
		}
	}
	var out []module
	for _, m := range seen {
		out = append(out, m)
	}
	slices.SortFunc(out, byPath)
	return out, nil
}

func byPath(a, b module) int { return strings.Compare(a.Path, b.Path) }

func addTarget(seen map[string]module, target string) error {
	goos, goarch, _ := strings.Cut(target, "/")
	cmd := exec.Command("go", "list", "-e", "-deps", "-json=Module", "./cmd/...")
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=1")
	data, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("go list for %s: %w", target, err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	for dec.More() {
		var p struct{ Module *module }
		if err := dec.Decode(&p); err != nil {
			return err
		}
		if p.Module != nil && p.Module.Path != product {
			seen[p.Module.Path] = *p.Module
		}
	}
	return nil
}

// moduleGraph returns which module requires which, by path, from go mod
// graph. Versions are dropped: only the selected ones are compiled in.
func moduleGraph() map[string][]string {
	graph := map[string][]string{}
	data, err := exec.Command("go", "mod", "graph").Output()
	if err != nil {
		return graph
	}
	for _, line := range strings.Split(string(data), "\n") {
		from, to, ok := strings.Cut(line, " ")
		if ok {
			f, t := pathOf(from), pathOf(to)
			graph[f] = append(graph[f], t)
		}
	}
	return graph
}

func pathOf(s string) string {
	p, _, _ := strings.Cut(s, "@")
	return p
}

func goVersion() string {
	data, err := exec.Command("go", "env", "GOVERSION").Output()
	if err != nil {
		return strings.TrimPrefix(runtime.Version(), "go")
	}
	return strings.TrimPrefix(strings.TrimSpace(string(data)), "go")
}

// sourceDir is where a module's source is, also for a replaced module.
func sourceDir(m module) string {
	if m.Replace != nil && m.Replace.Dir != "" {
		return m.Replace.Dir
	}
	return m.Dir
}

// licenseFile returns the text of a module's licence file, or "".
func licenseFile(dir string) string {
	for _, name := range []string{"LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING", "UNLICENSE"} {
		if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return string(data)
		}
	}
	return ""
}
