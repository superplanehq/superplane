#!/usr/bin/env bash
# Verify duration assignment and JUnit refresh with isolated fixtures.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
python3 - "$repo_root" <<'PY'
import json
import os
import pathlib
import subprocess
import sys
import tempfile

root = pathlib.Path(sys.argv[1])
splitter = root / "scripts/lib/test_shards.py"
refresh = root / "scripts/refresh_test_timings.py"

with tempfile.TemporaryDirectory() as directory:
    directory = pathlib.Path(directory)
    timings = directory / "timings.json"
    # Ten heavy items and ninety light items permit balance at all shard counts.
    weights = [20.5] * 10 + [1.25] * 90
    for group in ("unit", "e2e"):
        names = [f"./pkg/p{i:03}" if group == "unit" else f"Test{i:03}" for i in range(100)]
        known = dict(zip(names, weights))
        timings.write_text(json.dumps({group: known}))
        items = "".join(f"{name}\t2\n" for name in reversed(names))

        def select(index, count, path=timings, source=items):
            return subprocess.check_output(
                [sys.executable, str(splitter), group, str(index), str(count), str(path)],
                input=source, text=True,
            ).splitlines()

        for count in (1, 4, 10):
            shards = [select(index, count) for index in range(1, count + 1)]
            assert sorted(name for shard in shards for name in shard) == names, "coverage or duplicates"
            loads = [sum(known[name] for name in shard) for shard in shards]
            assert max(loads) <= sum(loads) / count * 1.15, loads
            assert shards == [select(index, count) for index in range(1, count + 1)], "not deterministic"
            assert shards == [select(index, count, source="".join(f"{n}\t2\n" for n in names))
                              for index in range(1, count + 1)], "input order changed assignment"
            missing = directory / "missing.json"
            assert sorted(n for i in range(1, count + 1) for n in select(i, count, missing)) == names
        print(f"PASS {group}: coverage, skewed balance, missing timings, deterministic assignment")

        # Known unit ratios are 5 and 15 seconds per file; median is 10.
        # E2E median is 20 seconds. Unknown must precede the 10-second item.
        timings.write_text(json.dumps({group: {names[0]: 10, names[1]: 30, "stale": 99999}}))
        unknown = "./pkg/new" if group == "unit" else "TestNew"
        source = f"{names[0]}\t2\n{names[1]}\t2\n{unknown}\t4\n"
        expected = [unknown, names[1], names[0]] if group == "unit" else [names[1], unknown, names[0]]
        assert select(1, 1, source=source) == expected, "incorrect median fallback"
        timings.write_text('{"broken":')
        assert sorted(select(1, 1, source=source)) == sorted([*names[:2], unknown])
        print(f"PASS {group}: median fallback, stale entries, invalid timings")

    report = directory / "unit.xml"
    report.write_text('''<testsuites><testsuite name="github.com/superplanehq/superplane/pkg/example" time="12.5">
      <testcase name="TestMain" time="50"/>
      <testcase name="TestParent/child" time="3"/>
      <testcase name="TestParent" time="4"/>
    </testsuite></testsuites>''')
    e2e = directory / "e2e.xml"
    e2e.write_text('''<testsuites><testsuite name="github.com/superplanehq/superplane/test/e2e" time="100">
      <testcase name="TestMain" time="50"/>
      <testcase name="TestBrowser/child" time="3"/>
      <testcase name="TestBrowser" time="5.5"/>
      <testcase name="TestChildren/a" time="2"/>
      <testcase name="TestChildren/b" time="4"/>
      <testcase name="TestSkipped" time="0"><skipped/></testcase>
    </testsuite></testsuites>''')
    subprocess.check_call([sys.executable, str(refresh), "--output", str(timings), str(report), str(e2e)])
    expected = {"unit": {"./pkg/example": 12.5}, "e2e": {"TestBrowser": 5.5, "TestChildren": 6.0}}
    assert json.loads(timings.read_text()) == expected, timings.read_text()
    first = timings.read_text()
    subprocess.check_call([sys.executable, str(refresh), "--output", str(timings), str(e2e), str(report)])
    assert timings.read_text() == first, "report order changed output"
    print("PASS refresh: package elapsed time, root tests, subtests, skipped tests, sorted output")
    later = directory / "later.xml"
    later.write_text('''<testsuites>
      <testsuite name="github.com/superplanehq/superplane/pkg/example" time="7.5">
        <testcase name="TestParent" time="4"/>
      </testsuite>
      <testsuite name="github.com/superplanehq/superplane/test/e2e" time="20">
        <testcase name="TestBrowser" time="6.5"/>
        <testcase name="TestBrowser" time="8.5"/>
      </testsuite>
    </testsuites>''')
    subprocess.check_call([sys.executable, str(refresh), "--output", str(timings),
                           str(report), str(e2e), str(later)])
    assert json.loads(timings.read_text()) == {
        "unit": {"./pkg/example": 10.0}, "e2e": {"TestBrowser": 6.5, "TestChildren": 6.0}}
    print("PASS refresh: average multiple reports and retry attempts")

    # Exercise discovery and the actual gotestsum selection in both shell callers.
    bin_dir = directory / "bin"
    bin_dir.mkdir()
    go = bin_dir / "go"
    go.write_text('''#!/usr/bin/env bash
set -eu
case "$*" in
  'list -m') printf '%s\\n' github.com/superplanehq/superplane ;;
  'list ./pkg/... ./ee/...') printf '%s\\n' github.com/superplanehq/superplane/pkg/p{00..11} ;;
  'tool cover -func=coverage-go.out') printf '%s\\n' 'total: 100%' ;;
  'run ./scripts/check_go_coverage_budget.go '*) ;;
  *) exit 1 ;;
esac
''')
    gotestsum = bin_dir / "gotestsum"
    gotestsum.write_text('''#!/usr/bin/env python3
import json, os, sys
with open(os.environ["CAPTURE"], "w") as output:
    json.dump(sys.argv[1:], output)
''')
    for executable in (go, gotestsum):
        executable.chmod(0o755)
    for index in range(12):
        package = directory / f"pkg/p{index:02}"
        package.mkdir(parents=True)
        (package / "example_test.go").touch()
    e2e_dir = directory / "test/e2e"
    e2e_dir.mkdir(parents=True)
    (e2e_dir / "example_test.go").write_text(
        "func TestMain(m *testing.M) {}\n" +
        "".join(f"func Test{i:02}(t *testing.T) {{}}\n" for i in range(12)))
    (e2e_dir / "duplicate_test.go").write_text("func Test00(t *testing.T) {}\n")
    timings.write_text(json.dumps({
        "unit": {f"./pkg/p{i:02}": 12 - i for i in range(6)},
        "e2e": {f"Test{i:02}": 12 - i for i in range(6)},
    }))
    capture = directory / "args.json"
    for group in ("unit", "e2e"):
        for timing_path in (timings, directory / "missing.json"):
            for count in (1, 4, 10):
                selected = []
                for index in range(1, count + 1):
                    capture.unlink(missing_ok=True)
                    env = dict(os.environ, PATH=f"{bin_dir}:{os.environ['PATH']}",
                               CAPTURE=str(capture), TEST_TIMINGS_FILE=str(timing_path),
                               SHARD_INDEX=str(index), SHARD_COUNT=str(count))
                    subprocess.check_call(["bash", str(root / f"scripts/test_{group}_autoparallel.sh")],
                                          cwd=directory, env=env, stdout=subprocess.DEVNULL)
                    if not capture.exists():
                        continue
                    args = json.loads(capture.read_text())
                    packages = next(arg for arg in args if arg.startswith("--packages="))
                    if group == "unit":
                        selected.extend(packages.split("=", 1)[1].split())
                    else:
                        assert packages == "--packages=./test/e2e/..."
                        assert "--rerun-fails=3" in args and "--rerun-fails-max-failures=1" in args
                        regex = args[args.index("-run") + 1]
                        assert regex.startswith("^(") and regex.endswith(")$")
                        selected.extend(regex[2:-2].split("|"))
                expected = [f"./pkg/p{i:02}" if group == "unit" else f"Test{i:02}" for i in range(12)]
                assert sorted(selected) == expected, (group, count, selected)
        print(f"PASS {group} shell caller: exact coverage for 1, 4, 10 shards with known and missing timings")
PY
