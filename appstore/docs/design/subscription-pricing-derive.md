# Subscription pricing derivation

## Decision

Add `asc subscriptions pricing derive` as a narrow, one-shot workflow for
setting one subscription's standard `UPFRONT` territory prices from another
subscription's currently effective territory prices.

```bash
asc subscriptions pricing derive \
  --source-subscription-id "MONTHLY_ID" \
  --target-subscription-id "YEARLY_ID" \
  --multiplier "10" \
  --round nearest \
  --dry-run

asc subscriptions pricing derive \
  --source-subscription-id "MONTHLY_ID" \
  --target-subscription-id "YEARLY_ID" \
  --multiplier "10" \
  --round nearest \
  --confirm
```

The command calculates `source customer price * multiplier` independently in
each territory. It then resolves that desired price against the target
subscription's Apple price-point ladder. It is a snapshot operation, not a
persistent relationship between subscriptions.

This command is intentionally separate from `pricing equalize`. Equalize uses
Apple's localized equalization matrix for one subscription. Derive preserves a
user-selected local price multiple between two subscriptions as closely as the
target price-point ladder permits.

## Placement and compatibility

Register `derive` under the existing stable `asc subscriptions pricing`
family. It is an additive command and does not change or deprecate existing
commands, flags, API requests, or JSON output.

The initial command supports standard `UPFRONT` prices for two distinct,
already-priced subscriptions. Monthly-with-12-month-commitment plan pricing is
out of scope because it has different availability and price constraints.
`UPFRONT` is Apple's billing-plan mode, not the subscription period: ordinary
`ONE_MONTH` and `ONE_YEAR` products both use it, so this covers the monthly to
yearly product relationship in the motivating example.

## Flags and validation

- `--source-subscription-id`: source subscription ID, product ID, or exact
  current name; required.
- `--target-subscription-id`: target subscription ID, product ID, or exact
  current name; required and must resolve to a different subscription.
- `--app`: app ID used when either selector is a product ID or name.
- `--multiplier`: positive finite decimal; required. Decimal arithmetic must be
  exact and must not use binary floating point.
- `--round`: `exact`, `nearest`, `up`, or `down`; default `nearest`.
- `--territory`: optional single territory ID, 2-letter code, or name for a
  focused preview or rollout; omitted means every current source territory.
- `--start-date`: optional `YYYY-MM-DD` schedule date.
- `--preserved`: preserve current prices for existing subscribers.
- `--auto-start-date`: when true, schedule approved/live target subscription
  changes for tomorrow if `--start-date` is omitted; default true.
- `--workers`: concurrent target price-point lookups and mutations; range
  1-32, default 8.
- `--dry-run`: calculate and print the complete plan without mutation.
- `--confirm`: required for mutation unless `--dry-run` is set.

Invalid or missing flags fail before client creation with usage exit code 2.
`--dry-run` and `--confirm` may not be combined. Positional arguments are not
accepted.

## Price-point resolution

For each territory, calculate the exact rational desired price and inspect the
target subscription's complete price-point ladder for that territory.

- `exact`: require a price point equal to the desired price.
- `nearest`: choose the smallest absolute price difference; an exact tie
  chooses the lower price to avoid an implicit overcharge.
- `up`: choose the lowest price greater than or equal to the desired price.
- `down`: choose the highest price less than or equal to the desired price.

Empty ladders, malformed prices, duplicate IDs for the selected customer
price, and missing upper or lower candidates are explicit row errors. Planning
is fail-closed: if any territory cannot resolve, print the full plan and make
no mutations.

The plan reports the source, desired, current target, and resolved target
prices, the requested and achieved multipliers, the price-point ID, action,
status, and any error for every territory. Rows are sorted by territory.

## API flow

The behavior is client-side orchestration over existing public API surfaces:

1. Resolve both subscription selectors.
2. Read current source and target `UPFRONT` prices with
   `GET /v1/subscriptions/{id}/prices`, including territory and price point.
3. Require the target subscription to have an existing pricing configuration.
   When a focused `--territory` read has no current target price, perform an
   unfiltered preflight to distinguish a missing territory price from an
   entirely unpriced target. Only the former may proceed with a planned update.
4. Read candidate target price points with a batched territory filter,
   `include=territory`, the endpoint's 8,000-resource page limit, and full
   pagination. Group the returned ladder by territory locally; do not issue a
   separately paginated ladder request for every territory.
5. Produce a complete plan and mark rows whose current target price point
   already matches as `noop`. Report them, but exclude them from mutation.
6. For apply, create changes with `POST /v1/subscriptionPrices`. The request
   uses `SubscriptionPriceCreateRequest`: optional `startDate`,
   `preserveCurrentPrice`, and `planType`, plus relationships to the target
   subscription, territory, and selected subscription price point.
7. Reconcile ambiguous transport failures against readback state, retry only
   retryable mutations, then read the effective target map back and verify it.

The command never changes subscription sale availability.

## Output and failure behavior

JSON uses exported camelCase result structs registered with the shared output
renderer. Table and Markdown output contain a summary followed by row-level
business fields. Data goes to stdout; progress and diagnostics go to stderr.

- Successful dry run: exit 0, `dryRun: true`, no mutation requests.
- Successful apply: exit 0 only after readback verification.
- Unresolved planning rows: print the plan, exit 1, make no mutation requests.
- Partial apply failure or verification mismatch: print all row results, exit
  1, and identify every failed territory.
- Already-matching rows are `noop` and do not produce mutation requests.

## Tests

Start with CLI-level RED coverage for command registration, required flags,
invalid decimals, identical subscriptions, invalid rounding modes, worker
bounds, positional arguments, confirmation, and mixed flag placement.

Add exact-decimal unit coverage for all four rounding strategies, exact hits,
nearest ties, unordered ladders, duplicate selected prices, malformed entries,
and missing bounds. Add `httptest` command coverage for paginated source/target
prices, paginated target ladders, deterministic rows, no-op suppression,
fail-closed planning, mutation request bodies, scheduled and preserved changes,
rate-limit reconciliation, readback verification, and API errors. Parse JSON
outputs and assert stdout, stderr, and exit behavior.

Finally, build `/tmp/asc`, inspect help and representative usage failures, run
generated-doc checks, affected package tests, and the full repository gate with
`ASC_BYPASS_KEYCHAIN=1`.

## Alternatives

1. Compose `prices list --resolved`, an external script, and CSV import. This
   works today but leaves rounding, audit output, fail-closed behavior, and
   verification to every caller.
2. Implement the broader spec-driven `pricing plan/apply` workflow from issue
   928. That remains a useful product direction, but it is materially larger
   than this concrete source-to-target pricing job. `derive` can later feed the
   same planning engine without committing the narrow command to a general
   pricing-template schema now.
