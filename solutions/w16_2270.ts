// github-issues.js
const axios = require('axios');

class GitHubIssues {
  constructor(token) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
  }

  async searchIssues({ 
    query = '', 
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
      let searchQuery = query;
      
      if (repo) {
        searchQuery += ` repo:${repo}`;
      }
      
      if (state) {
        searchQuery += ` state:${state}`;
      }
      
      if (labels) {
        const labelList = labels.split(',').map(l => l.trim());
        labelList.forEach(label => {
          searchQuery += ` label:"${label}"`;
        });
      }
      
      if (assignee) {
        searchQuery += ` assignee:${assignee}`;
      }
      
      if (author) {
        searchQuery += ` author:${author}`;
      }
      
      if (mentioned) {
        searchQuery += ` mentions:${mentioned}`;
      }

      const response = await axios.get(`${this.baseUrl}/search/issues`, {
        headers: {
          'Authorization': `token ${this.token}`,
          'Accept': 'application/vnd.github.v3+json'
        },
        params: {
          q: searchQuery.trim(),
          sort,
          direction,
          per_page: perPage,
          page
        }
      });

      return {
        totalCount: response.data.total_count,
        incompleteResults: response.data.incomplete_results,
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
            avatarUrl: assignee.avatar_url
          })),
          user: {
            login: issue.user.login,
            avatarUrl: issue.user.avatar_url
          },
          createdAt: issue.created_at,
          updatedAt: issue.updated_at,
          closedAt: issue.closed_at,
          comments: issue.comments,
          url: issue.html_url,
          repositoryUrl: issue.repository_url
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
        headers: {
          'Authorization': `token ${this.token}`,
          'Accept': 'application/vnd.github.v3+json'
        },
        params: {
          state: options.state || 'open',
          labels: options.labels,
          assignee: options.assignee,
          creator: options.creator,
          mentioned: options.mentioned,
          sort: options.sort || 'created',
          direction: options.direction || 'desc',
          per_page: options.perPage || 30,
          page: options.page || 1,
          since: options.since
        }
      });

      return response.data.map(issue => ({
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
          avatarUrl: assignee.avatar_url
        })),
        user: {
          login: issue.user.login,
          avatarUrl: issue.user.avatar_url
        },
        createdAt: issue.created_at,
        updatedAt: issue.updated_at,
        closedAt: issue.closed_at,
        comments: issue.comments,
        url: issue.html_url,
        milestone: issue.milestone ? {
          title: issue.milestone.title,
          state: issue.milestone.state,
          dueOn: issue.milestone.due_on
        } : null
      }));
    } catch (error) {
      if (error.response) {
        throw new Error(`GitHub API Error: ${error.response.status} - ${error.response.data.message}`);
      }
      throw new Error(`Network Error: ${error.message}`);
    }
  }
}

module.exports = GitHubIssues;
