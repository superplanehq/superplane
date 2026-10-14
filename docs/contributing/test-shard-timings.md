# Test shard timings

Semaphore runs four unit shards and ten E2E shards. Both scripts use
`scripts/test_timings.json` to assign the longest items to the lightest shard.
Names and shard indices break ties. Each worker computes the same assignment.

Unit weights use package elapsed time. E2E weights use top-level test elapsed
time. Parent times include subtests, so the refresh script does not add their
times again. `TestMain` and skipped cases do not supply test weights.

Unknown unit packages use test file count multiplied by the median seconds
per file across known packages in the current inventory. Unknown E2E tests
use the median known test duration. Without usable timings, these defaults
are one second per file and one second per E2E test. Old names are ignored.
A missing or invalid timing file does not prevent test selection.

## Refresh from main reports

Reports from a main Semaphore run give better weights than local reports.
Download the `junit-report.xml` reports from all unit and E2E shards in that
run. Give each report a distinct local path.

Run the refresh script from the repository root:

```sh
python3 scripts/refresh_test_timings.py /path/to/unit/*.xml /path/to/e2e/*.xml
```

The script accepts one or more report paths. It uses only the Python standard
library. Python 3 is available in the development app container.
It replaces the timing file with sorted output. Supply both groups to preserve
weights for both groups. Multiple observations of an item produce its mean
duration. Retry attempts use the mean root duration. Reports with only
subtests use the sum of leaf durations.

Review and commit `scripts/test_timings.json`. Do not commit JUnit reports.
Refresh after large test changes or when shard durations differ substantially.

## Local seed

The initial seed came from local runs on October 14, 2026:

```sh
make dev.up
DEV_SETUP_DBS="superplane_dev superplane_test" make dev.setup
make test
cp junit-report.xml /tmp/junit-unit.xml
make test.e2e
python3 scripts/refresh_test_timings.py /tmp/junit-unit.xml junit-report.xml
```

Both suites passed. The unit run reported 16,397 cases, with 19 skipped cases,
in 409 seconds. The E2E run reported 187 cases in 1,198 seconds.
The seed includes packages and top-level E2E tests that ran.
Local machine speed and database load differ from Semaphore.
After merge, check shard durations on Semaphore. Aim for a spread of about
25% or less in each group. Refresh from main reports if the spread is larger.

## Check the split

```sh
bash scripts/test_sharding_test.sh
```

The shell tests use fixture timings. They verify exact coverage at 1, 4, and
10 shards, skewed loads, fallback weights, missing files, and deterministic
output. They also verify the report refresh and both shell callers.
Set `TEST_TIMINGS_FILE` when you need to try a different timing file.
