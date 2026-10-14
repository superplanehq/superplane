#!/usr/bin/env python3
"""Build sorted shard weights from gotestsum JUnit reports (standard library only)."""

import argparse
import json
import math
import statistics
import xml.etree.ElementTree as ET
from collections import defaultdict
from pathlib import Path

MODULE = "github.com/superplanehq/superplane/"


def seconds(element):
    value = float(element.get("time", "0"))
    if not math.isfinite(value) or value < 0:
        raise ValueError("JUnit duration must be finite and nonnegative")
    return value


def collect_report(path, samples):
    for suite in ET.parse(path).getroot().iter("testsuite"):
        package = suite.get("name", "").removeprefix(MODULE).removeprefix("./")
        cases = [case for case in suite.findall("testcase")
                 if case.get("name", "").split("/")[0] != "TestMain"
                 and case.find("skipped") is None]
        if package.startswith(("pkg/", "ee/")):
            if cases:
                # Suite elapsed time includes setup and parallel tests without
                # double counting parent/subtest durations.
                samples["unit"][f"./{package}"].append(seconds(suite))
            continue
        if package != "test/e2e":
            continue
        roots = defaultdict(list)
        for case in cases:
            name = case.get("name", "")
            if name.startswith("Test"):
                roots[name.split("/")[0]].append(case)
        for name, tests in roots.items():
            parents = [case for case in tests if case.get("name") == name]
            if parents:
                # gotestsum can retain retry attempts. Average attempts rather
                # than treating a retry as another test.
                duration = statistics.mean(seconds(case) for case in parents)
            else:
                # Older reports may contain only leaf cases.
                leaves = [case for case in tests if not any(
                    other.get("name", "").startswith(case.get("name", "") + "/")
                    for other in tests)]
                duration = sum(seconds(case) for case in leaves)
            samples["e2e"][name].append(duration)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("reports", nargs="+", type=Path)
    parser.add_argument("--output", type=Path,
                        default=Path(__file__).with_name("test_timings.json"))
    args = parser.parse_args()
    samples = {"unit": defaultdict(list), "e2e": defaultdict(list)}
    for path in args.reports:
        collect_report(path, samples)
    timings = {group: {name: round(statistics.mean(values), 6)
                       for name, values in names.items()}
               for group, names in samples.items()}
    args.output.write_text(json.dumps(timings, indent=2, sort_keys=True) + "\n")


if __name__ == "__main__":
    main()
