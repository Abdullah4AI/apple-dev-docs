# Signing sync password rotation

## Problem

The encrypted signing repository accepts a repository password when assets are
pushed or pulled, but changing that password currently requires a manual
decrypt-and-republish workflow. That workflow is easy to interrupt, can omit
artifacts, and does not account for private identities whose normalized
PKCS#12 payload is itself protected by the repository password.

## Command

Add a focused command:

```bash
asc signing sync rotate-password \
  --repo git@github.com:team/certs.git \
  --password-file ~/.config/asc/signing-sync-password \
  --new-password-file ~/.config/asc/signing-sync-password-next \
  --confirm
```

The command also accepts the existing `--branch` and output flags. Both secret
files are required. They must be distinct protected regular files, and their
resolved values must differ. The command does not accept inline passwords or
environment fallbacks so that two long-lived secrets cannot be confused or
exposed in process arguments.

`--confirm` is required because a successful rotation makes the previous
password unusable for the branch head. The operator must distribute the new
secret before dependent jobs pull again.

## Transaction boundary

Rotation happens in one temporary clone and one Git commit:

1. Validate flags and both protected password files before cloning.
2. Clone the selected branch without creating it.
3. Enumerate every encrypted artifact and apply the existing path, file-count,
   cumulative-size, envelope, metadata, identity-context, profile-binding, and
   certificate-validity checks using the current password.
4. Re-encrypt every artifact with a fresh salt and nonce. Versioned artifacts
   retain their authenticated metadata. Legacy artifacts retain their legacy
   envelope so rotation does not silently change their format.
5. Decode each normalized PKCS#12 identity with the current password and
   re-encode it with the new password before encrypting its outer envelope.
6. Re-read and validate the complete rewritten repository with the new
   password.
7. Commit and push all rewritten artifacts together.

Any failure before the Git push discards the temporary clone. No partially
rotated repository is published. Git branch protection or a concurrent remote
update may reject the final push; the remote then remains unchanged.

The commit changes the selected branch head; it does not rewrite Git history.
Anyone who has the old password and access to an earlier commit may still
decrypt those historical artifacts. A credential-compromise response must also
rotate repository access and apply the team's history-retention policy.

## Compatibility and output

Existing `push` and `pull` behavior is unchanged. The rotation receipt uses the
existing signing sync result shape with `operation: "rotate-password"`, the
sanitized repository URL, the deterministic artifact list, and identity
presence and sensitivity summaries. Passwords and decrypted bytes never appear
in stdout, stderr, commit messages, or Git configuration.

## Failure behavior

- Invalid output options, missing required flags, unsafe password files, equal
  passwords, and missing `--confirm` fail before clone or repository writes.
- A wrong current password, corrupt artifact, invalid authenticated metadata,
  stale active identity certificate, or inconsistent identity graph aborts the
  entire rotation.
- An empty encrypted repository succeeds without a commit and reports an empty
  file list.
- A failed final Git push is reported as an error and does not claim success.

## Verification

Focused coverage must prove flag validation order, protected-file handling,
legacy and versioned artifact rotation, private-identity rewrapping, whole-store
validation before mutation, new-password validation before publication, one
commit containing every artifact, sanitized output, and current-password
rejection after success.
