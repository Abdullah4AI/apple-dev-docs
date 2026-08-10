# Workflow step retry and timeout

## Placement and invocation

This change extends the existing `.asc/workflow.json` long-form `run` step. It
does not add a command or flag, and it does not change registry placement.
`asc workflow validate`, `asc workflow run`, `--dry-run`, and `--resume` keep
their current invocation shapes.

```jsonc
{
  "name": "add_build_to_group",
  "run": "asc builds add-groups --build-id $BUILD_ID --group $GROUP_ID",
  "retry": {
    "max_attempts": 6,
    "delay": "10s"
  },
  "timeout": "2m"
}
```

`max_attempts` counts the initial execution. The delay is fixed, deterministic,
and occurs only between failed attempts. `timeout` applies independently to
each attempt, so the total step bound is the sum of attempt timeouts and retry
delays. Retry and timeout are opt-in; existing steps still execute once with no
new timeout.

## API and output contract

The runner is client-side orchestration and does not add or change an App Store
Connect request. The motivating command uses
`POST /v1/builds/{id}/relationships/betaGroups`, whose OpenAPI operation accepts
`BuildBetaGroupsLinkagesRequest`, returns `204` on success, and documents `404`
among its failures. The workflow layer intentionally does not infer HTTP
methods, status codes, idempotency, or mutation safety from a shell command.

Step command output continues to use the runner's command-output writer; the
CLI routes that writer to stderr so stdout remains one structured JSON result.
Retry diagnostics report the step, attempt number, total attempts, and next
delay on stderr. The structured step result records attempt status and failure
kind. Timeout and cancellation use stable failure kinds rather than requiring
callers to parse platform-specific process errors.

## Validation and compatibility

- `retry` is valid only on `run` steps and requires `max_attempts` from 2 through
  100 plus a positive, bounded Go duration in `delay`.
- `timeout` is valid only on `run` steps and must be a positive, bounded Go
  duration.
- Workflow-call steps reject both fields with an exact
  `workflows.<name>.steps[<index>]...` path.
- Bare string steps remain supported and keep single-attempt behavior.
- `before_all`, `after_all`, and `error` remain string hooks without retry or
  timeout configuration. They still honor caller cancellation, including
  process-tree termination.
- This is additive and requires no lifecycle migration or deprecation.

## Execution, state, and failure behavior

Each attempt receives a fresh output buffer. Declared outputs are extracted and
persisted only after a command exits successfully; output from failed attempts
cannot become step output. Output-extraction failures are not retried because a
successful command may already have performed a mutation. They are also marked
terminal and cannot be resumed automatically; the operator must inspect any
side effects before deciding whether to start a new run.

Caller cancellation stops immediately and never waits for or starts another
attempt. A timeout terminates the shell process tree before the runner records a
`timeout` attempt. On Unix, each command runs in its own process group; on
Windows, the runner uses the platform process-tree termination path. Hooks use
the same cancellation-aware command runner. A timeout-only failure is persisted
as terminal and cannot be resumed, because local termination does not prove a
remote mutation was rejected. Configuring `retry` is the explicit repeat-safety
signal that keeps retry-plus-timeout failures resumable.

Run state records attempt diagnostics for configured steps, including failed
attempts, and explicitly records whether the step enabled retry. A diagnostic-only
failed record is not a resume checkpoint. Resume requires a persisted successful
step or hook, or a retry-enabled failed step; it skips successes, restores their
outputs, and re-executes the failed step. The previous attempt history remains
available in state and the resumed result. Terminal output-extraction and
timeout-only failures reject `--resume` without executing any workflow command.
Omitted policies remain disabled, while explicit `retry: null` or `timeout: null`
is rejected with the step field path.

## RED-GREEN and verification

Focused tests cover validation paths, eventual success, exhaustion, timeout,
caller cancellation, process-tree termination, clean output capture, hook
ordering, and resume behavior. A built CLI fixture workflow verifies JSON-only
stdout, retry diagnostics on stderr, persisted state, timeout failure shape, and
successful resume. The repository gates are `make build`, `make format`,
`make check-docs`, `make lint`, and `ASC_BYPASS_KEYCHAIN=1 make test`.

No live mutation is appropriate: the behavior is local process orchestration,
and the user did not authorize mutating the documented Apple relationship.
Read-only verification is limited to current CLI help and the checked-in
OpenAPI contract.

## Residual risk

A timeout proves only that the local process tree was terminated. A remote
service may have accepted a mutating request before the deadline even though
the command returned no success result. A configured retry can therefore replay
that mutation. Operators must limit retry policies to commands whose duplicate
execution is acceptable or whose remote operation is idempotent.

## Alternatives

An exponential multiplier, jitter, exit-code filters, or HTTP-aware policy
would enlarge the schema and make timing or mutation behavior less predictable.
A fixed required delay is enough for eventually consistent endpoints and is
easy to audit. Applying policies to workflow-call steps or hooks would require
new nested state and hook schemas; rejecting those placements keeps this change
focused while still making every contained `run` step configurable.
