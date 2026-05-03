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
   * Parse search filter into structured query
   * @param {string} searchFilter - Free-form search filter
   * @returns {Object} Parsed search parameters
   */
  parseSearchFilter(searchFilter) {
    const params = {
      q: '',
      state: 'open',
      labels: [],
      assignee: null,
      author: null,
      mentions: null,
      milestone: null,
      sort: 'created',
      direction: 'desc',
      per_page: 30,
      page: 1
    };

    if (!searchFilter) return params;

    // Parse qualifiers from search filter
    const qualifiers = searchFilter.match(/(\w+):("[^"]*"|\S+)/g);
    if (qualifiers) {
      qualifiers.forEach(qualifier => {
        const [key, value] = qualifier.split(':');
        const cleanValue = value.replace(/"/g, '');
        
        switch(key) {
          case 'is':
            if (cleanValue === 'open' || cleanValue === 'closed') {
              params.state = cleanValue;
            }
            break;
          case 'label':
            params.labels.push(cleanValue);
            break;
          case 'assignee':
            params.assignee = cleanValue;
            break;
          case 'author':
            params.author = cleanValue;
            break;
          case 'mentions':
            params.mentions = cleanValue;
            break;
          case 'milestone':
            params.milestone = cleanValue;
            break;
          case 'sort':
            if (['created', 'updated', 'comments'].includes(cleanValue)) {
              params.sort = cleanValue;
            }
            break;
          case 'direction':
            if (['asc', 'desc'].includes(cleanValue)) {
              params.direction = cleanValue;
            }
            break;
        }
      });
    }

    // Extract free text search (remaining text after removing qualifiers)
    const freeText = searchFilter.replace(/\w+:"[^"]*"|\w+:\S+/g, '').trim();
    if (freeText) {
      params.q = freeText;
    }

    return params;
  }

  /**
   * Build GitHub API query string from parameters
   * @param {Object} params - Structured search parameters
   * @param {string} repo - Repository in format "owner/repo"
   * @returns {string} Formatted query string
   */
  buildQuery(params, repo) {
    let query = `repo:${repo}`;

    if (params.state) {
      query += ` is:${params.state}`;
    }

    if (params.labels.length > 0) {
      params.labels.forEach(label => {
        query += ` label:"${label}"`;
      });
    }

    if (params.assignee) {
      query += ` assignee:${params.assignee}`;
    }

    if (params.author) {
      query += ` author:${params.author}`;
    }

    if (params.mentions) {
      query += ` mentions:${params.mentions}`;
    }

    if (params.milestone) {
      query += ` milestone:"${params.milestone}"`;
    }

    if (params.q) {
      query += ` ${params.q}`;
    }

    return query;
  }

  /**
   * Fetch issues from GitHub repository
   * @param {string} repo - Repository in format "owner/repo"
   * @param {Object} options - Search options
   * @returns {Promise<Array>} List of issues
   */
  async fetchIssues(repo, options = {}) {
    try {
      // Parse search filter if provided
      const params = this.parseSearchFilter(options.searchFilter || '');
      
      // Override with explicit options if provided
      if (options.state) params.state = options.state;
      if (options.labels) params.labels = options.labels;
      if (options.assignee) params.assignee = options.assignee;
      if (options.author) params.author = options.author;
      if (options.mentions) params.mentions = options.mentions;
      if (options.milestone) params.milestone = options.milestone;
      if (options.sort) params.sort = options.sort;
      if (options.direction) params.direction = options.direction;
      if (options.per_page) params.per_page = options.per_page;
      if (options.page) params.page = options.page;

      // Build query
      const query = this.buildQuery(params, repo);
      
      // Make API request
      const response = await axios.get(`${this.baseUrl}/search/issues`, {
        headers: this.headers,
        params: {
          q: query,
          sort: params.sort,
          order: params.direction,
          per_page: params.per_page,
          page: params.page
        }
      });

      // Format and return issues
      return response.data.items.map(issue => ({
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
          avatar_url: assignee.avatar_url,
          html_url: assignee.html_url
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
        pull_request: !!issue.pull_request
      }));

    } catch (error) {
      if (error.response) {
        throw new Error(`GitHub API Error: ${error.response.status} - ${error.response.data.message}`);
      }
      throw new Error(`Network Error: ${error.message}`);
    }
  }

  /**
   * Get repository issues with pagination support
   * @param {string} repo - Repository in format "owner/repo"
   * @param {Object} options - Search options
   * @returns {Promise<Object>} Issues with pagination info
   */
  async getRepositoryIssues(repo, options = {}) {
    const issues = await this.fetchIssues(repo, options);
    
    return {
      issues,
      pagination: {
        page: options.page || 1,
        per_page: options.per_page || 30,
        total: issues.length
      },
      repo,
      timestamp: new Date().toISOString()
    };
  }
}

// Export for use in other modules
module.exports = GitHubIssueFetcher;

// Example usage
async function example() {
  const fetcher = new GitHubIssueFetcher(process.env.GITHUB_TOKEN);
  
  // Example 1: Simple search
  const result1 = await fetcher.getRepositoryIssues('facebook/react', {
    searchFilter: 'is:open label:"good first issue"'
  });
  console.log('Open good first issues:', result1.issues.length);

  // Example 2: Advanced search with individual fields
  const result2 = await fetcher.getRepositoryIssues('facebook/react', {
    state: 'closed',
    labels: ['bug', 'help wanted'],
    assignee: 'facebook',
    sort: 'updated',
    direction: 'desc',
    per_page: 50
  });
  console.log('Closed bugs assigned to facebook:', result2.issues.length);

  // Example 3: Search by author and mentions
  const result3 = await fetcher.getRepositoryIssues('facebook/react', {
    author: 'facebook',
    mentions: 'facebook',
    state: 'open'
  });
  console.log('Issues by facebook mentioning facebook:', result3.issues.length);
}

// Run example if executed directly
if (require.main === module) {
  example().catch(console.error);
}
