#!/usr/bin/env python3
"""
Daily analytics snapshot for App Store Connect CLI.

Captures GitHub stars, traffic (views/clones/referrers), and release
download counts. Stores one JSON file per day in analytics/data/.

GitHub only retains traffic data for 14 days, so this script should
run daily via GitHub Actions to preserve historical data.

Usage:
    python3 analytics/snapshot.py
"""

import json
import os
import subprocess
import sys
from datetime import datetime, timezone


REPO = "rudrankriyam/app-store-connect-cli"
DATA_DIR = os.path.join(os.path.dirname(__file__), "data")


def gh_api(path):
    result = subprocess.run(
        ["gh", "api", f"repos/{REPO}/{path}"],
        capture_output=True, text=True,
    )
    if result.returncode != 0:
        print(f"  Warning: gh api {path} failed: {result.stderr.strip()}", file=sys.stderr)
        return {}
    return json.loads(result.stdout)


def gh_api_paginate(path):
    result = subprocess.run(
        ["gh", "api", f"repos/{REPO}/{path}", "--paginate"],
        capture_output=True, text=True,
    )
    if result.returncode != 0:
        print(f"  Warning: gh api {path} (paginate) failed: {result.stderr.strip()}", file=sys.stderr)
        return []
    return json.loads(result.stdout)


def get_star_count():
    result = subprocess.run(
        ["gh", "repo", "view", f"{REPO}", "--json", "stargazerCount", "--jq", ".stargazerCount"],
        capture_output=True, text=True,
    )
    return int(result.stdout.strip()) if result.returncode == 0 else 0


def collect_snapshot():
    now = datetime.now(timezone.utc)
    today = now.strftime("%Y-%m-%d")

    print(f"Collecting snapshot for {today}...")

    stars = get_star_count()
    print(f"  Stars: {stars}")

    views = gh_api("traffic/views")
    clones = gh_api("traffic/clones")
    referrers = gh_api("traffic/popular/referrers")
    paths = gh_api("traffic/popular/paths")

    print(f"  Views (14d): {views.get('count', 'N/A')}")
    print(f"  Clones (14d): {clones.get('count', 'N/A')}")

    releases_raw = gh_api_paginate("releases")
    releases = []
    total_downloads = 0
    for release in releases_raw:
        dl = sum(a["download_count"] for a in release.get("assets", []))
        total_downloads += dl
        releases.append({
            "tag": release.get("tag_name"),
            "published_at": release.get("published_at"),
            "assets": [
                {"name": a["name"], "download_count": a["download_count"]}
                for a in release.get("assets", [])
            ],
            "total_downloads": dl,
        })

    print(f"  Releases: {len(releases)}")
    print(f"  Total downloads: {total_downloads}")

    # Extract today's traffic from the daily arrays
    today_views = 0
    today_views_uniques = 0
    for entry in views.get("views", []):
        if entry["timestamp"].startswith(today):
            today_views = entry["count"]
            today_views_uniques = entry["uniques"]

    today_clones = 0
    today_clones_uniques = 0
    for entry in clones.get("clones", []):
        if entry["timestamp"].startswith(today):
            today_clones = entry["count"]
            today_clones_uniques = entry["uniques"]

    snapshot = {
        "date": today,
        "timestamp": now.isoformat(),
        "stars": stars,
        "today": {
            "views": today_views,
            "views_uniques": today_views_uniques,
            "clones": today_clones,
            "clones_uniques": today_clones_uniques,
        },
        "traffic_14d": {
            "views": views,
            "clones": clones,
            "referrers": referrers,
            "paths": paths,
        },
        "total_downloads": total_downloads,
        "releases": releases,
    }

    return snapshot


def load_history():
    """Load all previous snapshots to build a summary timeline."""
    history = []
    if not os.path.isdir(DATA_DIR):
        return history
    for fname in sorted(os.listdir(DATA_DIR)):
        if fname.endswith(".json"):
            with open(os.path.join(DATA_DIR, fname)) as f:
                try:
                    history.append(json.load(f))
                except json.JSONDecodeError:
                    pass
    return history


def save_summary(history):
    """Save a lightweight timeline.json for easy charting."""
    timeline = []
    prev_downloads = 0
    prev_stars = 0
    for snap in history:
        dl = snap.get("total_downloads", 0)
        stars = snap.get("stars", 0)
        entry = {
            "date": snap["date"],
            "stars": stars,
            "stars_delta": stars - prev_stars if prev_stars else 0,
            "total_downloads": dl,
            "downloads_delta": dl - prev_downloads if prev_downloads else 0,
            "views": snap.get("today", {}).get("views", 0),
            "views_uniques": snap.get("today", {}).get("views_uniques", 0),
            "clones": snap.get("today", {}).get("clones", 0),
            "clones_uniques": snap.get("today", {}).get("clones_uniques", 0),
        }
        timeline.append(entry)
        prev_downloads = dl
        prev_stars = stars

    path = os.path.join(DATA_DIR, "timeline.json")
    with open(path, "w") as f:
        json.dump(timeline, f, indent=2)
    print(f"  Saved timeline to {path} ({len(timeline)} days)")


def main():
    os.makedirs(DATA_DIR, exist_ok=True)

    snapshot = collect_snapshot()

    path = os.path.join(DATA_DIR, f"{snapshot['date']}.json")
    with open(path, "w") as f:
        json.dump(snapshot, f, indent=2)
    print(f"  Saved snapshot to {path}")

    history = load_history()
    save_summary(history)

    print("Done.")


if __name__ == "__main__":
    main()
