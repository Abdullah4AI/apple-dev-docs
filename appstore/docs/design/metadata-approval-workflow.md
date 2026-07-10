# Metadata approval workflow design

Issue #1717 asks for agent-friendly metadata changes with a durable plan, explicit approval, and a guarded apply. Current `metadata apply --dry-run` computes a useful plan but does not persist a review artifact or bind approval to the exact plan that was reviewed.

## Chosen approach

Add `metadata plan`, `metadata approve`, and `metadata status` as thin artifact workflow commands around the existing metadata push/apply planner. `metadata plan` runs the same dry-run planner as `metadata apply --dry-run`, writes `.asc/metadata/review/plan.json`, and records a SHA-256 hash over a stable payload containing the normalized command options plus the planned changes. `metadata approve` reads that plan and writes `.asc/metadata/review/approved.json` with the reviewed plan hash and approved change keys. `metadata apply --review-dir ... --confirm` recomputes the plan, compares hashes, verifies all current change keys are approved, then delegates to the existing executor.

## Compatibility

Direct `metadata apply` and `metadata push` stay compatible. Approval is enforced only when `--review-dir` is supplied to `metadata apply`. Existing `--dry-run` behavior remains available for quick previews.

## Trade-offs

This first slice requires all currently planned changes to be approved before guarded apply. It supports selective approval artifacts for review/status, but partial application is not enabled yet because the current executor applies a whole computed plan. That keeps the first implementation deterministic and avoids introducing a second mutation engine.

## Verification

Tests should cover artifact writes, approval selection, status output, drift rejection before mutation, and successful guarded apply. Live verification should run the artifact workflow against one real app only after inspecting the generated plan.
