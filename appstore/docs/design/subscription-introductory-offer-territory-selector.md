# Subscription introductory-offer territory selector

> Status: the `--territory ALL` compatibility spelling described below was removed in 5.0.0; `ALL` is now validated like any other territory value. Use `--all-territories`. See `migrate-to-5-0.mdx`.

## Decision

`asc subscriptions offers introductory create` requires exactly one territory
mode before authentication or HTTP:

```text
(--territory "USA" | --all-territories)
```

`--territory` accepts alpha-2, alpha-3, or exact English country names and is
normalized to the canonical App Store Connect territory ID. An explicitly
blank value is invalid. Combining a concrete territory with
`--all-territories` is invalid.

The shipped `--territory ALL` spelling remains a deprecated compatibility alias
for `--all-territories`. It emits a migration warning and is omitted from
examples. Combining the alias with `--all-territories` is also invalid.

## API contract

The command uses `POST /v1/subscriptionIntroductoryOffers` with
`SubscriptionIntroductoryOfferCreateRequest` and returns
`SubscriptionIntroductoryOfferResponse`. OpenAPI requires the attributes and
subscription relationship but marks the territory relationship optional. This
CLI intentionally applies a stricter workflow contract because App Store
Connect rejects practical create requests that omit a territory.

Single-territory mode preserves the normalized territory relationship and an
optional subscription price-point relationship. All-territories mode preserves
the existing availability lookup, existing-offer reconciliation, per-territory
create requests, summary output, dry-run behavior, and partial-failure handling.
The lower-level API client remains unchanged because it follows the OpenAPI
schema and has callers outside this command.

`--dry-run` suppresses the create request in both territory modes and prints the
same summary shape. Single-territory mode reports `total: 1` with the resolved
territory and issues no requests of its own, because the territory is already
resolved locally and the availability lookup only exists to enumerate territories
for the bulk path. In this mode the `subscriptionId` summary field echoes the
selector supplied by the caller; it is not resolved or existence-checked. The
single planned create is therefore reported as `created: 1` even when an
existing offer might cause the real create to be rejected. All-territories mode
retains its existing availability and offer reconciliation reads, so its
`created` and `skipped` counts remain provider-backed.

## Compatibility and verification

Valid concrete and all-territories invocations keep their existing request and
output behavior. Missing, blank, conflicting, and invalid selectors return exit
code 2 with empty stdout and one precise stderr diagnostic before client setup.
The deprecated ALL alias remains accepted with a direct replacement warning.

RED-GREEN coverage includes every validation edge, canonicalization, the
single-create relationship payload, bulk payloads, the alias transition, live
help, generated command documentation, and built-binary stdout, stderr, and
exit behavior. No live mutation is needed because the provider failure is
reproduced and the exact request schema is available offline.

Removing `--territory ALL` immediately was rejected because it would break a
stable spelling. Allowing the provider to reject missing selectors was rejected
because it spends authentication and network work on a locally detectable
invalid workflow.
