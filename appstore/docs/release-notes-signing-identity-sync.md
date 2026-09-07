# Signing identity sync release note

> Status: the legacy password sources described below were removed in 5.0.0.
> `--password` is no longer registered and `ASC_MATCH_PASSWORD` is no longer
> read; use `--password-file` or `ASC_SIGNING_SYNC_PASSWORD`. See
> `migrate-to-5-0.mdx`.

`asc signing sync push` can now pair a local PKCS#12 or RSA/EC private key with
the certificate embedded in the selected provisioning profile, then store one
canonical encrypted identity for other agents and CI workers. New identity
envelopes use scrypt plus authenticated metadata, while existing
certificate/profile repository behavior stays compatible.

Use `--password-file` or `ASC_SIGNING_SYNC_PASSWORD` for the signing repository
password. The older `--password` flag and `ASC_MATCH_PASSWORD` environment
variable remain available through 4.x with a deprecation warning and will be
rejected in 5.0.0.
