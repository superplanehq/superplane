# Semaphore CI Pipelines

This directory contains Semaphore CI pipeline configurations for the SuperPlane project.

## Pipelines

### Main CI Pipeline (`semaphore.yml`)
The main CI pipeline that runs on every commit and pull request. It includes:
- License checking
- Linting
- Unit tests
- E2E tests
- Database structure validation
- Format checking
- Frontend build and tests

### Fix Formatting Pipeline (`fix-formatting.yaml`)
A manual pipeline that can be triggered to automatically fix formatting issues in a pull request.

#### How to Use

This pipeline can be triggered manually through the Semaphore UI or API to fix formatting issues in a specific pull request.

**Via Semaphore UI:**
1. Go to the Semaphore dashboard for this project
2. Navigate to Workflows
3. Click "Run Workflow"
4. Select "Fix Formatting" pipeline
5. Enter the PR number in the `PR_NUMBER` parameter
6. Click "Start"

**Via Semaphore API:**
```bash
# Using curl
curl -X POST \
  -H "Authorization: Token YOUR_SEMAPHORE_API_TOKEN" \
  -d "project_id=YOUR_PROJECT_ID&reference=main&pipeline_file=.semaphore/fix-formatting.yaml&env_vars[PR_NUMBER]=123" \
  https://YOUR_ORG.semaphoreci.com/api/v1alpha/plumber-workflows

# Or using sem CLI
sem create workflow \
  --project-id YOUR_PROJECT_ID \
  --branch main \
  --pipeline-file .semaphore/fix-formatting.yaml \
  --param PR_NUMBER=123
```

#### What It Does

The pipeline will:
1. Check out the specified pull request
2. Run `gofmt -s -w .` to format Go code
3. Run `prettier` to format TypeScript/TSX files (via `npm run format` in web_src)
4. If changes are detected:
   - **For internal PRs**: Commit and push the changes directly to the PR branch
   - **For external PRs (from forks)**: Post a comment on the PR with the formatting diff and instructions for the author

#### Requirements

- A GitHub token must be configured in Semaphore secrets as `github-token` with the `GITHUB_TOKEN` environment variable
- The token must have permissions to:
  - Read PR information
  - Push to branches (for internal PRs)
  - Comment on PRs (for external PRs)

#### Notes

- The pipeline uses `sem-version` to install Go 1.22 and Node.js 20
- For external PRs from forks, the pipeline cannot directly push changes due to GitHub security restrictions. Instead, it posts a helpful comment with the diff and instructions.
- The bot commits will be authored by "SuperPlane Bot <bot@superplane.app>"
