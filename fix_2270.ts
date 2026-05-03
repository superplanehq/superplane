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
   * Build search query from individual fields
   */
  buildSearchQuery({ 
    repo, 
    state = 'open', 
    labels = '', 
    assignee = '', 
    author = '', 
    mentioned = '',
    isIssue = true,
    isPR = false,
    sort = 'created',
    direction = 'desc'
  }) {
    let query = `repo:${repo}`;
    
    if (state) query += ` state:${state}`;
    if (labels) query += ` label:${labels}`;
    if (assignee) query += ` assignee:${assignee}`;
    if (author) query += ` author:${author}`;
    if (mentioned) query += ` mentions:${mentioned}`;
    
    if (isIssue && !isPR) query += ' type:issue';
    if (isPR && !isIssue) query += ' type:pr';
    
    return { query, sort, direction };
  }

  /**
   * Parse top-level search filter
   */
  parseSearchFilter(filter) {
    const params = {};
    
    // Extract repo
    const repoMatch = filter.match(/repo:([^\s]+)/);
    if (repoMatch) params.repo = repoMatch[1];
    
    // Extract state
    const stateMatch = filter.match(/state:(\w+)/);
    if (stateMatch) params.state = stateMatch[1];
    
    // Extract labels
    const labelMatch = filter.match(/label:([^\s]+)/);
    if (labelMatch) params.labels = labelMatch[1];
    
    // Extract assignee
    const assigneeMatch = filter.match(/assignee:([^\s]+)/);
    if (assigneeMatch) params.assignee = assigneeMatch[1];
    
    // Extract author
    const authorMatch = filter.match(/author:([^\s]+)/);
    if (authorMatch) params.author = authorMatch[1];
    
    // Extract mentions
    const mentionsMatch = filter.match(/mentions:([^\s]+)/);
    if (mentionsMatch) params.mentioned = mentionsMatch[1];
    
    // Extract type
    const typeMatch = filter.match(/type:(\w+)/);
    if (typeMatch) {
      params.isIssue = typeMatch[1] === 'issue';
      params.isPR = typeMatch[1] === 'pr';
    }
    
    // Extract sort
    const sortMatch = filter.match(/sort:(\w+)/);
    if (sortMatch) params.sort = sortMatch[1];
    
    // Extract direction
    const directionMatch = filter.match(/direction:(\w+)/);
    if (directionMatch) params.direction = directionMatch[1];
    
    return params;
  }

  /**
   * Fetch issues from GitHub
   */
  async fetchIssues(params, page = 1, perPage = 30) {
    try {
      const searchParams = this.buildSearchQuery(params);
      const url = `${this.baseUrl}/search/issues`;
      
      const response = await axios.get(url, {
        headers: this.headers,
        params: {
          q: searchParams.query,
          sort: searchParams.sort,
          direction: searchParams.direction,
          page: page,
          per_page: perPage
        }
      });
      
      return {
        issues: response.data.items,
        totalCount: response.data.total_count,
        incompleteResults: response.data.incomplete_results
      };
    } catch (error) {
      console.error('Error fetching issues:', error.response?.data || error.message);
      throw error;
    }
  }

  /**
   * Fetch all issues with pagination
   */
  async fetchAllIssues(params, maxPages = 10) {
    let allIssues = [];
    let page = 1;
    let hasMore = true;
    
    while (hasMore && page <= maxPages) {
      const result = await this.fetchIssues(params, page);
      allIssues = allIssues.concat(result.issues);
      
      hasMore = result.issues.length === 30;
      page++;
    }
    
    return allIssues;
  }

  /**
   * Export issues to JSON format
   */
  exportToJSON(issues) {
    return JSON.stringify(issues.map(issue => ({
      id: issue.id,
      number: issue.number,
      title: issue.title,
      state: issue.state,
      body: issue.body,
      labels: issue.labels.map(l => l.name),
      assignees: issue.assignees?.map(a => a.login) || [],
      author: issue.user.login,
      created_at: issue.created_at,
      updated_at: issue.updated_at,
      closed_at: issue.closed_at,
      url: issue.html_url
    })), null, 2);
  }

  /**
   * Export issues to CSV format
   */
  exportToCSV(issues) {
    const headers = ['ID', 'Number', 'Title', 'State', 'Labels', 'Assignees', 'Author', 'Created', 'Updated', 'Closed', 'URL'];
    const rows = issues.map(issue => [
      issue.id,
      issue.number,
      `"${issue.title.replace(/"/g, '""')}"`,
      issue.state,
      `"${issue.labels.map(l => l.name).join('; ')}"`,
      `"${(issue.assignees || []).map(a => a.login).join('; ')}"`,
      issue.user.login,
      issue.created_at,
      issue.updated_at,
      issue.closed_at || '',
      issue.html_url
    ]);
    
    return [headers.join(','), ...rows.map(row => row.join(','))].join('\n');
  }
}

// Example usage
async function main() {
  const fetcher = new GitHubIssueFetcher(process.env.GITHUB_TOKEN);
  
  // Example 1: Using individual fields
  const params = {
    repo: 'octocat/Hello-World',
    state: 'open',
    labels: 'bug,enhancement',
    assignee: 'octocat',
    sort: 'updated',
    direction: 'desc'
  };
  
  try {
    const issues = await fetcher.fetchIssues(params);
    console.log(`Found ${issues.totalCount} issues`);
    console.log('JSON Export:');
    console.log(fetcher.exportToJSON(issues.issues));
    
    // Example 2: Using top-level search filter
    const filter = 'repo:octocat/Hello-World state:open label:bug sort:created direction:asc';
    const parsedParams = fetcher.parseSearchFilter(filter);
    const filteredIssues = await fetcher.fetchAllIssues(parsedParams);
    console.log(`\nFiltered issues count: ${filteredIssues.length}`);
    console.log('CSV Export:');
    console.log(fetcher.exportToCSV(filteredIssues));
    
  } catch (error) {
    console.error('Failed to fetch issues:', error.message);
  }
}

// Run if called directly
if (require.main === module) {
  main();
}

module.exports = GitHubIssueFetcher;
