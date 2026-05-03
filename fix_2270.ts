// GitHub Issue Search Module
const axios = require('axios');

class GitHubIssueSearch {
  constructor(token) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
  }

  async searchIssues({
    repo,
    state = 'open',
    labels = '',
    assignee = '',
    author = '',
    involved = '',
    searchFilter = '',
    page = 1,
    perPage = 30
  }) {
    try {
      let query = `repo:${repo}`;
      
      if (state) query += ` state:${state}`;
      if (labels) query += ` label:${labels}`;
      if (assignee) query += ` assignee:${assignee}`;
      if (author) query += ` author:${author}`;
      if (involved) query += ` involves:${involved}`;
      if (searchFilter) query += ` ${searchFilter}`;

      const response = await axios.get(`${this.baseUrl}/search/issues`, {
        headers: {
          Authorization: `Bearer ${this.token}`,
          Accept: 'application/vnd.github.v3+json'
        },
        params: {
          q: query,
          page,
          per_page: perPage
        }
      });

      return {
        success: true,
        data: response.data,
        total: response.data.total_count,
        issues: response.data.items
      };
    } catch (error) {
      return {
        success: false,
        error: error.response?.data?.message || error.message
      };
    }
  }

  async getRepositoryIssues(repo, options = {}) {
    const result = await this.searchIssues({
      repo,
      ...options
    });

    if (!result.success) {
      throw new Error(`Failed to fetch issues: ${result.error}`);
    }

    return result;
  }
}

// Usage example
async function main() {
  const token = 'YOUR_GITHUB_TOKEN';
  const searcher = new GitHubIssueSearch(token);

  try {
    // Example: Get open issues for a repository
    const issues = await searcher.getRepositoryIssues('octocat/Hello-World', {
      state: 'open',
      labels: 'bug,enhancement',
      assignee: 'octocat',
      perPage: 10
    });

    console.log(`Total issues found: ${issues.total}`);
    issues.issues.forEach(issue => {
      console.log(`#${issue.number}: ${issue.title} (${issue.state})`);
    });
  } catch (error) {
    console.error('Error:', error.message);
  }
}

// Export for use in other modules
module.exports = GitHubIssueSearch;
