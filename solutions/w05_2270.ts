// GitHub Issue Search Tool - Complete Working Solution
const axios = require('axios');

class GitHubIssueSearch {
  constructor(token) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
    this.headers = {
      'Authorization': `token ${this.token}`,
      'Accept': 'application/vnd.github.v3+json'
    };
  }

  async searchIssues({
    searchFilter = '',
    repo = '',
    state = 'open',
    labels = '',
    assignee = '',
    author = '',
    mentioned = '',
    sort = 'created',
    direction = 'desc',
    perPage = 30,
    page = 1
  } = {}) {
    try {
      // Build search query
      let query = searchFilter;
      
      if (repo) {
        query += ` repo:${repo}`;
      }
      
      if (state) {
        query += ` state:${state}`;
      }
      
      if (labels) {
        const labelList = labels.split(',').map(l => l.trim());
        labelList.forEach(label => {
          query += ` label:"${label}"`;
        });
      }
      
      if (assignee) {
        query += ` assignee:${assignee}`;
      }
      
      if (author) {
        query += ` author:${author}`;
      }
      
      if (mentioned) {
        query += ` mentions:${mentioned}`;
      }

      // Make API request
      const response = await axios.get(`${this.baseUrl}/search/issues`, {
        headers: this.headers,
        params: {
          q: query.trim(),
          sort: sort,
          order: direction,
          per_page: perPage,
          page: page
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
          html_url: issue.html_url,
          created_at: issue.created_at,
          updated_at: issue.updated_at,
          closed_at: issue.closed_at,
          labels: issue.labels.map(label => ({
            name: label.name,
            color: label.color
          })),
          assignees: issue.assignees.map(assignee => ({
            login: assignee.login,
            avatar_url: assignee.avatar_url
          })),
          user: {
            login: issue.user.login,
            avatar_url: issue.user.avatar_url
          },
          comments: issue.comments,
          body: issue.body
        }))
      };
    } catch (error) {
      if (error.response) {
        throw new Error(`GitHub API Error: ${error.response.status} - ${error.response.data.message}`);
      }
      throw new Error(`Network Error: ${error.message}`);
    }
  }

  async getRepositoryIssues(owner, repo, options = {}) {
    try {
      const response = await axios.get(`${this.baseUrl}/repos/${owner}/${repo}/issues`, {
        headers: this.headers,
        params: {
          state: options.state || 'open',
          labels: options.labels || '',
          assignee: options.assignee || '',
          sort: options.sort || 'created',
          direction: options.direction || 'desc',
          per_page: options.perPage || 30,
          page: options.page || 1
        }
      });

      return response.data.map(issue => ({
        id: issue.id,
        number: issue.number,
        title: issue.title,
        state: issue.state,
        html_url: issue.html_url,
        created_at: issue.created_at,
        updated_at: issue.updated_at,
        closed_at: issue.closed_at,
        labels: issue.labels.map(label => ({
          name: label.name,
          color: label.color
        })),
        assignees: issue.assignees.map(assignee => ({
          login: assignee.login,
          avatar_url: assignee.avatar_url
        })),
        user: {
          login: issue.user.login,
          avatar_url: issue.user.avatar_url
        },
        comments: issue.comments,
        body: issue.body
      }));
    } catch (error) {
      if (error.response) {
        throw new Error(`GitHub API Error: ${error.response.status} - ${error.response.data.message}`);
      }
      throw new Error(`Network Error: ${error.message}`);
    }
  }
}

// Usage Example
async function main() {
  const githubToken = process.env.GITHUB_TOKEN || 'YOUR_GITHUB_TOKEN_HERE';
  const searcher = new GitHubIssueSearch(githubToken);

  try {
    // Example 1: Search for open issues with specific labels
    const searchResults = await searcher.searchIssues({
      repo: 'facebook/react',
      state: 'open',
      labels: 'bug,good first issue',
      sort: 'updated',
      direction: 'desc',
      perPage: 10
    });

    console.log('Search Results:');
    console.log(`Total issues found: ${searchResults.total_count}`);
    searchResults.issues.forEach(issue => {
      console.log(`#${issue.number}: ${issue.title} (${issue.state})`);
    });

    // Example 2: Get all issues for a specific repository
    const repoIssues = await searcher.getRepositoryIssues('facebook', 'react', {
      state: 'open',
      sort: 'created',
      direction: 'desc',
      perPage: 5
    });

    console.log('\nRepository Issues:');
    repoIssues.forEach(issue => {
      console.log(`#${issue.number}: ${issue.title}`);
    });

  } catch (error) {
    console.error('Error:', error.message);
  }
}

// Export for use in other modules
module.exports = GitHubIssueSearch;

// Run if executed directly
if (require.main === module) {
  main();
}
