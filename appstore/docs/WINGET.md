# WinGet Packaging

This document captures the Windows Package Manager packaging flow from
[GitHub Discussion #1552](https://github.com/rorkai/App-Store-Connect-CLI/discussions/1552).

The goal is to make the common install command work:

```powershell
winget install asc
```

Keep the exact package identifier documented as the scripting fallback:

```powershell
winget install --id Rorkai.ASC --exact
```

## Package Identity

Use this identity for `microsoft/winget-pkgs` submissions:

```yaml
PackageIdentifier: Rorkai.ASC
PackageName: asc
Moniker: asc
Commands:
  - asc
```

`Moniker: asc` is what lets `winget install asc` resolve to this package when
there is no ambiguity. `PackageIdentifier: Rorkai.ASC` remains the stable,
non-ambiguous form for automation.

## First Submission

WinGet packages live in the public
[`microsoft/winget-pkgs`](https://github.com/microsoft/winget-pkgs) repository,
not in this repository. For the initial submission, create a portable package
from the Windows release asset:

```yaml
PackageIdentifier: Rorkai.ASC
PackageVersion: 1.5.1
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.6.0
```

```yaml
PackageIdentifier: Rorkai.ASC
PackageVersion: 1.5.1
PackageLocale: en-US
Publisher: Rorkai
PackageName: asc
PackageUrl: https://github.com/rorkai/App-Store-Connect-CLI
License: MIT
LicenseUrl: https://github.com/rorkai/App-Store-Connect-CLI/blob/main/LICENSE
ShortDescription: Fast, scriptable CLI for the App Store Connect API.
Moniker: asc
Tags:
  - app-store-connect
  - app-store-connect-api
  - cli
  - ios
  - testflight
ReleaseNotesUrl: https://github.com/rorkai/App-Store-Connect-CLI/releases/tag/1.5.1
ManifestType: defaultLocale
ManifestVersion: 1.6.0
```

```yaml
PackageIdentifier: Rorkai.ASC
PackageVersion: 1.5.1
InstallerType: portable
Commands:
  - asc
Installers:
  - Architecture: x64
    InstallerUrl: https://github.com/rorkai/App-Store-Connect-CLI/releases/download/1.5.1/asc_1.5.1_windows_amd64.exe
    InstallerSha256: REPLACE_WITH_RELEASE_SHA256
ManifestType: installer
ManifestVersion: 1.6.0
```

The release workflow generates these files with
`scripts/generate_winget_manifests.py` using the SHA256 from
`asc_<version>_checksums.txt`.

## Release Automation

`.github/workflows/release.yml` generates the WinGet manifests after the GitHub
release is published, in parallel with the Homebrew tap update. It then opens or
updates a PR against `microsoft/winget-pkgs`.

Required release configuration:

- `secrets.WINGET_GITHUB_TOKEN`: token that can create/use a fork of
  `microsoft/winget-pkgs`, push branches to that fork, and open PRs against
  `microsoft/winget-pkgs`.
- `vars.WINGET_FORK_OWNER`: optional organization/user that owns the fork. When
  unset, the workflow uses the authenticated token owner.

The submission step logs the token's `gh api rate_limit` quota first, then runs
every `gh` and `git` network call through a bounded retry helper (five attempts,
exponential backoff from 10s capped at 120s, with jitter). Only GitHub rate
limits and transient transport failures are retried; authentication,
permission, validation, and scope-check failures fail immediately. Clone and PR
creation are idempotent under retry, so a lost response cannot open a duplicate
`microsoft/winget-pkgs` PR.

If the `winget` job still fails after the retries, re-run that job (or dispatch
the release workflow for the same version). If the preflight showed a primary
quota (`core`, `graphql`, or `search`) at zero, wait for that bucket's logged
reset time first; for secondary throttles, 5xx responses, or transport
failures the reset times are unrelated, so re-run after a few minutes. The job
is safe to re-run: it reuses the published release artifact,
recognizes an existing `rorkai-asc-<version>` branch, and skips PR creation
when one is already open. Fall back to a manual submission only when a re-run
fails for a non-transient reason.

The generated branch name is:

```text
rorkai-asc-<version>
```

The manifest path in `winget-pkgs` is:

```text
manifests/r/Rorkai/ASC/<version>/
```

To generate manifests locally for a release asset directory:

```bash
python3 scripts/generate_winget_manifests.py \
  --version 1.5.1 \
  --release-dir release \
  --output-dir /tmp/asc-winget
```

## Acceptance Check

After the manifest PR is merged and WinGet sources update, run the manual
`WinGet smoke` workflow or use a clean Windows machine:

```powershell
winget source update
winget search asc --source winget
winget install asc --source winget --accept-source-agreements --accept-package-agreements
asc --help
asc version
winget uninstall --id Rorkai.ASC --exact
```

If `winget install asc` prompts for disambiguation, keep the README fallback as
the documented automation-safe command:

```powershell
winget install --id Rorkai.ASC --exact
```
