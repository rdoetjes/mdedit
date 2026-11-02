# Cybersecurity risk assessment (Article 13(2), Annex VII point 3)

Product: mdedit 0.1.0. Assessed 2 November 2026. Update on every substantial change.

## Use and environment

**Intended purpose:** one person writing Markdown, with live preview, spell check and
PDF export. **Foreseeable misuse:** opening documents from strangers; sensitive text on
shared computers. **Environment:** the user's own desktop, or a browser with a static
web server behind an HTTPS proxy. **Lifetime:** see [support-period.md](support-period.md).

**Assets:** the user's documents; the user's account and files (a program could misuse
them); the integrity of the program and its updates; availability of unsaved work.

**Not present:** accounts, passwords, a server holding user data, telemetry, plug-ins,
script execution from documents.

**Attack surface:** Markdown files, including anything pasted; images; dictionary files;
installed fonts; links clicked by the user; three outgoing HTTPS connections
(dictionaries from raw.githubusercontent.com at a pinned commit, YouTube stills, and
the browser's own loading); the `mdedit-web` static server.

## Risks

Likelihood and impact: L low, M medium, H high, before and after measures.

| # | Threat | Before | Measures | After | Status |
|---|---|---|---|---|---|
| R1 | A crafted link starts a program or reaches an application through a URL scheme | H | Only http, https and mailto are opened (`safeLink`); a local non-Markdown file asks first. Fixed in this assessment; tested | L | Done |
| R2 | Known vulnerabilities in Go or libraries | H | Found 35 reachable (Go 1.25.5, x/image); updated to Go 1.27.3 and current modules; `make security` | L | Done, keep scanning |
| R3 | A malicious file crashes or hangs the editor through a parser bug | M | Memory-safe language, no unsafe parsing of our own; recursion limited for compounds; recovery copy on window close. **No fuzzing; no time limit on rule matching** | M | Open, B4 |
| R4 | A local image is pulled into a PDF the user then shares | M | Image files only, shown in the preview first; documented for users | L | Accepted, documented |
| R5 | A tampered dictionary | M | HTTPS, pinned commit, 64 MB limit; dictionaries are data, not code. Parsing as R3 | L | Done |
| R6 | A tampered update or download | H | **No signed releases or published hashes** | H | Open, B2 |
| R7 | Users keep running a vulnerable version | H | Browser version updates with the site. **Desktop has no update notification** | H | Open, B1 |
| R8 | Compromised or vulnerable dependency | M | `go.sum` hashes; few dependencies; SBOM; Gio copy in the repository, which needs manual follow-up | L | Done, keep watching |
| R9 | Access to local documents by other users or programs | L | Plain files under the user's account; recovery files 0600 in a 0700 folder; the operating system protects them; no encryption | L | Accepted, documented |
| R10 | Path traversal or disclosure through `mdedit-web` | M | Cleaned paths, GET and HEAD only, regular files only, runs as an unprivileged user, serves one folder of public files, `nosniff` | L | Done |
| R11 | Tracking through remote content | L | Remote images blocked, except YouTube stills over HTTPS with size limit and timeout | L | Accepted, documented |
| R12 | A malicious font or settings file | L | Needs write access to the user's account, which already defeats the product's protections | L | Accepted |

## Annex I Part I: applicability and status

"Met" means the assessment found no gap, not that an authority has confirmed it.

| Point | Requirement | Assessment | Status |
|---|---|---|---|
| (1) | Appropriate level of cybersecurity based on the risks | Follows from the table above | Open until R3, R6, R7 close |
| 2(a) | No known exploitable vulnerabilities | No findings on macOS, Windows and browser targets; Linux and FreeBSD GUI not scanned | **Partly, B3** |
| 2(b) | Secure by default, reset possible | No open ports on desktop, no accounts, network only on request; reset by deleting the settings folder | Met |
| 2(c) | Security updates, automatic by default where applicable, notification, postponing | Browser: yes, by the site. Desktop: none | **Not met, B1** |
| 2(d) | Protection from unauthorised access | No accounts or remote access in the program; files protected by the operating system. Static server holds public files only | Met for this use |
| 2(e) | Confidentiality of data | HTTPS for all traffic; data at rest not encrypted, left to the disk encryption of the platform, as for any plain-text editor; documented | Met, by decision |
| 2(f) | Integrity of data and programs | Plain files by design; failed saves are reported. Program integrity depends on B2 | **Partly, B2** |
| 2(g) | Data minimisation | Keeps only settings, recent files and recovery copies; no telemetry | Met |
| 2(h) | Availability of basic functions, resilience | Recovery copy on close; browser autosave. No desktop autosave, DoS by hostile files possible | **Partly, B4** |
| 2(i) | Minimise impact on other services | Requests only on user action, once per item; failures aren't retried for a minute | Met |
| 2(j) | Limit attack surfaces | No listening port on desktop, no plug-ins, schemes and hosts allow-listed | Met |
| 2(k) | Reduce the impact of an incident | Memory-safe language; no privileges needed. Server runs unprivileged in a jail. No extra sandboxing on desktop | Met, by decision |
| 2(l) | Record security-related activity, with opt-out | Not applicable to a single-user local editor without accounts or services: there is nothing to attribute. Server logs go to its log file | Not applicable, justified |
| 2(m) | Remove all data and settings securely and easily | By deleting folders, as instructed. No export of data to other systems beyond plain files | **Partly, B5** |
