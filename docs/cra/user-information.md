# Information and instructions for users (Annex II)

## Manufacturer

Phonax-Soft, Lijsterbesstraat 232, 3434AK, Nieuwegen, NL, Obtain Email from Github,
<https://github.com/rdoetjes/mdedit>

## Reporting vulnerabilities

Single point of contact and disclosure policy: [SECURITY.md](../../SECURITY.md)

## Product

mdedit, a Markdown editor, version shown in Help → About. Desktop program for Linux,
FreeBSD, macOS and Windows, and a browser version.

## Intended purpose and security properties

Writing Markdown documents with a live preview, spelling checks and PDF export, by
one person on their own computer, or in a browser for the same purpose. mdedit has no
user accounts, stores nothing on a server, and has no telemetry. It only goes on the
network to check for updates at start-up, when you ask for a dictionary, or when a
document shows a YouTube still.

## Known risks

- **Documents from others are untrusted.** mdedit doesn't run them, but links do lead elsewhere. It opens http, https and mailto links in your browser, and refuses other kinds. A link to a local file other than Markdown asks first: answer Open only if you trust the document, because the file could be a program.
- **Images.** A document can show any image your account can read, and a PDF export embeds it. Check documents from others before you export and share the PDF.
- **YouTube stills** are fetched from YouTube when shown, which reveals your IP address to it.
- **Files aren't encrypted.** Documents, settings and recovery copies are stored as plain files. Use disk encryption for sensitive work.
- **Browser version.** Your text is stored in the browser's local storage on that computer. Don't use it on a shared computer for sensitive text.
- **Very large or malformed files** can make mdedit slow or stop it responding.

## Support

Security updates are free until the end of the support period: see
[support-period.md](support-period.md).

## Secure use

- **Install** from the official source only: <https://github.com/rdoetjes/mdedit/releases>.
  **Check the download first** with `gh attestation verify FILE --repo rdoetjes/mdedit`,
  or against `SHA256SUMS` in the release. mdedit isn't signed with an Apple or Microsoft
  certificate, so the system asks once whether to allow it. Allow it only after the check
  passed: on macOS, System Settings → Privacy & Security → Open Anyway; on Windows,
  More info → Run anyway.
- **Throughout use:** keep mdedit, your browser and your operating system up to date.
- **Changes:** a build you changed yourself, including its dependencies, loses the assurances here. Changes can affect how your data is handled.

## Updates

- **Browser version:** updated by the site; reload the page.
- **Desktop:** at start-up mdedit asks GitHub whether there is a newer version. If so, a
  dialog offers its release page; **Later** postpones it to the next start. Download and
  check the new version as under Install, and replace the old one. Help → Check for
  Updates checks at any time. Help → **Don't Check for Updates at Start** turns the check
  off; it reveals your IP address to GitHub, nothing else. Without internet, the check
  is skipped.

## Removing mdedit and your data

Delete the program, then the folders below. Your documents stay where you saved them.

| Data | Linux, FreeBSD | macOS | Windows |
|---|---|---|---|
| Settings and dictionaries | `~/.config/mdedit` | `~/Library/Application Support/mdedit` | `%AppData%\mdedit` |
| Recovery copies, YouTube stills | `~/.cache/mdedit` | `~/Library/Caches/mdedit` | `%LocalAppData%\mdedit` |

Deleting the settings folder also resets mdedit to its original state. The location
follows `XDG_CONFIG_HOME` and `XDG_CACHE_HOME` if set. Browser version: clear this
site's data in the browser's settings.

## Integrators

mdedit isn't intended for integration into other products.

## Software bill of materials

Published in the repository under [docs/cra/sbom/](sbom/).

## Declaration of conformity

A copy comes with mdedit, as `EU-declaration-of-conformity.md` next to `LICENSE`
(Help → About says where). [FILL: remove this note until the declaration is signed]
