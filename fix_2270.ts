// GitHub Issue Fetcher - Complete Solution
const axios = require('axios');

class GitHubIssueFetcher {
  constructor(token = null) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
    this.headers = {
      'Accept': 'application/vnd.github.v3+json',
      'User-Agent': 'GitHub-Issue-Fetcher/1.0'
    };
    
    if (this.token) {
      this.headers['Authorization'] = `token ${this.token}`;
    }
  }

  /**
   * Parse search filter string into individual qualifiers
   * @param {string} searchFilter - Top-level search filter
   * @returns {Object} Parsed qualifiers
   */
  parseSearchFilter(searchFilter) {
    const qualifiers = {
      state: 'open',
      labels: [],
      assignee: null,
      author: null,
      mentioned: null,
      milestone: null,
      is: [],
      sort: 'created',
      direction: 'desc',
      per_page: 30,
      page: 1
    };

    if (!searchFilter) return qualifiers;

    // Parse state
    if (searchFilter.includes('is:open')) qualifiers.state = 'open';
    if (searchFilter.includes('is:closed')) qualifiers.state = 'closed';
    if (searchFilter.includes('is:issue')) qualifiers.is.push('issue');
    if (searchFilter.includes('is:pr')) qualifiers.is.push('pull-request');

    // Parse labels
    const labelMatch = searchFilter.match(/label:["']?([^"'\s]+)["']?/g);
    if (labelMatch) {
      qualifiers.labels = labelMatch.map(l => l.replace(/label:["']?/, '').replace(/["']?$/, ''));
    }

    // Parse assignee
    const assigneeMatch = searchFilter.match(/assignee:(\S+)/);
    if (assigneeMatch) qualifiers.assignee = assigneeMatch[1];

    // Parse author
    const authorMatch = searchFilter.match(/author:(\S+)/);
    if (authorMatch) qualifiers.author = authorMatch[1];

    // Parse mentioned
    const mentionedMatch = searchFilter.match(/mentions:(\S+)/);
    if (mentionedMatch) qualifiers.mentioned = mentionedMatch[1];

    // Parse milestone
    const milestoneMatch = searchFilter.match(/milestone:["']?([^"'\s]+)["']?/);
    if (milestoneMatch) qualifiers.milestone = milestoneMatch[1];

    // Parse sort
    const sortMatch = searchFilter.match(/sort:(\S+)/);
    if (sortMatch) qualifiers.sort = sortMatch[1];

    // Parse direction
    const directionMatch = searchFilter.match(/direction:(\S+)/);
    if (directionMatch) qualifiers.direction = directionMatch[1];

    return qualifiers;
  }

  /**
   * Build search query from individual fields
   * @param {Object} fields - Individual field controls
   * @returns {string} Search query string
   */
  buildSearchQuery(fields) {
    const parts = [];
    
    if (fields.state) parts.push(`is:${fields.state}`);
    if (fields.is && fields.is.length > 0) {
      fields.is.forEach(i => parts.push(`is:${i}`));
    }
    if (fields.labels && fields.labels.length > 0) {
      fields.labels.forEach(label => parts.push(`label:"${label}"`));
    }
    if (fields.assignee) parts.push(`assignee:${fields.assignee}`);
    if (fields.author) parts.push(`author:${fields.author}`);
    if (fields.mentioned) parts.push(`mentions:${fields.mentioned}`);
    if (fields.milestone) parts.push(`milestone:"${fields.milestone}"`);
    
    return parts.join(' ');
  }

  /**
   * Fetch issues from GitHub repository
   * @param {string} owner - Repository owner
   * @param {string} repo - Repository name
   * @param {Object} options - Search options
   * @returns {Promise<Array>} List of issues
   */
  async fetchIssues(owner, repo, options = {}) {
    try {
      // Parse search filter if provided
      const qualifiers = this.parseSearchFilter(options.searchFilter || '');
      
      // Merge with individual field controls
      const mergedOptions = {
        ...qualifiers,
        ...options,
        owner: owner || options.owner,
        repo: repo || options.repo
      };

      // Build search query
      const queryParts = [`repo:${mergedOptions.owner}/${mergedOptions.repo}`];
      
      if (mergedOptions.state) queryParts.push(`is:${mergedOptions.state}`);
      if (mergedOptions.is && mergedOptions.is.length > 0) {
        mergedOptions.is.forEach(i => queryParts.push(`is:${i}`));
      }
      if (mergedOptions.labels && mergedOptions.labels.length > 0) {
        mergedOptions.labels.forEach(label => queryParts.push(`label:"${label}"`));
      }
      if (mergedOptions.assignee) queryParts.push(`assignee:${mergedOptions.assignee}`);
      if (mergedOptions.author) queryParts.push(`author:${mergedOptions.author}`);
      if (mergedOptions.mentioned) queryParts.push(`mentions:${mergedOptions.mentioned}`);
      if (mergedOptions.milestone) queryParts.push(`milestone:"${mergedOptions.milestone}"`);

      const query = queryParts.join(' ');

      // Make API request
      const response = await axios.get(`${this.baseUrl}/search/issues`, {
        headers: this.headers,
        params: {
          q: query,
          sort: mergedOptions.sort || 'created',
          order: mergedOptions.direction || 'desc',
          per_page: Math.min(mergedOptions.per_page || 30, 100),
          page: mergedOptions.page || 1
        }
      });

      // Format response
      return {
        total_count: response.data.total_count,
        incomplete_results: response.data.incomplete_results,
        issues: response.data.items.map(item => ({
          id: item.id,
          number: item.number,
          title: item.title,
          state: item.state,
          body: item.body,
          labels: item.labels.map(l => ({
            name: l.name,
            color: l.color,
            description: l.description
          })),
          assignees: item.assignees.map(a => ({
            login: a.login,
            avatar_url: a.avatar_url,
            html_url: a.html_url
          })),
          user: {
            login: item.user.login,
            avatar_url: item.user.avatar_url,
            html_url: item.user.html_url
          },
          milestone: item.milestone ? {
            title: item.milestone.title,
            state: item.milestone.state,
            due_on: item.milestone.due_on
          } : null,
          created_at: item.created_at,
          updated_at: item.updated_at,
          closed_at: item.closed_at,
          html_url: item.html_url,
          comments: item.comments,
          pull_request: item.pull_request ? {
            url: item.pull_request.html_url,
            merged_at: item.pull_request.merged_at
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

  /**
   * Get issues with full field control
   * @param {Object} config - Configuration object
   * @returns {Promise<Array>} List of issues
   */
  async getIssues(config) {
    const {
      owner,
      repo,
      searchFilter,
      state,
      labels,
      assignee,
      author,
      mentioned,
      milestone,
      is,
      sort,
      direction,
      per_page,
      page
    } = config;

    // Build options from individual fields
    const options = {
      owner,
      repo,
      searchFilter,
      state: state || 'open',
      labels: labels || [],
      assignee: assignee || null,
      author: author || null,
      mentioned: mentioned || null,
      milestone: milestone || null,
      is: is || [],
      sort: sort || 'created',
      direction: direction || 'desc',
      per_page: per_page || 30,
      page: page || 1
    };

    return await this.fetchIssues(owner, repo, options);
  }
}

// Example usage and testing
async function main() {
  // Initialize fetcher (optional: add GitHub token for higher rate limits)
  const fetcher = new GitHubIssueFetcher(process.env.GITHUB_TOKEN);

  // Example 1: Simple search with filter
  const result1 = await fetcher.getIssues({
    owner: 'facebook',
    repo: 'react',
    searchFilter: 'is:open label:bug sort:created-desc',
    per_page: 5
  });
  console.log('Example 1 - Open bugs in React:', result1.total_count, 'issues found');

  // Example 2: Full field control
  const result2 = await fetcher.getIssues({
    owner: 'microsoft',
    repo: 'vscode',
    state: 'closed',
    labels: ['bug', 'help wanted'],
    assignee: 'username',
    sort: 'updated',
    direction: 'desc',
    per_page: 10
  });
  console.log('Example 2 - Closed bugs assigned to user:', result2.total_count, 'issues found');

  // Example 3: Export format for Jira/Slack
  const result3 = await fetcher.getIssues({
    owner: 'vercel',
    repo: 'next.js',
    state: 'open',
    labels: ['documentation'],
    per_page: 20
  });
  
  // Format for export
  const exportData = result3.issues.map(issue => ({
    id: issue.number,
    title: issue.title,
    status: issue.state,
    url: issue.html_url,
    created: issue.created_at,
    labels: issue.labels.map(l => l.name).join(', ')
  }));
  console.log('Example 3 - Export ready data:', JSON.stringify(exportData, null, 2));
}

// Run if called directly
if (require.main === module) {
  main().catch(console.error);
}

module.exports = GitHubIssueFetcher;
