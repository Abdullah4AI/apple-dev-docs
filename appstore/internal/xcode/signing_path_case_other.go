//go:build !darwin && !windows && !linux

package xcode

// Non-Darwin callers have explicit Windows handling in signingLexicalPathKey.
// Other supported hosts use case-sensitive lexical semantics for missing
// paths; existing aliases still go through rooted identity checks.
func signingCaseInsensitiveVolumeFor(string) (caseInsensitive, known bool) {
	return false, true
}
