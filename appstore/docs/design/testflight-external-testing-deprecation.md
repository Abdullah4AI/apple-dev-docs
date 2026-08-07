# TestFlight external-testing deprecation

## Command and API contract

`asc testflight distribution edit` is the canonical command for updating a
build beta detail. Its supported PATCH shape is:

```console
asc testflight distribution edit --id "DETAIL_ID" --auto-notify
```

The offline App Store Connect OpenAPI snapshot defines
`PATCH /v1/buildBetaDetails/{id}` with
`BuildBetaDetailUpdateRequest.data.attributes.autoNotifyEnabled` as its only
update attribute. The response model still includes `internalBuildState` and
`externalBuildState`; those fields remain available when decoding API
responses.

Release 0.35.3 exposed `--external-testing`, translating its boolean value to
`externalBuildState`. Removing that stable parser surface immediately would
break existing invocations without the repository's required deprecation
window. The flag therefore remains recognized and is marked deprecated in
help, but every invocation fails with usage exit code 2 before client creation
or HTTP. Stdout remains empty; stderr contains a deprecation warning followed
by value-specific migration guidance.

## Migration

The old boolean does not identify a beta group, cannot decide whether review
submission is appropriate, and cannot identify which existing group
assignments should be removed. Callers must use the explicit distribution
operations instead:

```console
asc builds add-groups --build-id "BUILD_ID" --group "GROUP_ID" --submit --confirm
asc builds remove-groups --build-id "BUILD_ID" --group "GROUP_ID" --confirm
```

`--external-testing=true` points to `builds add-groups`; `false` points to
`builds remove-groups`. Supplying `--auto-notify` together with the deprecated
flag still fails closed rather than sending a partial PATCH.

## Verification and alternatives

Transition tests cover both boolean values, the mixed-flag case, warning and
error text, empty stdout, and the no-client/no-transport boundary. The existing
HTTP contract test continues to assert that `--auto-notify` sends a PATCH whose
attributes contain only `autoNotifyEnabled`. Built-binary checks cover help,
invalid boolean parsing, stdout, stderr, and exit codes.

Immediate removal was rejected because the flag shipped as stable. Continuing
to translate the boolean into `externalBuildState` was rejected because the
update schema does not allow it. Guessing group assignment or review behavior
was rejected because a boolean lacks the inputs needed to do either safely.
