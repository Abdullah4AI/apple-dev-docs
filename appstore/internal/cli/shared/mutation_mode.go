package shared

// RequireConfirmUnlessDryRun enforces the apply decision for commands that turn
// file input into App Store Connect mutations: the caller must either preview
// the plan with --dry-run or accept the mutation with --confirm.
//
// When both flags are set, --dry-run wins and nothing is mutated. That matches
// the existing convention in `asc release stage`, `asc review submit`, and
// `asc subscriptions pricing equalize`, so contradictory input resolves to the
// safe side in every command instead of being rejected in some and honored in
// others.
func RequireConfirmUnlessDryRun(dryRun, confirm bool) error {
	if dryRun || confirm {
		return nil
	}
	return UsageError("--confirm is required unless --dry-run is set")
}
