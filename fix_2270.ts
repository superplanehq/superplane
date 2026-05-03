import requests
import json
from typing import List, Dict, Optional
from datetime import datetime

class GitHubIssueFetcher:
    def __init__(self, token: Optional[str] = None):
        self.base_url = "https://api.github.com"
        self.headers = {
            "Accept": "application/vnd.github.v3+json"
        }
        if token:
            self.headers["Authorization"] = f"token {token}"
    
    def search_issues(
        self,
        repo_owner: str,
        repo_name: str,
        state: Optional[str] = None,
        labels: Optional[List[str]] = None,
        assignee: Optional[str] = None,
        author: Optional[str] = None,
        mentioned: Optional[str] = None,
        is_issue: bool = True,
        is_pull_request: bool = False,
        sort: str = "created",
        direction: str = "desc",
        per_page: int = 30,
        page: int = 1,
        search_filter: Optional[str] = None
    ) -> Dict:
        """
        Search and list issues for a GitHub repository.
        
        Args:
            repo_owner: Repository owner (user or organization)
            repo_name: Repository name
            state: Issue state (open, closed, all)
            labels: List of labels to filter by
            assignee: Filter by assignee username
            author: Filter by author username
            mentioned: Filter by mentioned username
            is_issue: Filter for issues only
            is_pull_request: Filter for pull requests only
            sort: Sort field (created, updated, comments)
            direction: Sort direction (asc, desc)
            per_page: Results per page (max 100)
            page: Page number
            search_filter: Additional search query string
            
        Returns:
            Dictionary with issues data and pagination info
        """
        # Build search query
        query_parts = [f"repo:{repo_owner}/{repo_name}"]
        
        if state and state != "all":
            query_parts.append(f"state:{state}")
        
        if labels:
            for label in labels:
                query_parts.append(f'label:"{label}"')
        
        if assignee:
            query_parts.append(f"assignee:{assignee}")
        
        if author:
            query_parts.append(f"author:{author}")
        
        if mentioned:
            query_parts.append(f"mentions:{mentioned}")
        
        if is_issue and not is_pull_request:
            query_parts.append("is:issue")
        elif is_pull_request and not is_issue:
            query_parts.append("is:pull-request")
        
        if search_filter:
            query_parts.append(search_filter)
        
        query = " ".join(query_parts)
        
        # Make API request
        params = {
            "q": query,
            "sort": sort,
            "order": direction,
            "per_page": min(per_page, 100),
            "page": page
        }
        
        response = requests.get(
            f"{self.base_url}/search/issues",
            headers=self.headers,
            params=params
        )
        
        if response.status_code != 200:
            raise Exception(f"GitHub API error: {response.status_code} - {response.text}")
        
        data = response.json()
        
        # Parse and format issues
        issues = []
        for item in data.get("items", []):
            issue = {
                "number": item["number"],
                "title": item["title"],
                "state": item["state"],
                "body": item["body"],
                "html_url": item["html_url"],
                "created_at": item["created_at"],
                "updated_at": item["updated_at"],
                "closed_at": item.get("closed_at"),
                "labels": [label["name"] for label in item["labels"]],
                "assignees": [assignee["login"] for assignee in item["assignees"]],
                "user": item["user"]["login"],
                "comments": item["comments"],
                "is_pull_request": "pull_request" in item
            }
            issues.append(issue)
        
        return {
            "total_count": data["total_count"],
            "incomplete_results": data["incomplete_results"],
            "issues": issues,
            "page": page,
            "per_page": per_page,
            "has_next": len(issues) == per_page
        }
    
    def get_repository_issues(
        self,
        repo_owner: str,
        repo_name: str,
        state: str = "open",
        labels: Optional[List[str]] = None,
        assignee: Optional[str] = None,
        sort: str = "created",
        direction: str = "desc",
        per_page: int = 30,
        page: int = 1
    ) -> Dict:
        """
        Get issues for a repository using the Issues API (not search).
        This is more efficient for simple queries.
        """
        url = f"{self.base_url}/repos/{repo_owner}/{repo_name}/issues"
        
        params = {
            "state": state,
            "sort": sort,
            "direction": direction,
            "per_page": min(per_page, 100),
            "page": page
        }
        
        if labels:
            params["labels"] = ",".join(labels)
        
        if assignee:
            params["assignee"] = assignee
        
        response = requests.get(url, headers=self.headers, params=params)
        
        if response.status_code != 200:
            raise Exception(f"GitHub API error: {response.status_code} - {response.text}")
        
        issues = response.json()
        
        # Parse and format issues
        formatted_issues = []
        for item in issues:
            issue = {
                "number": item["number"],
                "title": item["title"],
                "state": item["state"],
                "body": item["body"],
                "html_url": item["html_url"],
                "created_at": item["created_at"],
                "updated_at": item["updated_at"],
                "closed_at": item.get("closed_at"),
                "labels": [label["name"] for label in item["labels"]],
                "assignees": [assignee["login"] for assignee in item["assignees"]],
                "user": item["user"]["login"],
                "comments": item["comments"],
                "is_pull_request": "pull_request" in item
            }
            formatted_issues.append(issue)
        
        # Check if there are more pages
        link_header = response.headers.get("Link", "")
        has_next = 'rel="next"' in link_header
        
        return {
            "total_count": len(formatted_issues),
            "issues": formatted_issues,
            "page": page,
            "per_page": per_page,
            "has_next": has_next
        }


# Example usage
if __name__ == "__main__":
    # Initialize fetcher (optional: add GitHub token for higher rate limits)
    fetcher = GitHubIssueFetcher()
    
    # Example 1: Search for open issues with specific labels
    result = fetcher.search_issues(
        repo_owner="octocat",
        repo_name="Hello-World",
        state="open",
        labels=["bug", "enhancement"],
        sort="updated",
        per_page=10
    )
    
    print(f"Found {result['total_count']} issues")
    for issue in result['issues']:
        print(f"#{issue['number']}: {issue['title']} ({issue['state']})")
    
    # Example 2: Get issues assigned to a specific user
    result = fetcher.get_repository_issues(
        repo_owner="octocat",
        repo_name="Hello-World",
        state="open",
        assignee="octocat"
    )
    
    print(f"\nIssues assigned to octocat: {len(result['issues'])}")
    
    # Example 3: Search with custom filter
    result = fetcher.search_issues(
        repo_owner="octocat",
        repo_name="Hello-World",
        search_filter="is:open is:issue no:assignee"
    )
    
    print(f"\nUnassigned open issues: {result['total_count']}")
