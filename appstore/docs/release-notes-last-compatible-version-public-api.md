# Last-compatible version settings move to the public API

App Store Connect's Last-Compatible Version Settings screen is the nullable
`downloadable` attribute on `appStoreVersions`. Both directions are now on the
public API path.

Read it with the versions JSON output, which preserves the attribute whenever
Apple returns it:

```bash
asc versions list --app "APP_ID" --paginate --output json
asc versions view --version-id "VERSION_ID" --output json
```

Write it with a new tri-state `--downloadable` flag on `asc versions update`:

```bash
asc versions update --version-id "VERSION_ID" --downloadable true
asc versions update --version-id "VERSION_ID" --downloadable false --confirm
```

Leaving `--downloadable` unset sends no `downloadable` attribute, so an
unrelated `asc versions update` never changes download availability. Setting it
to `false` makes a previously released version unavailable for download on
older operating systems and devices and is not reversible from every state, so
that direction requires `--confirm`:

```text
Error: --confirm is required with --downloadable false because making a released version unavailable for download is not reversible from every state. No request was sent.
```

`--confirm` is rejected in any other combination rather than silently ignored,
including an explicit `--confirm=false`:

```text
Error: --confirm applies only to --downloadable false; remove it or pass --downloadable false
```

`--downloadable false --confirm=false` is not confirmation either, so the write
is still refused.

`asc versions update --output json` now also echoes `downloadable` in its
receipt when Apple returns the attribute, matching `asc versions view`.

Use `--paginate` on the list read. The versions this setting usually targets are
the oldest ones, which fall past the first page for apps with a long release
history.

The short-lived web-session read `asc web apps last-compatible-version view` is
removed. It shipped after `4.11.0` and was never part of a tagged release, so it
never reached the `stable` rung of the command lifecycle and is removed directly
rather than deprecated. It was read-only, required an interactive web session,
and returned the same `downloadable` attribute the public API now exposes.

Migration:

| Retired | Replacement |
| --- | --- |
| `asc web apps last-compatible-version view --app APP_ID` | `asc versions list --app APP_ID --paginate --output json` |
| (no write existed) | `asc versions update --version-id VERSION_ID --downloadable true` |
| (no write existed) | `asc versions update --version-id VERSION_ID --downloadable false --confirm` |

The `capabilities` inventory row for this feature moves from `partial` to
`cli-supported` and no longer lists a web-session command.
