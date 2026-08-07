# Subscription promotional offer inline prices

## Decision

`asc subscriptions offers promotional create` remains in the existing
subscription-offers taxonomy. Its required `--prices` flag supports both the
released linkage form and compound inline creation:

- existing promotional-offer price IDs, for example `PRICE_ID,PRICE_ID`
- inline territory-only entries, for example `US,France`
- inline territory and price-point entries, for example
  `US:PRICE_POINT_ID,France:PRICE_POINT_ID`

Territories accept alpha-2, alpha-3, or exact English country names and are
normalized to App Store Connect alpha-3 IDs. Inline inputs become temporary
local linkages in `data.relationships.prices` and matching
`subscriptionPromotionalOfferPrices` resources in top-level `included`.
Existing price IDs remain ordinary linkage references and do not emit
`included`. One request may combine territory-only and territory/price-point
inline entries, but it cannot mix linkage references with inline entries.

## API contract

The command calls `POST /v1/subscriptionPromotionalOffers` with no query
parameters. Its request is `SubscriptionPromotionalOfferCreateRequest`; the
required primary resource has type `subscriptionPromotionalOffers`, the five
existing required attributes, and required `subscription` and `prices`
relationships. The request schema permits top-level
`SubscriptionPromotionalOfferPriceInlineCreate` resources. The inline schema
requires only `type`; both `territory` and `subscriptionPricePoint`
relationships are optional, and it defines no conditional relationship rule
for any `offerMode`. Each local ID used by an inline price relationship exactly
matches one included resource. A successful request returns status 201 with
`SubscriptionPromotionalOfferResponse`.

Apple's current official 4.4.1 OpenAPI download is byte-for-byte identical to
`docs/openapi/latest.json`. It documents create, update, and delete operations
for promotional offers but no standalone create operation for their prices.
The generated Swift SDK and tddworks/asc-cli use this same atomic compound POST
with temporary linkages and inline territory and price-point relationships.

The command continues to write its response through the shared TTY-aware output
path: explicit `--output` wins, data is written to stdout, and validation or API
errors are written to stderr. Missing required flags and malformed price inputs
exit with status 2 before authentication or HTTP.

## Compatibility and migration

The flag name and released bare-ID behavior remain supported. Inputs containing
a colon are parsed as compound `TERRITORY:PRICE_POINT_ID` entries. If every bare
entry resolves as a territory, the command creates territory-only inline
resources. If no entry resolves as a territory, bare entries are preserved as
existing promotional-offer price IDs and sent with the original linkage-only
payload. A list containing both shapes is rejected before authentication rather
than sending a territory as a price ID. This auto-detection keeps existing
scripts working while allowing the new atomic create form.

Mode does not change parsing or payload validation because the exact OpenAPI
schema defines no relationship constraint based on `offerMode`. In particular,
territory-only inline resources are accepted for paid modes, and a price-point
relationship is not rejected for `FREE_TRIAL`. These are schema-supported
request shapes, not claims that every combination will pass Apple's
account-specific business validation.

The separate promotional-offer `update --prices` behavior is outside this
change: that endpoint updates relationships to prices belonging to an existing
offer and is not a substitute for creating the initial inline resources.

## RED-GREEN and verification

The client regression preserves the old linkage-only payload assertion and adds
exact bodies for temporary local IDs, optional territory and price-point
relationships, and top-level included resources. CLI coverage asserts legacy
bare IDs, territory-only and compound inline inputs across offer modes, mixed
shape rejection, JSON stdout, empty stderr, and usage exit status. The focused
tests run RED against the strict mode-specific parser, then GREEN after the
backward-compatible parser and client changes.

Black-box verification uses a freshly built binary to check help, stdout,
stderr, and exit status. Live verification on disposable app `6759231657`
created a territory-only `FREE_TRIAL` offer on subscription `6759789022`, read
the created offer and its single price, then deleted the exact returned offer
ID. A final list was empty and a read of the deleted ID returned not found. This
proves the territory-only compound path and cleanup boundary; the other
mode/relationship combinations remain schema-backed rather than live-proven.

## Alternatives

Unconditionally replacing existing price IDs was rejected because it breaks a
released CLI contract even if many callers prefer inline creation. Adding a
separate mode flag was also rejected because the shapes are distinguishable
without another public option. Separate territory and price-point flags would
make multi-territory pairing ambiguous. Auto-detection retains the legacy
payload while keeping the compact inline grammar.
