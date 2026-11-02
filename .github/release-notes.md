## Downloads

| File | For |
|---|---|
| `mdedit-*-linux-amd64.tar.gz` | Linux (x86-64) |
| `mdedit-*-macos-arm64.zip` | macOS on Apple Silicon |
| `mdedit-*-windows-amd64.zip` | Windows (x86-64) |
| `mdedit-*-web.tar.gz` | The browser version, for your own web server |
| `mdedit-*.cdx.json` | Software bill of materials (CycloneDX) |
| `SHA256SUMS` | Checksums of all files |

## First start on macOS and Windows

mdedit isn't signed with an Apple or Microsoft certificate, so the system asks once
whether to allow it.

- **macOS:** unzip and open MDEdit. macOS says it can't check it: click Done. Then open
  **System Settings → Privacy & Security**, scroll down, and click **Open Anyway** next to
  MDEdit. From then on it opens normally.
- **Windows:** unzip and start `mdedit.exe`. If "Windows protected your PC" appears, click
  **More info → Run anyway**.

## Check your download

Every file is signed by GitHub when it is built here. With the GitHub CLI:

```
gh attestation verify FILE --repo rdoetjes/mdedit
```

or compare it with `SHA256SUMS` (`sha256sum -c SHA256SUMS`, or `shasum -a 256` on macOS).
Only allow mdedit to run if the check passes.
