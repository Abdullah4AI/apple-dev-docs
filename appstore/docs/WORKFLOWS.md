# Workflow Patterns

Use the high-level workflow surfaces deliberately:

- `asc publish appstore`: canonical App Store shipping path
- `asc publish testflight`: canonical high-level TestFlight publish path
- `asc workflow`: user-defined orchestration for repo-specific pipelines

`asc workflow` lets you compose existing `asc` commands and shell commands into
repeatable release pipelines once you know which top-level path you want.

## Verified local Xcode -> TestFlight workflow

This pattern was validated against a real app using:

- `asc builds next-build-number` to choose the next build number for a version
- `asc xcode inject` to materialize deployment metadata into generated Xcode
  plist/config files and asset paths before archiving
- `asc xcode archive` to create a deterministic `.xcarchive`
- `asc xcode export` to create a deterministic `.ipa`
- `asc publish testflight --group ... --wait` to upload, wait for processing,
  and add the build to a TestFlight group
- `--submit --confirm` on `asc publish testflight` when the target is an
  external group that should trigger beta app review submission

`asc xcode export` generates archive-specific App Store Connect export options
automatically when `--export-options` is omitted. It chooses a unique
archive-adjacent path and never overwrites an existing file. The command result
reports the exact path used. To persist a plist at a deterministic path for
inspection or reuse, generate it after the archive:

```bash
asc xcode export-options generate \
  --archive-path .asc/artifacts/App.xcarchive \
  --output-path .asc/export-options-app-store.plist \
  --overwrite
```

The standalone command defaults to `.asc/export-options-app-store.plist` and
requires `--overwrite` if that file already exists. Automatic generation is
cross-platform and only reads archive metadata. Manual signing resolution is
Darwin-only because it inspects local Xcode signing identities and provisioning
profiles.

Create `.asc/deployment.json`:

```json
{
  "values": {
    "bundle_id": "com.example.app",
    "app_name": "Example",
    "version": "",
    "build_number": ""
  },
  "outputs": [
    {
      "type": "plist",
      "path": "../Generated/Info.generated.plist",
      "values": {
        "CFBundleIdentifier": "${bundle_id}",
        "CFBundleDisplayName": "${app_name}",
        "CFBundleShortVersionString": "${version}",
        "CFBundleVersion": "${build_number}"
      }
    },
    {
      "type": "text",
      "path": "../Generated/Deployment.xcconfig",
      "contents": "PRODUCT_BUNDLE_IDENTIFIER = ${bundle_id}\nMARKETING_VERSION = ${version}\nCURRENT_PROJECT_VERSION = ${build_number}\n"
    },
    {
      "type": "copy",
      "source": "../Assets/AppIcon.appiconset/Contents.json",
      "path": "../Generated/Assets.xcassets/AppIcon.appiconset/Contents.json"
    }
  ]
}
```

Point your Xcode project at generated files such as `Generated/Info.generated.plist`
or include `Generated/Deployment.xcconfig` from the build configuration. Then run
`asc xcode inject` before archive time to fill in the release-specific values
that previously came from Fastlane scripts.

Create `.asc/workflow.json`:

```json
{
  "env": {
    "APP_ID": "1234567890",
    "PROJECT_PATH": "App.xcodeproj",
    "SCHEME": "App",
    "CONFIGURATION": "Release",
    "TESTFLIGHT_GROUP": "Beta",
    "VERSION": ""
  },
  "workflows": {
    "testflight_beta": {
      "description": "Archive, export, upload, and distribute an app to a TestFlight group.",
      "steps": [
        {
          "name": "validate_version",
          "run": "if [ -z \"$VERSION\" ]; then echo \"VERSION is required\" >&2; exit 1; fi"
        },
        {
          "name": "resolve_next_build",
          "run": "asc builds next-build-number --app \"$APP_ID\" --version \"$VERSION\" --platform IOS --initial-build-number 1 --output json",
          "outputs": {
            "BUILD_NUMBER": "$.nextBuildNumber"
          }
        },
        {
          "name": "inject_metadata",
          "run": "asc xcode inject --manifest .asc/deployment.json --set version=\"$VERSION\" --set build_number=${steps.resolve_next_build.BUILD_NUMBER} --overwrite --output json",
          "outputs": {
            "GENERATED_FILES": "$.outputs"
          }
        },
        {
          "name": "archive",
          "run": "asc xcode archive --project \"$PROJECT_PATH\" --scheme \"$SCHEME\" --configuration \"$CONFIGURATION\" --archive-path \".asc/artifacts/App-$VERSION-${steps.resolve_next_build.BUILD_NUMBER}.xcarchive\" --clean --overwrite --xcodebuild-flag=-destination --xcodebuild-flag=generic/platform=iOS --xcodebuild-flag=-allowProvisioningUpdates --xcodebuild-flag=MARKETING_VERSION=$VERSION --xcodebuild-flag=CURRENT_PROJECT_VERSION=${steps.resolve_next_build.BUILD_NUMBER} --output json",
          "outputs": {
            "ARCHIVE_PATH": "$.archive_path",
            "VERSION": "$.version",
            "BUILD_NUMBER": "$.build_number"
          }
        },
        {
          "name": "export",
          "run": "asc xcode export --archive-path ${steps.archive.ARCHIVE_PATH} --ipa-path \".asc/artifacts/App-$VERSION-${steps.archive.BUILD_NUMBER}.ipa\" --overwrite --timeout 10m --xcodebuild-flag=-allowProvisioningUpdates --output json",
          "outputs": {
            "IPA_PATH": "$.ipa_path",
            "VERSION": "$.version",
            "BUILD_NUMBER": "$.build_number"
          }
        },
        {
          "name": "publish",
          "run": "asc publish testflight --app \"$APP_ID\" --ipa ${steps.export.IPA_PATH} --group \"$TESTFLIGHT_GROUP\" --wait --poll-interval 10s --output json",
          "outputs": {
            "BUILD_ID": "$.buildId",
            "BUILD_NUMBER": "$.buildNumber"
          }
        }
      ]
    }
  }
}
```

Run it:

```bash
asc workflow validate
asc workflow run --dry-run testflight_beta VERSION:1.2.3
asc workflow run testflight_beta VERSION:1.2.3
```

Notes:

- `VERSION` must be a valid next marketing version for your app. If the latest
  App Store version is already `READY_FOR_DISTRIBUTION`, reusing that same
  version can cause App Store Connect to reject the upload.
- `TESTFLIGHT_GROUP` accepts either a beta group name or group ID.
- Add `"ASC_BYPASS_KEYCHAIN": "1"` to the top-level `env` block if you want the
  workflow to resolve credentials from environment variables or config instead
  of the macOS keychain.
- Output-producing step names only need to stay unique within workflows that
  can execute together in the same run graph. Independent workflows can reuse
  names like `archive` or `publish`.

Example:

```json
{
  "workflows": {
    "testflight_beta": {
      "steps": [
        {
          "name": "archive",
          "run": "printf '{\"buildId\":\"beta\"}'",
          "outputs": {
            "BUILD_ID": "$.buildId"
          }
        }
      ]
    },
    "appstore_release": {
      "steps": [
        {
          "name": "archive",
          "run": "printf '{\"buildId\":\"release\"}'",
          "outputs": {
            "BUILD_ID": "$.buildId"
          }
        }
      ]
    }
  }
}
```

This is valid because those workflows are independent. If a third workflow can
call both of them in the same run, the duplicate `archive` producers still need
to be renamed.
