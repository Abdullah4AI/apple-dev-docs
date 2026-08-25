# Xcode Cloud status ID alias

`asc xcode-cloud status` now accepts `--id` as a deprecated compatibility alias
for `--run-id`. Existing alias calls continue after printing this warning to
stderr:

```text
Warning: `--id` is deprecated. Use `--run-id`.
```

New scripts should use `asc xcode-cloud status --run-id "BUILD_RUN_ID"`.
Supplying both `--id` and `--run-id` is a usage error, even when the values
match. The alias may be removed in the next major release after the standard
deprecation window.
