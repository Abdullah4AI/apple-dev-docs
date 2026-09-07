# Xcode Cloud status ID alias

> Status: the `--id` alias described below was removed in 5.0.0; `asc xcode-cloud status` accepts only `--run-id`. See `migrate-to-5-0.mdx`.

## Decision

`asc xcode-cloud status` keeps `--run-id` as its canonical build-run selector
and accepts `--id` as a deprecated compatibility alias. The alias is visible in
command help during the migration window because repeated agent invocations show
that callers infer it from nearby Xcode Cloud commands.

The alias is normalized before required-input checks, authentication, or HTTP.
Supplying both spellings is a usage error even when the values match. Conflict
diagnostics and telemetry name only the canonical `--run-id` parameter.

## Compatibility and migration

Existing `--run-id` invocations and JSON output are unchanged. An invocation
that uses only `--id` performs the same request and emits one warning on stderr:

```text
Warning: `--id` is deprecated. Use `--run-id`.
```

New scripts should use `--run-id`. The alias may be removed in the next major
release after the standard deprecation window. The migration is recorded in
[`release-notes-xcode-cloud-status-id-alias.md`](../release-notes-xcode-cloud-status-id-alias.md).

## Verification

Tests cover visible deprecated help, alias-only JSON output, the exact request
path, dual-spelling rejection, exit code 2, canonical telemetry, and the
no-request conflict path.
