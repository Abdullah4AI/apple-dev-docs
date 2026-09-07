# Canonical `--build-id` selector

> Status: the hidden `--build` compatibility alias described below was removed
> in 5.0.0 on every command in the table (and on `asc versions attach-build`).
> `--build` is now an unknown flag; use `--build-id`. See `migrate-to-5-0.mdx`.

Release 4.7.0 standardizes the build-ID selector across the remaining command
paths that previously used `--build`. New automation should use
`--build-id`; it is now the documented flag and appears in command help.

The affected commands are:

| Command | Build-ID use |
| --- | --- |
| `asc review submit` | Build to attach to the review submission |
| `asc publish testflight` | Existing build to distribute |
| `asc validate testflight` | Build to validate |
| `asc release stage` | Build to attach during staging |
| `asc build-localizations list` | Build whose localizations are listed |
| `asc build-localizations create` | Build whose localization is created |
| `asc build-bundles list` | Build whose bundles are listed |
| `asc testflight crashes list` | Build-ID filter |
| `asc testflight feedback list` | Build-ID filter |
| `asc performance metrics view` | Build whose metrics are viewed |
| `asc performance diagnostics list` | Build-ID filter |
| `asc performance download` | Build whose diagnostics are downloaded |
| `asc encryption declarations list` | Build-ID filter |
| `asc encryption declarations assign-builds` | Build IDs to assign |
| `asc apps app-encryption-declarations list` | Build-ID filter |

## Compatibility window

Existing scripts may continue to pass `--build` during the 4.x deprecation
window. The flag is hidden from help, retains its existing behavior, and emits
one migration warning on stderr pointing to `--build-id`; stdout and API
semantics remain unchanged. Supplying both spellings with different values is
a usage error before any API side effect, while identical values continue with
the deprecation warning.

Migrate automation to `--build-id` before the next major release, 5.0.0,
where deprecated compatibility flags may be removed under the CLI stability
ladder.
