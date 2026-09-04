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

## Tax Categories and Transaction Tax Reports

Verified against the App Store Connect OpenAPI snapshot in `docs/openapi/` (spec version 4.4.1):

- There is no tax-category endpoint, and no tax-category attribute on `apps`, `appInfos`, or `inAppPurchases`. The App Store Connect UI is the only way to read or set an app or in-app purchase tax category.
- `GET /v1/financeReports` accepts only `FINANCIAL` and `FINANCE_DETAIL` in `filter[reportType]`, and `GET /v1/salesReports` has no tax report type, so Transaction Tax reports cannot be generated or downloaded through the public API.
- Both surfaces still need a live web-session endpoint capture before any `asc web` command can be shipped: the request method, path, headers, request body, and response body for the App Information tax category read and write, and for the Payments and Financial Reports "Create Reports" Transaction Tax generate, poll, and download calls. See issue #2299.
- `asc capabilities --area monetization` reports the tax category gap, and `asc capabilities --status not-public-api` reports both gaps.

## Sandbox Testers

- `asc web sandbox create` requires `--first-name`, `--last-name`, `--email`, `--password`, and `--territory`
- Password must include uppercase, lowercase, and a number (8+ chars)
- Historical public v1 create also required password confirmation, a secret question/answer, and a birth date; that removed v1 contract does not establish that those fields are accepted by the current private web flow
- Sandbox territory inputs accept alpha-2, alpha-3, and exact English country names, but the CLI sends canonical 3-letter App Store territory codes (for example, `US`, `USA`, and `United States` all resolve to `USA`)
- This normalization is limited to verified ASC alpha-3 territory surfaces, including customer-review filters; public storefront and finance region flags keep their existing namespaces
- List, view, update, and clear-history use the v2 API through `asc sandbox`
- `asc web sandbox create` currently sends three private web-session requests: `POST /sandbox/v2/account/validateFields` with `firstName`, `lastName`, and `acAccountName`; the same path with `acAccountPassword` added; then `POST /sandbox/v2/account/create` with `firstName`, `lastName`, `acAccountName`, `acAccountPassword`, and `storeFront`. This is the source-backed client request shape; Apple acceptance of extra portal fields has not been live-captured. See issue #2294.
- Public `asc sandbox` does not expose create or delete, and the current web-session CLI has no delete path. Do not infer a private delete endpoint from the removed v1 surface without a fresh capture.

## App Store Regulations & Permits declarations

- The public App Store Connect API has no declaration surface. `appInfos` in `docs/openapi/latest.json` exposes no `isRegulatedMedicalDevice`, `isPersonalService`, trader, or DSA attribute, and a case-insensitive scan of the whole snapshot finds no `medical`, `personalService`, `trader`, or `digitalServicesAct` field anywhere. The only trader-adjacent values are read-only `TerritoryAvailability.contentStatuses` reasons (`TRADER_STATUS_NOT_PROVIDED`, `TRADER_STATUS_VERIFICATION_FAILED`, `TRADER_STATUS_VERIFICATION_STATUS_MISSING`), which report a consequence rather than let anything be declared.
- Declarations therefore live on the web-session `ppm/complianceform/v1` service, which is neither JSON:API nor `/ci/api` plain JSON; requests need the App Store Connect UI headers (`X-Csrf-Itc: itc`, `Origin`, and a `/apps/{id}/distribution/info` `Referer`).
- `GET /ppm/complianceform/v1/accounts/{accountId}/requirements?contentId={appId}` lists every declaration Apple tracks for the app. Each row carries `id`, `name`, `ref`, `status`, `dateSigned`, `formId`, and `isRequired`. `requirementData` is keyed by `contentId`; prefer the entry whose `contentId` matches the app and fall back to the entry with an empty `contentId`. `asc web apps declarations list` reads exactly this.
- `GET /ppm/complianceform/v1/accounts/{accountId}/requirements/{requirementId}/forms?contentId={appId}` returns the stored answer alongside `constraints`, an object of JSONPath keys to `{attributeName, options[{value, listValues}]}` validation metadata. The constraint keys are rooted at `$[*]`, so the stored answer is returned as an array; readers accept the answer at the top level, under a `data` object, or as the first element of a `data` array. `asc web apps medical-device view` reads `medicalDeviceData.declaration` (`no`, `yes`, or absent while outstanding) from it.
- `POST /ppm/complianceform/v1/accounts/{accountId}/contents/{appId}/requirements/{requirementId}/forms` saves an answer. The captured body is `{accountId, contentId, requirementId, requirementName, countriesOrRegions, medicalDeviceData:{declaration:"no"}}`, where `countriesOrRegions` comes from the form's own `countriesOrRegions` constraint options with `EU` normalized to `EEA`. `asc web apps medical-device set --declared false` sends this only when the stored declaration is not already `no` with the requirement at `COLLECTED`; otherwise it reports `changed: false` without writing.
- The affirmative medical-device path is not implemented: the extra `medicalDeviceData` attributes Apple requires for a "Yes" answer (regulatory contact and evidence fields) have never been captured, and the constraint metadata alone does not establish the request body.
- The personal-service declaration is likewise not implemented. No capture in this repository or in any reachable reference records its requirement `name` or its form attribute, so there is nothing to send. `asc web apps declarations list` surfaces whatever requirement rows Apple returns, which is how the missing names should be captured.
- EU DSA trader status is account-level rather than app-level: it is read from `GET /ppm/v1/accounts/{id}/sellerInfo` and filed by `POST /ppm/v1/legalEntities/{id}/sellerInfo`, whose body carries contact details, an `isAppTraderOverride` flag, base64 identity documents, and a `jwtToken` minted by a separate `authenticationDetail` call and validated interactively against `id.apple.com`. Every `ppm/v1` record also carries an `optimisticLock`. A legal filing behind an interactive identity check is out of scope for an unattended CLI write.

## Web-session API keys

- `asc web api-keys list` reuses the iris v1 team-key list (`GET /iris/v1/apiKeys?include=createdBy,revokedBy,provider`) and the iris v2 individual-key list (`GET /iris/v2/apiKeys?include=visibleApps,createdByActor,revokedByActor`) already used by `asc web auth capabilities`. Both readers follow `links.next` internally, so the command has no `--paginate` flag. Individual keys sometimes carry an empty `roles` array on that list payload; list does not issue per-key actor lookups. Use `asc web auth capabilities --key-id` to resolve actor-backed roles for one key.
- `asc web api-keys view --key-id` uses the existing iris v1 team-key resource (`GET /iris/v1/apiKeys/{id}?include=provider`). Individual keys appear in `list` but are not loaded by `view`. The issue proposed `get`; current CLI taxonomy uses `view` for this leaf.
- Those payloads expose key ID, nickname, roles, `isActive`, key type, and last-used. They do not include a creation date, so list/view omit that column rather than inventing one. Private key material is never copied into command output.
- Revoke and `--individual` create still need a live web-session endpoint capture.

## Web app availability (iris)

- `GET /iris/v1/apps/{id}/appAvailabilityV2` returns `availableInNewTerritories` and a links-only `relationships.territoryAvailabilities`. It does not include `availableTerritories.data`. Adding `?include=availableTerritories&limit[availableTerritories]=200` returns 400 `PARAMETER_ERROR.INVALID`.
- The readable source is the iris v2 related collection: `GET /iris/v2/appAvailabilities/{id}/territoryAvailabilities?include=territory&limit=200`. Follow `links.next`. `filter[available]=true` is rejected with 400 `PARAMETER_ERROR.ILLEGAL`; filter client-side on `attributes.available`.
- `asc web apps delete` uses this collection for the "removed from sale in all territories" preflight. The public API counterpart is `/v2/appAvailabilities/{id}/territoryAvailabilities`.

## Web-session Resolution Center

- Resolution Center has no official App Store Connect API surface; the OpenAPI snapshot contains no `resolutionCenter*` or `reviewRejection*` path. Every reader below is a web-session (`/iris/v1`) call and needs Apple ID auth, not an API key.
- Threads have two scopes and they are not interchangeable. `asc web review show` resolves the submission scope (`GET /iris/v1/resolutionCenterThreads?filter[reviewSubmission]={id}&include=reviewSubmission`), which only returns threads Apple attached to that review submission. `asc web review threads --app` reads the app scope (`GET /iris/v1/apps/{appId}/resolutionCenterThreads?include=appStoreVersions,app,appMessageThreadDetail,build,betaBackgroundAssetReviewSubmission&limit[appStoreVersions]=2000&filter[threadType]=REJECTION_BINARY,REJECTION_METADATA,REJECTION_REVIEW_SUBMISSION,APP_MESSAGE_ARC,APP_MESSAGE_ARB,APP_MESSAGE_COMM,APP_MESSAGE_INFORMATIONAL`), which also returns binary, metadata, and informational threads that no submission owns. `show` reports the app-scoped threads the selected submission does not cover under `appThreads`.
- The app-scoped relationship is sent with the review center's captured `filter[threadType]` set rather than a narrowed one. Unsupported include or filter shapes on these surfaces answer 400 (for example `include=fromActor,rejections,resolutionCenterThread` on `resolutionCenterMessages`), so the known-good query shapes are sent verbatim.
- A thread's unsent draft reply lives at `GET /iris/v1/resolutionCenterThreads/{threadId}/resolutionCenterDraftMessage?include=resolutionCenterMessageAttachments,fromActor&limit[resolutionCenterMessageAttachments]=1000`. It is a single-resource document: a thread with no draft answers with a null `data` member, and the relationship can also answer 404. Both mean "no draft" rather than an error. `asc web review threads --drafts` reads it read-only, keeps Apple's raw HTML body, and never returns the attachments' signed download URLs.
- All of these readers follow `links.next` internally, so the commands have no `--paginate` flag.
- Sending a reply or a draft is not implemented; only reads are supported.

## Web-session app distribution method

- The public App Store Connect API has no distribution-method surface: `App` and `AppUpdateRequest` in `docs/openapi/latest.json` expose only `contentRightsDeclaration`, `streamlinedPurchasingEnabled`, subscription status URLs, and identity fields, and `AppAvailabilityV2` only carries `availableInNewTerritories`. The setting is web-session only.
- `asc web apps distribution view --app APP_ID` reads the internal app resource (`GET /iris/v1/apps/{id}`) and reports the `distributionType` and `educationDiscountType` attributes verbatim, alongside `name` and `bundleId`. No sparse fieldset or include is requested, because those attributes are returned on the plain resource read and an unknown `fields[apps]` value would fail the request outright.
- Observed values are `APP_STORE` (public App Store distribution) and `CUSTOM` (private distribution through Apple Business Manager or Apple School Manager). Apple omits the attribute for accounts or apps that never carried it; the command reports `unknown` in table output and omits the field in JSON rather than defaulting it to `APP_STORE`.
- Writes are not shipped. The observed write contract pairs `distributionType` with `educationDiscountType` in a single app PATCH, and public/private transitions carry Apple-side eligibility restrictions that are not observable from the read payload, so the CLI fails closed and leaves the change to the App Store Connect web UI.
- Unlisted App Store distribution is a request form reviewed by Apple, not an attribute value on this resource. There is no captured endpoint for it, so no flag is offered.

## Last-compatible version settings (`downloadable`)

- App Store Connect's Last-Compatible Version Settings screen has no dedicated resource and no `lastCompatibleVersion` attribute. The feature is carried by the boolean `downloadable` attribute on the existing `appStoreVersions` resource; `lastCompatibleVersion` is only a client-side label App Store Connect puts on the `appStoreVersions` collection it reads back.
- The public API covers both directions. `docs/openapi/latest.json` documents `downloadable` on `AppStoreVersion` and as a nullable attribute on `AppStoreVersionUpdateRequest`. `asc versions list/view --output json` preserves the attribute when Apple returns it, and `asc versions update --downloadable true|false` writes it. The default versions table does not include the field.
- `--downloadable` is tri-state: unset sends no `downloadable` attribute at all, so an unrelated `asc versions update` never changes download availability. `--downloadable false` makes a previously released version unavailable for download on older operating systems and devices, is not reversible from every state, and therefore requires `--confirm`.
- Apple omits `downloadable` on versions that never carried the setting. Reads report the attribute as absent rather than defaulting it to `true`, so a missing key means "Apple did not say", not "downloadable".
- `appStoreState` and `appVersionState` are both returned inconsistently across versions. App Store Connect's web client populates `appStoreState` from `appVersionState` and applies legacy remapping (`READY_FOR_DISTRIBUTION` to `READY_FOR_SALE`) purely client-side. The CLI does not reproduce that remapping.
- A web-session read (`asc web apps last-compatible-version view`) briefly existed for this screen and was retired before it reached a release. It mirrored App Store Connect's own iris request (`GET /iris/v1/apps/{id}?include=appStoreVersions&fields[appStoreVersions]=...,downloadable,...&limit[appStoreVersions]=2000`), which required a web session and offered no write. The public API path supersedes it in both directions, so no web-session command is needed here.

## Web-session app status history

- The public App Store Connect API has no status-history endpoint. fastlane's only status-history path is the retired legacy tunes API (`GET ra/apps/{id}/stateHistory?platform=...`), which has no app-level iris counterpart.
- The modern equivalent is version-scoped: `GET /iris/v1/appStoreVersions/{appStoreVersionId}/appStoreVersionStateChanges`, whose resources carry `appStoreState`, `date`, and `initiator`. `initiator` is the actor App Store Connect shows for the change.
- There is no app-level history endpoint, so `asc web apps history --app APP_ID` lists the app's versions with `GET /iris/v1/apps/{id}/appStoreVersions` and then fans out one state-change request per version. `--version-id` scopes the read to a single version and skips the fan-out after verifying that version's app relationship matches `--app`.
- Both readers follow `links.next` internally, so the command has no `--paginate` flag, matching `asc web api-keys list`.
- The fan-out is serial and shares one request timeout, so an app with a long release history can exceed the 30s default. Scope the read with `--version-id`, or raise `ASC_TIMEOUT`. Requests are not parallelized, to avoid hammering a web session with concurrent internal-API calls.
- `AppStatusHistory` is a role capability in App Store Connect, so accounts without it can get an authorization error on the state-change read even when the app list succeeds.

## Web-session review submissions (iris)

- `GET /iris/v1/reviewSubmissions/{id}/items` rejects `include=appStoreVersionExperimentV2` with HTTP 400 `PARAMETER_ERROR.INVALID` even though the public OpenAPI snapshot lists that relationship. Verified live 2026-09-03. `asc web review show` omits it from the items include and keeps the iris-accepted names, including `inAppPurchaseVersion`, `subscriptionVersion`, and `subscriptionGroupVersion`.

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

## Xcode Cloud workflows

- `GET /v1/ciWorkflows/{id}` returns relationships with links only by default: `repository` and `buildRuns` come back without a `data` linkage, and `product`, `xcodeVersion`, and `macOsVersion` are absent from the response entirely. `POST /v1/ciWorkflows` requires all four linkages, so any read-then-recreate flow must request `?include=product,repository,xcodeVersion,macOsVersion`, which populates them.
- `GET /v1/ciWorkflows/{id}` also emits JSON `null` for optional action and start-condition properties (`destination`, `testConfiguration`, `filesAndFoldersRule`) that `CiWorkflowCreateRequest` does not mark nullable. `workflows duplicate` omits those nulls so the create body stays schema-clean; unused nullable start conditions are omitted rather than sent as `null`.
- `CiAction` has no post-actions: the public workflow schema covers `BUILD`, `ANALYZE`, `TEST`, and `ARCHIVE` actions plus `buildDistributionAudience`, but TestFlight post-actions (beta group and tester assignment) exist only in the private `/ci/api/` workflow payload. A workflow recreated through the public API therefore loses its TestFlight post-actions.
- Workflow-scoped environment variables and secrets are also absent from `CiWorkflowCreateRequest`; they live on the private `/ci/api/` workflow payload. `workflows duplicate` cannot copy them. Use `asc web xcode-cloud env-vars` after creating the copy.

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

## Subscription Plan Availability

- Reading: `GET /v1/subscriptions/{id}/planAvailabilities` accepts `include=availableTerritories`, but `limit[availableTerritories]` is capped at 50 while a plan can be available in every storefront. The complete set comes from `GET /v1/subscriptionPlanAvailabilities/{id}/relationships/availableTerritories`, whose `limit` maximum is 200 with cursor pagination. `asc subscriptions pricing plan-availability show` prints Apple's include envelope unmodified and warns on stderr when paging metadata shows the include was truncated.
- Writing: `PATCH /v1/subscriptionPlanAvailabilities/{id}` replaces the `availableTerritories` linkage array wholesale, so the request body must carry the complete desired territory set, not a delta. `SubscriptionPlanAvailabilityUpdateRequest` accepts only `availableInNewTerritories` as a mutable attribute; `planType` is create-only through `POST /v1/subscriptionPlanAvailabilities`. After a write, `set` verifies territories through the paginated relationship endpoint and `availableInNewTerritories` through a fresh `GET /v1/subscriptionPlanAvailabilities/{id}` rather than the mutation response.
- Apple's internal web (iris) API uses the same resource, path shape, and PATCH body; `asc web subscriptions availability remove-from-sale` uses it only because emptying `availableTerritories` removes an approved subscription from sale, which Apple restricts to the Account Holder. Everything else about plan availability is available through the public API, so `asc subscriptions pricing plan-availability show|set` uses the public endpoints.
- `availableInNewTerritories` is not supported for `MONTHLY` plan availability.

## Developer Portal session (web session)

- Bundle IDs, App Groups, and agreements share one Developer Portal session helper: `POST /services-account/QH65B2/account/listTeams.action` bootstraps CSRF and the team list, then every later portal request carries the selected `teamId`. Same-origin redirects are enforced; cookies and CSRF tokens are never written to stdout, stderr, or debug logs.
- `--developer-team` (ID, or exact team name) is accepted only on Developer Portal-backed commands (`web bundle-ids capabilities enable`, every `web app-groups` subcommand, and `web agreements`). It is not a global web-session flag. There is no `ASC_DEVELOPER_TEAM` env fallback; `--apple-id` / `--provider-id` likewise have none.
- Team resolution: an explicit `--developer-team` wins (case-insensitive ID, then exact name) and fails closed with the available IDs and names if nothing matches. Without a selector, a previously persisted team ID is reused when it is still in the list; otherwise the selected App Store Connect provider is matched by public provider ID, then exact name, then a name-prefix heuristic only when exactly one team matches. A single remaining team is used. Multiple unmatched teams fail closed and ask for `--developer-team`. The resolved team ID is stored in the web session cache next to the provider selection; a new `--developer-team` value overrides and re-persists. `asc web auth status` reports it as additive `developerTeamId`.
- App Groups mutations still refresh CSRF from `listApplicationGroups.action` in that endpoint's scope after the shared bootstrap. Bundle ID capability and App Group assign/set/unassign paths still read the complete relationship graph, skip already-satisfied writes, and abort rather than rewrite from incomplete data.

## Developer Portal Agreements (web session)

- The public API has no agreements endpoint. `asc web agreements` uses the cookie-authenticated Developer Portal account services: `POST /services-account/QH65B2/account/getAgreementHistory` and `POST /services-account/QH65B2/account/acceptAgreements`, both with a JSON body carrying `teamId` (accept also carries an `agreementIds` array, so several agreements can be accepted in one request). They answer HTTP 200 even on failure; `resultCode` carries the outcome (`0` success).
- Each history record includes `agreementDownloadUrl`, observed as a root-relative Developer Portal path such as `/services-account/agreement/{agreementId}/content/pdf`. `asc web agreements download` resolves it against the Developer Portal origin and only follows HTTPS, same-origin targets and redirects, and rejects empty or HTML responses. The URL is treated as potentially signed: the `download` receipt and its error text never include it. `asc web agreements status` and the verified `accept` receipt still expose it as `downloadUrl` in JSON, so treat those outputs accordingly.
- `acceptAgreements` returns the updated history, but the CLI re-reads `getAgreementHistory` after the write and reports the re-read state (`dateAccepted >= dateEffective`) instead of trusting the mutation response.

## Pass Type IDs

- Live API rejects `include=passTypeId` and `fields[passTypeIds]` on `/v1/passTypeIds/{id}/certificates` despite the OpenAPI spec allowing them.
- The CLI does not expose those parameters for `pass-type-ids certificates list` to avoid API errors.

## Sparse Fieldsets Combined with Includes

Observed 2026-09-02 against a live App Store Connect team. The CLI does not add included relationship names to the primary fieldset for these list commands.

- `GET /v1/profiles` with `fields[profiles]=name&include=devices` returns HTTP 200 and still puts related devices in `included`. Apple omits `relationships` on each profile unless `devices` is also listed in `fields[profiles]`. `fields[devices]=name,udid` still sparse-filters those included devices.
- `GET /v1/certificates` with `fields[certificates]=displayName&include=passTypeId` returns HTTP 200. This team has no `PASS_TYPE_ID` certificates, so `included` was absent both with that sparse fieldset and with `include=passTypeId` alone. Non-pass certificates expose `relationships.passTypeId.data=null` only when the relationship is in the fieldset (or when no certificate fieldset is sent).

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

## Developer Portal App Groups (web session)

- App Group resources and their Bundle ID associations are not exposed by the public App Store Connect API. `asc web app-groups` uses the Developer Portal legacy form endpoints under `/services-account/QH65B2/account/ios/identifiers/` (`listApplicationGroups.action`, `addApplicationGroup.action`, `deleteApplicationGroup.action`) plus the cookie-authenticated JSON:API proxy under `/services-account/v1`.
- Legacy form endpoints are `POST` with `application/x-www-form-urlencoded` bodies that always carry `teamId`, return a `resultCode` envelope (`0` is success; `userString`/`resultString` carry Apple's message), and require the `csrf`/`csrf_ts` headers captured from a preceding `listApplicationGroups.action` response in the same endpoint scope.
- `deleteApplicationGroup.action` takes `teamId` and `applicationGroup` (the opaque App Group resource ID shown in `asc web app-groups list`) and returns the bare `resultCode` envelope. This contract is inferred from the sibling list/create actions and the long-standing third-party portal client contract; it was not live-verified when added because no Developer Portal web session was available. The CLI verifies every delete by re-reading the team's App Group list with pagination and fails if the group is still listed; on that strict read a success envelope whose `applicationGroupList` or `totalRecords` is absent or null, whose `totalRecords` is smaller than the records returned, whose `pageNumber` does not match the requested page, whose pages stop before `totalRecords` is reached, or that repeats an App Group across pages is treated as unverified rather than as a short list. A 2xx delete response whose body cannot be parsed or that carries no `resultCode` is likewise reported as an accepted but unverified delete, because the group may already be gone; an explicit non-zero `resultCode` is a refused delete.
- Bundle ID association submits the complete group set: the CLI reads the Bundle ID with its `bundleIdCapabilities` graph, rewrites only the `APP_GROUPS` capability's `appGroups` relationship together with its `enabled` attribute, and `PATCH`es `/services-account/v1/bundleIds/{id}`. `assign`, `unassign`, and `set` share this path with different computed sets, and all three abort before any write when the Bundle ID read returns a different resource ID than requested, omits the `bundleIdCapabilities` relationship, returns it with a null `data` collection or with a reference whose type is not `bundleIdCapabilities` or whose ID is empty, or repeats an included capability ID with conflicting representations, or carries an `APP_GROUPS` capability without a readable `appGroups` collection or without a boolean `enabled` attribute, because the PATCH would otherwise rewrite the graph from incomplete data. When the portal accepts a write but the verification read fails or disagrees, the client returns a `DeveloperAppGroupUnverifiedError` and the CLI warns that the change should be assumed applied. A write that fails without a verdict (transport error, dropped connection, or context deadline after the request was handed off) is settled by the same re-read: the requested state means it applied and the command succeeds, the prior state means it did not and the transport error is returned as a retry-safe failure, and anything else is reported as unverified. Explicit HTTP error statuses are refusals and are never re-read. `delete` settles its own form POST the same way against the App Group list. The portal can return a disabled `APP_GROUPS` capability that still lists groups in its relationship data; those groups count as referenced for delete preflight, so `unassign` and `set` operate on the raw relationship list rather than only the enabled set (`assign`/`set` enable the capability, `unassign` keeps a disabled capability disabled and disables it when the last group is removed). `assign`, `set`, and `unassign` re-read the Bundle ID afterwards and fail when the raw group list or `enabled` state differs from what was written.
- Before deleting, the CLI lists every Bundle ID in the team through the proxied `POST /services-account/v1/bundleIds` read (`X-HTTP-Method-Override: GET`, `include=bundleIdCapabilities,bundleIdCapabilities.capability,bundleIdCapabilities.appGroups`, `limit=200`, following `links.next`) and refuses when any `APP_GROUPS` capability references the group. A Bundle ID whose capability graph is missing from `included`, a list entry that is not a `bundleIds` resource, a Bundle ID repeated across pages, a `meta.paging.total` that disagrees with the records returned, a final full page of 200 without a `links.next` cursor or paging total, two conflicting `included` representations of the same capability ID, a capability reference with the wrong type or an empty ID, an `APP_GROUPS` capability without a readable `appGroups` collection, or a response whose `data` collection is absent or `null`, is treated as an error, never as unassigned. This list read is also inferred from the single-resource read contract and not yet live-verified.

## Apple Ads Platform API v1 in 4.4.0

- Release 4.4.0 makes Platform API v1 the direct `asc ads` resource surface. Its host, request payloads, response envelopes, pagination, and ad-account context differ from Campaign Management API v5. The intermediate nested prototype is intentionally removed before release.
- Apple scheduled Campaign Management API v5 retirement for January 26, 2027. Every runnable v5 leaf moves under `asc ads v5`, emits a deprecation warning on invocation, and keeps its existing endpoint and output behavior. Use the direct v1 replacement where one exists; the seven v5 leaves without a one-command replacement retain explicit migration guidance.
- Platform account-scoped requests use `X-AP-Context: adAccountId=<AD_ACCOUNT_ID>;`. Resolve the account independently from the legacy organization context with `--ad-account`, `ASC_ADS_AD_ACCOUNT_ID`, the selected profile's `ad_account_id`, or root `ads.ad_account_id` when no named profile is selected. `ASC_ADS_ORG_ID` and `--org` are not fallbacks for an ad-account ID.
- `/v1/ad-accounts` is method-dependent: `POST /v1/ad-accounts` creates an account without `X-AP-Context`; `GET /v1/ad-accounts/{id}` and `PUT /v1/ad-accounts/{id}` require `X-AP-Context: adAccountId=<id>;`, and the header account must match the path ID.
- Authentication validation and discovery use Platform API v1. `asc ads auth login --network` and `asc ads auth status --validate` exchange an OAuth client-credentials token when needed and call `GET /v1/me` without an ad-account context. `asc ads auth discover` calls Platform API v1 `GET /v1/me` and `GET /v1/acls` without an ad-account context. A supplied `ASC_ADS_ACCESS_TOKEN` skips token exchange.
- The deprecated `asc ads v5 reports preset` warning follows `--level`: campaigns, ad groups, ads, keywords, and search terms point to their matching `asc ads reports apps` command; the two ad-group-specific keyword levels point to v1's consolidated `keywords` or `search-terms` report.
