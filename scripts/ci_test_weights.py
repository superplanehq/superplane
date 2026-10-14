#!/usr/bin/env python3
"""Collect successful Go test timings and refresh CI shard weights."""

import argparse
import json
import math
import re
import statistics
import sys
import xml.etree.ElementTree as ET
from collections import defaultdict
from pathlib import Path

MODULE = "github.com/superplanehq/superplane/"
E2E_ROOT = re.compile(r"^func\s+(Test[A-Za-z0-9_]*)\s*\(", re.MULTILINE)
MIN_UNIT_SECONDS = 7
SHARD_REPORT = re.compile(r"^([1-9][0-9]*)-of-([1-9][0-9]*)\.xml$")


def read_report(path):
    root = ET.parse(path).getroot()
    if root.tag != "testsuites":
        raise ValueError(f"{path}: expected testsuites root")
    if int(root.get("failures", "0")) or int(root.get("errors", "0")):
        raise ValueError(f"{path}: report contains failures")
    return root


def shard_reports(directory):
    paths = sorted(directory.glob("*.xml"))
    matches = [SHARD_REPORT.fullmatch(path.name) for path in paths]
    if not paths or any(match is None for match in matches):
        raise ValueError(f"missing or invalid shard reports in {directory}")
    counts = {int(match.group(2)) for match in matches}
    if len(counts) != 1:
        raise ValueError(f"mixed shard counts in {directory}")
    count = counts.pop()
    expected = {f"{index}-of-{count}.xml" for index in range(1, count + 1)}
    if {path.name for path in paths} != expected:
        raise ValueError(f"incomplete shard reports in {directory}")
    return paths


def collect(reports, output, sha):
    e2e_reports = shard_reports(reports / "e2e")
    unit_reports = shard_reports(reports / "unit")

    e2e = {}
    unit = {}
    for path in e2e_reports:
        for suite in read_report(path).findall("testsuite"):
            roots = defaultdict(lambda: {"root": None, "children": 0.0})
            for case in suite.findall("testcase"):
                name = case.get("name", "")
                if not name.startswith("Test"):
                    continue
                root_name = name.split("/", 1)[0]
                seconds = float(case.get("time", "0"))
                if "/" in name:
                    roots[root_name]["children"] += seconds
                else:
                    roots[root_name]["root"] = seconds
            for name, timing in roots.items():
                if name in e2e:
                    raise ValueError(f"duplicate E2E root: {name}")
                e2e[name] = max(timing["root"] or 0, timing["children"])

    for path in unit_reports:
        for suite in read_report(path).findall("testsuite"):
            name = suite.get("name", "")
            if not name.startswith(MODULE):
                raise ValueError(f"unknown package in {path}: {name}")
            package = name.removeprefix(MODULE)
            if package in unit:
                raise ValueError(f"duplicate unit package: {package}")
            unit[package] = float(suite.get("time", "0"))

    if not e2e or not unit:
        raise ValueError("test timing reports contain no tests")
    output.parent.mkdir(parents=True, exist_ok=True)
    output.write_text(json.dumps({"sha": sha, "e2e": e2e, "unit": unit}, sort_keys=True) + "\n")
    print(f"Stored {len(e2e)} E2E roots and {len(unit)} unit packages")


def source_e2e_roots(repo):
    roots = set()
    for path in (repo / "test/e2e").rglob("*_test.go"):
        roots.update(name for name in E2E_ROOT.findall(path.read_text()) if name != "TestMain")
    return roots


def source_unit_packages(repo):
    packages = {}
    for folder in ("pkg", "ee", "cmd", "test/support"):
        root = repo / folder
        if not root.exists():
            continue
        for path in root.rglob("*_test.go"):
            package = path.parent.relative_to(repo).as_posix()
            packages[package] = packages.get(package, 0) + 1
    return packages


def read_weights(path):
    weights = {}
    for line in path.read_text().splitlines():
        if line and not line.startswith("#"):
            name, value = line.split()
            weights[name] = int(value)
    return weights


def stable_weight(old, measured):
    if old is None:
        return measured
    if abs(measured - old) < max(5, math.ceil(old * 0.2)):
        return old
    return measured


def write_weights(path, comment, values):
    content = comment + "\n" + "".join(f"{name} {values[name]}\n" for name in sorted(values))
    if path.read_text() != content:
        path.write_text(content)
        print(f"Updated {path}")
    else:
        print(f"No change to {path}")


def refresh(history, repo, runs, minimum):
    files = sorted(history.glob("*.json"), reverse=True)
    selected = []
    seen = set()
    for path in files:
        data = json.loads(path.read_text())
        sha = data["sha"]
        if sha in seen:
            continue
        seen.add(sha)
        selected.append(data)
        if len(selected) == runs:
            break
    if len(selected) < minimum:
        raise ValueError(f"need {minimum} distinct successful main runs; found {len(selected)}")

    e2e_names = source_e2e_roots(repo)
    unit_packages = source_unit_packages(repo)
    if not e2e_names or not unit_packages:
        raise ValueError("could not discover current tests")

    e2e_path = repo / "scripts/ci_e2e_test_weights.tsv"
    unit_path = repo / "scripts/ci_unit_package_weights.tsv"
    prior_e2e = read_weights(e2e_path)
    prior_unit = read_weights(unit_path)
    e2e = {}
    for name in e2e_names:
        samples = [run["e2e"][name] for run in selected if name in run["e2e"]]
        if not samples:
            raise ValueError(f"no timing for current E2E test {name}")
        measured = max(1, math.ceil(statistics.median(samples)))
        e2e[name] = stable_weight(prior_e2e.get(name), measured)

    unit = {}
    for package in unit_packages:
        samples = [run["unit"][package] for run in selected if package in run["unit"]]
        if not samples:
            raise ValueError(f"no timing for current unit package {package}")
        measured = math.ceil(statistics.median(samples))
        if measured >= MIN_UNIT_SECONDS or package in prior_unit:
            weight = stable_weight(prior_unit.get(package), measured)
            if weight >= MIN_UNIT_SECONDS:
                unit[package] = weight

    provenance = f"# Median test times from {len(selected)} successful main runs.\n# Refresh weekly with the Semaphore Refresh test weights task."
    write_weights(e2e_path, provenance + "\n# Root E2E seconds; new tests use 30 seconds until measured.", e2e)
    write_weights(unit_path, provenance + "\n# Extra unit package seconds above test-file count; unlisted packages use file count.", unit)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    commands = parser.add_subparsers(dest="command", required=True)
    collect_cmd = commands.add_parser("collect")
    collect_cmd.add_argument("--reports", type=Path, required=True)
    collect_cmd.add_argument("--output", type=Path, required=True)
    collect_cmd.add_argument("--sha", required=True)
    refresh_cmd = commands.add_parser("refresh")
    refresh_cmd.add_argument("--history", type=Path, required=True)
    refresh_cmd.add_argument("--repo", type=Path, default=Path("."))
    refresh_cmd.add_argument("--runs", type=int, default=5)
    refresh_cmd.add_argument("--min-runs", type=int, default=3)
    args = parser.parse_args()
    try:
        if args.command == "collect":
            collect(args.reports, args.output, args.sha)
        else:
            refresh(args.history, args.repo, args.runs, args.min_runs)
    except (ValueError, KeyError, ET.ParseError, OSError) as error:
        print(f"test weight refresh: {error}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
