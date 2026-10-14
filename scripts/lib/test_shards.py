#!/usr/bin/env python3
"""Select a shard from stdin lines of name<TAB>test file count.

Both callers supply the full inventory. Names and shard indices break ties so
each worker computes the same longest-first, lightest-shard assignment.
"""

import json
import math
import statistics
import sys
from pathlib import Path


def read_timings(path, group):
    try:
        timings = json.loads(Path(path).read_text()).get(group, {})
        return {
            name: float(seconds)
            for name, seconds in timings.items()
            if isinstance(seconds, (int, float))
            and not isinstance(seconds, bool)
            and math.isfinite(seconds)
            and seconds >= 0
        }
    except (OSError, ValueError, AttributeError):
        # Timing data is optional; a damaged or missing file must not skip tests.
        return {}


def weighted_items(items, timings, group):
    samples = [
        timings[name] / files if group == "unit" else timings[name]
        for name, files in items
        if name in timings and (group != "unit" or files > 0)
    ]
    fallback = statistics.median(samples) if samples else 1.0
    return [
        (name, timings.get(name, files * fallback if group == "unit" else fallback))
        for name, files in items
    ]


def select_shard(items, shard_index, shard_count):
    loads = [0.0] * shard_count
    for name, seconds in sorted(items, key=lambda item: (-item[1], item[0])):
        shard = min(range(shard_count), key=lambda index: loads[index])
        loads[shard] += seconds
        if shard + 1 == shard_index:
            print(name)


def main():
    group, index, count, path = sys.argv[1:]
    items = []
    for line in sys.stdin:
        name, files = line.rstrip("\n").split("\t")
        items.append((name, int(files)))
    select_shard(weighted_items(items, read_timings(path, group), group), int(index), int(count))


if __name__ == "__main__":
    main()
