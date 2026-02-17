# Apple Developer Documentation CLI

Search Apple Developer Documentation, frameworks, APIs, and WWDC videos directly from the terminal. Built for AI agents and developers.

- 1,267 WWDC sessions indexed locally (2014-2025)
- Direct integration with developer.apple.com
- Zero external dependencies
- Score-based search across all indexed sessions
- Topic and year filtering with 20 categories

## Quick Start

No setup required. Works out of the box with Node.js.

```bash
# Search documentation
node cli.js search "SwiftUI animation"

# Search WWDC sessions
node cli.js wwdc-search "swift concurrency"

# Get video details
node cli.js wwdc-video 2024-10169
```

## Commands

### Documentation

| Command | Description |
|---------|-------------|
| `search "query"` | Search Apple Developer Documentation |
| `symbols "UIView"` | Search framework classes, structs, protocols |
| `doc "/path/to/doc"` | Get detailed documentation by path |

### API Exploration

| Command | Description |
|---------|-------------|
| `apis "UIViewController"` | Find related APIs |
| `platform "UIScrollView"` | Check platform/version compatibility |
| `similar "UIPickerView"` | Find recommended alternatives |

### Technology Browsing

| Command | Description |
|---------|-------------|
| `tech` | List all Apple technologies |
| `overview "SwiftUI"` | Get technology overview guide |
| `samples "SwiftUI"` | Find sample code projects |
| `updates` | Latest documentation updates |

### WWDC Videos

| Command | Description |
|---------|-------------|
| `wwdc-search "async"` | Search WWDC sessions |
| `wwdc-video 2024-100` | Get video details and transcript |
| `wwdc-topics` | List 20 topic categories |
| `wwdc-topic "swift"` | List videos by topic |
| `wwdc-years` | List years with video counts |
| `wwdc-year 2025` | List all videos for a year |

## Options

| Option | Description |
|--------|-------------|
| `--limit <n>` | Limit number of results |
| `--year <YYYY>` | Filter WWDC by year |
| `--topic <slug>` | Filter WWDC by topic |
| `--no-transcript` | Skip transcript for WWDC videos |

## WWDC Index

The skill includes a pre-built index of 1,267 WWDC sessions with titles, durations, and topic classifications. To rebuild:

```bash
node build-wwdc-index.js
```

This fetches all session data directly from developer.apple.com.

### Coverage

| Year | Sessions |
|------|----------|
| 2025 | 122 |
| 2024 | 123 |
| 2023 | 181 |
| 2022 | 185 |
| 2021 | 207 |
| 2020 | 209 |
| 2019 | 153 |
| 2018 | 18 |
| 2017 | 36 |
| 2016 | 16 |
| 2015 | 11 |
| 2014 | 6 |

## How It Works

Documentation searches query developer.apple.com directly. WWDC data is indexed locally for fast offline search. Individual video detail pages are fetched live from Apple when requested, providing descriptions, related sessions, and transcripts.

All data comes from developer.apple.com. No third-party packages or external APIs.

## License

MIT
