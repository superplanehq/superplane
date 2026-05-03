// GitHub Issue Fetcher - Complete Solution
const axios = require('axios');

class GitHubIssueFetcher {
  constructor(token = null) {
    this.baseURL = 'https://api.github.com';
    this.headers = {
      'Accept': 'application/vnd.github.v3+json',
      'User-Agent': 'GitHub-Issue-Fetcher/1.0'
    };
    if (token) {
      this.headers['Authorization'] = `token ${token}`;
    }
  }

  async getIssues(repo, options = {}) {
    const {
      state = 'open',
      labels = null,
      assignee = null,
      creator = null,
      mentioned = null,
      milestone = null,
      sort = 'created',
      direction = 'desc',
      per_page = 100,
      page = 1,
      searchFilter = null
    } = options;

    let query = `repo:${repo} type:issue state:${state}`;

    if (labels) query += ` label:${labels}`;
    if (assignee) query += ` assignee:${assignee}`;
    if (creator) query += ` author:${creator}`;
    if (mentioned) query += ` involves:${mentioned}`;
    if (milestone) query += ` milestone:${milestone}`;
    if (searchFilter) query += ` ${searchFilter}`;

    try {
      const response = await axios.get(`${this.baseURL}/search/issues`, {
        headers: this.headers,
        params: {
          q: query,
          sort,
          direction,
          per_page,
          page
        }
      });

      return {
        total_count: response.data.total_count,
        incomplete_results: response.data.incomplete_results,
        issues: response.data.items.map(issue => ({
          id: issue.id,
          number: issue.number,
          title: issue.title,
          state: issue.state,
          body: issue.body,
          labels: issue.labels.map(label => ({
            name: label.name,
            color: label.color
          })),
          assignees: issue.assignees.map(assignee => ({
            login: assignee.login,
            avatar_url: assignee.avatar_url
          })),
          created_at: issue.created_at,
          updated_at: issue.updated_at,
          closed_at: issue.closed_at,
          html_url: issue.html_url,
          user: {
            login: issue.user.login,
            avatar_url: issue.user.avatar_url
          },
          comments: issue.comments,
          milestone: issue.milestone ? {
            title: issue.milestone.title,
            state: issue.milestone.state
          } : null
        }))
      };
    } catch (error) {
      if (error.response) {
        throw new Error(`GitHub API Error: ${error.response.status} - ${error.response.data.message}`);
      }
      throw new Error(`Network Error: ${error.message}`);
    }
  }

  async getIssueByNumber(repo, issueNumber) {
    try {
      const response = await axios.get(`${this.baseURL}/repos/${repo}/issues/${issueNumber}`, {
        headers: this.headers
      });
      return response.data;
    } catch (error) {
      if (error.response) {
        throw new Error(`GitHub API Error: ${error.response.status} - ${error.response.data.message}`);
      }
      throw new Error(`Network Error: ${error.message}`);
    }
  }

  async getAllIssues(repo, options = {}) {
    let allIssues = [];
    let page = 1;
    let hasMore = true;

    while (hasMore) {
      const result = await this.getIssues(repo, { ...options, page });
      allIssues = allIssues.concat(result.issues);
      
      if (result.issues.length < (options.per_page || 100)) {
        hasMore = false;
      } else {
        page++;
      }
    }

    return allIssues;
  }
}

// Usage example
async function main() {
  const fetcher = new GitHubIssueFetcher(process.env.GITHUB_TOKEN);
  
  try {
    // Example 1: Get open issues
    const openIssues = await fetcher.getIssues('octocat/Hello-World', {
      state: 'open',
      sort: 'updated',
      direction: 'desc'
    });
    console.log('Open Issues:', openIssues.total_count);

    // Example 2: Get issues with specific label
    const labeledIssues = await fetcher.getIssues('octocat/Hello-World', {
      labels: 'bug',
      state: 'all'
    });
    console.log('Bug Issues:', labeledIssues.total_count);

    // Example 3: Get all issues (paginated)
    const allIssues = await fetcher.getAllIssues('octocat/Hello-World', {
      state: 'all',
      per_page: 50
    });
    console.log('Total All Issues:', allIssues.length);

    // Example 4: Get specific issue
    const issue = await fetcher.getIssueByNumber('octocat/Hello-World', 1);
    console.log('Issue #1:', issue.title);

  } catch (error) {
    console.error('Error:', error.message);
  }
}

// Export for module usage
module.exports = GitHubIssueFetcher;

// Run if executed directly
if (require.main === module) {
  main();
}
