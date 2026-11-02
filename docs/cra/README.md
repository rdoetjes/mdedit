# EU Cyber Resilience Act: conformity file for mdedit

Regulation (EU) 2024/2847. Route: internal control, Module A (Annex VIII, Part I).
State as of 2 November 2026, version 0.1.0.

**Status: not ready to declare.** The declaration stays unsigned until the
blocking items below are closed. This file was prepared with AI assistance and
isn't legal advice; the manufacturer declares conformity on its sole responsibility.

## Dates

| Date | What applies | Article |
|---|---|---|
| 11 June 2026 | Notified bodies; doesn't affect Module A | 71(2) |
| **11 Sept 2026, in force now** | Reporting of actively exploited vulnerabilities and severe incidents, also for products already on the market | 14, 69(3) |
| 11 Dec 2027 | All other obligations, including this conformity file | 71(2) |

A product placed on the market before 11 December 2027 only needs the rest if it is
substantially modified after that date (Article 69(2)). Don't rely on this:
mdedit is under active development.

## The file

| Document | Requirement |
|---|---|
| [technical-documentation.md](technical-documentation.md) | Annex VII: the main file, which refers to the others |
| [risk-assessment.md](risk-assessment.md) | Article 13(2) and (3); Annex VII point 3, Annex I Part I |
| [vulnerability-handling.md](vulnerability-handling.md) | Annex I Part II; Article 14 reporting procedure |
| [support-period.md](support-period.md) | Article 13(8), Annex VII point 4 |
| [test-report.md](test-report.md) | Annex VII point 6 |
| [user-information.md](user-information.md) | Annex II |
| [declaration-of-conformity.md](declaration-of-conformity.md) | Annexes V and VI: **draft, unsigned** |
| [sbom/](sbom/) | Annex I Part II point 1 (`make sbom`) |
| [../../SECURITY.md](../../SECURITY.md) | Coordinated vulnerability disclosure policy, single point of contact |

## Open items

### Do now (Article 14 already applies)

- [X] **Create an account on ENISA's Single Reporting Platform**, so a report doesn't wait for registration when the clock is running: <https://portal.cra-srp.enisa.europa.eu>
- [X] Fill in the `[FILL: …]` fields: postal address, security contact address, KvK number.
- [X] Turn on GitHub private vulnerability reporting for the repository (Settings → Advanced Security).
- [X] Deploy the fixed version to markdown.phonax.com: the live site still runs on the vulnerable Go and libraries.

### Blocking the declaration (before 11 December 2027)

- [ ] **B1. Security updates, desktop.** There is no update check or notification (Annex I Part I, 2(c)). Needed: at least a notification of a new version, and a decision on automatic updates with an opt-out.
- [x] **B2. Secure distribution.** `.github/workflows/release.yml` builds every push to `main` into a tagged release with SHA256SUMS and GitHub build attestations (Sigstore signatures), verifiable with `gh attestation verify`. Only Linux and the browser version are published. Check after the first run. **Keep releases for at least 10 years: don't delete them** (Article 13(9)). **macOS builds delivered to clients** come from `make macos` and aren't covered: record each delivery (version, client, SHA-256 of the zip), and give the client the hash so they can check it.
- [x] **B3. Scan Linux.** Done in CI for Linux and macOS on every release, and once by hand in Ubuntu 24.04. **FreeBSD:** `make security` there before each FreeBSD build.
- [ ] **B4. Hostile-input testing.** Markdown, Hunspell and font parsing are untested against malicious input (fuzzing), and compound and rule matching have no time limit.
- [ ] **B5. Data removal.** Removal is by hand, per the instructions. Decide whether that is "easy" enough, or add a menu item (Annex I Part I, 2(m)).
- [ ] **B6. Support period.** Set and publish the end date (month and year): [support-period.md](support-period.md).
- [ ] **B7. Harmonised standards.** None was cited in the Official Journal when this was written. Check again before signing; if one is cited, consider applying it.
- [ ] **B8. Simplified form.** Article 33(5) lets small businesses use a simplified form of the technical documentation. Check whether the Commission has published it.
- [ ] Sign the declaration, and put the CE marking on it or on the website (Article 30(1)).

### Keep doing

- [ ] `make security` at least monthly, also without releases (the workflow only scans on a push); update Go and modules.
- [ ] Each release: the workflow publishes the SBOM with it. Keep this file and the releases for at least 10 years or the support period, whichever is longer (Article 13(13)).
- [ ] Review the risk assessment on every substantial change (Article 13(3)).

## Sources checked

Final text of Regulation (EU) 2024/2847 through third-party reproductions (EUR-Lex
refused automated access). **Verify the articles against EUR-Lex before signing.**
The Commission's guidance of July 2026 (C(2026) 5252) was seen only in summaries.
