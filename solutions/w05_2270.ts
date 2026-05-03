// github-issue-fetcher.js
const axios = require('axios');

class GitHubIssueFetcher {
  constructor(token) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
    this.headers = {
      'Authorization': `token ${token}`,
      'Accept': 'application/vnd.github.v3+json'
    };
  }

  async getIssues(repoOwner, repoName, filters = {}) {
    try {
      let query = `repo:${repoOwner}/${repoName}`;
      
      // Build search query from filters
      if (filters.state) query += ` state:${filters.state}`;
      if (filters.label) query += ` label:${filters.label}`;
      if (filters.assignee) query += ` assignee:${filters.assignee}`;
      if (filters.author) query += ` author:${filters.author}`;
      if (filters.involves) query += ` involves:${filters.involves}`;
      if (filters.milestone) query += ` milestone:${filters.milestone}`;
      if (filters.is) query += ` is:${filters.is}`;
      if (filters.created) query += ` created:${filters.created}`;
      if (filters.updated) query += ` updated:${filters.updated}`;
      if (filters.comments) query += ` comments:${filters.comments}`;
      if (filters.no) query += ` no:${filters.no}`;
      if (filters.language) query += ` language:${filters.language}`;
      if (filters.user) query += ` user:${filters.user}`;
      if (filters.org) query += ` org:${filters.org}`;
      if (filters.repo) query += ` repo:${filters.repo}`;
      if (filters.team) query += ` team:${filters.team}`;
      if (filters.type) query += ` type:${filters.type}`;
      if (filters.sort) query += ` sort:${filters.sort}`;
      if (filters.order) query += ` order:${filters.order}`;
      if (filters.per_page) query += ` per_page:${filters.per_page}`;
      if (filters.page) query += ` page:${filters.page}`;

      // Add search filter if provided
      if (filters.search) query = `${filters.search} ${query}`;

      const response = await axios.get(`${this.baseUrl}/search/issues`, {
        headers: this.headers,
        params: { q: query }
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
            color: label.color,
            description: label.description
          })),
          assignees: issue.assignees.map(assignee => ({
            login: assignee.login,
            avatar_url: assignee.avatar_url
          })),
          milestone: issue.milestone ? {
            title: issue.milestone.title,
            state: issue.milestone.state,
            due_on: issue.milestone.due_on
          } : null,
          created_at: issue.created_at,
          updated_at: issue.updated_at,
          closed_at: issue.closed_at,
          html_url: issue.html_url,
          user: {
            login: issue.user.login,
            avatar_url: issue.user.avatar_url
          },
          comments: issue.comments,
          pull_request: issue.pull_request ? {
            url: issue.pull_request.url,
            html_url: issue.pull_request.html_url
          } : null
        }))
      };
    } catch (error) {
      console.error('Error fetching issues:', error.message);
      throw error;
    }
  }

  async getRepositoryIssues(repoOwner, repoName, options = {}) {
    const filters = {
      state: options.state || 'open',
      ...options
    };
    
    return this.getIssues(repoOwner, repoName, filters);
  }
}

// Example usage
async function main() {
  const fetcher = new GitHubIssueFetcher('YOUR_GITHUB_TOKEN');
  
  // Example: Get open issues with specific label
  const issues = await fetcher.getRepositoryIssues('octocat', 'Hello-World', {
    state: 'open',
    label: 'bug',
    sort: 'created',
    order: 'desc',
    per_page: 10
  });
  
  console.log('Total issues:', issues.total_count);
  console.log('Issues:', issues.issues);
}

// Export for use in other modules
module.exports = GitHubIssueFetcher;
