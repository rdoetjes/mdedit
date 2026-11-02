# Technical documentation (Annex VII)

Manufacturer: Phonax-Soft, sole proprietorship, the Netherlands, KvK: 52165205.
Keep for at least 10 years after placing on the market, or the support period if longer
(Article 13(13)). Available to market surveillance authorities on request.

## Scope and classification

mdedit is a **product with digital elements** (Article 3) placed on the market in the
course of a commercial activity: commercial use is sold under licence. It is **not free and
open-source software**, since its licence restricts use (PolyForm Noncommercial 1.0.0), so
the open-source provisions don't apply.

It is **not in Annex III or IV**: an editor isn't a browser, password manager or any other
listed category, even though one variant runs inside a browser. So **internal control
(Module A)** is available (Article 32(1)(a)), without a notified body.

Phonax-Soft is the manufacturer and, as a micro business, may use the simplified form of
this documentation once published (Article 33(5)).

## 1. General description

- **Intended purpose:** see [user-information.md](user-information.md).
- **Versions:** 0.1.0. Identified by the version in Help → About and `mdedit -version`.
- **Hardware:** not applicable, software only.
- **User information:** [user-information.md](user-information.md).

## 2. Design, development, production

### (a) Design and architecture

Written in Go, a memory-safe language. Full architecture: [HOW_IT_WORKS.md](../../HOW_IT_WORKS.md).

```text
keyboard, mouse -> ui (Gio window) -> edit (text commands)
                       |-> mdmodel (Markdown to blocks) -> render (positioned primitives)
                                                              |-> ui preview
                                                              |-> pdfout (PDF file)
                   spell: built-in list, downloaded Hunspell dictionaries (HTTPS, pinned)
                   imgcache: YouTube stills (HTTPS, allow-listed)
                   sysfont: installed fonts, desktop only
mdedit-web: static server for the browser build, behind an HTTPS proxy (Caddy)
```

Documents are data: nothing in a document is executed. Links are handled by
`ui/files.go: followLink`; remote images by `youtube.IsThumbnail`.

### (b) Vulnerability handling

[vulnerability-handling.md](vulnerability-handling.md), [SECURITY.md](../../SECURITY.md),
SBOM in [sbom/](sbom/). **Contact address:** in SECURITY.md. **Secure distribution of
updates:** browser version over HTTPS; desktop still open (B1, B2).

### (c) Production and monitoring

Source in Git at <https://github.com/rdoetjes/mdedit>. Dependencies pinned in `go.mod`
and verified through `go.sum`; Gio as a reviewed copy in `third_party/` with
`gioui.org.patch`. Builds with `make`, from source, with a fixed toolchain in `go.mod`.
Every change passes `make test`, including a code-style check. One person develops and
reviews, with AI assistance; no independent review. No automated CI yet.

## 3. Risk assessment

[risk-assessment.md](risk-assessment.md), including how Annex I Part I applies.

## 4. Support period

[support-period.md](support-period.md).

## 5. Standards and specifications

No harmonised standard was cited in the Official Journal when this was written, and no
common specification or certification scheme applies. The solutions adopted are
described in this file. Other specifications used: CycloneDX 1.6 (SBOM), CVSS (severity),
TLS through Go's standard library.

## 6. Test reports

[test-report.md](test-report.md).

## 7. EU declaration of conformity

[declaration-of-conformity.md](declaration-of-conformity.md): draft, unsigned.

## 8. Software bill of materials

[sbom/](sbom/). Available to authorities on reasoned request.
