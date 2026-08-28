# API Notes

Quirks and tips for specific App Store Connect API endpoints.

## Apple Ads Profile Context Isolation

- Apple Ads named profiles use only the context stored on that profile: they do not inherit `ads.org_id` or `ads.ad_account_id` from root config or another profile. This prevents a selected profile from silently sending a request to the wrong organization or ad account. Profile-less access-token and environment authentication can still use matching root context.

## Public App Store Ranking

- `asc apps public rank` is an unauthenticated experimental storefront command, not an App Store Connect OpenAPI operation or an Apple Ads metric.
- iOS ranking inspects up to 200 results from the public iTunes `/search` endpoint. Apple TV ranking uses the undocumented MZStore search endpoint with `X-Apple-Store-Front: <numeric-storefront-id>,33`, so it is available only for countries with a known numeric storefront ID.
- `found: false` means only that the app was absent from Apple's returned result window. Storefront order, window size, and the Apple TV response schema can change independently of the CLI.

## Analytics & Sales Reports

- Although Apple's current Sales Reports documentation describes `YYYY-MM-DD` for non-daily dates, the live endpoint requires `YYYY-MM` for monthly reports and `YYYY` for yearly reports. The CLI accepts either form and reduces full monthly or yearly dates to those live period identifiers before the request.
- Vendor number comes from Sales and Trends → Reports URL (`vendorNumber=...`)
- Sales Reports validates the complete report type/subtype/frequency/version tuple against Apple's endpoint table. Although the current table lists `SUBSCRIPTION` `1_3`, live verification in PR #1842 proved `1_4` succeeds and is required by some accounts, so both are accepted and `1_4` remains the default.
- Use `--paginate` with `asc analytics view --processing-date` to search every report page; the CLI forwards the value as `filter[processingDate]` when fetching instances. To resume from a saved report-page `links.next` URL, pass it with `--next <links.next> --paginate`.
- Use `--granularity "DAILY,WEEKLY,MONTHLY"` with `asc analytics view` to filter instances by one or more documented granularities
- Long analytics runs may require raising `ASC_TIMEOUT`

## Finance Reports

Finance reports use Apple fiscal months (`YYYY-MM`), not calendar months.

**API Report Types (mapping to App Store Connect UI):**

| API `--report-type` | UI Option                               | `--region` Code(s)      |
|---------------------|-----------------------------------------|-------------------------|
| `FINANCIAL`         | All Countries or Regions (Single File)  | `ZZ` (consolidated)     |
| `FINANCIAL`         | All Countries or Regions (Multiple Files) | `US`, `EU`, `JP`, etc. |
| `FINANCE_DETAIL`    | All Countries or Regions (Detailed)     | `Z1` (required)         |
| Not available       | Transaction Tax (Single File)           | N/A                     |

**Important:**
- `FINANCE_DETAIL` reports require region code `Z1` (the only valid region for detailed reports)
- Transaction Tax reports are NOT available via API; download manually from App Store Connect
- Region codes reference: https://developer.apple.com/help/app-store-connect/reference/financial-report-regions-and-currencies/
- Use `asc finance regions` to see all available region codes

## Sandbox Testers

- Required fields: email, first/last name, password + confirm, secret question/answer, birth date, territory
- Password must include uppercase, lowercase, and a number (8+ chars)
- Sandbox territory inputs accept alpha-2, alpha-3, and exact English country names, but the CLI sends canonical 3-letter App Store territory codes (for example, `US`, `USA`, and `United States` all resolve to `USA`)
- This normalization is limited to verified ASC alpha-3 territory surfaces, including customer-review filters; public storefront and finance region flags keep their existing namespaces
- List/get use the v2 API; create/delete use v1 endpoints (may be unavailable on some accounts)
- Update/clear-history use the v2 API

## TestFlight Distribution

- `asc testflight distribution edit --external-testing` shipped in 0.35.3 but App Store Connect does not allow `externalBuildState` in the build beta detail PATCH request. The flag remains parseable during its deprecation window and fails before HTTP instead of sending an unsupported update.
- Migrate `--external-testing=true` to `asc builds add-groups --build-id "BUILD_ID" --group "GROUP_ID" --submit --confirm`. Migrate `--external-testing=false` to `asc builds remove-groups --build-id "BUILD_ID" --group "GROUP_ID" --confirm`; the old boolean cannot identify which group assignments to remove.
- App Store Connect can briefly return a build-specific 404 from `POST /v1/builds/{id}/relationships/betaGroups` after an uploaded build is already readable and valid. `asc publish testflight` confirms the uploaded build with `GET /v1/builds/{id}` and retries only that post-upload propagation error with bounded backoff, reporting retry attempts on stderr. A confirmation in processing state `FAILED` or `INVALID` stops immediately without retrying distribution. A later post-upload failure emits a partial publish result with the recoverable `buildId`, terminal processing or notification outcome, and completed stages before exiting non-zero; notification follow-up failures use `failureStage=notification` after beta-group distribution succeeds.

## Game Center

- Most Game Center endpoints require a Game Center detail ID, resolved via `/v1/apps/{id}/gameCenterDetail`.
- If Game Center is not enabled for the app, the detail lookup returns 404.
- Releases are required to make achievements/leaderboards/leaderboard-sets live (create a release after creating the resource).
- Image uploads follow a three-step flow: reserve upload slot → upload file → commit upload (using upload operations).
- The `challengesMinimumPlatformVersions` relationship on `gameCenterDetails` uses `appStoreVersions` linkages (live API rejects `gameCenterAppVersions` for this relationship).
- The relationship endpoint is replace-only (PATCH); GET relationship requests are rejected with "does not allow 'GET_RELATIONSHIP'... Allowed operation is: REPLACE".
- Setting `challengesMinimumPlatformVersions` requires a live App Store version; non-live versions fail with `ENTITY_ERROR.RELATIONSHIP.INVALID.MIN_CHALLENGES_VERSION_MUST_BE_LIVE` ("must be live to be set as a minimum challenges version.").
- App Store Connect has no direct GET for a leaderboard-set member localization. `asc game-center leaderboard-sets member-localizations view --id` resolves the localization's leaderboard and leaderboard set through their to-one endpoints, then finds the exact ID in the doubly filtered collection across all pages.
- App Store Connect exposes a group's challenge relationships as read-only. `asc game-center groups challenges set` remains registered during a deprecation window and returns migration guidance without making an HTTP request; create a group-owned challenge with `asc game-center challenges create --group-id` instead.
- `asc game-center details list` is backed by the app's single Game Center detail. Its legacy `--limit`, `--next`, and `--paginate` flags remain registered during a deprecation window but return precise guidance to omit the unsupported flag.

## Apple Ads Platform API v1

- Platform API v1 uses `https://api.ads.apple.com/v1/` and sends
  `X-AP-Context: adAccountId=<id>;` for ad-account-scoped calls. Its
  `--ad-account` value is independent from the Campaign Management API v5
  `--org` value.
- V1 query and report requests use Platform API JSON schemas and preserve
  Apple's response envelopes. Report pagination belongs in the request body;
  the v1 report commands do not use the legacy `--paginate` flag.
- The v5 command tree remains runnable under `asc ads v5` in CLI 4.4.0 with a deprecation warning.
  Apple retires Campaign Management API v5 on January 26, 2027. The legacy raw
  `asc ads v5 api request` command stays a v5 request and is not rewritten; raw v1
  requests use `asc ads api request`.
- The version-neutral `asc ads auth discover` command calls Platform API v1
  `GET /v1/me` and `GET /v1/acls`. The direct `asc ads me view` and
  `asc ads acls list` commands expose those resources separately.
- Platform v1 has one negative-keywords resource for campaign and ad-group
  scope, and does not provide a bulk-delete operation. Product-page countries,
  product-page devices, and custom impression-share report list/view likewise
  have no one-command v1 replacement in 4.4.0.

## Authentication & Rate Limiting

- JWTs issued for App Store Connect are valid for 10 minutes (handled internally).
- For App Store Connect API requests, GET/HEAD requests automatically retry transient 408/429/5xx responses and transient transport failures. Ordinary POST/PATCH/PUT/DELETE requests are automatically replayed only after App Store Connect rejects them with 429; ambiguous 408/5xx responses and transport failures are surfaced without replay. Explicitly idempotent mutations use the broader transient retry policy because their exact payloads are safe to replay. Mutation bodies are buffered and sent identically on each retry. Set `ASC_MAX_RETRIES=0` to disable retries. Presigned uploads follow the upload-specific rules below.
- Retry-After headers are honored when they do not exceed `ASC_MAX_DELAY` and fit within the remaining request context. Hints above the cap or beyond the context budget fail fast with the requested delay and the applicable limit. Configure retry settings via `ASC_MAX_RETRIES`, `ASC_BASE_DELAY`, `ASC_MAX_DELAY`, `ASC_RETRY_LOG`.
- Uploads to the presigned URLs Apple returns in `uploadOperations` retry per part rather than per file: a PUT part is retried on 408/429/500/502/503/504 and on transient transport failures, using the same retry settings and honoring Retry-After only up to `ASC_MAX_DELAY`. Over-cap hints fail fast; parts that use any other method are never replayed, and each attempt is bounded by `ASC_UPLOAD_TIMEOUT`. This applies to build, screenshot, Game Center, App Clip, subscription, in-app purchase, and app event asset uploads.
- Unauthenticated public storefront reads used by `asc apps public view`, `asc apps public search`, `asc apps public prices`, `asc apps public descriptions`, `asc apps public rank`, and `asc reviews ratings` are idempotent GET requests. They retry 429 and 5xx responses with the shared backoff settings; Apple sends `Retry-After` as either seconds or an HTTP date on these endpoints, and both forms are capped at `ASC_MAX_DELAY`. `ASC_MAX_RETRIES=0` disables the retries. Successful stdout (including table and JSON renderers) and terminal public-storefront status errors remain unchanged; non-retryable statuses, transport failures, decode failures, and context cancellation are not replayed.
- The public storefront retry path is validated with deterministic `httptest` coverage for status boundaries, Retry-After parsing/capping, response-body draining, request replay, concurrency, and cancellation. It does not perform live mutations. The additive behavior can increase latency and request volume during transient failures, and Apple's undocumented storefront responses remain an external compatibility risk.
- `--api-debug` and `ASC_DEBUG=api` log each response's raw `X-Rate-Limit` value to stderr without changing stdout.
- Some endpoints return 403 when the API key role lacks permission (e.g., finance reports, reviews).

## Builds

- `GET /v1/apps/{id}/builds` has no documented default order and rejects `sort` with 400 `PARAMETER_ERROR.ILLEGAL`; with `limit=1` it can return a weeks-stale build that reads as "latest". Use the top-level collection instead: `GET /v1/builds?filter[app]={id}&sort=-uploadedDate&limit=1`.
- General shape of the trap: a relationship endpoint (`/v1/{parent}/{id}/{children}`) and its top-level collection (`/v1/{children}?filter[{parent}]=`) accept different query parameters, so a `sort` or `filter` that works on one can 400 on the other.

## Devices

- No DELETE endpoint; devices can only be enabled/disabled via PATCH.
- Registration requires a UDID (iOS) or Hardware UUID (macOS).
- Device management UI lives in the Apple Developer portal, not App Store Connect.
- Device reset is limited to once per membership year; disabling does not free slots.

## Subscription Offer Codes

- `POST /v1/subscriptionOfferCodes`: the `prices` relationship is required for every offer mode. For `FREE_TRIAL`, each inline price selects a territory but must omit `subscriptionPricePoint`; including one returns 409 `ENTITY_ERROR.RELATIONSHIP.INVALID`. Use `--prices "DEU,FRA"` for `FREE_TRIAL` and `--prices "DEU:PRICE_POINT_ID"` for paid modes.

## Monthly Subscriptions with a 12-Month Commitment

- Apple announced Monthly Subscriptions with a 12-Month Commitment on April 27, 2026:
  - https://developer.apple.com/news/?id=agq42lxe
  - https://developer.apple.com/help/app-store-connect/manage-subscriptions/set-availability-for-an-auto-renewable-subscription/
- The App Store Connect help docs describe this as a billing option on a regular 1-year subscription, with separate `1 Year Upfront` and `Monthly with 12-Month Commitment` availability sections for the same product.
- App Store Connect API 4.4 exposes `subscriptionPlanAvailabilities` with a `planType` attribute and `/v1/subscriptions/{id}/planAvailabilities` for reading the upfront/monthly plan availability set. Use `planType=MONTHLY` for Monthly with 12-Month Commitment, and keep `subscriptionAvailability` for the default/upfront availability.
- App Store Connect API 4.4.1 adds `/v1/subscriptionPricePoints/{id}/adjustedEqualizations`. Although OpenAPI models `filter[planType]` as an unconstrained string array, the live endpoint rejects `UPFRONT` and reports `MONTHLY` as the only supported value.
- Monthly commitment remains unavailable in the United States and Singapore; the CLI removes `USA` and `SGP` from requested monthly-commitment territories before writing plan availability.

## Pass Type IDs

- Live API rejects `include=passTypeId` and `fields[passTypeIds]` on `/v1/passTypeIds/{id}/certificates` despite the OpenAPI spec allowing them.
- The CLI does not expose those parameters for `pass-type-ids certificates list` to avoid API errors.

## App Store Connect API 4.4.1

- Apple added discrete versions for in-app purchases, subscriptions, and subscription groups. Their v2 localizations and images are version-scoped; pass a version ID rather than the legacy product, subscription, or group ID.
- Review submissions accept `inAppPurchaseVersions`, `subscriptionVersions`, and `subscriptionGroupVersions` through `reviewSubmissionItems`. The CLI preserves both relationship data and `included` resources in JSON output.
- API 4.4.1 has no item-detail GET operation. List a parent submission's items with `asc review items list --submission "SUBMISSION_ID"`.
- API 4.4.1 has no marketplace-webhook instance GET operation. `asc marketplace webhooks view` preserves its released behavior by selecting the exact ID across all pages of the supported collection GET.
- Review-item updates accept only nullable `resolved` and `removed` attributes. The response-only `state` attribute cannot be patched; use `--resolved`, `--removed`, or their matching `--clear-*` flags. Setting `removed=true` requires `--confirm`.
- Review-submission updates expose nullable `platform`, `submitted`, and `canceled` values plus matching `--clear-*` flags. Setting `submitted=true` or `canceled=true` requires `--confirm`; false, null, and platform-only updates do not.
- The create schema names its second experiment relationship `appStoreVersionExperimentV2`, but its linked resource type remains `appStoreVersionExperiments`. The CLI selector is `appStoreVersionExperimentsV2`. Experiment treatments are not valid review-item create relationships.
- Review items require `appCustomProductPageVersions`; `appCustomProductPages` is not an accepted item type because a page ID cannot be silently converted to a version ID.
- The v1 localization/image commands and submission shortcuts remain available during their deprecation window. Each direct invocation warns on stderr and preserves the existing endpoint, flags, stdout, and exit behavior. The two localization `sync` leaves are experimental; the other 27 direct leaves are stable. No v1 localization or image command is removed in this release.
- 4.0.0 removes the review item-detail surfaces: `asc review items view` and `asc review items-get` → `asc review items list --submission "SUBMISSION_ID"`; `asc review items update --state` / `items-update --state` → `--resolved` or `--removed`.
- The 3.x `--item-type appStoreVersionExperimentV2` alias is removed in 4.0.0; the value is rejected with guidance naming the canonical `appStoreVersionExperimentsV2`.
- `asc iap setup` and `asc subscriptions setup` remain supported, but warn when localization flags request their legacy v1 localization steps. Setup calls without those flags do not warn.
- Migration mapping:
  - IAP localizations/images → create or resolve an IAP version, then use `asc iap versions localizations ...` / `asc iap versions images ...`.
  - Subscription localizations/images → create or resolve a subscription version, then use `asc subscriptions versions localizations ...` / `asc subscriptions versions images ...`.
  - Subscription group localizations → create or resolve a group version, then use `asc subscriptions groups versions localizations ...`.
  - IAP submissions → `asc review items add --submission "SUBMISSION_ID" --item-type inAppPurchaseVersions --item-id "IAP_VERSION_ID"`.
  - Subscription submissions → `asc review items add --submission "SUBMISSION_ID" --item-type subscriptionVersions --item-id "SUBSCRIPTION_VERSION_ID"`.
  - Subscription group submissions → `asc review items add --submission "SUBMISSION_ID" --item-type subscriptionGroupVersions --item-id "GROUP_VERSION_ID"`.
- There is no one-command version-scoped replacement for the two experimental legacy localization `sync` leaves. Reconcile entries through the matching version-localization list/create/update/delete commands.
- A legacy IAP image file re-upload has no one-to-one v2 update. Create the replacement version image, then delete the old version image if needed. Subscription v2 image updates do not expose the legacy checksum flag; use the version image upload workflow for a new file.
- The 33 exported `internal/asc.Client` methods that target these legacy resources remain callable and are marked with Go `Deprecated:` documentation naming their version-scoped or review-item replacement.
- Nullable v2 localization updates distinguish omitted, value, and JSON `null`; use the corresponding `--clear-*` flag for explicit clears.

## Apple Ads Platform API v1 in 4.4.0

- Release 4.4.0 makes Platform API v1 the direct `asc ads` resource surface. Its host, request payloads, response envelopes, pagination, and ad-account context differ from Campaign Management API v5. The intermediate nested prototype is intentionally removed before release.
- Apple scheduled Campaign Management API v5 retirement for January 26, 2027. Every runnable v5 leaf moves under `asc ads v5`, emits a deprecation warning on invocation, and keeps its existing endpoint and output behavior. Use the direct v1 replacement where one exists; the seven v5 leaves without a one-command replacement retain explicit migration guidance.
- Platform account-scoped requests use `X-AP-Context: adAccountId=<AD_ACCOUNT_ID>;`. Resolve the account independently from the legacy organization context with `--ad-account`, `ASC_ADS_AD_ACCOUNT_ID`, the selected profile's `ad_account_id`, or root `ads.ad_account_id` when no named profile is selected. `ASC_ADS_ORG_ID` and `--org` are not fallbacks for an ad-account ID.
- `/v1/ad-accounts` is method-dependent: `POST /v1/ad-accounts` creates an account without `X-AP-Context`; `GET /v1/ad-accounts/{id}` and `PUT /v1/ad-accounts/{id}` require `X-AP-Context: adAccountId=<id>;`, and the header account must match the path ID.
- Authentication validation and discovery use Platform API v1. `asc ads auth login --network` and `asc ads auth status --validate` exchange an OAuth client-credentials token when needed and call `GET /v1/me` without an ad-account context. `asc ads auth discover` calls Platform API v1 `GET /v1/me` and `GET /v1/acls` without an ad-account context. A supplied `ASC_ADS_ACCESS_TOKEN` skips token exchange.
- The deprecated `asc ads v5 reports preset` warning follows `--level`: campaigns, ad groups, ads, keywords, and search terms point to their matching `asc ads reports apps` command; the two ad-group-specific keyword levels point to v1's consolidated `keywords` or `search-terms` report.
