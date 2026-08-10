# Webhook contract corrections

## Placement and command shape

This change stays within the existing public `asc webhooks` and deprecated
`asc marketplace webhooks` command families. Public delivery listing remains:

```text
asc webhooks deliveries --webhook-id "WEBHOOK_ID" [--created-after TIMESTAMP] [--created-before TIMESTAMP]
```

Both date flags are optional and may be combined. Public webhook create and
update continue to accept comma-separated `--events`, but only values from the
documented `WebhookEventType` enum are accepted. The released marketplace
`view` command and `GetMarketplaceWebhook` client method remain source
compatible, but resolve an exact ID locally from the supported collection GET.

## API contract

`GET /v1/webhooks/{id}/deliveries` has two independent optional array query
parameters: `filter[createdDateGreaterThanOrEqualTo]` and
`filter[createdDateLessThan]`. Its response is a webhook-delivery collection.
Neither filter is required or mutually exclusive.

Webhook create and update payloads use the exact `WebhookEventType` enum from
the create/update request schema. Invalid examples such as
`SUBSCRIPTION.CREATED` must not be sent to Apple.

`/v1/marketplaceWebhooks/{id}` exposes only `PATCH` and `DELETE` in the current
OpenAPI snapshot. A live read-only `GET` probe also returns `GET_INSTANCE` as
not allowed, so the existing mocked instance GET is not an Apple operation.
`GET /v1/marketplaceWebhooks` is the supported read operation. It accepts
`fields[marketplaceWebhooks]` and a maximum `limit` of 200 and returns a
paginated `MarketplaceWebhooksResponse`. Marketplace `view` therefore requests
the collection at that maximum page size, follows every `links.next` URL until
it finds the exact requested ID or exhausts the collection, and rejects a
repeated next URL rather than silently truncating or looping.
## CLI behavior and compatibility

Successful commands keep their existing TTY-aware output behavior and write
data to stdout. Validation errors use stderr with usage exit code 2. Delivery
date flags become more permissive, which is backward compatible. Event
validation rejects previously accepted invalid values before authentication or
HTTP side effects. Marketplace `view` keeps its released invocation, output
shape, help, missing-ID usage exit code 2, API-error propagation, and not-found
exit code 4 without calling the unsupported instance GET operation.

## RED-GREEN and verification

- Replace delivery validation tests that encode required/mutually-exclusive
  filters with zero-, one-, and two-filter HTTP query assertions.
- Add create and update CLI tests proving invalid event values fail locally and
  valid enum values are normalized.
- Replace the marketplace instance-GET mock with page-one, page-two,
  repeated-next, empty, not-found, and API-error coverage for collection-backed
  exact-ID selection. Add CLI help, JSON output, and exit-code regressions.
- Regenerate command documentation, build `/tmp/asc`, and verify stdout,
  stderr, and exit codes.
- Re-run the collection list and a deliberately nonexistent marketplace
  webhook view as safe live reads with the explicit file-backed profile; do not
  mutate any real resource.
- Run focused tests, adjacent packages, formatting, documentation checks,
  lint, and the full test suite.

Edge cases include pagination URLs (whose embedded query remains authoritative),
repeated next URLs, exact rather than prefix ID matching, empty comma-separated
event input, case normalization, duplicate date values, API errors, and a
marketplace account with no collection entries.

## Alternatives

Direct removal would accurately reflect the missing instance operation, but it
would break a released stable command without the repository's required
deprecation lifecycle. A deprecated shim that only directs callers to `list`
would preserve the command name but not its successful behavior or structured
output.

Collection-backed exact-ID selection preserves the public Go method and CLI
shape while using only Apple's supported operation. Its extra requests are
bounded by collection pagination, and repeated-next detection makes malformed
pagination fail visibly instead of looping or returning a false not-found.
