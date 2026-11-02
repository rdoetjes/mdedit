package main

import "testing"

func TestDetectLicense(t *testing.T) {
	tests := map[string]string{
		"Permission is hereby granted, free of charge, to any person":                                "MIT",
		"Redistribution and use in source and binary forms ... Neither the name of Google":           "BSD-3-Clause",
		"Redistribution and use in source and binary forms, with or without modification":            "BSD-2-Clause",
		"This project is provided under the terms of\n\nSPDX-License-Identifier: Unlicense OR MIT\n": "Unlicense OR MIT",
		"Proprietary.": "",
	}
	for text, want := range tests {
		if got := detectLicense(text); got != want {
			t.Errorf("detectLicense(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestKeepDropsWhatIsntCompiledIn(t *testing.T) {
	refs := map[string]string{"a": "pkg:a", "b": "pkg:b"}
	got := keep([]string{"b", "x", "a", "b"}, refs)
	if len(got) != 2 || got[0] != "pkg:a" || got[1] != "pkg:b" {
		t.Errorf("keep = %q", got)
	}
}

func TestSerialIsAUUID(t *testing.T) {
	if s := serial(); len(s) != len("urn:uuid:")+36 || s[9+14] != '4' {
		t.Errorf("serial %q", s)
	}
}
