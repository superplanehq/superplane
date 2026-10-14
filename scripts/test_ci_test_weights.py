import json
import tempfile
import unittest
from pathlib import Path

from ci_test_weights import collect, refresh


class TestTimingRefresh(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.reports = self.root / "reports"
        self.repo = self.root / "repo"
        (self.repo / "test/e2e").mkdir(parents=True)
        (self.repo / "pkg/example").mkdir(parents=True)
        (self.repo / "scripts").mkdir()
        (self.repo / "test/e2e/example_test.go").write_text("func TestCurrent(t *testing.T) {}\n")
        (self.repo / "pkg/example/example_test.go").write_text("package example\n")
        (self.repo / "scripts/ci_e2e_test_weights.tsv").write_text("TestCurrent 30\nTestDeleted 40\n")
        (self.repo / "scripts/ci_unit_package_weights.tsv").write_text("pkg/deleted 20\n")

    def report(self, kind, index, suite):
        path = self.reports / kind / f"{index}.xml"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(f'<testsuites failures="0" errors="0">{suite}</testsuites>')

    def test_collect_requires_every_successful_shard(self):
        self.report("e2e", 0, '<testsuite><testcase name="TestCurrent" time="12" /></testsuite>')
        with self.assertRaisesRegex(ValueError, "expected 10 E2E and 4 unit"):
            collect(self.reports, self.root / "summary.json", "abc")

    def test_collect_uses_root_or_child_time(self):
        for index in range(1, 11):
            self.report("e2e", index,
                        f'<testsuite><testcase name="TestCase{index}" time="0" />'
                        f'<testcase name="TestCase{index}/step" time="12" /></testsuite>')
        for index in range(1, 5):
            self.report("unit", index,
                        f'<testsuite name="github.com/superplanehq/superplane/pkg/part{index}" time="9" />')
        output = self.root / "summary.json"
        collect(self.reports, output, "abc")
        summary = json.loads(output.read_text())
        self.assertEqual(summary["e2e"]["TestCase1"], 12)
        self.assertEqual(summary["unit"]["pkg/part1"], 9)

    def test_median_refresh_keeps_noise_and_removes_deleted_tests(self):
        history = self.root / "history"
        history.mkdir()
        for index, seconds in enumerate((33, 34, 90), start=1):
            (history / f"{index}.json").write_text(json.dumps({
                "sha": f"commit-{index}",
                "e2e": {"TestCurrent": seconds},
                "unit": {"pkg/example": 9},
            }))

        refresh(history, self.repo, runs=5, minimum=3)
        e2e = (self.repo / "scripts/ci_e2e_test_weights.tsv").read_text()
        unit = (self.repo / "scripts/ci_unit_package_weights.tsv").read_text()
        self.assertIn("TestCurrent 30\n", e2e)
        self.assertNotIn("TestDeleted", e2e)
        self.assertIn("pkg/example 9\n", unit)
        self.assertNotIn("pkg/deleted", unit)

    def test_refresh_rejects_missing_current_test(self):
        history = self.root / "history"
        history.mkdir()
        for index in range(3):
            (history / f"{index}.json").write_text(json.dumps({
                "sha": f"commit-{index}", "e2e": {}, "unit": {"pkg/example": 10}
            }))
        with self.assertRaisesRegex(ValueError, "no timing for current E2E test"):
            refresh(history, self.repo, runs=5, minimum=3)


if __name__ == "__main__":
    unittest.main()
