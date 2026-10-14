# CI test weights

Semaphore divides Go tests between four unit jobs and ten E2E jobs. The files
`scripts/ci_unit_package_weights.tsv` and `scripts/ci_e2e_test_weights.tsv`
control job assignment. They do not control test discovery. The CI shard check
fails if a Go test file is outside the covered folders.

## Automatic refresh

Each successful `main` CI run that executes backend tests stores one timing
summary in Semaphore project artifacts under `ci-test-timings/`. The summary
contains the times from all four unit jobs and all ten E2E jobs. Failed runs do
not enter the history.

The **Refresh test weights** Semaphore task runs each Monday at 09:00 UTC. It
uses the five most recent successful `main` commits. It requires at least
three distinct commits. It uses the median test time, which limits the effect
of one slow run. The task removes deleted tests, adds measured new tests, and
opens or updates a pull request when a weight changes by at least five seconds
and 20%. New E2E tests use a 30-second estimate until a successful `main` run
measures them. Unit packages below seven seconds use their test-file count.

The task uses the `github-release-token` Semaphore secret. Its
`GITHUB_RELEASE_TOKEN` must have **Contents: write** and **Pull requests: write**
permission for `superplanehq/superplane`. The task does not push to `main`.
Review and merge its pull request to apply new weights.

## Run the refresh now

1. Open the SuperPlane project in Semaphore.
2. Open **Tasks** and select **Refresh test weights**.
3. Select **Run now**. Keep the branch set to `main`.

To run the calculation inside a Semaphore job with project artifacts, use:

```sh
artifact pull project ci-test-timings
python3 scripts/ci_test_weights.py refresh --history ci-test-timings
```

The command updates the two TSV files in the checkout. Review the diff before
you commit it. If fewer than three successful `main` runs exist, wait for more
runs and start the task again. If a current test has no timing, check that all
shards published reports. Do not invent a weight for a missing report.
