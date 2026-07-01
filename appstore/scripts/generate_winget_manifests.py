#!/usr/bin/env python3
from __future__ import annotations

import argparse
import re
from pathlib import Path


PACKAGE_IDENTIFIER = "Rorkai.ASC"
PACKAGE_PUBLISHER = "Rorkai"
PACKAGE_NAME = "asc"
PACKAGE_MONIKER = "asc"
PACKAGE_COMMAND = "asc"
PACKAGE_DESCRIPTION = "Fast, scriptable CLI for the App Store Connect API."
REPOSITORY_URL = "https://github.com/rorkai/App-Store-Connect-CLI"
MANIFEST_VERSION = "1.6.0"


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate WinGet manifests for an asc release.")
    parser.add_argument("--version", required=True, help="Release version, without a leading v.")
    parser.add_argument(
        "--release-dir",
        type=Path,
        default=Path("release"),
        help="Directory containing release artifacts and checksums.",
    )
    parser.add_argument(
        "--output-dir",
        type=Path,
        required=True,
        help="Directory where the WinGet manifest tree should be written.",
    )
    return parser.parse_args()


def validate_version(version: str) -> str:
    normalized = version.strip()
    if not re.fullmatch(r"\d+\.\d+\.\d+", normalized):
        raise SystemExit(f"version must be plain semver x.y.z, got {version!r}")
    return normalized


def checksum_for_asset(checksums_path: Path, asset_name: str) -> str:
    if not checksums_path.exists():
        raise SystemExit(f"missing checksums file: {checksums_path}")

    for line in checksums_path.read_text(encoding="utf-8").splitlines():
        parts = line.split()
        if len(parts) >= 2 and parts[1].lstrip("*") == asset_name:
            return parts[0].upper()
    raise SystemExit(f"{asset_name} not found in {checksums_path}")


def manifest_dir(output_dir: Path, version: str) -> Path:
    return output_dir / "manifests" / "r" / "Rorkai" / "ASC" / version


def write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content, encoding="utf-8")


def generate(version: str, release_dir: Path, output_dir: Path) -> list[Path]:
    asset_name = f"asc_{version}_windows_amd64.exe"
    checksums_path = release_dir / f"asc_{version}_checksums.txt"
    sha256 = checksum_for_asset(checksums_path, asset_name)
    installer_url = f"{REPOSITORY_URL}/releases/download/{version}/{asset_name}"

    target = manifest_dir(output_dir, version)
    version_file = target / f"{PACKAGE_IDENTIFIER}.yaml"
    locale_file = target / f"{PACKAGE_IDENTIFIER}.locale.en-US.yaml"
    installer_file = target / f"{PACKAGE_IDENTIFIER}.installer.yaml"

    write(
        version_file,
        f"""# Created with App-Store-Connect-CLI/scripts/generate_winget_manifests.py
PackageIdentifier: {PACKAGE_IDENTIFIER}
PackageVersion: {version}
DefaultLocale: en-US
ManifestType: version
ManifestVersion: {MANIFEST_VERSION}
""",
    )
    write(
        locale_file,
        f"""# Created with App-Store-Connect-CLI/scripts/generate_winget_manifests.py
PackageIdentifier: {PACKAGE_IDENTIFIER}
PackageVersion: {version}
PackageLocale: en-US
Publisher: {PACKAGE_PUBLISHER}
PackageName: {PACKAGE_NAME}
PackageUrl: {REPOSITORY_URL}
License: MIT
LicenseUrl: {REPOSITORY_URL}/blob/main/LICENSE
ShortDescription: {PACKAGE_DESCRIPTION}
Moniker: {PACKAGE_MONIKER}
Tags:
  - app-store-connect
  - app-store-connect-api
  - cli
  - ios
  - testflight
ReleaseNotesUrl: {REPOSITORY_URL}/releases/tag/{version}
ManifestType: defaultLocale
ManifestVersion: {MANIFEST_VERSION}
""",
    )
    write(
        installer_file,
        f"""# Created with App-Store-Connect-CLI/scripts/generate_winget_manifests.py
PackageIdentifier: {PACKAGE_IDENTIFIER}
PackageVersion: {version}
InstallerType: portable
Commands:
  - {PACKAGE_COMMAND}
Installers:
  - Architecture: x64
    InstallerUrl: {installer_url}
    InstallerSha256: {sha256}
ManifestType: installer
ManifestVersion: {MANIFEST_VERSION}
""",
    )
    return [version_file, locale_file, installer_file]


def main() -> None:
    args = parse_args()
    version = validate_version(args.version)
    files = generate(version, args.release_dir, args.output_dir)
    for path in files:
        print(path)


if __name__ == "__main__":
    main()
