// github-issues.js
const axios = require('axios');

class GitHubIssues {
  constructor(token) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
  }

  async searchIssues({ query, state, labels, assignee, author, involved, sort, direction, perPage, page }) {
    try {
      let searchQuery = query || '';
      
      if (state) searchQuery += ` state:${state}`;
      if (labels) searchQuery += ` label:${labels}`;
      if (assignee) searchQuery += ` assignee:${assignee}`;
      if (author) searchQuery += ` author:${author}`;
      if (involved) searchQuery += ` involves:${involved}`;

      const response = await axios.get(`${this.baseUrl}/search/issues`, {
        headers: {
          Authorization: `token ${this.token}`,
          Accept: 'application/vnd.github.v3+json'
        },
        params: {
          q: searchQuery.trim(),
          sort: sort || 'created',
          direction: direction || 'desc',
          per_page: perPage || 30,
          page: page || 1
        }
      });

      return {
        total_count: response.data.total_count,
        incomplete_results: response.data.incomplete_results,
        items: response.data.items.map(issue => ({
          id: issue.id,
          number: issue.number,
          title: issue.title,
          state: issue.state,
          labels: issue.labels.map(label => ({
            name: label.name,
            color: label.color,
            description: label.description
          })),
          assignees: issue.assignees.map(assignee => ({
            login: assignee.login,
            avatar_url: assignee.avatar_url,
            html_url: assignee.html_url
          })),
          user: {
            login: issue.user.login,
            avatar_url: issue.user.avatar_url,
            html_url: issue.user.html_url
          },
          created_at: issue.created_at,
          updated_at: issue.updated_at,
          closed_at: issue.closed_at,
          html_url: issue.html_url,
          body: issue.body,
          comments: issue.comments,
          pull_request: issue.pull_request ? {
            url: issue.pull_request.url,
            html_url: issue.pull_request.html_url
          } : null
        }))
      };
    } catch (error) {
      throw new Error(`GitHub API error: ${error.response?.data?.message || error.message}`);
    }
  }

  async listRepositoryIssues(owner, repo, options = {}) {
    try {
      const response = await axios.get(`${this.baseUrl}/repos/${owner}/${repo}/issues`, {
        headers: {
          Authorization: `token ${this.token}`,
          Accept: 'application/vnd.github.v3+json'
        },
        params: {
          state: options.state || 'open',
          labels: options.labels,
          assignee: options.assignee,
          creator: options.creator,
          mentioned: options.mentioned,
          sort: options.sort || 'created',
          direction: options.direction || 'desc',
          since: options.since,
          per_page: options.perPage || 30,
          page: options.page || 1,
          milestone: options.milestone
        }
      });

      return response.data.map(issue => ({
        id: issue.id,
        number: issue.number,
        title: issue.title,
        state: issue.state,
        labels: issue.labels.map(label => ({
          name: label.name,
          color: label.color,
          description: label.description
        })),
        assignees: issue.assignees.map(assignee => ({
          login: assignee.login,
          avatar_url: assignee.avatar_url,
          html_url: assignee.html_url
        })),
        user: {
          login: issue.user.login,
          avatar_url: issue.user.avatar_url,
          html_url: issue.user.html_url
        },
        created_at: issue.created_at,
        updated_at: issue.updated_at,
        closed_at: issue.closed_at,
        html_url: issue.html_url,
        body: issue.body,
        comments: issue.comments,
        pull_request: issue.pull_request ? {
          url: issue.pull_request.url,
          html_url: issue.pull_request.html_url
        } : null,
        milestone: issue.milestone ? {
          title: issue.milestone.title,
          number: issue.milestone.number,
          state: issue.milestone.state
        } : null
      }));
    } catch (error) {
      throw new Error(`GitHub API error: ${error.response?.data?.message || error.message}`);
    }
  }

  async getIssue(owner, repo, issueNumber) {
    try {
      const response = await axios.get(`${this.baseUrl}/repos/${owner}/${repo}/issues/${issueNumber}`, {
        headers: {
          Authorization: `token ${this.token}`,
          Accept: 'application/vnd.github.v3+json'
        }
      });

      const issue = response.data;
      return {
        id: issue.id,
        number: issue.number,
        title: issue.title,
        state: issue.state,
        labels: issue.labels.map(label => ({
          name: label.name,
          color: label.color,
          description: label.description
        })),
        assignees: issue.assignees.map(assignee => ({
          login: assignee.login,
          avatar_url: assignee.avatar_url,
          html_url: assignee.html_url
        })),
        user: {
          login: issue.user.login,
          avatar_url: issue.user.avatar_url,
          html_url: issue.user.html_url
        },
        created_at: issue.created_at,
        updated_at: issue.updated_at,
        closed_at: issue.closed_at,
        html_url: issue.html_url,
        body: issue.body,
        comments: issue.comments,
        pull_request: issue.pull_request ? {
          url: issue.pull_request.url,
          html_url: issue.pull_request.html_url
        } : null,
        milestone: issue.milestone ? {
          title: issue.milestone.title,
          number: issue.milestone.number,
          state: issue.milestone.state
        } : null
      };
    } catch (error) {
      throw new Error(`GitHub API error: ${error.response?.data?.message || error.message}`);
    }
  }
}

module.exports = GitHubIssues;
