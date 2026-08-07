# TestFlight feedback screenshot fieldset

## Placement and command shape

The change stays in the existing `asc testflight feedback list` command and its
shared `internal/cli/feedback` implementation. The public invocation remains:

```sh
asc testflight feedback list --app "APP_ID" --include-screenshots
```

Current help describes `--include-screenshots` as including screenshot URLs in
feedback output. No command, flag, registry, help, or generated-command-doc
change is needed.

## API contract

The command performs `GET /v1/apps/{id}/betaFeedbackScreenshotSubmissions`.
The App Store Connect 4.4.1
OpenAPI operation accepts filters for device, platform, build, pre-release
version, and tester; `sort`; sparse fields; `limit`; and `include=build,tester`.
Its `200` response is `BetaFeedbackScreenshotSubmissionsResponse`, containing
`BetaFeedbackScreenshotSubmission` resources and optional included builds or
beta testers.

`screenshots` is an attribute selectable through
`fields[betaFeedbackScreenshotSubmissions]`, not an included relationship. When
the flag is enabled, the fieldset must contain all 20 regular feedback
attributes plus `screenshots` and both relationship fields:

```text
createdDate,comment,email,deviceModel,osVersion,locale,timeZone,architecture,connectionType,pairedAppleWatch,appUptimeInMilliseconds,diskBytesAvailable,diskBytesTotal,batteryPercentage,screenWidthInPoints,screenHeightInPoints,appPlatform,devicePlatform,deviceFamily,buildBundleId,screenshots,build,tester
```

The `build` and `tester` field names preserve relationship linkages on the
primary feedback resources. If `--include build`, `--include tester`, or both
are also present, the separate `include` query asks Apple to return those
related resources as compound `included` data.

## Support and history evidence

Apple's current [endpoint
documentation](https://developer.apple.com/documentation/appstoreconnectapi/get-v1-apps-_id_-betafeedbackscreenshotsubmissions)
describes this GET workflow and lists `screenshots` among the supported sparse
fields. Apple introduced TestFlight feedback screenshot retrieval in the
[App Store Connect API 4.0 release
notes](https://developer.apple.com/documentation/appstoreconnectapi/app-store-connect-api-4-0-release-notes).
The current official OpenAPI download is byte-for-byte identical to the
repository snapshot.

The maintained [AppStoreConnect Swift
SDK](https://github.com/AvdLee/appstoreconnect-swift-sdk/blob/651b56950c917d30d7aaa863baadd290f2e28cb7/Sources/OpenAPI/Generated/Paths/PathsV1AppsWithIDBetaFeedbackScreenshotSubmissions.swift)
defines this exact GET operation and the complete documented field enum.

The flag originated in PR #20, which used all seven feedback attributes modeled
at that time plus `screenshots`. A later model expansion added 13 documented
attributes without updating the sparse fieldset or its mock. Neither PR history
nor review discussion indicates that discarding those attributes was intended.
Live read-only GETs against an explicit disposable app ID returned HTTP 200 with
and without the current screenshot fieldset; the app had no feedback rows, so
response-field preservation remains covered deterministically by HTTP and
command regression tests.

## Behavior and compatibility

Output selection remains TTY-aware: explicit `--output` wins,
`ASC_DEFAULT_OUTPUT` otherwise pins the default, and without either setting
interactive terminals use table while pipes and CI use minified JSON. Table and
Markdown keep their existing conditional Screenshots column. Data stays on
stdout, diagnostics stay on stderr, and existing success and usage exit codes
are unchanged. The fix is backward-compatible and additive: it restores the 13
feedback attributes that the current eight-name sparse fieldset suppresses
while preserving screenshots and requested relationships. No lifecycle,
migration, or deprecation change is required.

## Verification

RED coverage will assert the exact method, path, and complete query fieldset at
the query-builder, HTTP-client, and canonical command levels. The mocked
response will contain every feedback attribute plus a screenshot, and the
command test will prove JSON retains representative restored fields and the
screenshot. Focused tests will be followed by a built-binary check, read-only
live requests with explicit app ID and file-backed authentication, and the full
format, docs, lint, and test gates.

Edge cases include combining screenshots with build/tester includes, empty
feedback collections, and pagination through Apple's `links.next` URL. API
errors continue through the existing client error path.

## Alternatives

Omitting the sparse fieldset would avoid suppressing normal attributes, but it
does not explicitly request the opt-in screenshot data and would undo the
flag's purpose. Requesting only `screenshots` is even more destructive. Sending
the complete documented attribute fieldset is explicit, additive, and keeps
the existing public contract intact.
