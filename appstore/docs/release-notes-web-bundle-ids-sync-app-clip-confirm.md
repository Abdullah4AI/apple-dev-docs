# Web bundle-ids sync-app-clip confirmation and idempotency

`asc web bundle-ids capabilities sync-app-clip` now requires `--confirm`, in
line with every other `asc web` mutation. Syncing rewrites the App Clip Bundle
ID capability graph, and Apple invalidates existing provisioning profiles that
contain the changed App ID.

```bash
asc web bundle-ids capabilities sync-app-clip \
  --bundle-id "CLIP_BUNDLE_ID" \
  --parent-bundle-id "PARENT_BUNDLE_ID" \
  --capability "PUSH_NOTIFICATIONS" \
  --confirm
```

The previous invocation shape still parses, but it fails with usage exit code
`2` before a web session is resolved or any request is sent, and prints this
migration guidance to stderr:

```text
Warning: web bundle-ids capabilities sync-app-clip now requires --confirm because syncing rewrites the App Clip Bundle ID capability graph and invalidates existing provisioning profiles; re-run with --confirm to acknowledge. No request was sent.
Error: --confirm is required
```

Because a no-op write still invalidates profiles, the command is now
idempotent. It reads the current capability graph first and, when the
capability is already enabled with the requested `parentBundleId` (and the
requested settings, when `--settings-json` is passed), it sends no PATCH and
reports `changed: false` with status `already-synced`. When it does write, it
preserves every unrelated capability and relationship (including to-many
relationships such as App Groups and iCloud containers, which previously failed
to parse), sends only the writable `enabled` and `settings` capability
attributes, warns on stderr that existing provisioning profiles are now
invalid, and caches any refreshed session cookies.

The JSON receipt gains two additive fields, `changed` and `status`
(`synced` or `already-synced`); the table and Markdown outputs gain matching
`Changed` and `Status` columns.
