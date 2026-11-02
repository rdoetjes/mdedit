# Test report (Annex VII point 6)

Run 2 November 2026 on macOS (arm64), Go 1.27.3, at the state of the repository
holding this report. Repeat for each release.

| Test | Command | Result |
|---|---|---|
| Unit and UI tests | `make test` | 118 passed, 0 failed; 3 skipped by design, run separately below |
| Browser build check | `GOOS=js GOARCH=wasm go vet ./...` | Passed |
| Known vulnerabilities, macOS | `govulncheck ./...` v1.9.0, database of 27 Oct 2026 | No vulnerabilities found |
| Known vulnerabilities, browser build | same, `GOOS=js GOARCH=wasm` | No vulnerabilities found |
| Known vulnerabilities, Windows | same, `GOOS=windows`; loaded packages | No vulnerabilities found |
| Known vulnerabilities, Linux, FreeBSD | same |No vulnerabilities found |
| Dictionaries | `MDEDIT_DICTS=… go test ./internal/spell -run Real`, 59 dictionaries | Passed |
| Fonts | `MDEDIT_FONTS=… go test ./internal/sysfont -run RealFonts`, 80 families, PDFs viewed | Passed |
| Patched Gio | `go test ./text/` in `third_party/gioui.org` | Passed |
| Patched Gio, widgets | `go test ./widget/` | 1 failure, `TestIndexPositionRunes`; the same test fails on unmodified Gio v0.10.3 (works as expected as we have extended Gio). Editor tests pass |
| SBOM | Validated against the CycloneDX 1.6 JSON schema | 0 errors |
| Browser build | Loaded in headless Chrome, typed text, checked storage | Text stored, no page errors |
| Links | `internal/ui/links_test.go` | `file:`, `smb:`, `javascript:`, app schemes and local paths refused; local files ask |

## Before these fixes

The same scan on Go 1.25.5 found 35 vulnerabilities reachable from the code. Updating
Go and modules removed them; all tests passed afterwards.

## Not tested

No fuzzing or penetration test (B4). No test on a real Linux or FreeBSD desktop. From
now on every push to `main` runs the tests on Linux and macOS and the scan on Linux,
macOS and Windows; a failure stops the release. The scan finds only
published vulnerabilities in dependencies, not unknown ones or flaws in mdedit's own logic.
