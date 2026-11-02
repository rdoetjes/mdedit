package main

import (
	"crypto/rand"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"
)

// The parts of a CycloneDX 1.6 document that are used here.
type (
	// BOM is the document.
	BOM struct {
		Format       string       `json:"bomFormat"`
		Spec         string       `json:"specVersion"`
		Serial       string       `json:"serialNumber"`
		Version      int          `json:"version"`
		Metadata     Metadata     `json:"metadata"`
		Components   []Component  `json:"components"`
		Dependencies []Dependency `json:"dependencies"`
	}
	// Metadata describes the document and the product.
	Metadata struct {
		Timestamp string       `json:"timestamp"`
		Component Component    `json:"component"`
		Supplier  Organization `json:"supplier"`
	}
	// Organization is a supplier.
	Organization struct {
		Name string   `json:"name"`
		URL  []string `json:"url,omitempty"`
	}
	// Component is one part of the product.
	Component struct {
		Type        string    `json:"type"`
		Ref         string    `json:"bom-ref,omitempty"`
		Name        string    `json:"name"`
		Version     string    `json:"version,omitempty"`
		Description string    `json:"description,omitempty"`
		Purl        string    `json:"purl,omitempty"`
		Licenses    []License `json:"licenses,omitempty"`
		Pedigree    *Pedigree `json:"pedigree,omitempty"`
	}
	// License is an SPDX licence or an expression over them.
	License struct {
		License    LicenseID `json:"license,omitzero"`
		Expression string    `json:"expression,omitempty"`
	}
	// LicenseID names a licence by SPDX id, or else by name.
	LicenseID struct {
		ID   string `json:"id,omitempty"`
		Name string `json:"name,omitempty"`
	}
	// Pedigree says a component differs from the original.
	Pedigree struct {
		Notes string `json:"notes"`
	}
	// Dependency says what a component needs.
	Dependency struct {
		Ref       string   `json:"ref"`
		DependsOn []string `json:"dependsOn,omitempty"`
	}
)

const productRef = "pkg:generic/" + product

// build makes the document.
func build(version string, mods []module, graph map[string][]string) BOM {
	bom := BOM{Format: "CycloneDX", Spec: "1.6", Serial: serial(), Version: 1}
	bom.Metadata = Metadata{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Supplier:  Organization{Name: supplier, URL: []string{website}},
		Component: Component{Type: "application", Ref: productRef, Name: product, Version: version,
			Licenses: []License{{License: LicenseID{ID: license}}}},
	}
	refs := map[string]string{}
	for _, m := range mods {
		c := moduleComponent(m)
		refs[m.Path] = c.Ref
		bom.Components = append(bom.Components, c)
	}
	bom.Components = append(bom.Components, goComponent())
	bom.Components = append(bom.Components, embedded...)
	bom.Dependencies = dependencies(refs, graph)
	return bom
}

func goComponent() Component {
	v := goVersion()
	return Component{Type: "platform", Ref: "pkg:golang/stdlib@" + v, Name: "Go toolchain and standard library",
		Version: v, Purl: "pkg:golang/stdlib@" + v, Licenses: []License{{License: LicenseID{ID: "BSD-3-Clause"}}}}
}

func moduleComponent(m module) Component {
	purl := "pkg:golang/" + m.Path + "@" + m.Version
	c := Component{Type: "library", Ref: purl, Name: m.Path, Version: m.Version, Purl: purl}
	if id := detectLicense(licenseFile(sourceDir(m))); id != "" {
		c.Licenses = licenses(id)
	}
	if m.Replace != nil {
		c.Pedigree = &Pedigree{Notes: "Modified by the supplier: copy in third_party/gioui.org with the changes in " +
			"third_party/gioui.org.patch (shaper cache size, line layout, hiding the caret)."}
	}
	return c
}

func licenses(id string) []License {
	if strings.Contains(id, " ") {
		return []License{{Expression: id}}
	}
	return []License{{License: LicenseID{ID: id}}}
}

var spdxLine = regexp.MustCompile(`(?m)^\s*(?:SPDX-License-Identifier:\s*)?((?:Unlicense|MIT|BSD-[23]-Clause)(?: OR (?:Unlicense|MIT|BSD-[23]-Clause))+)\s*$`)

// detectLicense recognises the licences in use from their wording.
func detectLicense(text string) string {
	if m := spdxLine.FindStringSubmatch(text); m != nil {
		return m[1]
	}
	flat := strings.Join(strings.Fields(text), " ")
	switch {
	case strings.Contains(flat, "Permission is hereby granted, free of charge"):
		return "MIT"
	case strings.Contains(flat, "Neither the name of"):
		return "BSD-3-Clause"
	case strings.Contains(flat, "Redistribution and use in source and binary forms"):
		return "BSD-2-Clause"
	}
	return ""
}

// dependencies gives the product its direct modules, and each module the
// modules it requires, among those compiled in.
func dependencies(refs map[string]string, graph map[string][]string) []Dependency {
	out := []Dependency{{Ref: productRef, DependsOn: keep(graph[product], refs)}}
	for path, ref := range refs {
		out = append(out, Dependency{Ref: ref, DependsOn: keep(graph[path], refs)})
	}
	slices.SortFunc(out, byRef)
	return out
}

func byRef(a, b Dependency) int { return strings.Compare(a.Ref, b.Ref) }

// keep maps paths to refs, dropping modules that aren't compiled in.
func keep(paths []string, refs map[string]string) []string {
	var out []string
	for _, p := range paths {
		if r, ok := refs[p]; ok && !slices.Contains(out, r) {
			out = append(out, r)
		}
	}
	slices.Sort(out)
	return out
}

// serial is a random UUID, as CycloneDX asks.
func serial() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		fmt.Fprintln(os.Stderr, "sbom:", err)
		os.Exit(1)
	}
	b[6], b[8] = b[6]&0x0f|0x40, b[8]&0x3f|0x80
	return fmt.Sprintf("urn:uuid:%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}
