#!/usr/bin/env python3
"""Protect CI runner, build, artifact, and security-check contracts."""

import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
PR_WORKFLOW = ROOT / ".github/workflows/pr-checks.yml"
MAIN_WORKFLOW = ROOT / ".github/workflows/main-branch.yml"
RELEASE_WORKFLOW = ROOT / ".github/workflows/release.yml"
WEBSITE_WORKFLOW = ROOT / ".github/workflows/website-checks.yml"
GOVULNCHECK_WORKFLOW = ROOT / ".github/workflows/govulncheck.yml"
DEPENDABOT_CONFIG = ROOT / ".github/dependabot.yml"
MAKEFILE = ROOT / "Makefile"


def assert_go_toolchain_source() -> None:
    workflow_dir = ROOT / ".github/workflows"
    workflows = sorted([*workflow_dir.glob("*.yml"), *workflow_dir.glob("*.yaml")])
    assert_go_toolchain_workflows([(path, path.read_text()) for path in workflows])


def assert_go_toolchain_workflows(workflows: list[tuple[Path, str]]) -> None:
    setup_go_count = 0

    for path, workflow in workflows:
        assert "go-version:" not in workflow, f"{path}: source Go versions from go.mod"
        lines = workflow.splitlines()
        for index, line in enumerate(lines):
            uses = re.match(r"^(\s*)(-\s*)?uses:\s*(.*?)\s*$", line)
            if not uses:
                continue
            uses_value = normalize_yaml_scalar(uses.group(3))
            if not uses_value.startswith("actions/setup-go@") or uses_value == "actions/setup-go@":
                continue

            setup_go_count += 1
            uses_indent = len(uses.group(1)) + len(uses.group(2) or "")
            assert setup_go_step_uses_go_mod(lines[index + 1 :], uses_indent), (
                f"{path}: every setup-go step must source go.mod"
            )

    assert setup_go_count > 0, "expected at least one setup-go step"


def normalize_yaml_scalar(value: str) -> str:
    scalar = value.strip()
    quote = ""
    escaped = False
    index = 0

    while index < len(scalar):
        char = scalar[index]
        if quote == '"':
            if escaped:
                escaped = False
            elif char == "\\":
                escaped = True
            elif char == quote:
                quote = ""
        elif quote == "'":
            if char == quote:
                if index + 1 < len(scalar) and scalar[index + 1] == quote:
                    index += 1
                else:
                    quote = ""
        elif char in {'"', "'"}:
            quote = char
        elif char == "#" and (index == 0 or scalar[index - 1].isspace()):
            scalar = scalar[:index].rstrip()
            break
        index += 1

    if len(scalar) >= 2 and scalar[0] == scalar[-1] and scalar[0] in {'"', "'"}:
        scalar = scalar[1:-1]
        if value.strip().startswith("'"):
            scalar = scalar.replace("''", "'")
    return scalar


def setup_go_step_uses_go_mod(lines: list[str], uses_indent: int) -> bool:
    for index, line in enumerate(lines):
        if line.strip() and len(line) - len(line.lstrip()) < uses_indent:
            return False

        match = re.match(r"^(\s*)with:\s*$", line)
        if not match or len(match.group(1)) != uses_indent:
            continue

        input_indent = 0
        for setting in lines[index + 1 :]:
            if not setting.strip() or setting.lstrip().startswith("#"):
                continue
            setting_indent = len(setting) - len(setting.lstrip())
            if setting_indent <= uses_indent:
                break
            if input_indent == 0:
                input_indent = setting_indent
            if setting_indent != input_indent:
                continue

            go_version_file = re.match(r"^\s*go-version-file:\s*(.*?)\s*$", setting)
            if go_version_file and normalize_yaml_scalar(go_version_file.group(1)) == "go.mod":
                return True
        return False
    return False


def assert_go_toolchain_source_accepts_normalized_scalars() -> None:
    valid_workflow = """jobs:
  test:
    steps:
      - uses: "actions/setup-go@v6"
        with:
          go-version-file: "go.mod" # quoted source of truth
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod # source of truth
"""
    assert_go_toolchain_workflows([(Path("normalized-scalars.yml"), valid_workflow)])


def assert_go_toolchain_source_rejects_step_local_violations() -> None:
    invalid_workflows = {
        "missing-before-valid.yml": """jobs:
  test:
    steps:
      - uses: "actions/setup-go@v6"
      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
""",
        "comment-only.yml": """jobs:
  test:
    steps:
      - uses: actions/setup-go@v6
        with:
          go-version-file: # go.mod
""",
    }

    for filename, workflow in invalid_workflows.items():
        path = Path(filename)
        expected = f"{path}: every setup-go step must source go.mod"
        try:
            assert_go_toolchain_workflows([(path, workflow)])
        except AssertionError as error:
            if str(error) != expected:
                raise
        else:
            raise AssertionError(f"{filename} must fail: {expected}")


def assert_govulncheck_version_source() -> None:
    makefile = MAKEFILE.read_text()
    workflow = GOVULNCHECK_WORKFLOW.read_text()

    assert re.search(r"^GOVULNCHECK_VERSION \?= v\d+\.\d+\.\d+$", makefile, re.MULTILINE)
    assert "golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)" in makefile
    assert "run: make install-govulncheck" in workflow
    assert "golang.org/x/vuln/cmd/govulncheck@" not in workflow
    assert "govulncheck@latest" not in makefile


def job_block(workflow: str, job: str) -> str:
    marker = f"  {job}:\n"
    start = workflow.find(marker)
    if start < 0:
        raise AssertionError(f"missing job {job!r}")

    end = len(workflow)
    offset = start + len(marker)
    for line in workflow[offset:].splitlines(keepends=True):
        content = line.rstrip("\r\n")
        if content.startswith("  ") and not content.startswith("    ") and content.endswith(":"):
            end = offset
            break
        offset += len(line)
    return workflow[start:end]


def assert_optimized_workflow(path: Path, test_job: str) -> None:
    assert_optimized_workflow_text(path, path.read_text(), test_job)


def assert_optimized_workflow_text(path: Path, workflow: str, test_job: str) -> None:
    assert "actions/upload-artifact" not in workflow, f"{path}: development artifacts must not be uploaded"
    changes = job_block(workflow, "changes")
    assert "scope: ${{ steps.scope.outputs.scope }}" in changes
    assert "website_affected: ${{ steps.scope.outputs.website_affected }}" in changes
    assert "python3 scripts/ci_change_scope.py --github-output" in changes
    assert "git diff --name-only --no-renames" in changes
    for guarded_path in (
        ".github/workflows/*",
        "scripts/ci_change_scope.py",
        "scripts/test_ci_change_scope.py",
        "scripts/test_ci_workflows.py",
    ):
        assert guarded_path in changes
    assert changes.index("force_full=false") < changes.index(
        "python3 scripts/ci_change_scope.py --github-output"
    )
    assert 'if [ "$force_full" = true ]; then' in changes
    assert "website_affected=true" in changes
    assert "wall|docs|website|telemetry|full" in changes
    assert "invalid CI scope" in changes

    wall_only = job_block(workflow, "wall-only-check")
    assert "runs-on: ubuntu-latest" in wall_only
    assert "make check-wall-of-apps" in wall_only, f"{path}: wall-only-check must validate the Wall source"
    assert "runs-on: ubuntu-latest" in job_block(workflow, "format-and-lint")
    quality = job_block(workflow, "quality-checks")
    assert "runs-on: ubuntu-latest" in quality
    assert "python3 scripts/test_ci_change_scope.py" in quality
    assert "contains(fromJSON('[\"telemetry\", \"full\"]'), needs.changes.outputs.scope)" in quality
    # Agents.md: formatting, documentation, and lint must keep running on PR and main.
    for command in (
        "make format-check",
        "python3 scripts/test_check_docs.py",
        "make check-docs",
        "make check-wall-of-apps",
        "make lint",
    ):
        assert command in quality, f"{path}: quality-checks must run {command!r}"
    # Every test job runs on ubuntu, so darwin- and windows-gated sources and tests
    # are never type-checked unless CI vets them explicitly. golangci-lint covers
    # GOOS=linux with tests enabled, so only the two absent platforms need a pass.
    for goos in ("darwin", "windows"):
        assert f"GOOS={goos} go vet ./..." in quality, (
            f"{path}: quality-checks must type-check {goos}-gated code"
        )
    website = job_block(workflow, "website-checks")
    assert "uses: ./.github/workflows/website-checks.yml" in website
    assert "needs.changes.outputs.website_affected == 'true'" in website
    tests = job_block(workflow, test_job)
    assert "runs-on: ubuntu-latest" in tests
    assert "needs.changes.outputs.scope == 'full'" in tests
    # Agents.md: the Go test suite must keep running, over the whole module, with the
    # keychain bypass that stops runners from prompting or leaking host profile state.
    assert "python3 scripts/go_test_shard.py" in tests, f"{path}: {test_job} must run the sharded Go test suite"
    assert "--packages ./..." in tests, f"{path}: {test_job} must cover every package"
    assert "go test" in tests, f"{path}: {test_job} must invoke go test"
    assert "ASC_BYPASS_KEYCHAIN=1" in tests, f"{path}: {test_job} must bypass the keychain"

    build_platforms = job_block(workflow, "build-platforms")
    assert "needs.changes.outputs.scope == 'full'" in build_platforms
    for runner in ("macos-latest", "ubuntu-latest", "windows-latest"):
        assert f"runner: {runner}" in build_platforms, f"{path}: missing native build runner {runner}"
    assert "go test -short ./internal/screenshots" in build_platforms, f"{path}: missing Darwin-only tests"
    for arch in ("amd64", "arm64"):
        command = f"CGO_ENABLED=1 GOOS=darwin GOARCH={arch} go build"
        assert command in build_platforms, f"{path}: missing cgo-enabled Darwin {arch} build"
        assert f"asc_dev_macos_{arch}" in build_platforms
    for os_name, arch in (("linux", "amd64"), ("linux", "arm64"), ("windows", "amd64")):
        command = f"CGO_ENABLED=0 GOOS={os_name} GOARCH={arch} go build"
        assert command in build_platforms, f"{path}: missing cgo-disabled {os_name} {arch} build"

    ordinary_build = job_block(workflow, "ordinary-build")
    assert "needs.changes.outputs.scope == 'telemetry'" in ordinary_build
    assert "runner: ubuntu-latest" in ordinary_build
    assert "runner: macos-latest" in ordinary_build
    assert "run: go build ." in ordinary_build
    assert "go build ./..." not in ordinary_build

    build = job_block(workflow, "build")
    assert "needs: [changes, build-platforms, ordinary-build]" in build
    assert "if: always()" in build
    assert "needs.build-platforms.result" in build
    assert "needs.ordinary-build.result" in build


def assert_optimized_workflow_rejects_weakened_checks() -> None:
    """Prove the guard fails when a required check is deleted from PR or main CI."""
    for path, test_job in ((PR_WORKFLOW, "unit-test-shards"), (MAIN_WORKFLOW, "test-shards")):
        workflow = path.read_text()
        for command in (
            "make format-check",
            "python3 scripts/test_check_docs.py",
            "make check-docs",
            "make check-wall-of-apps",
            "make lint",
            "GOOS=darwin go vet ./...",
            "GOOS=windows go vet ./...",
            "python3 scripts/go_test_shard.py",
            "--packages ./...",
            "ASC_BYPASS_KEYCHAIN=1",
            "CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build",
            "CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build",
            "CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build",
            "CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build",
            "CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build",
        ):
            assert command in workflow, f"{path}: expected to find {command!r}"
            weakened = workflow.replace(command, "true")
            try:
                assert_optimized_workflow_text(path, weakened, test_job)
            except AssertionError:
                continue
            raise AssertionError(f"{path}: guard accepts CI with {command!r} removed")


def run_security_target(path: str) -> subprocess.CompletedProcess[str]:
    make = shutil.which("make")
    if make is None:
        raise AssertionError("make is required to test Makefile contracts")

    env = os.environ.copy()
    env["PATH"] = path
    return subprocess.run(
        [
            make,
            "--no-print-directory",
            "-f",
            str(MAKEFILE),
            "VERSION=test",
            "COMMIT=test",
            "DATE=test",
            "GOBIN=/tmp/asc-test-bin",
            "GO_TOOLCHAIN_VERSION=test",
            "security",
        ],
        cwd=ROOT,
        env=env,
        capture_output=True,
        text=True,
        check=False,
    )


def assert_security_target_contract() -> None:
    with tempfile.TemporaryDirectory() as tmpdir:
        fake_go = Path(tmpdir) / "go"
        fake_go.write_text("#!/bin/sh\nexit 0\n")
        fake_go.chmod(0o755)

        missing = run_security_target(tmpdir)
        assert missing.returncode == 0, missing.stderr
        assert "Install gosec for security checks" in missing.stdout

        fake_gosec = Path(tmpdir) / "gosec"
        fake_gosec.write_text("#!/bin/sh\necho scanner-result >&2\nexit 23\n")
        fake_gosec.chmod(0o755)

        finding = run_security_target(tmpdir)
        assert finding.returncode != 0, (
            "make security must fail when gosec fails; "
            f"stdout:\n{finding.stdout}\nstderr:\n{finding.stderr}"
        )
        assert "scanner-result" in finding.stderr
        assert "Install gosec for security checks" not in finding.stdout

        fake_gosec.write_text("#!/bin/sh\nexit 0\n")
        success = run_security_target(tmpdir)
        assert success.returncode == 0, success.stderr
        assert "Install gosec for security checks" not in success.stdout


def main() -> None:
    assert_go_toolchain_source_accepts_normalized_scalars()
    assert_go_toolchain_source_rejects_step_local_violations()
    assert_go_toolchain_source()
    assert_govulncheck_version_source()
    assert_optimized_workflow(PR_WORKFLOW, "unit-test-shards")
    assert_optimized_workflow(MAIN_WORKFLOW, "test-shards")
    assert_optimized_workflow_rejects_weakened_checks()

    pr = PR_WORKFLOW.read_text()
    for required_job in ("format-and-lint", "unit-tests", "build"):
        assert "if: always()" in job_block(pr, required_job), f"required job {required_job} must always resolve"
    quality_gate = job_block(pr, "format-and-lint")
    assert "needs: [changes, wall-only-check, quality-checks, website-checks]" in quality_gate
    assert "needs.website-checks.result" in quality_gate

    website = WEBSITE_WORKFLOW.read_text()
    assert "workflow_call:" in website
    assert "runs-on: ubuntu-latest" in job_block(website, "website")
    assert "make check-website-docs" in website

    main = MAIN_WORKFLOW.read_text()
    assert "git diff-tree --no-commit-id --name-only --no-renames -r" in main
    main_windows = job_block(main, "telemetry-windows-tests")
    assert "[\"telemetry\", \"full\"]" in main_windows
    assert "needs.telemetry-windows-tests.result" in job_block(main, "test")

    release = RELEASE_WORKFLOW.read_text()
    assert "actions/upload-artifact" in release, "release workflow must retain official artifact publication"

    subprocess.run(
        ["go", "test", ".", "-run", "^TestReleaseRehearsalWorkflowContract", "-count=1"],
        cwd=ROOT,
        check=True,
    )
    subprocess.run([sys.executable, str(ROOT / "scripts/test_release_rehearsal.py")], check=True)

    dependabot = DEPENDABOT_CONFIG.read_text()
    assert dependabot == """version: 2

updates:
  - package-ecosystem: \"gomod\"
    directory: \"/\"
    schedule:
      interval: \"weekly\"

  - package-ecosystem: \"github-actions\"
    directory: \"/\"
    schedule:
      interval: \"weekly\"
""", "Dependabot must monitor root Go modules and GitHub Actions weekly"

    assert_security_target_contract()

    print("CI workflow contracts passed")


if __name__ == "__main__":
    main()
