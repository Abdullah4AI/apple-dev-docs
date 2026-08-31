# Design: install a local IPA on a connected iOS device

Issue: #2235

## Placement and current surface

The feature belongs under the existing `asc xcode` local-tooling command
group. The current group exposes build, archive, export, export-options,
validate, and version helpers; it has no command that changes a connected
device. The new leaf is therefore `asc xcode install` and is visible in help on
all platforms, while execution is explicitly macOS-only.

The command accepts an IPA and one exact CoreDevice identifier:

```text
asc xcode install --ipa ./App.ipa --device-id COREDEVICE_IDENTIFIER --timeout 5m --output json
```

The Xcode-provided `devicectl device install app` contract requires `--device`
plus a path whose extension is `.app`; it does not accept an enclosing IPA.
`devicectl list devices` returns the CoreDevice identifier as the JSON
`result.devices[].identifier` field. `devicectl device info apps` supports exact
`--bundle-id` filtering and is used after install for verification. The command
resolves `devicectl` from the active Xcode developer directory rather than
assuming the default Command Line Tools selection.

There is no App Store Connect API endpoint in this flow. It is a local,
read-only artifact inspection followed by a local device mutation through the
Xcode-provided tool.

## Execution design

1. Validate `--ipa`, `--device-id`, `--timeout`, positional arguments, and
   output format before side effects. Open the selected path with a rooted,
   no-follow file operation and require a regular, single-link file owned by
   the current user.
2. Snapshot the open IPA into a private mode-0600 temporary file. Run the
   existing `distribution.InspectIPAContext` validation against that snapshot
   with device membership included. Continue only for an iOS IPA whose
   non-expired development or ad-hoc profile is internally consistent, has at
   least one provisioned device, and has a verified complete main-app code
   signature. Installation applies its own explicit profile and main-app
   requirements; generic preparation warnings for metadata or embedded targets
   do not independently block a valid install.
3. Resolve `devicectl` through `/usr/bin/xcrun --find devicectl` using the
   sanitized child environment. Run `devicectl list devices` into an
   unpredictable, mode-0600 JSON file inside a mode-0700 temporary directory.
   Parse only the supported structured schema and find exactly one physical,
   paired, usable iOS device whose `identifier` equals `--device-id`. Never
   select by name, prefix, fuzzy match, or list order. If a hardware UDID is
   present, require that it belongs to the inspected provisioning profile;
   absence of a usable UDID fails closed.
4. Re-open the same private snapshot and materialize only its one inspected
   `Payload/<app>.app` subtree into a fresh unpredictable mode-0700 temporary
   directory. Reuse the existing archive path, encryption, symlink, regular
   member, duplicate/collision, entry-count, declared-expansion, and streamed
   expansion limits. Create output files with rooted no-follow operations and
   preserve executable permission bits. The materialized bytes are derived
   from the same immutable snapshot and digest used for inspection; the input
   IPA is never changed.
5. Invoke only `devicectl device install app --device <resolved identifier>
   <temporary app path>` with bounded diagnostics and structured JSON output.
   Parse the command type, one of the supported JSON schema versions 4 or 5,
   outcome, target device, and installed bundle identity strictly. A documented
   v5 `_deprecationNotice` is accepted and ignored. Future versions and unknown
   fields remain rejected. Do not expose raw tool output or device identifiers
   in errors or command output.
6. On a successful install, run the existing exact-bundle app observation
   shape (`device info apps --bundle-id`) and require the installed bundle's
   version and build to match the inspected IPA. A tool success without this
   proof is reported as `installed=true, verified=false` and remains nonzero.
   All temporary directories and files are removed on every return path.

## Output, streams, and exits

The computed result is an exported camelCase `asc.XcodeInstallResult` with
`schemaVersion`, `operation: "xcode.install"`, `success`, `installed`,
`verified`, nested `ipa` artifact identity, an optional nested `device` with a
one-way identifier digest, platform, pairing/connection state, and duration.
Operational failures additionally carry only closed `failureStage` and
`failureCode` values. It contains no raw device identifier, UDID, selector,
source path, temporary path, tool JSON, or profile device list. JSON is the
stable machine-readable contract; table and Markdown renderers expose the same
redacted fields. Data goes to stdout and diagnostics go to stderr.

Exit status is 0 only after post-install verification succeeds. Deterministic
usage errors return 2. The default timeout is 5m and accepted values range
from 5s through 10m. Platform, artifact, device, tool, timeout, structured
output, and post-install verification failures return 1. No interactive
prompts, App Store Connect credentials, keychain access, network calls, or
persistent device state are introduced beyond the requested install.

## RED-GREEN validation

Start with command-level tests for required flags, platform gating, output
shape, stdout/stderr separation, and usage classification. Add core tests for
IPA materialization, path containment, mode preservation, exact device
selection, profile membership, strict devicectl schemas, install argument
construction, timeout handling, post-install mismatch, and cleanup. Use fake
devicectl runners and generated signed/fixture IPAs; never require a device in
unit tests. Run the built binary for help and deterministic usage cases. A
live smoke test may be run only when an actual connected device and a valid
installable IPA are available; otherwise the limitation is reported rather
than bypassed.

## Compatibility and alternatives

This is additive: no existing command or flag changes, and no stable behavior
is removed or deprecated. Existing connected-device observation remains
unchanged. A future implementation may add a user-facing device-name lookup,
but this first slice deliberately requires the exact CoreDevice identifier to
avoid ambiguity and privacy leakage.

An alternative is to pass the IPA directly to `devicectl`; its install contract
requires an app bundle, so that would fail for valid IPA inputs. Another is to
use a shell extractor or a broad temporary tree; that would duplicate or
weaken the repository's bounded archive and rooted filesystem guarantees. A
single immutable snapshot plus bounded rooted extraction keeps inspection,
installation, and privacy guarantees aligned.
