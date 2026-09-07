# Local notarization ticket stapling and validation

## Placement and current behavior

This change extends the existing `asc notarization` command group with two
local, macOS-only artifact operations. The current 4.11.0 command surface is
`submit`, `status`, `log`, and `list`; those commands use the Notary API and
remain unchanged. No App Store Connect endpoint or request schema is involved
in this change.

The new invocations are:

```text
asc notarization staple --file PATH --confirm [--output FORMAT] [--pretty]
asc notarization validate --file PATH [--output FORMAT] [--pretty]
```

`staple` calls Apple's local `xcrun stapler staple PATH` operation and then
calls `xcrun stapler validate PATH` before returning success. `validate` calls
only the validation operation and never mutates the artifact. Apple supports
UDIF disk images, signed flat installer packages, and supported code-signed
bundles. A ZIP is intentionally rejected as a direct target because it must be
recreated after stapling its contained item.

## Design

The command layer validates the invocation and target before any tool or auth
work. Because stapling mutates the artifact in place, `staple` requires an
explicit `--confirm` flag before it inspects the target or invokes Apple's
tool. Trimming is used only to detect an empty flag, so leading and trailing
whitespace in a real filename is preserved; the supplied path is then cleaned
once and resolved to an absolute path. It rejects NULs, missing paths, final
symlinks, unsafe parent symlinks, special files, and empty regular files, and
accepts regular files or directory bundles for Apple's tool to classify. The
final component's kind is classified by a rooted no-follow `Lstat` probe, so
only a proven regular file falls back to the regular-file path; a probed
special file is rejected as invalid input, while traversal and open failures
stay operational instead of being retried as files. No classification decision
reads the text of an open error, so an artifact whose own pathname contains a
diagnostic phrase cannot be misreported as invalid input.

The probe establishes every semantic precondition, which fixes where the
usage/runtime boundary sits. Missing paths, symlinks, special files, direct
`.zip` targets, and empty files are operator input errors. Once the probe has
succeeded, a disagreement at the open that follows it — including `ENOENT`, a
symlink appearing at the final component, a kind flip, or a different object
of the same kind — is a replacement race, so it keeps runtime classification
and its sanitized diagnostic rather than being reported as invalid input with
the artifact path attached.

The opened artifact descriptor is retained for the whole operation instead of
being closed after its metadata is recorded. Holding it keeps the artifact's
inode allocated, so an attacker who unlinks the target cannot have a
replacement receive the recycled file ID and then satisfy `os.SameFile`
against the recorded identity. The retained `rootfs.Root` pins only the
filesystem root, which is why the artifact handle is pinned separately.

Direct `.zip` paths fail with a usage diagnostic. Parent and final checks use
the existing no-follow/rooted
filesystem helpers. Stable macOS `/etc`, `/tmp`, and `/var` filesystem aliases
are accepted at the volume boundary, while symlinks below the selected
artifact parent are rejected. The path is passed to the child as one argv
element and is never interpolated into a shell command.

The reusable local runner lives in `internal/xcode`. It requires the Darwin
platform, resolves `xcrun`, verifies that `xcrun --find stapler` succeeds, and
uses the existing command-construction and bounded wait seams. Child stdout and
stderr are directed to the caller's diagnostic writer so structured command
output remains parseable. Context cancellation is propagated through
`exec.CommandContext`; no API client or credential lookup is performed.

Both runner entry points accept an optional stage verifier that runs
immediately before and after every child process, after `xcrun --find stapler`
has resolved. The staple flow guards its staple and validation children; the
validation-only flow guards its single validation child. This closes the window
between tool resolution and the child, so a replacement that is still in place
when the child would run is rejected before the child starts, instead of
Apple's tool inspecting one artifact while the command reports another. A
replacement that is fully reverted before the pre-child check is not detectable
by the wrapper, and the run may still be reported as verified; in that case the
child observed the original artifact, so the reported state matches what was
inspected. A verifier failure is returned as a typed stage error that keeps the
caller's cause available for classification.

Directory bundles receive an additional bounded content binding at the command
boundary. After a successful staple child and before the follow-up validation
child, the command captures a private inventory of the bundle; validation-only
captures one immediately before its child. The inventory is a deterministic
SHA-256 over every relative entry's kind, normalized relative path, permission
mode, size, and file SHA-256, together with total byte and entry counts. The
scanner is rooted at the already pinned filesystem root, opens directories and
files without following final symlinks. Contained relative symlinks are
accepted and recorded as link entries using their raw target length and a
SHA-256 of the target text; absolute or lexically escaping links are rejected,
and no link target is followed by the scanner. Special files remain rejected.
Every entry is rechecked for identity, size, mode, and (for links) unchanged
target text after it is read. It checks the active context between entries and
reads and fails closed at bounded path, entry, and total-byte limits. A second
inventory is required after validation; any mismatch is an unverified artifact
change and cannot produce success. The inventory is comparison evidence only:
paths, names, digests, and raw scanner causes never enter public output or
telemetry. Extended attributes and filesystem ACLs remain outside this content
digest and are not represented as unchanged by it.

Regular-file targets receive the same bounded byte binding at each stage
boundary. The command captures a size and SHA-256 fingerprint after stapling
and compares it before and after the follow-up validation; validate-only
captures before validation and compares after. A same-inode rewrite therefore
cannot pass only because its outer device/inode identity was retained. Relative
paths retain an open descriptor for the physical current directory while the
operation runs and revalidate that directory identity before each stage, so a
cwd rename/replacement is rejected before a child receives the old pathname.

A stage boundary distinguishes a proven replacement from a boundary that could
not be evaluated. A `SameFile` mismatch, a vanished target, a kind flip, or a
replacement by a symlink is reported as a changed artifact and names the stage;
an operational reopen or stat failure such as a revoked permission, a
descriptor limit, or an I/O error is reported as the sanitized filesystem
diagnostic instead, because the corrective action differs. Neither outcome can
produce a success result.

When a stapling child exits non-zero, the runner preserves its status in a
typed error. The CLI converts a real child status to the repository's private
process-exit marker after writing one concise stage diagnostic. Because a
staple child that was started can fail after it has already written part of
the artifact, and the post-stage check recaptures the resulting state as the
next baseline rather than comparing it with the pre-staple evidence, every
failure from a started staple child is classified as a possible partial
mutation: the CLI names the failing stage, warns that the artifact may have
been modified without being verified, and still returns the child status. A
failure before the child starts stays an ordinary error. A successful staple
followed by a failed validation is reported specifically as an unverified
mutation and returns the validation child status. Lookup, platform, and
cancellation failures retain the ordinary generic runtime mapping, except that
a concrete resolver launch failure observed together with a late cancellation
keeps its resolution-stage classification alongside the context error. A start
or signal failure represented by the typed runner error keeps its underlying
cause for internal classification but emits only a stable stage diagnostic, so
an executable or temporary path from the operating system is not exposed.
Non-NotFound `xcrun` lookup failures use the same closed resolution-stage
diagnostic while retaining the lookup cause internally. If a staple child has
already completed successfully but its diagnostic writer fails, the runner
returns the writer/process cause as a partial-mutation failure and does not
claim an unverified success result.

Successful computed output is represented by exported structs in
`internal/asc/output_notary.go` and registered with the normal output registry:

```json
{
  "filePath": "/absolute/path/MyApp.dmg",
  "operation": "staple",
  "stapled": true,
  "validated": true
}
```

```json
{
  "filePath": "/absolute/path/MyApp.dmg",
  "operation": "validate",
  "validated": true
}
```

JSON is written to stdout; table and Markdown render the same stage state.
Progress, child diagnostics, and corrective guidance are written to stderr.
Failed operations never emit a success result.

## Compatibility and scope

This is additive behavior. Existing Notary API commands, polling, output
shapes, authentication, and telemetry remain unchanged. The new commands do
not extract ZIPs, submit artifacts, re-sign, package, upload, un-staple, or run
Gatekeeper policy assessment. Apple's stapler may require network access for
both operations, but the CLI itself does not resolve App Store Connect auth.

## RED-GREEN and verification

Tests begin with CLI usage and output cases for required/empty `--file`,
missing `--confirm` (including the usage exit and no target/tool work),
positional arguments, direct ZIP rejection, invalid output/pretty combinations,
help discoverability, JSON/table/Markdown rendering, and unchanged existing
commands. Runner tests cover exact argv, path preservation, tool resolution,
stdout/stderr routing, child status preservation, staple-then-validate
ordering, validation-only behavior, cancellation, unsupported hosts, and
missing tools. Filesystem tests cover final and parent symlinks, missing,
special, empty, regular-file, and directory-bundle targets. Directory-bundle
tests additionally cover nested replacement before and after validation, stable
inventories, symlinks, special files, cancellation during scanning, and
entry/byte/path bounds. Signal termination after the staple child starts is
classified as a partial mutation and retains its internal process cause while
emitting only the unverified warning.

After focused tests pass, verify a built binary's help, output streams, and
usage status. Run `make build`, `make format`, `make generate-command-docs`,
`make check-docs`, `make lint`, and `ASC_BYPASS_KEYCHAIN=1 make test`. On macOS,
use a disposable copy of an existing Accepted, Developer ID-signed artifact to
run staple followed by validate; do not create a new notarization submission.

## Alternatives considered

Keeping the operations only in the API-facing command would leave the final
distributed artifact unverifiable. Calling `stapler` through a shell would
make paths with spaces or shell metacharacters unsafe. Reusing the Notary API
client would add credentials and server behavior to a purely local operation.
The direct `xcrun` argv runner keeps the boundary narrow while reusing the
repository's existing macOS command and process-test seams.

## Release and push requirements

The implementation and generated command documentation must be committed and
pushed as additive changes; existing commits in the feature branch must not be
rewritten. Before release, rerun the focused tests and the repository's build,
format, documentation, lint, and full test gates. Record the exact release
commit and pushed branch in the handoff. A macOS smoke test with an existing
Accepted artifact is required when such a disposable artifact is available.

## Unresolved risks

The automated suite uses deterministic command seams and cannot prove Apple's
ticket service behavior or the exact stapler result for every supported bundle,
disk image, and package format. No rollback is possible if Apple completes an
in-place staple before cancellation or if the follow-up validation fails; the
command reports that state but does not restore the artifact. ZIP extraction,
repackaging, and Gatekeeper assessment remain outside this change and require
separate workflows.

The wrapper keeps the validated target identity anchored through each local
stage, but Apple's stapler accepts a pathname rather than the wrapper's open
descriptor. A concurrent replacement can still occur after an identity check
and before or during the child process; the wrapper detects replacements at the
stage boundaries and the bounded directory/regular-file bindings detect
content changes between those boundaries, but neither can eliminate the
provider/path-based TOCTOU window during the child itself. Regular-file
fingerprints are comparison evidence and are not a rollback or atomicity
guarantee.
