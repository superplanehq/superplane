---
app: "github"
label: "On Push"
name: "github.onPush"
type: "trigger"
---

# On Push

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

