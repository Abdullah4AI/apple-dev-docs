# Structured Xcode version editor

## Placement and current behavior

This change stays under the existing `asc xcode version` command group. No new
top-level command is added and the registry does not change.

Today, `view`, `edit`, and `bump` require macOS and Apple Generic Versioning.
Resolved reads use `agvtool` and, for modern projects, `xcodebuild
-showBuildSettings`. Modern writes scan `project.pbxproj` line by line and
replace every line beginning with `MARKETING_VERSION =` or
`CURRENT_PROJECT_VERSION =`. Consequently writes are project-wide, depend on
formatting, cannot update xcconfig-backed settings, and rewrite the file with
mode `0600`.

## Public command shape

The existing invocations remain valid. The following flags are added:

```text
asc xcode version view \
  [--target NAME] [--configuration NAME]

asc xcode version edit \
  [--version VER] [--build-number NUM] \
  [--target NAME] [--configuration NAME]

asc xcode version bump --type major|minor|patch|build \
  [--target NAME] [--configuration NAME]
```

`--target` and `--configuration` select the exact target/configuration pair.
Supplying neither retains project-wide edit behavior. Supplying only a target
updates all configurations for that target. Supplying only a configuration
updates that configuration across the project and all targets.

Remote-safe build numbers are available without a shell pipeline:

```text
asc xcode version edit --next-build-number --app APP \
  [--version VER] [--platform PLATFORM] [--initial-build-number N]

asc xcode version bump --type build --next-build-number --app APP \
  [--platform PLATFORM] [--initial-build-number N]
```

`--next-build-number` is mutually exclusive with `--build-number`, requires
`--app` or `ASC_APP_ID`, and is accepted by `bump` only with `--type build`. The selected local
marketing version is the remote version filter unless `edit --version` is
provided. Remote selection reuses the same processed-build and in-flight-upload
logic as `asc builds next-build-number`. When the version is inferred, every
configuration selected for mutation must resolve the same marketing version;
otherwise the command fails before the App Store Connect lookup.

The remote flags are `--app`, `--platform`, `--processing-state`,
`--exclude-expired`, and `--initial-build-number`. They have the same meanings
and validation as the canonical builds command.

No command prompts. Data remains on stdout and errors remain on stderr. Usage
errors return exit code 2.

## Structured output

JSON remains backward compatible: the existing top-level version/build fields
remain. Mutation results add:

- `target` and `configuration` when scoped;
- `changedFiles`, a stable sorted list of paths;
- `changes`, a stable list containing setting, old value, new value, target,
  configuration, path, and source (`pbxproj` or `xcconfig`).

View results add source paths for the selected values. Table and Markdown output
keep the existing version/build presentation and add scope/source information
only when present.

## Implementation

`github.com/bitrise-io/go-xcode/xcodeproject/xcodeproj` at `v1.3.3` (MIT)
parses the pbxproj object graph. The editor walks project and target configuration lists
instead of scanning lines. Existing settings are mutated in place; settings are
not invented in unrelated layers.

An internal lossless xcconfig scanner handles assignments, line and block
comments, CRLF/LF endings, conditional keys, `#include`, and `#include?`.
Includes resolve relative to the containing file, are cycle-safe, and may be
shared by configurations. Unrelated bytes remain unchanged. Existing quote
delimiters are preserved; `+=` and `?=` are normalized to `=` when editing a
version setting so the requested effective value is guaranteed. All matching
variants of `MARKETING_VERSION` or `CURRENT_PROJECT_VERSION` in the selected
include graph are updated.

Unreadable xcconfig graphs fail when the selected scope depends on them. A
broken graph on an unrelated configuration does not block a direct pbxproj
view or edit; mutation stays conservative when that graph makes shared-file
consumer discovery uncertain.

Reads resolve direct pbxproj settings, registered xcconfig values, and simple
build-setting references locally, including `$(inherited)` across the next
lower target xcconfig or project layer. Target xcconfig inheritance is seeded
from the matching project configuration, matching Xcode's layer order. Unresolved build-system variables
produce an explicit error rather than an invented value; callers can use Xcode
when SDK-specific resolution is required. Conditional-only values are editable,
but cannot be used as a view or bump baseline without an unconditional value.
Projects that define only one of the two structured version settings retain the
macOS/agvtool fallback, as do legacy projects with versions stored only in
Info.plist. An unscoped remote build bump remains available on that fallback by
passing the resolved number to `agvtool new-version -all`; scoped legacy bumps
are rejected before any project-wide write. Legacy fallback requires a
discoverable, parseable Xcode project that lacks the structured settings;
missing, unreadable, and ambiguous project paths remain discovery errors.

Every output file is prepared and validated before mutation. Writes use a
same-directory temporary file, preserve the original mode, `fsync`, and rename.
If a later file fails, already-written files are restored from their captured
original bytes. The staged pbxproj is reparsed before commit.
Mutation values containing comment syntax or build-setting expressions are
rejected before staging so reported and subsequently resolved values cannot
diverge. Each selected leaf configuration must either resolve the requested
value or schedule an actual mutation; unresolved protected settings cannot
return a false-success result.

## Compatibility and lifecycle

This is a stable-command implementation improvement plus additive flags and
JSON fields. Existing project-wide invocations remain project-wide. Existing
human-readable output remains recognizable. No deprecation is required.

Modern pbxproj/xcconfig operations become cross-platform. Operations that need
Xcode's build-system resolution or legacy agvtool behavior still fail with a
specific macOS/Xcode requirement.

## RED-GREEN and verification

Characterization and regression coverage will include:

- current project-wide behavior with no scope flags;
- target-only, configuration-only, and target-plus-configuration writes;
- project-level and target-level settings;
- settings inherited from one or multiple recursive xcconfig includes;
- optional/missing includes, include cycles, shared includes, comments, quoted
  values, assignment operators, conditional keys, inherited values, CRLF, no
  final newline, and unchanged byte regions;
- malformed pbxproj/xcconfig, missing settings, ambiguous targets/configurations,
  multiple application targets, conditional-only baselines, partially migrated
  projects, unsafe values, symlinks, permissions, write failures, rollback, and
  atomic replacement;
- cross-platform view/edit/bump without `agvtool` for modern projects;
- remote next-build-number validation, API errors, in-flight uploads, and JSON;
- Xcode 26 and Xcode 27 project copies, reparsed and checked with
  `xcodebuild -list`/`-showBuildSettings` after mutations;
- focused unit/CLI tests, a built `/tmp/asc` black-box check, then the full
  format/docs/lint/test gate.

## Alternatives

Keeping the line editor is smaller but cannot safely scope or understand
xcconfig inheritance. Porting `rork-xcode` would provide a richer model but
would create a new parser maintenance burden. Calling Fastlane or Ruby's
`xcodeproj` violates the single-binary/no-runtime-dependency contract. The
Bitrise parser is already used in production Go tooling and provides the
smallest credible structured base; the repository-owned xcconfig and atomic
write layers cover the behavior it lacks.
