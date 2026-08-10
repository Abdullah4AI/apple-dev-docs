# Xcode export signing passthrough

## Placement and current behavior

`asc xcode export` and local-build `asc publish testflight` / `asc publish
appstore` already generate App Store Connect export options from an archive when
no plist is supplied. Those implicit paths currently hard-code automatic
signing even though `asc xcode export-options generate` supports the shared
`automatic|manual` signing-style contract and an optional team override.

This change extends the existing commands; it does not add a registry entry or
an App Store Connect API operation. The offline OpenAPI snapshot has no bearing
on local `xcodebuild -exportArchive` option construction.

## Public command shape

The three archive-derived generation paths accept:

```text
asc xcode export ... [--signing-style automatic|manual] [--team-id TEAM_ID]
asc publish testflight ... [--signing-style automatic|manual] [--team-id TEAM_ID]
asc publish appstore ... [--signing-style automatic|manual] [--team-id TEAM_ID]
```

Both flags configure generated export options only. `automatic` remains the
default. `--team-id` is optional for either style and overrides team metadata
read from the archive. Manual generation may infer the team from the archive
and uses the existing local certificate/profile matcher; no profile UUID flag
is added.

The signing-style enum is normalized and validated by the same
`internal/xcode` helper used by `export-options generate`. Invalid values are
usage errors before Xcode, filesystem generation, authentication, or network
work.

## Precedence and compatibility

An explicit `--export-options` plist remains authoritative, but combining it
with an explicitly set `--signing-style` or `--team-id` is rejected as a usage
error instead of silently ignoring generation flags.

Local publish keeps its existing conventional-plist fallback:

1. explicit `--export-options`, with no generation flags;
2. generated options when either generation flag is explicitly set;
3. existing `.asc/export-options-app-store.plist`;
4. generated automatic options.

This preserves every existing invocation. It also makes a manual-signing
request override the implicit conventional-plist fallback without altering or
overwriting that file. All implicit generation continues to use a unique,
archive-adjacent output path. Publish always uses `destination=export`; `xcode
export --wait` continues to use `destination=upload`.

## Output compatibility

This change adds no signing-material fields to xcode export or publish output.
Those commands continue to report the export-options path they used. The
existing standalone export-options generator keeps its structured output
contract unchanged, including its certificate selector and profile mappings.
No command logs plist contents or adds output beyond its existing result shape.

## RED-GREEN and verification

Focused coverage establishes:

- help advertises both flags on xcode export and both publish commands;
- automatic remains the default and an explicit team reaches generation;
- manual style and team reach archive-derived generation unchanged;
- both local publish commands use the shared post-archive generation path;
- explicit-plist conflicts, invalid styles, and non-local publish use fail as
  usage errors before side effects;
- generation flags bypass, but never overwrite, a conventional publish plist;
- generated plist payloads retain automatic/manual behavior.

Verification uses focused package tests, a built CLI for help and early-error
stdout/stderr/exit behavior, and the repository format/docs/lint/full-test gate.
An actual archive export is attempted only when a disposable signed fixture is
available; this change neither uploads nor mutates App Store Connect state.

## Alternatives

Silently letting an explicit plist win would preserve precedence but would
accept ignored flags, violating the CLI contract. Adding provisioning-profile
flags would expose unstable per-target UUIDs and duplicate the existing local
matcher. Making manual signing the default would break automatic-signing users.
