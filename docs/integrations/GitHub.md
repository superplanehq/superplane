---
title: "GitHub"
sidebar:
  label: "GitHub"
type: "application"
name: "github"
label: "GitHub"
---

Manage and react to changes in your GitHub repositories

### Components

- [Create Issue](#create-issue)
- [Create Release](#create-release)
- [Delete Release](#delete-release)
- [Get Issue](#get-issue)
- [Publish Commit Status](#publish-commit-status)
- [Run Workflow](#run-workflow)
- [Update Issue](#update-issue)
- [Update Release](#update-release)

### Triggers

- [On Branch Created](#on-branch-created)
- [On Issue](#on-issue)
- [On Issue Comment](#on-issue-comment)
- [On Pull Request](#on-pull-request)
- [On PR Review Comment](#on-pr-review-comment)
- [On Push](#on-push)
- [On Release](#on-release)
- [On Tag Created](#on-tag-created)

## Components

### Create Issue

Create a new issue in a GitHub repository

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| title | Title | string | yes | - |
| body | Body | text | no | - |
| assignees | Assignees | list | no | - |
| labels | Labels | list | no | - |

## Example Output

```json
{
  "data": {
    "html_url": "https://github.com/acme/widgets/issues/42",
    "id": 101,
    "number": 42,
    "state": "open",
    "title": "Fix flaky build",
    "user": {
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.issue"
}
```

### Create Release

Create a new release in a GitHub repository

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | string | yes | - |
| versionStrategy | Version Strategy | select | yes | How to determine the release version |
| tagName | Tag Name | string | no | The name of the tag to create the release for |
| name | Release Name | string | no | The title of the release |
| draft | Draft | boolean | no | Mark this release as a draft |
| prerelease | Prerelease | boolean | no | Mark this release as a prerelease |
| generateReleaseNotes | Generate release notes | boolean | no | Automatically generate release notes from commits since the last release |
| body | Additional notes | text | no | Optional text to append after auto-generated release notes. If auto-generation is off, this becomes the entire release description. |

## Example Output

```json
{
  "data": {
    "draft": false,
    "html_url": "https://github.com/acme/widgets/releases/tag/v1.2.3",
    "id": 3001,
    "name": "Release 1.2.3",
    "prerelease": false,
    "tag_name": "v1.2.3"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.release"
}
```

### Delete Release

Delete a release from a GitHub repository

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | string | yes | - |
| releaseStrategy | Release to Delete | select | yes | How to identify which release to delete |
| tagName | Tag Name | string | no | Git tag identifying the release to delete. Supports template variables from previous steps. |
| deleteTag | Also delete Git tag | boolean | no | When enabled, also deletes the associated Git tag from the repository |

## Example Output

```json
{
  "data": {
    "deleted_at": "2026-01-16T17:55:00Z",
    "draft": false,
    "html_url": "https://github.com/acme/widgets/releases/tag/v1.2.3",
    "id": 3001,
    "name": "Release 1.2.3",
    "prerelease": false,
    "tag_deleted": true,
    "tag_name": "v1.2.3"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.release"
}
```

### Get Issue

Get a GitHub issue by number

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| issueNumber | Issue Number | string | yes | - |

## Example Output

```json
{
  "data": {
    "comments": 3,
    "html_url": "https://github.com/acme/widgets/issues/42",
    "id": 101,
    "number": 42,
    "state": "open",
    "title": "Fix flaky build",
    "user": {
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.issue"
}
```

### Publish Commit Status

Publish a status check to a GitHub commit

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| sha | Commit SHA | string | yes | The full SHA of the commit to attach the status to |
| state | State | select | yes | - |
| context | Context | string | yes | A label to identify this status check |
| description | Description | text | no | Short description of the status (max ~140 characters) |
| targetUrl | Target URL | string | no | e.g. Link to build logs, test results, ... |

## Example Output

```json
{
  "data": {
    "context": "ci/build",
    "created_at": "2026-01-16T17:45:00Z",
    "description": "All checks passed",
    "state": "success",
    "target_url": "https://ci.example.com/builds/123",
    "updated_at": "2026-01-16T17:45:10Z"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.commitStatus"
}
```

### Run Workflow

Run GitHub Actions workflow

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| passed | Passed | - |
| failed | Failed | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| workflowFile | Workflow file | string | yes | - |
| ref | Branch or tag | git-ref | yes | - |
| inputs | Inputs | list | no | - |

## Example Output

```json
{
  "data": {
    "workflow_run": {
      "conclusion": "success",
      "html_url": "https://github.com/acme/widgets/actions/runs/9001",
      "id": 9001,
      "status": "completed"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.workflow.finished"
}
```

### Update Issue

Update a GitHub issue

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| issueNumber | Issue Number | number | yes | - |
| title | Title | string | no | - |
| body | Body | text | no | - |
| state | State | select | no | - |
| assignees | Assignees | list | no | - |
| labels | Labels | list | no | - |

## Example Output

```json
{
  "data": {
    "html_url": "https://github.com/acme/widgets/issues/42",
    "id": 101,
    "number": 42,
    "state": "closed",
    "title": "Fix flaky build",
    "updated_at": "2026-01-16T17:40:00Z"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.issue"
}
```

### Update Release

Update an existing release in a GitHub repository

## Output Channels

| Name | Label | Description |
| --- | --- | --- |
| default | Default | - |

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | string | yes | - |
| releaseStrategy | Release Strategy | select | yes | How to identify which release to update |
| tagName | Tag Name | string | no | Git tag identifying the release to update. Supports template variables from previous steps. |
| name | Release Name | string | no | Update the release title (leave empty to keep current) |
| generateReleaseNotes | Generate release notes | boolean | no | Automatically generate release notes from commits since the last release. If body is also provided, custom text is appended. |
| body | Release Notes | text | no | Update release description (leave empty to keep current) |
| draft | Draft | boolean | no | Mark release as draft or publish it |
| prerelease | Prerelease | boolean | no | Mark as prerelease or stable release |

## Example Output

```json
{
  "data": {
    "draft": false,
    "html_url": "https://github.com/acme/widgets/releases/tag/v1.2.3",
    "id": 3001,
    "name": "Release 1.2.3",
    "prerelease": false,
    "tag_name": "v1.2.3",
    "updated_at": "2026-01-16T17:50:00Z"
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.release"
}
```

## Triggers

### On Branch Created

Listen to GitHub branch creation events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| branches | Branches | any-predicate-list | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "description": "Example repository for webhook payloads",
    "master_branch": "main",
    "pusher_type": "user",
    "ref": "feature/new-endpoint",
    "ref_type": "branch",
    "repository": {
      "full_name": "acme/widgets",
      "html_url": "https://github.com/acme/widgets",
      "id": 123456
    },
    "sender": {
      "html_url": "https://github.com/octocat",
      "id": 101,
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.branchCreated"
}
```

### On Issue

Listen to issue events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| actions | Actions | multi-select | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "action": "opened",
    "assignee": null,
    "issue": {
      "html_url": "https://github.com/acme/widgets/issues/42",
      "number": 42,
      "state": "open",
      "title": "Fix flaky build",
      "user": {
        "login": "octocat"
      }
    },
    "repository": {
      "full_name": "acme/widgets",
      "html_url": "https://github.com/acme/widgets",
      "id": 123456
    },
    "sender": {
      "html_url": "https://github.com/octocat",
      "id": 101,
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.issue"
}
```

### On Issue Comment

Listen to issue comment events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| contentFilter | Content Filter | string | no | Optional regex pattern to filter comments by content |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "action": "created",
    "comment": {
      "body": "I can reproduce this",
      "html_url": "https://github.com/acme/widgets/issues/42#issuecomment-5001",
      "id": 5001
    },
    "issue": {
      "html_url": "https://github.com/acme/widgets/issues/42",
      "number": 42,
      "title": "Fix flaky build"
    },
    "repository": {
      "full_name": "acme/widgets",
      "html_url": "https://github.com/acme/widgets",
      "id": 123456
    },
    "sender": {
      "html_url": "https://github.com/octocat",
      "id": 101,
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.issueComment"
}
```

### On Pull Request

Listen to pull request events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| actions | Actions | multi-select | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "action": "opened",
    "assignee": null,
    "number": 101,
    "pull_request": {
      "html_url": "https://github.com/acme/widgets/pull/101",
      "number": 101,
      "state": "open",
      "title": "Add new endpoint",
      "user": {
        "login": "octocat"
      }
    },
    "repository": {
      "full_name": "acme/widgets",
      "html_url": "https://github.com/acme/widgets",
      "id": 123456
    },
    "sender": {
      "html_url": "https://github.com/octocat",
      "id": 101,
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.pullRequest"
}
```

### On PR Review Comment

Listen to pull request review comment events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| contentFilter | Content Filter | string | no | Optional regex pattern to filter comments by content |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "action": "created",
    "comment": {
      "body": "Looks good to me",
      "html_url": "https://github.com/acme/widgets/pull/101#discussion_r7001",
      "id": 7001
    },
    "pull_request": {
      "html_url": "https://github.com/acme/widgets/pull/101",
      "number": 101,
      "title": "Add new endpoint"
    },
    "repository": {
      "full_name": "acme/widgets",
      "html_url": "https://github.com/acme/widgets",
      "id": 123456
    },
    "sender": {
      "html_url": "https://github.com/octocat",
      "id": 101,
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.pullRequestReviewComment"
}
```

### On Push

Listen to GitHub push events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| refs | Refs | any-predicate-list | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "after": "4f9c2e1a7b3d45c0d1e9f23456789abcdeffed01",
    "base_ref": null,
    "before": "1a2b3c4d5e6f708192a3b4c5d6e7f8090a1b2c3d",
    "commits": [
      {
        "added": [],
        "author": {
          "date": "2026-03-10T14:22:11+01:00",
          "email": "alex.doe@example.com",
          "name": "Alex Doe",
          "username": "alexdoe"
        },
        "committer": {
          "date": "2026-03-10T14:22:11+01:00",
          "email": "noreply@example.com",
          "name": "GitHub",
          "username": "web-flow"
        },
        "distinct": true,
        "id": "4f9c2e1a7b3d45c0d1e9f23456789abcdeffed01",
        "message": "feat: add lightweight metrics endpoint (#42)\n\nAdds a basic /metrics handler with a minimal gauge.",
        "modified": [
          "cmd/server/main.go",
          "pkg/metrics/handler.go",
          "docs/metrics.md"
        ],
        "removed": [],
        "timestamp": "2026-03-10T14:22:11+01:00",
        "tree_id": "7a8b9c0d1e2f3a4b5c6d7e8f90123456789abcde",
        "url": "https://github.com/example-org/example-repo/commit/4f9c2e1a7b3d45c0d1e9f23456789abcdeffed01"
      }
    ],
    "compare": "https://github.com/example-org/example-repo/compare/1a2b3c4d5e6f...4f9c2e1a7b3d",
    "created": false,
    "deleted": false,
    "forced": false,
    "head_commit": {
      "added": [],
      "author": {
        "date": "2026-03-10T14:22:11+01:00",
        "email": "alex.doe@example.com",
        "name": "Alex Doe",
        "username": "alexdoe"
      },
      "committer": {
        "date": "2026-03-10T14:22:11+01:00",
        "email": "noreply@example.com",
        "name": "GitHub",
        "username": "web-flow"
      },
      "distinct": true,
      "id": "4f9c2e1a7b3d45c0d1e9f23456789abcdeffed01",
      "message": "feat: add lightweight metrics endpoint (#42)\n\nAdds a basic /metrics handler with a minimal gauge.",
      "modified": [
        "cmd/server/main.go",
        "pkg/metrics/handler.go",
        "docs/metrics.md"
      ],
      "removed": [],
      "timestamp": "2026-03-10T14:22:11+01:00",
      "tree_id": "7a8b9c0d1e2f3a4b5c6d7e8f90123456789abcde",
      "url": "https://github.com/example-org/example-repo/commit/4f9c2e1a7b3d45c0d1e9f23456789abcdeffed01"
    },
    "organization": {
      "avatar_url": "https://avatars.githubusercontent.com/u/12345678?v=4",
      "description": "Example organization for demo data",
      "events_url": "https://api.github.com/orgs/example-org/events",
      "hooks_url": "https://api.github.com/orgs/example-org/hooks",
      "id": 12345678,
      "issues_url": "https://api.github.com/orgs/example-org/issues",
      "login": "example-org",
      "members_url": "https://api.github.com/orgs/example-org/members{/member}",
      "node_id": "O_kgDOBb1AaA",
      "public_members_url": "https://api.github.com/orgs/example-org/public_members{/member}",
      "repos_url": "https://api.github.com/orgs/example-org/repos",
      "url": "https://api.github.com/orgs/example-org"
    },
    "pusher": {
      "email": "alex.doe@example.com",
      "name": "alexdoe"
    },
    "ref": "refs/heads/main",
    "repository": {
      "allow_forking": true,
      "archive_url": "https://api.github.com/repos/example-org/example-repo/{archive_format}{/ref}",
      "archived": false,
      "assignees_url": "https://api.github.com/repos/example-org/example-repo/assignees{/user}",
      "blobs_url": "https://api.github.com/repos/example-org/example-repo/git/blobs{/sha}",
      "branches_url": "https://api.github.com/repos/example-org/example-repo/branches{/branch}",
      "clone_url": "https://github.com/example-org/example-repo.git",
      "collaborators_url": "https://api.github.com/repos/example-org/example-repo/collaborators{/collaborator}",
      "comments_url": "https://api.github.com/repos/example-org/example-repo/comments{/number}",
      "commits_url": "https://api.github.com/repos/example-org/example-repo/commits{/sha}",
      "compare_url": "https://api.github.com/repos/example-org/example-repo/compare/{base}...{head}",
      "contents_url": "https://api.github.com/repos/example-org/example-repo/contents/{+path}",
      "contributors_url": "https://api.github.com/repos/example-org/example-repo/contributors",
      "created_at": 1746900000,
      "custom_properties": {},
      "default_branch": "main",
      "deployments_url": "https://api.github.com/repos/example-org/example-repo/deployments",
      "description": "Example repository for webhook payloads",
      "disabled": false,
      "downloads_url": "https://api.github.com/repos/example-org/example-repo/downloads",
      "events_url": "https://api.github.com/repos/example-org/example-repo/events",
      "fork": false,
      "forks": 2,
      "forks_count": 2,
      "forks_url": "https://api.github.com/repos/example-org/example-repo/forks",
      "full_name": "example-org/example-repo",
      "git_commits_url": "https://api.github.com/repos/example-org/example-repo/git/commits{/sha}",
      "git_refs_url": "https://api.github.com/repos/example-org/example-repo/git/refs{/sha}",
      "git_tags_url": "https://api.github.com/repos/example-org/example-repo/git/tags{/sha}",
      "git_url": "git://github.com/example-org/example-repo.git",
      "has_discussions": false,
      "has_downloads": true,
      "has_issues": true,
      "has_pages": false,
      "has_projects": true,
      "has_wiki": false,
      "homepage": null,
      "hooks_url": "https://api.github.com/repos/example-org/example-repo/hooks",
      "html_url": "https://github.com/example-org/example-repo",
      "id": 987654321,
      "is_template": false,
      "issue_comment_url": "https://api.github.com/repos/example-org/example-repo/issues/comments{/number}",
      "issue_events_url": "https://api.github.com/repos/example-org/example-repo/issues/events{/number}",
      "issues_url": "https://api.github.com/repos/example-org/example-repo/issues{/number}",
      "keys_url": "https://api.github.com/repos/example-org/example-repo/keys{/key_id}",
      "labels_url": "https://api.github.com/repos/example-org/example-repo/labels{/name}",
      "language": "TypeScript",
      "languages_url": "https://api.github.com/repos/example-org/example-repo/languages",
      "license": {
        "key": "apache-2.0",
        "name": "Apache License 2.0",
        "node_id": "MDc6TGljZW5zZTI=",
        "spdx_id": "Apache-2.0",
        "url": "https://api.github.com/licenses/apache-2.0"
      },
      "master_branch": "main",
      "merges_url": "https://api.github.com/repos/example-org/example-repo/merges",
      "milestones_url": "https://api.github.com/repos/example-org/example-repo/milestones{/number}",
      "mirror_url": null,
      "name": "example-repo",
      "node_id": "R_kgDOAbCdEf",
      "notifications_url": "https://api.github.com/repos/example-org/example-repo/notifications{?since,all,participating}",
      "open_issues": 5,
      "open_issues_count": 5,
      "organization": "example-org",
      "owner": {
        "avatar_url": "https://avatars.githubusercontent.com/u/12345678?v=4",
        "email": null,
        "events_url": "https://api.github.com/users/example-org/events{/privacy}",
        "followers_url": "https://api.github.com/users/example-org/followers",
        "following_url": "https://api.github.com/users/example-org/following{/other_user}",
        "gists_url": "https://api.github.com/users/example-org/gists{/gist_id}",
        "gravatar_id": "",
        "html_url": "https://github.com/example-org",
        "id": 12345678,
        "login": "example-org",
        "name": "example-org",
        "node_id": "O_kgDOBb1AaA",
        "organizations_url": "https://api.github.com/users/example-org/orgs",
        "received_events_url": "https://api.github.com/users/example-org/received_events",
        "repos_url": "https://api.github.com/users/example-org/repos",
        "site_admin": false,
        "starred_url": "https://api.github.com/users/example-org/starred{/owner}{/repo}",
        "subscriptions_url": "https://api.github.com/users/example-org/subscriptions",
        "type": "Organization",
        "url": "https://api.github.com/users/example-org",
        "user_view_type": "public"
      },
      "private": false,
      "pulls_url": "https://api.github.com/repos/example-org/example-repo/pulls{/number}",
      "pushed_at": 1760000000,
      "releases_url": "https://api.github.com/repos/example-org/example-repo/releases{/id}",
      "size": 48200,
      "ssh_url": "git@github.com:example-org/example-repo.git",
      "stargazers": 3,
      "stargazers_count": 3,
      "stargazers_url": "https://api.github.com/repos/example-org/example-repo/stargazers",
      "statuses_url": "https://api.github.com/repos/example-org/example-repo/statuses/{sha}",
      "subscribers_url": "https://api.github.com/repos/example-org/example-repo/subscribers",
      "subscription_url": "https://api.github.com/repos/example-org/example-repo/subscription",
      "svn_url": "https://github.com/example-org/example-repo",
      "tags_url": "https://api.github.com/repos/example-org/example-repo/tags",
      "teams_url": "https://api.github.com/repos/example-org/example-repo/teams",
      "topics": [],
      "trees_url": "https://api.github.com/repos/example-org/example-repo/git/trees{/sha}",
      "updated_at": "2026-03-10T13:50:00Z",
      "url": "https://api.github.com/repos/example-org/example-repo",
      "visibility": "public",
      "watchers": 3,
      "watchers_count": 3,
      "web_commit_signoff_required": false
    },
    "sender": {
      "avatar_url": "https://avatars.githubusercontent.com/u/87654321?v=4",
      "events_url": "https://api.github.com/users/octo-user/events{/privacy}",
      "followers_url": "https://api.github.com/users/octo-user/followers",
      "following_url": "https://api.github.com/users/octo-user/following{/other_user}",
      "gists_url": "https://api.github.com/users/octo-user/gists{/gist_id}",
      "gravatar_id": "",
      "html_url": "https://github.com/octo-user",
      "id": 87654321,
      "login": "octo-user",
      "node_id": "MDQ6VXNlcjg3NjU0MzIx",
      "organizations_url": "https://api.github.com/users/octo-user/orgs",
      "received_events_url": "https://api.github.com/users/octo-user/received_events",
      "repos_url": "https://api.github.com/users/octo-user/repos",
      "site_admin": false,
      "starred_url": "https://api.github.com/users/octo-user/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/octo-user/subscriptions",
      "type": "User",
      "url": "https://api.github.com/users/octo-user",
      "user_view_type": "public"
    }
  },
  "timestamp": "2026-03-10T13:35:00.31254162Z",
  "type": "github.push"
}
```

### On Release

Listen to release events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| actions | Actions | multi-select | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "action": "published",
    "release": {
      "html_url": "https://github.com/acme/widgets/releases/tag/v1.2.3",
      "id": 3001,
      "name": "Release 1.2.3",
      "tag_name": "v1.2.3"
    },
    "repository": {
      "full_name": "acme/widgets",
      "html_url": "https://github.com/acme/widgets",
      "id": 123456
    },
    "sender": {
      "html_url": "https://github.com/octocat",
      "id": 101,
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.release"
}
```

### On Tag Created

Listen to GitHub tag creation events

## Configuration

| Name | Label | Type | Required | Description |
| --- | --- | --- | --- | --- |
| repository | Repository | app-installation-resource | yes | - |
| tags | Tags | any-predicate-list | yes | - |
| customName | Run title (optional) | string | no | Optional run title template. Supports expressions like {{ $.data }}. |

## Example Data

```json
{
  "data": {
    "description": "Example repository for webhook payloads",
    "master_branch": "main",
    "pusher_type": "user",
    "ref": "v1.2.3",
    "ref_type": "tag",
    "repository": {
      "full_name": "acme/widgets",
      "html_url": "https://github.com/acme/widgets",
      "id": 123456
    },
    "sender": {
      "html_url": "https://github.com/octocat",
      "id": 101,
      "login": "octocat"
    }
  },
  "timestamp": "2026-01-16T17:56:16.680755501Z",
  "type": "github.tagCreated"
}
```

