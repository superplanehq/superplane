// GitHub Issue Fetcher - Complete Solution
const axios = require('axios');

class GitHubIssueFetcher {
  constructor(token = null) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
    this.headers = {
      'Accept': 'application/vnd.github.v3+json',
      ...(token && { 'Authorization': `token ${token}` })
    };
  }

  /**
   * Parse search filter string into qualifiers
   * @param {string} searchFilter - e.g., "is:open label:bug"
   * @returns {Object} Parsed qualifiers
   */
  parseSearchFilter(searchFilter) {
    const qualifiers = {
      state: 'open',
      labels: [],
      assignee: null,
      author: null,
      mentions: null,
      milestone: null,
      is: []
    };

    if (!searchFilter) return qualifiers;

    const parts = searchFilter.split(' ');
    parts.forEach(part => {
      const [key, value] = part.split(':');
      
      switch(key) {
        case 'is':
          qualifiers.is.push(value);
          if (value === 'open' || value === 'closed') {
            qualifiers.state = value;
          }
          break;
        case 'label':
          qualifiers.labels.push(value);
          break;
        case 'assignee':
          qualifiers.assignee = value;
          break;
        case 'author':
          qualifiers.author = value;
          break;
        case 'mentions':
          qualifiers.mentions = value;
          break;
        case 'milestone':
          qualifiers.milestone = value;
          break;
        case 'no':
          qualifiers[`no:${value}`] = true;
          break;
      }
    });

    return qualifiers;
  }

  /**
   * Build GitHub search query from qualifiers
   * @param {Object} qualifiers - Search qualifiers
   * @returns {string} Search query string
   */
  buildSearchQuery(qualifiers) {
    const parts = [];

    // State filter
    if (qualifiers.state) {
      parts.push(`is:${qualifiers.state}`);
    }

    // Labels
    if (qualifiers.labels && qualifiers.labels.length > 0) {
      qualifiers.labels.forEach(label => {
        parts.push(`label:"${label}"`);
      });
    }

    // Assignee
    if (qualifiers.assignee) {
      parts.push(`assignee:${qualifiers.assignee}`);
    }

    // Author
    if (qualifiers.author) {
      parts.push(`author:${qualifiers.author}`);
    }

    // Mentions
    if (qualifiers.mentions) {
      parts.push(`mentions:${qualifiers.mentions}`);
    }

    // Milestone
    if (qualifiers.milestone) {
      parts.push(`milestone:"${qualifiers.milestone}"`);
    }

    // No label/milestone/assignee
    if (qualifiers['no:label']) {
      parts.push('no:label');
    }
    if (qualifiers['no:milestone']) {
      parts.push('no:milestone');
    }
    if (qualifiers['no:assignee']) {
      parts.push('no:assignee');
    }

    return parts.join(' ');
  }

  /**
   * Fetch issues from GitHub repository
   * @param {string} owner - Repository owner
   * @param {string} repo - Repository name
   * @param {Object} options - Search options
   * @param {string} options.searchFilter - Top-level search filter
   * @param {string} options.state - Issue state (open/closed/all)
   * @param {string[]} options.labels - Filter by labels
   * @param {string} options.assignee - Filter by assignee
   * @param {string} options.author - Filter by author
   * @param {string} options.mentions - Filter by mentions
   * @param {string} options.milestone - Filter by milestone
   * @param {number} options.perPage - Results per page (max 100)
   * @param {number} options.page - Page number
   * @returns {Promise<Object>} Issues and pagination info
   */
  async getIssues(owner, repo, options = {}) {
    try {
      // Parse search filter if provided
      let qualifiers = {};
      if (options.searchFilter) {
        qualifiers = this.parseSearchFilter(options.searchFilter);
      }

      // Override with individual field values
      if (options.state) qualifiers.state = options.state;
      if (options.labels) qualifiers.labels = options.labels;
      if (options.assignee) qualifiers.assignee = options.assignee;
      if (options.author) qualifiers.author = options.author;
      if (options.mentions) qualifiers.mentions = options.mentions;
      if (options.milestone) qualifiers.milestone = options.milestone;

      // Build search query
      const searchQuery = this.buildSearchQuery(qualifiers);
      
      // Determine API endpoint
      let endpoint;
      let params = {
        per_page: options.perPage || 30,
        page: options.page || 1
      };

      if (searchQuery) {
        // Use search API for complex queries
        endpoint = `${this.baseUrl}/search/issues`;
        params.q = `repo:${owner}/${repo} ${searchQuery}`;
        params.sort = options.sort || 'created';
        params.order = options.order || 'desc';
      } else {
        // Use issues API for simple queries
        endpoint = `${this.baseUrl}/repos/${owner}/${repo}/issues`;
        if (options.state) params.state = options.state;
        params.sort = options.sort || 'created';
        params.direction = options.order || 'desc';
      }

      // Make API request
      const response = await axios.get(endpoint, {
        headers: this.headers,
        params: params
      });

      // Parse response
      let issues;
      let totalCount;
      
      if (searchQuery) {
        issues = response.data.items;
        totalCount = response.data.total_count;
      } else {
        issues = response.data;
        totalCount = parseInt(response.headers['x-total-count'] || issues.length);
      }

      // Extract pagination info
      const linkHeader = response.headers.link;
      const pagination = this.parseLinkHeader(linkHeader);

      return {
        issues: issues.map(issue => ({
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
          user: {
            login: issue.user.login,
            avatar_url: issue.user.avatar_url
          },
          milestone: issue.milestone ? {
            title: issue.milestone.title,
            state: issue.milestone.state
          } : null,
          created_at: issue.created_at,
          updated_at: issue.updated_at,
          closed_at: issue.closed_at,
          comments: issue.comments,
          pull_request: !!issue.pull_request,
          html_url: issue.html_url
        })),
        pagination: {
          total: totalCount,
          page: params.page,
          perPage: params.per_page,
          ...pagination
        }
      };
    } catch (error) {
      if (error.response) {
        throw new Error(`GitHub API Error: ${error.response.status} - ${error.response.data.message}`);
      }
      throw error;
    }
  }

  /**
   * Parse GitHub Link header for pagination
   * @param {string} linkHeader - Link header string
   * @returns {Object} Pagination URLs
   */
  parseLinkHeader(linkHeader) {
    if (!linkHeader) return {};

    const links = {};
    linkHeader.split(',').forEach(part => {
      const section = part.split(';');
      const url = section[0].replace(/<(.*)>/, '$1').trim();
      const name = section[1].replace(/rel="(.*)"/, '$1').trim();
      links[name] = url;
    });

    return {
      first: links.first || null,
      prev: links.prev || null,
      next: links.next || null,
      last: links.last || null
    };
  }
}

// Example usage
async function main() {
  const fetcher = new GitHubIssueFetcher(process.env.GITHUB_TOKEN);

  try {
    // Example 1: Simple query - open issues
    console.log('=== Open Issues ===');
    const openIssues = await fetcher.getIssues('facebook', 'react', {
      state: 'open',
      perPage: 5
    });
    console.log(`Found ${openIssues.pagination.total} open issues`);
    openIssues.issues.forEach(issue => {
      console.log(`#${issue.number}: ${issue.title} (${issue.state})`);
    });

    // Example 2: Using search filter
    console.log('\n=== Issues with search filter ===');
    const filteredIssues = await fetcher.getIssues('facebook', 'react', {
      searchFilter: 'is:open label:"Component: DOM"',
      perPage: 5
    });
    console.log(`Found ${filteredIssues.pagination.total} issues matching filter`);
    filteredIssues.issues.forEach(issue => {
      console.log(`#${issue.number}: ${issue.title}`);
      console.log(`  Labels: ${issue.labels.map(l => l.name).join(', ')}`);
    });

    // Example 3: Full control with individual fields
    console.log('\n=== Issues with individual fields ===');
    const customIssues = await fetcher.getIssues('facebook', 'react', {
      state: 'closed',
      labels: ['bug', 'good first issue'],
      assignee: 'gaearon',
      sort: 'updated',
      order: 'desc',
      perPage: 3
    });
    console.log(`Found ${customIssues.pagination.total} issues`);
    customIssues.issues.forEach(issue => {
      console.log(`#${issue.number}: ${issue.title}`);
      console.log(`  Assignee: ${issue.assignees.map(a => a.login).join(', ')}`);
      console.log(`  Updated: ${issue.updated_at}`);
    });

  } catch (error) {
    console.error('Error:', error.message);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = GitHubIssueFetcher;
