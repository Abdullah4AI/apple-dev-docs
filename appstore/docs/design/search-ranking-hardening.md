# Search ranking hardening

## Placement and current behavior

`asc search` remains in `internal/cli/search` and stays registered as the existing
top-level utility command. Its invocation and help remain unchanged:

```text
asc search [flags] <query>
```

The current scorer treats any query term of three or more characters as a match
when it appears anywhere inside an indexed token. That makes `ship` match
`relationships`. It also has no first-class notion of canonical workflows, so
deep resource commands can outrank `asc publish appstore` for release-oriented
queries.

## Proposed behavior

- Match indexed fields by exact normalized tokens plus conservative singular and
  plural stemming.
- Match hyphenated token components independently, so `app` still matches
  `app-store-releases` without restoring arbitrary substring behavior.
- Apply deterministic intent boosts to canonical publish workflows when a query
  contains both a shipping action and the corresponding App Store or TestFlight
  subject.
- Keep fuzzy typo matching as the fallback only when a term has no direct token
  or stem match.

This change is local-only and does not call an App Store Connect endpoint.

## Compatibility and output

Flags, accepted values, output formats, stdout/stderr placement, exit codes, and
the JSON response schema remain unchanged. Result ordering, scores, and `matched`
reasons intentionally change. Existing scripts must not treat scores as stable
across CLI versions.

## Tests and verification

RED-GREEN coverage will assert that:

- `ship` does not match the `relationships` token;
- singular queries still match plural command tokens;
- `asc search "ship app"` ranks `asc publish appstore` first;
- `asc search "release app"` includes and prioritizes the canonical publish
  workflow within a five-result limit;
- `asc search "upload build app"` keeps the direct `asc builds upload` command
  ahead of broader publish workflows;
- `asc search "app review"` does not apply a publish-workflow boost to the
  generic review intent;
- the built binary emits valid JSON to stdout, nothing to stderr, and exits zero
  for both natural-language queries.

Focused package tests, adjacent CLI tests, a `/tmp/asc` build, and the repository
validation gate cover the remaining regression surface. No live API call is
needed because ranking is deterministic and offline.

## Alternatives

Blacklisting `relationships` would fix only the reported collision and leave all
other substring false positives intact. Penalizing deep command paths alone would
improve some ordering but would neither remove the false match nor reliably route
shipping language to the documented canonical workflow. Exact token/stem matching
plus narrow intent boosts fixes both causes without changing the command surface.
