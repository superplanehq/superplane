// GitHub Issues Fetcher - Complete Solution
// File: github-issues-fetcher.js

class GitHubIssuesFetcher {
  constructor(token = null) {
    this.baseUrl = 'https://api.github.com';
    this.token = token;
    this.headers = {
      'Accept': 'application/vnd.github.v3+json',
      'Content-Type': 'application/json'
    };
    if (this.token) {
      this.headers['Authorization'] = `Bearer ${this.token}`;
    }
  }

  /**
   * Fetch issues from a GitHub repository with advanced filtering
   * @param {string} owner - Repository owner
   * @param {string} repo - Repository name
   * @param {Object} filters - Search filters
   * @param {string} [filters.state] - 'open', 'closed', or 'all'
   * @param {string} [filters.label] - Comma-separated labels
   * @param {string} [filters.assignee] - Username or 'none' or '*'
   * @param {string} [filters.author] - Username
   * @param {string} [filters.involves] - Username
   * @param {string} [filters.milestone] - Milestone number or '*'
   * @param {string} [filters.sort] - 'created', 'updated', 'comments'
   * @param {string} [filters.direction] - 'asc' or 'desc'
   * @param {number} [filters.per_page] - Results per page (max 100)
   * @param {number} [filters.page] - Page number
   * @returns {Promise<Array>} - Array of issue objects
   */
  async getIssues(owner, repo, filters = {}) {
    try {
      const queryParams = this.buildQueryParams(filters);
      const url = `${this.baseUrl}/repos/${owner}/${repo}/issues?${queryParams}`;
      
      const response = await fetch(url, {
        method: 'GET',
        headers: this.headers
      });

      if (!response.ok) {
        throw new Error(`GitHub API error: ${response.status} ${response.statusText}`);
      }

      const issues = await response.json();
      const linkHeader = response.headers.get('Link');
      
      return {
        issues: issues,
        pagination: this.parseLinkHeader(linkHeader),
        total: parseInt(response.headers.get('X-Total-Count') || issues.length)
      };
    } catch (error) {
      console.error('Error fetching issues:', error);
      throw error;
    }
  }

  /**
   * Build query string from filters
   * @param {Object} filters
   * @returns {string}
   */
  buildQueryParams(filters) {
    const params = new URLSearchParams();
    
    if (filters.state) params.append('state', filters.state);
    if (filters.label) params.append('labels', filters.label);
    if (filters.assignee) params.append('assignee', filters.assignee);
    if (filters.sort) params.append('sort', filters.sort);
    if (filters.direction) params.append('direction', filters.direction);
    if (filters.per_page) params.append('per_page', Math.min(filters.per_page, 100));
    if (filters.page) params.append('page', filters.page);
    
    return params.toString();
  }

  /**
   * Search issues across repositories with advanced qualifiers
   * @param {string} query - Search query
   * @param {Object} qualifiers - Search qualifiers
   * @param {number} [per_page] - Results per page
   * @param {number} [page] - Page number
   * @returns {Promise<Object>} - Search results
   */
  async searchIssues(query, qualifiers = {}, per_page = 30, page = 1) {
    try {
      let searchQuery = query;
      
      // Build qualifiers
      if (qualifiers.repo) searchQuery += ` repo:${qualifiers.repo}`;
      if (qualifiers.state) searchQuery += ` state:${qualifiers.state}`;
      if (qualifiers.label) searchQuery += ` label:${qualifiers.label}`;
      if (qualifiers.assignee) searchQuery += ` assignee:${qualifiers.assignee}`;
      if (qualifiers.author) searchQuery += ` author:${qualifiers.author}`;
      if (qualifiers.involves) searchQuery += ` involves:${qualifiers.involves}`;
      if (qualifiers.milestone) searchQuery += ` milestone:${qualifiers.milestone}`;
      if (qualifiers.is) searchQuery += ` is:${qualifiers.is}`;
      if (qualifiers.created) searchQuery += ` created:${qualifiers.created}`;
      if (qualifiers.updated) searchQuery += ` updated:${qualifiers.updated}`;
      if (qualifiers.comments) searchQuery += ` comments:${qualifiers.comments}`;
      if (qualifiers.no) searchQuery += ` no:${qualifiers.no}`;
      if (qualifiers.language) searchQuery += ` language:${qualifiers.language}`;
      if (qualifiers.user) searchQuery += ` user:${qualifiers.user}`;
      if (qualifiers.org) searchQuery += ` org:${qualifiers.org}`;

      const url = `${this.baseUrl}/search/issues?q=${encodeURIComponent(searchQuery)}&per_page=${per_page}&page=${page}`;
      
      const response = await fetch(url, {
        method: 'GET',
        headers: this.headers
      });

      if (!response.ok) {
        throw new Error(`GitHub API error: ${response.status} ${response.statusText}`);
      }

      const data = await response.json();
      return {
        total_count: data.total_count,
        incomplete_results: data.incomplete_results,
        items: data.items,
        pagination: this.parseLinkHeader(response.headers.get('Link'))
      };
    } catch (error) {
      console.error('Error searching issues:', error);
      throw error;
    }
  }

  /**
   * Parse GitHub Link header for pagination
   * @param {string} linkHeader
   * @returns {Object|null}
   */
  parseLinkHeader(linkHeader) {
    if (!linkHeader) return null;
    
    const links = {};
    const parts = linkHeader.split(',');
    
    parts.forEach(part => {
      const section = part.split(';');
      if (section.length !== 2) return;
      
      const url = section[0].replace(/<(.*)>/, '$1').trim();
      const name = section[1].replace(/rel="(.*)"/, '$1').trim();
      
      links[name] = url;
    });
    
    return links;
  }

  /**
   * Get all issues (handles pagination automatically)
   * @param {string} owner
   * @param {string} repo
   * @param {Object} filters
   * @returns {Promise<Array>}
   */
  async getAllIssues(owner, repo, filters = {}) {
    let allIssues = [];
    let page = 1;
    let hasMore = true;
    
    while (hasMore) {
      const result = await this.getIssues(owner, repo, { ...filters, page, per_page: 100 });
      allIssues = allIssues.concat(result.issues);
      
      if (result.pagination && result.pagination.next) {
        page++;
      } else {
        hasMore = false;
      }
    }
    
    return allIssues;
  }

  /**
   * Format issues for export (CSV compatible)
   * @param {Array} issues
   * @returns {Array}
   */
  formatIssuesForExport(issues) {
    return issues.map(issue => ({
      id: issue.id,
      number: issue.number,
      title: issue.title,
      state: issue.state,
      body: issue.body,
      created_at: issue.created_at,
      updated_at: issue.updated_at,
      closed_at: issue.closed_at,
      labels: issue.labels.map(l => l.name).join(', '),
      assignees: issue.assignees.map(a => a.login).join(', '),
      author: issue.user.login,
      milestone: issue.milestone ? issue.milestone.title : null,
      comments: issue.comments,
      url: issue.html_url
    }));
  }
}

// Export for use in Node.js or browser
if (typeof module !== 'undefined' && module.exports) {
  module.exports = GitHubIssuesFetcher;
} else {
  window.GitHubIssuesFetcher = GitHubIssuesFetcher;
}
