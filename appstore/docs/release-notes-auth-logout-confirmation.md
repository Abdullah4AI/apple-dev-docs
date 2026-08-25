# Auth logout confirmation migration

`asc auth logout` now accepts `--confirm` so scripts can migrate before
confirmation becomes mandatory in 5.0.0. New invocations should name the target
explicitly:

```bash
asc auth logout --name "PROFILE" --confirm
asc auth logout --all --confirm
```

During the compatibility window, existing logout calls still remove credentials
but print this warning to stderr:

```text
Warning: auth logout without --confirm is deprecated and will be rejected in 5.0.0; pass --confirm to acknowledge credential removal.
```

Bare `asc auth logout` continues to target all stored App Store Connect
credentials. Stray positional arguments and an explicit `--confirm=false` are
now rejected before any credential store is changed.
