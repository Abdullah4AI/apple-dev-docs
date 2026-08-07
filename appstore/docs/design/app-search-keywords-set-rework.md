# Rework the app search-keywords setter

## Placement and command shape

`asc apps search-keywords list` remains the low-level app-keyword read surface.
The released sibling spelling `asc apps search-keywords set` remains available,
but its write is routed through the supported App Store version-localization
resource instead of the app's read-only `searchKeywords` relationship.

The supported invocation is:

```bash
asc apps search-keywords set \
  --app "APP_ID" \
  --version "1.2.3" \
  --locale "en-US" \
  --platform IOS \
  --keywords "kw1,kw2" \
  --confirm
```

`--app`, `--version`, `--locale`, `--keywords`, and `--confirm` are required.
`--platform` is optional when the app has only one App Store version with that
version string. If the same version string exists on multiple platforms, the
command fails before mutation and asks for `--platform`.

On success the command writes the updated App Store version-localization
response to stdout. Usage failures, resolution failures, and API diagnostics go
to stderr through the normal CLI runner. Invalid or missing flags exit 2;
not-found, ambiguity, pagination, and API failures exit 1.

## API contract

The current OpenAPI defines this three-request flow:

1. `GET /v1/apps/{id}/appStoreVersions` with
   `filter[versionString]`, optional `filter[platform]`, and `limit=200` resolves
   the App Store version. Every response page is considered.
2. `GET /v1/appStoreVersions/{id}/appStoreVersionLocalizations` with
   `filter[locale]` and `limit=200` resolves exactly one existing localization.
   Every response page is considered.
3. `PATCH /v1/appStoreVersionLocalizations/{id}` sends an
   `AppStoreVersionLocalizationUpdateRequest` containing only
   `data.attributes.keywords`.

The update returns an `AppStoreVersionLocalizationResponse`. The keyword field
is a comma-separated string and is subject to App Store Connect's 100-character
limit.

Apple exposes only `GET` at `/v1/apps/{id}/relationships/searchKeywords` and
`GET /v1/apps/{id}/searchKeywords`. The original
`PATCH /v1/apps/{id}/relationships/searchKeywords` transport is not in the
OpenAPI and must not be restored. Localization-level `searchKeywords`
relationship POST and DELETE operations do exist, but they accept opaque
`appKeywords` linkage IDs rather than raw keyword text and are unrelated to
this setter.

## Maintained implementation evidence

Codemagic CLI Tools exposes `keywords` on App Store version
localization create and modify actions. Its maintained client sends keyword
text as a localization attribute, not as app-keyword relationship linkage.

Relevant current sources:

- [`app_store_version_localizations.py`](https://github.com/codemagic-ci-cd/cli-tools/blob/feb80b2d944923402e56b825f40c038a39f35b64/src/codemagic/apple/app_store_connect/versioning/app_store_version_localizations.py)
- [`app_store_version_localizations_action_group.py`](https://github.com/codemagic-ci-cd/cli-tools/blob/feb80b2d944923402e56b825f40c038a39f35b64/src/codemagic/tools/app_store_connect/action_groups/app_store_version_localizations_action_group.py)

These implementations agree with the local OpenAPI snapshot and the existing
`asc localizations update --version ... --locale ... --keywords ...` and
`asc metadata keywords ...` implementations.

## History and compatibility

Commit `56e162bb7ea0e2a9e7c3471ea2ffedf163411c1f` introduced `set` in PR #346
for issue #317. The OpenAPI snapshot at that commit already exposed only GET
for the app relationship. The PR's live smoke exercised `list`, not `set`; its
only set evidence was a mock that accepted the invented PATCH and returned 204.

The public command spelling and the existing `--app`, `--keywords`,
`--confirm`, and output flags are preserved. An old invocation without
`--version` and `--locale` cannot be preserved safely: app ID plus keyword text
does not identify the version-localized record Apple requires. Such an
invocation now fails with migration guidance before authentication or network
access. No automatic "editable version", latest version, primary locale, or
first-match selection is performed.

The direct alternatives remain valid:

- `asc localizations update --version "VERSION_ID" --locale "en-US"
  --keywords "kw1,kw2"` when the version ID is already known;
- `asc metadata keywords push --version-id "VERSION_ID" --input
  "./keywords.json"` for locale-keyed direct input; and
- `asc metadata keywords apply --app "APP_ID" --version "1.2.3" --dir
  "./metadata" --confirm` for repository-backed metadata.

## Verification design

RED coverage first exercises the retained spelling and proves it cannot perform
the required supported flow. GREEN coverage asserts the exact methods, paths,
filters, pagination URLs, PATCH body, JSON output, and exit behavior. It also
covers missing required flags, invalid platform and locale values, missing
versions and localizations, version ambiguity across platforms, duplicate
locale ambiguity, pagination failures, and API failures.

The built binary is checked for help, stdout, stderr, and exit status. Live
verification is read-only: resolve versions on the disposable app and list the
selected version's localizations. Live mutation uses only disposable app
`6759231657` when the current value can be restored and verified safely.

The final built binary resolved iOS version `1.0` and its single `en-US`
localization on that disposable app. It changed `keywords` from
`baseline,copy` to `pr1878-live-temporary`, a separate read observed the new
value, then the same command restored `baseline,copy`; a final read confirmed
the restoration. No disposable state was left behind.

## Alternatives considered

Accepting only a version-localization ID would reduce requests, but it would
discard the released app-oriented command shape and make `--app` meaningless.
Automatically selecting a latest or editable version and primary locale would
keep the old flag count, but could silently mutate the wrong platform, version,
or language. Explicit version and locale selection with optional platform is
the smallest supported, deterministic rework.
