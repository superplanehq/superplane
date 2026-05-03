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

  /**
   * Fetch issues from a GitHub repository
   * @param {string} repo - Repository in format "owner/repo"
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
   * @param {string} [filters.since] - ISO 8601 timestamp
   * @returns {Promise<Array>} - Array of issue objects
   */
  async getIssues(repo, filters = {}) {
    try {
      const queryParams = this._buildQueryParams(filters);
      const url = `${this.baseURL}/repos/${repo}/issues?${queryParams}`;
      
      const response = await axios.get(url, { headers: this.headers });
      return response.data;
    } catch (error) {
      this._handleError(error);
    }
  }

  /**
   * Search issues across all repositories
   * @param {string} query - Search query string
   * @param {Object} filters - Additional filters
   * @returns {Promise<Object>} - Search results with items and total_count
   */
  async searchIssues(query, filters = {}) {
    try {
      const queryParams = this._buildSearchParams(query, filters);
      const url = `${this.baseURL}/search/issues?${queryParams}`;
      
      const response = await axios.get(url, { headers: this.headers });
      return response.data;
    } catch (error) {
      this._handleError(error);
    }
  }

  /**
   * Get a single issue by number
   * @param {string} repo - Repository in format "owner/repo"
   * @param {number} issueNumber - Issue number
   * @returns {Promise<Object>} - Issue object
   */
  async getIssue(repo, issueNumber) {
    try {
      const url = `${this.baseURL}/repos/${repo}/issues/${issueNumber}`;
      const response = await axios.get(url, { headers: this.headers });
      return response.data;
    } catch (error) {
      this._handleError(error);
    }
  }

  /**
   * Get all issues (paginated)
   * @param {string} repo - Repository in format "owner/repo"
   * @param {Object} filters - Search filters
   * @param {number} [maxPages=10] - Maximum pages to fetch
   * @returns {Promise<Array>} - Array of all issues
   */
  async getAllIssues(repo, filters = {}, maxPages = 10) {
    let allIssues = [];
    let page = 1;
    let hasMore = true;

    while (hasMore && page <= maxPages) {
      const issues = await this.getIssues(repo, { ...filters, page, per_page: 100 });
      allIssues = allIssues.concat(issues);
      
      hasMore = issues.length === 100;
      page++;
    }

    return allIssues;
  }

  /**
   * Build query string for repository issues endpoint
   * @param {Object} filters - Filter parameters
   * @returns {string} - URL query string
   */
  _buildQueryParams(filters) {
    const params = new URLSearchParams();
    
    const validFilters = {
      milestone: filters.milestone,
      state: filters.state,
      assignee: filters.assignee,
      creator: filters.author,
      mentioned: filters.involves,
      labels: filters.label,
      sort: filters.sort,
      direction: filters.direction,
      per_page: filters.per_page || 30,
      page: filters.page || 1,
      since: filters.since
    };

    Object.entries(validFilters).forEach(([key, value]) => {
      if (value !== undefined && value !== null && value !== '') {
        params.append(key, value);
      }
    });

    return params.toString();
  }

  /**
   * Build query string for search endpoint
   * @param {string} query - Base search query
   * @param {Object} filters - Additional filters
   * @returns {string} - URL query string
   */
  _buildSearchParams(query, filters) {
    const params = new URLSearchParams();
    
    // Build search query with qualifiers
    let searchQuery = query;
    
    if (filters.repo) searchQuery += ` repo:${filters.repo}`;
    if (filters.state) searchQuery += ` state:${filters.state}`;
    if (filters.label) searchQuery += ` label:${filters.label}`;
    if (filters.assignee) searchQuery += ` assignee:${filters.assignee}`;
    if (filters.author) searchQuery += ` author:${filters.author}`;
    if (filters.involves) searchQuery += ` involves:${filters.involves}`;
    if (filters.milestone) searchQuery += ` milestone:${filters.milestone}`;
    if (filters.is) searchQuery += ` is:${filters.is}`;
    
    params.append('q', searchQuery);
    params.append('sort', filters.sort || 'created');
    params.append('order', filters.direction || 'desc');
    params.append('per_page', filters.per_page || 30);
    params.append('page', filters.page || 1);
    
    return params.toString();
  }

  /**
   * Handle API errors
   * @param {Error} error - Axios error object
   */
  _handleError(error) {
    if (error.response) {
      const { status, data } = error.response;
      throw new Error(`GitHub API Error ${status}: ${data.message || 'Unknown error'}`);
    } else if (error.request) {
      throw new Error('Network error: Unable to reach GitHub API');
    } else {
      throw new Error(`Error: ${error.message}`);
    }
  }
}

// Example usage and CLI interface
class GitHubIssueCLI {
  constructor() {
    this.fetcher = new GitHubIssueFetcher(process.env.GITHUB_TOKEN);
  }

  async run() {
    const args = process.argv.slice(2);
    const command = args[0] || 'help';

    switch (command) {
      case 'list':
        await this.listIssues(args.slice(1));
        break;
      case 'search':
        await this.searchIssues(args.slice(1));
        break;
      case 'get':
        await this.getSingleIssue(args.slice(1));
        break;
      case 'export':
        await this.exportIssues(args.slice(1));
        break;
      default:
        this.showHelp();
    }
  }

  async listIssues(args) {
    if (args.length < 1) {
      console.error('Usage: node script.js list <owner/repo> [options]');
      return;
    }

    const repo = args[0];
    const filters = this.parseFilters(args.slice(1));

    try {
      const issues = await this.fetcher.getIssues(repo, filters);
      this.displayIssues(issues);
    } catch (error) {
      console.error('Error:', error.message);
    }
  }

  async searchIssues(args) {
    if (args.length < 1) {
      console.error('Usage: node script.js search <query> [options]');
      return;
    }

    const query = args[0];
    const filters = this.parseFilters(args.slice(1));

    try {
      const results = await this.fetcher.searchIssues(query, filters);
      console.log(`Found ${results.total_count} issues:`);
      this.displayIssues(results.items);
    } catch (error) {
      console.error('Error:', error.message);
    }
  }

  async getSingleIssue(args) {
    if (args.length < 2) {
      console.error('Usage: node script.js get <owner/repo> <issue_number>');
      return;
    }

    try {
      const issue = await this.fetcher.getIssue(args[0], parseInt(args[1]));
      console.log(JSON.stringify(issue, null, 2));
    } catch (error) {
      console.error('Error:', error.message);
    }
  }

  async exportIssues(args) {
    if (args.length < 1) {
      console.error('Usage: node script.js export <owner/repo> [options]');
      return;
    }

    const repo = args[0];
    const filters = this.parseFilters(args.slice(1));

    try {
      const issues = await this.fetcher.getAllIssues(repo, filters);
      console.log(JSON.stringify(issues, null, 2));
    } catch (error) {
      console.error('Error:', error.message);
    }
  }

  parseFilters(args) {
    const filters = {};
    
    args.forEach(arg => {
      const [key, value] = arg.split('=');
      if (key && value) {
        filters[key.replace('--', '')] = value;
      }
    });

    return filters;
  }

  displayIssues(issues) {
    issues.forEach(issue => {
      console.log(`#${issue.number} [${issue.state}] ${issue.title}`);
      console.log(`  Labels: ${issue.labels.map(l => l.name).join(', ') || 'none'}`);
      console.log(`  Assignee: ${issue.assignee ? issue.assignee.login : 'unassigned'}`);
      console.log(`  Created: ${issue.created_at}`);
      console.log(`  URL: ${issue.html_url}`);
      console.log('---');
    });
  }

  showHelp() {
    console.log(`
GitHub Issue Fetcher - CLI Tool

Usage:
  node script.js list <owner/repo> [options]     List issues for a repository
  node script.js search <query> [options]         Search issues across repositories
  node script.js get <owner/repo> <number>        Get a single issue
  node script.js export <owner/repo> [options]    Export all issues as JSON

Options:
  --state=open|closed|all
  --label=bug,feature
  --assignee=username|none|*
  --author=username
  --involves=username
  --milestone=number|*
  --sort=created|updated|comments
  --direction=asc|desc
  --per_page=30
  --page=1

Environment Variables:
  GITHUB_TOKEN    GitHub personal access token (optional, increases rate limit)

Examples:
  node script.js list facebook/react --state=open --label=bug
  node script.js search "react component" --repo=facebook/react --state=open
  node script.js get facebook/react 12345
  node script.js export facebook/react --state=all > issues.json
    `);
  }
}

// Run the CLI if executed directly
if (require.main === module) {
  const cli = new GitHubIssueCLI();
  cli.run().catch(console.error);
}

module.exports = GitHubIssueFetcher;
