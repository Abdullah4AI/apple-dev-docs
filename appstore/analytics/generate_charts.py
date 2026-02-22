#!/usr/bin/env python3
"""
Generate sponsor-ready growth charts for App Store Connect CLI.

Usage:
    python3 analytics/generate_charts.py

Outputs PNG charts to analytics/ directory.
"""

import json
import subprocess
import sys
from collections import Counter, defaultdict
from datetime import datetime, timedelta

import matplotlib.pyplot as plt
import matplotlib.dates as mdates
import matplotlib.ticker as ticker
import numpy as np

OUTPUT_DIR = "analytics"
REPO = "rudrankriyam/app-store-connect-cli"


def fetch_stars():
    """Fetch all stargazer timestamps via gh CLI."""
    with open("/tmp/asc_stars.txt") as f:
        lines = [l.strip() for l in f if l.strip()]
    return sorted([datetime.fromisoformat(ts.replace("Z", "+00:00")) for ts in lines])


def fetch_downloads():
    """Fetch release download data via gh CLI."""
    with open("/tmp/asc_downloads.txt") as f:
        lines = [l.strip() for l in f if l.strip()]

    releases = []
    for line in lines:
        parts = line.split("\t")
        if len(parts) == 3:
            tag, published, total = parts
            releases.append({
                "tag": tag,
                "date": datetime.fromisoformat(published.replace("Z", "+00:00")),
                "downloads": int(total),
            })
    return sorted(releases, key=lambda r: r["date"])


def style_chart(ax, title, xlabel, ylabel):
    """Apply consistent styling to axes."""
    ax.set_title(title, fontsize=16, fontweight="bold", pad=15)
    ax.set_xlabel(xlabel, fontsize=12)
    ax.set_ylabel(ylabel, fontsize=12)
    ax.spines["top"].set_visible(False)
    ax.spines["right"].set_visible(False)
    ax.grid(axis="y", alpha=0.3, linestyle="--")
    ax.tick_params(labelsize=10)


def chart_cumulative_stars(star_dates):
    """Chart 1: Cumulative stars over time -- the headline growth chart."""
    fig, ax = plt.subplots(figsize=(14, 7))

    cumulative = list(range(1, len(star_dates) + 1))

    ax.fill_between(star_dates, cumulative, alpha=0.15, color="#0A84FF")
    ax.plot(star_dates, cumulative, linewidth=2.5, color="#0A84FF")

    # Annotate milestones
    milestones = [100, 250, 500, 750, 1000]
    for m in milestones:
        if m <= len(star_dates):
            idx = m - 1
            ax.annotate(
                f"{m} stars",
                xy=(star_dates[idx], m),
                xytext=(15, 15),
                textcoords="offset points",
                fontsize=9,
                fontweight="bold",
                color="#333",
                arrowprops=dict(arrowstyle="->", color="#666", lw=1.2),
                bbox=dict(boxstyle="round,pad=0.3", facecolor="white", edgecolor="#ccc"),
            )

    # Annotate current total
    ax.annotate(
        f"{len(star_dates)} stars\n(current)",
        xy=(star_dates[-1], len(star_dates)),
        xytext=(-60, 20),
        textcoords="offset points",
        fontsize=11,
        fontweight="bold",
        color="#0A84FF",
        arrowprops=dict(arrowstyle="->", color="#0A84FF", lw=1.5),
        bbox=dict(boxstyle="round,pad=0.4", facecolor="#E8F4FD", edgecolor="#0A84FF"),
    )

    # Calculate days since creation
    days_alive = (star_dates[-1] - star_dates[0]).days
    avg_per_day = len(star_dates) / max(days_alive, 1)

    # Add subtitle with key stats
    fig.text(
        0.5, 0.92,
        f"Created {star_dates[0].strftime('%b %d, %Y')}  |  "
        f"{days_alive} days  |  "
        f"avg {avg_per_day:.0f} stars/day",
        ha="center", fontsize=11, color="#666",
    )

    style_chart(ax, "App Store Connect CLI -- GitHub Stars Growth", "", "Cumulative Stars")
    ax.xaxis.set_major_formatter(mdates.DateFormatter("%b %d"))
    ax.xaxis.set_major_locator(mdates.WeekdayLocator(interval=1))
    fig.autofmt_xdate(rotation=30)
    plt.tight_layout(rect=[0, 0, 1, 0.93])
    path = f"{OUTPUT_DIR}/stars_cumulative.png"
    fig.savefig(path, dpi=200, bbox_inches="tight", facecolor="white")
    plt.close(fig)
    print(f"  Saved {path}")


def chart_stars_per_day(star_dates):
    """Chart 2: Stars per day -- shows daily momentum."""
    fig, ax = plt.subplots(figsize=(14, 6))

    day_counts = Counter(d.strftime("%Y-%m-%d") for d in star_dates)
    all_days = []
    start = star_dates[0].date()
    end = star_dates[-1].date()
    current = start
    while current <= end:
        all_days.append(current)
        current += timedelta(days=1)

    counts = [day_counts.get(d.strftime("%Y-%m-%d"), 0) for d in all_days]
    day_dts = [datetime.combine(d, datetime.min.time()) for d in all_days]

    # Color bars by intensity
    max_count = max(counts) if counts else 1
    colors = [plt.cm.Blues(0.3 + 0.7 * (c / max_count)) for c in counts]

    ax.bar(day_dts, counts, width=0.85, color=colors, edgecolor="none")

    # 3-day moving average
    if len(counts) >= 3:
        window = 3
        ma = np.convolve(counts, np.ones(window) / window, mode="valid")
        ma_dates = day_dts[window - 1:]
        ax.plot(ma_dates, ma, color="#FF3B30", linewidth=2, label=f"{window}-day moving avg", zorder=5)
        ax.legend(fontsize=10, loc="upper left")

    # Find and annotate peak day
    peak_idx = counts.index(max(counts))
    peak_date = all_days[peak_idx]
    ax.annotate(
        f"Peak: {max(counts)} stars\n{peak_date.strftime('%b %d')}",
        xy=(day_dts[peak_idx], max(counts)),
        xytext=(0, 20),
        textcoords="offset points",
        fontsize=9,
        fontweight="bold",
        ha="center",
        color="#FF3B30",
        arrowprops=dict(arrowstyle="->", color="#FF3B30", lw=1.2),
        bbox=dict(boxstyle="round,pad=0.3", facecolor="#FFF0F0", edgecolor="#FF3B30"),
    )

    style_chart(ax, "Daily GitHub Stars", "", "Stars per Day")
    ax.xaxis.set_major_formatter(mdates.DateFormatter("%b %d"))
    ax.xaxis.set_major_locator(mdates.WeekdayLocator(interval=1))
    ax.yaxis.set_major_locator(ticker.MaxNLocator(integer=True))
    fig.autofmt_xdate(rotation=30)
    plt.tight_layout()
    path = f"{OUTPUT_DIR}/stars_per_day.png"
    fig.savefig(path, dpi=200, bbox_inches="tight", facecolor="white")
    plt.close(fig)
    print(f"  Saved {path}")


def chart_cumulative_downloads(releases):
    """Chart 3: Cumulative downloads over time."""
    fig, ax = plt.subplots(figsize=(14, 7))

    dates = [r["date"] for r in releases]
    cumulative = []
    total = 0
    for r in releases:
        total += r["downloads"]
        cumulative.append(total)

    ax.fill_between(dates, cumulative, alpha=0.15, color="#34C759")
    ax.plot(dates, cumulative, linewidth=2.5, color="#34C759", marker="o", markersize=4)

    # Annotate current total
    ax.annotate(
        f"{cumulative[-1]:,} total\ndownloads",
        xy=(dates[-1], cumulative[-1]),
        xytext=(-80, 20),
        textcoords="offset points",
        fontsize=11,
        fontweight="bold",
        color="#34C759",
        arrowprops=dict(arrowstyle="->", color="#34C759", lw=1.5),
        bbox=dict(boxstyle="round,pad=0.4", facecolor="#E8FBF0", edgecolor="#34C759"),
    )

    style_chart(ax, "Cumulative Release Downloads", "", "Total Downloads")
    ax.xaxis.set_major_formatter(mdates.DateFormatter("%b %d"))
    ax.xaxis.set_major_locator(mdates.WeekdayLocator(interval=1))
    fig.autofmt_xdate(rotation=30)
    plt.tight_layout()
    path = f"{OUTPUT_DIR}/downloads_cumulative.png"
    fig.savefig(path, dpi=200, bbox_inches="tight", facecolor="white")
    plt.close(fig)
    print(f"  Saved {path}")


def chart_downloads_per_release(releases):
    """Chart 4: Downloads per release -- shows adoption velocity."""
    fig, ax = plt.subplots(figsize=(16, 6))

    # Only show releases with > 0 downloads, and limit to avoid clutter
    filtered = [r for r in releases if r["downloads"] > 0]
    if len(filtered) > 30:
        filtered = filtered[-30:]  # last 30 releases

    tags = [r["tag"] for r in filtered]
    downloads = [r["downloads"] for r in filtered]

    max_dl = max(downloads) if downloads else 1
    colors = [plt.cm.Greens(0.3 + 0.7 * (d / max_dl)) for d in downloads]

    bars = ax.bar(range(len(tags)), downloads, color=colors, edgecolor="none")

    # Label top bars
    for i, (bar, dl) in enumerate(zip(bars, downloads)):
        if dl >= max_dl * 0.5:
            ax.text(
                bar.get_x() + bar.get_width() / 2, bar.get_height() + max_dl * 0.02,
                str(dl), ha="center", fontsize=8, fontweight="bold", color="#333",
            )

    ax.set_xticks(range(len(tags)))
    ax.set_xticklabels(tags, rotation=55, ha="right", fontsize=8)

    style_chart(ax, "Downloads per Release (last 30)", "Release Version", "Downloads")
    plt.tight_layout()
    path = f"{OUTPUT_DIR}/downloads_per_release.png"
    fig.savefig(path, dpi=200, bbox_inches="tight", facecolor="white")
    plt.close(fig)
    print(f"  Saved {path}")


def chart_combined_dashboard(star_dates, releases):
    """Chart 5: Combined 4-panel dashboard for sponsors."""
    fig, axes = plt.subplots(2, 2, figsize=(18, 12))

    # -- Panel 1: Cumulative Stars --
    ax = axes[0, 0]
    cumulative_stars = list(range(1, len(star_dates) + 1))
    ax.fill_between(star_dates, cumulative_stars, alpha=0.15, color="#0A84FF")
    ax.plot(star_dates, cumulative_stars, linewidth=2, color="#0A84FF")
    style_chart(ax, f"GitHub Stars ({len(star_dates):,} total)", "", "Stars")
    ax.xaxis.set_major_formatter(mdates.DateFormatter("%b %d"))
    ax.xaxis.set_major_locator(mdates.WeekdayLocator(interval=1))
    for label in ax.get_xticklabels():
        label.set_rotation(30)
        label.set_ha("right")

    # -- Panel 2: Stars per Day --
    ax = axes[0, 1]
    day_counts = Counter(d.strftime("%Y-%m-%d") for d in star_dates)
    all_days_set = sorted(day_counts.keys())
    start = star_dates[0].date()
    end = star_dates[-1].date()
    all_days = []
    current = start
    while current <= end:
        all_days.append(current)
        current += timedelta(days=1)
    counts = [day_counts.get(d.strftime("%Y-%m-%d"), 0) for d in all_days]
    day_dts = [datetime.combine(d, datetime.min.time()) for d in all_days]
    ax.bar(day_dts, counts, width=0.85, color="#5AC8FA", edgecolor="none")
    if len(counts) >= 3:
        ma = np.convolve(counts, np.ones(3) / 3, mode="valid")
        ax.plot(day_dts[2:], ma, color="#FF3B30", linewidth=1.5, label="3-day avg")
        ax.legend(fontsize=8)
    style_chart(ax, "Stars per Day", "", "Daily Stars")
    ax.xaxis.set_major_formatter(mdates.DateFormatter("%b %d"))
    ax.xaxis.set_major_locator(mdates.WeekdayLocator(interval=1))
    ax.yaxis.set_major_locator(ticker.MaxNLocator(integer=True))
    for label in ax.get_xticklabels():
        label.set_rotation(30)
        label.set_ha("right")

    # -- Panel 3: Cumulative Downloads --
    ax = axes[1, 0]
    dates = [r["date"] for r in releases]
    cum_dl = []
    total = 0
    for r in releases:
        total += r["downloads"]
        cum_dl.append(total)
    ax.fill_between(dates, cum_dl, alpha=0.15, color="#34C759")
    ax.plot(dates, cum_dl, linewidth=2, color="#34C759", marker="o", markersize=3)
    style_chart(ax, f"Cumulative Downloads ({total:,} total)", "", "Downloads")
    ax.xaxis.set_major_formatter(mdates.DateFormatter("%b %d"))
    ax.xaxis.set_major_locator(mdates.WeekdayLocator(interval=1))
    for label in ax.get_xticklabels():
        label.set_rotation(30)
        label.set_ha("right")

    # -- Panel 4: Weekly Stars (bar chart for trend) --
    ax = axes[1, 1]
    week_counts = defaultdict(int)
    for d in star_dates:
        week_start = d.date() - timedelta(days=d.weekday())
        week_counts[week_start] += 1
    weeks_sorted = sorted(week_counts.keys())
    week_vals = [week_counts[w] for w in weeks_sorted]
    week_dts = [datetime.combine(w, datetime.min.time()) for w in weeks_sorted]
    wmax = max(week_vals) if week_vals else 1
    wcolors = [plt.cm.Purples(0.3 + 0.7 * (v / wmax)) for v in week_vals]
    ax.bar(week_dts, week_vals, width=5, color=wcolors, edgecolor="none")
    style_chart(ax, "Stars per Week", "", "Weekly Stars")
    ax.xaxis.set_major_formatter(mdates.DateFormatter("%b %d"))
    ax.xaxis.set_major_locator(mdates.WeekdayLocator(interval=1))
    ax.yaxis.set_major_locator(ticker.MaxNLocator(integer=True))
    for label in ax.get_xticklabels():
        label.set_rotation(30)
        label.set_ha("right")

    fig.suptitle(
        "App Store Connect CLI -- Growth Dashboard",
        fontsize=20, fontweight="bold", y=0.98,
    )

    days_alive = (star_dates[-1] - star_dates[0]).days
    fig.text(
        0.5, 0.945,
        f"Created {star_dates[0].strftime('%b %d, %Y')}  |  "
        f"{days_alive} days old  |  "
        f"{len(star_dates):,} stars  |  "
        f"{total:,} downloads  |  "
        f"75 releases",
        ha="center", fontsize=12, color="#666",
    )

    plt.tight_layout(rect=[0, 0, 1, 0.93])
    path = f"{OUTPUT_DIR}/dashboard.png"
    fig.savefig(path, dpi=200, bbox_inches="tight", facecolor="white")
    plt.close(fig)
    print(f"  Saved {path}")


def main():
    import os
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    print("Fetching data...")
    star_dates = fetch_stars()
    releases = fetch_downloads()

    print(f"  {len(star_dates)} stars, {len(releases)} releases")
    print()
    print("Generating charts...")

    chart_cumulative_stars(star_dates)
    chart_stars_per_day(star_dates)
    chart_cumulative_downloads(releases)
    chart_downloads_per_release(releases)
    chart_combined_dashboard(star_dates, releases)

    # Print summary stats
    days_alive = (star_dates[-1] - star_dates[0]).days
    total_downloads = sum(r["downloads"] for r in releases)
    day_counts = Counter(d.strftime("%Y-%m-%d") for d in star_dates)
    peak_day = max(day_counts, key=day_counts.get)
    peak_count = day_counts[peak_day]

    print()
    print("=" * 55)
    print("  GROWTH SUMMARY")
    print("=" * 55)
    print(f"  Total stars:       {len(star_dates):,}")
    print(f"  Total downloads:   {total_downloads:,}")
    print(f"  Releases:          {len(releases)}")
    print(f"  Project age:       {days_alive} days")
    print(f"  Avg stars/day:     {len(star_dates) / max(days_alive, 1):.1f}")
    print(f"  Peak day:          {peak_day} ({peak_count} stars)")
    print(f"  Last 7 days stars: {sum(1 for d in star_dates if d >= star_dates[-1] - timedelta(days=7))}")
    print("=" * 55)
    print()
    print(f"Charts saved to {OUTPUT_DIR}/")


if __name__ == "__main__":
    main()
