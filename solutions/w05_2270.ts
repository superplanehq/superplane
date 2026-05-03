// GitHub Issue Fetcher - Complete Solution
const axios = require('axios');

class GitHubIssueFetcher {
  constructor(token = null) {
    this.token = token;
    this.baseURL = 'https://api.github.com';
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
    labels = [], 
    assignee = null, 
    author = null, 
    mentioned = null,
    milestone = null,
    isIssue = true,
    isPR = false,
    sort = 'created',
    direction = 'desc'
  }) {
    let query = `repo:${repo}`;
    
    // State filter
    if (state === 'open' || state === 'closed') {
      query += ` state:${state}`;
    }
    
    // Type filter
    if (isIssue && !isPR) {
      query += ' type:issue';
    } else if (!isIssue && isPR) {
      query += ' type:pr';
    }
    
    // Labels
    if (labels.length > 0) {
      labels.forEach(label => {
        query += ` label:"${label}"`;
      });
    }
    
    // Assignee
    if (assignee) {
      query += ` assignee:${assignee}`;
    }
    
    // Author
    if (author) {
      query += ` author:${author}`;
    }
    
    // Mentioned
    if (mentioned) {
      query += ` mentions:${mentioned}`;
    }
    
    // Milestone
    if (milestone) {
      query += ` milestone:"${milestone}"`;
    }
    
    return { query, sort, direction };
  }

  /**
   * Search issues using GitHub's search API
   */
  async searchIssues(searchParams, page = 1, perPage = 30) {
    try {
      const { query, sort, direction } = searchParams;
      
      const response = await axios.get(`${this.baseURL}/search/issues`, {
        headers: this.headers,
        params: {
          q: query,
          sort: sort,
          order: direction,
          page: page,
          per_page: perPage
        }
      });
      
      return {
        total_count: response.data.total_count,
        incomplete_results: response.data.incomplete_results,
        issues: response.data.items.map(this.formatIssue)
      };
    } catch (error) {
      throw new Error(`Search failed: ${error.message}`);
    }
  }

  /**
   * Get issues for a repository directly
   */
  async getRepoIssues(repo, params = {}) {
    try {
      const response = await axios.get(`${this.baseURL}/repos/${repo}/issues`, {
        headers: this.headers,
        params: {
          state: params.state || 'open',
          labels: params.labels ? params.labels.join(',') : undefined,
          assignee: params.assignee,
          creator: params.author,
          mentioned: params.mentioned,
          milestone: params.milestone,
          sort: params.sort || 'created',
          direction: params.direction || 'desc',
          per_page: params.perPage || 30,
          page: params.page || 1
        }
      });
      
      return response.data.map(this.formatIssue);
    } catch (error) {
      throw new Error(`Failed to get repo issues: ${error.message}`);
    }
  }

  /**
   * Format issue data
   */
  formatIssue(issue) {
    return {
      id: issue.id,
      number: issue.number,
      title: issue.title,
      state: issue.state,
      body: issue.body,
      url: issue.html_url,
      api_url: issue.url,
      labels: issue.labels.map(label => ({
        name: label.name,
        color: label.color,
        description: label.description
      })),
      assignees: issue.assignees ? issue.assignees.map(user => ({
        login: user.login,
        avatar_url: user.avatar_url,
        url: user.html_url
      })) : [],
      creator: {
        login: issue.user.login,
        avatar_url: issue.user.avatar_url,
        url: issue.user.html_url
      },
      milestone: issue.milestone ? {
        title: issue.milestone.title,
        state: issue.milestone.state,
        due_on: issue.milestone.due_on
      } : null,
      created_at: issue.created_at,
      updated_at: issue.updated_at,
      closed_at: issue.closed_at,
      comments: issue.comments,
      pull_request: issue.pull_request ? {
        url: issue.pull_request.html_url,
        merged: issue.pull_request.merged_at ? true : false
      } : null,
      is_pull_request: !!issue.pull_request
    };
  }

  /**
   * Get all issues with pagination
   */
  async getAllIssues(repo, params = {}, maxPages = 10) {
    let allIssues = [];
    let page = 1;
    let hasMore = true;
    
    while (hasMore && page <= maxPages) {
      const issues = await this.getRepoIssues(repo, { ...params, page });
      
      if (issues.length === 0) {
        hasMore = false;
      } else {
        allIssues = allIssues.concat(issues);
        page++;
        
        // Rate limit check
        if (issues.length < (params.perPage || 30)) {
          hasMore = false;
        }
      }
    }
    
    return allIssues;
  }

  /**
   * Export issues to JSON format
   */
  exportToJSON(issues, filename = 'issues.json') {
    const fs = require('fs');
    fs.writeFileSync(filename, JSON.stringify(issues, null, 2));
    return filename;
  }

  /**
   * Export issues to CSV format
   */
  exportToCSV(issues, filename = 'issues.csv') {
    const fs = require('fs');
    const headers = ['Number', 'Title', 'State', 'Labels', 'Assignees', 'Creator', 'Created', 'Updated', 'URL'];
    
    let csv = headers.join(',') + '\n';
    
    issues.forEach(issue => {
      const row = [
        issue.number,
        `"${issue.title.replace(/"/g, '""')}"`,
        issue.state,
        `"${issue.labels.map(l => l.name).join('; ')}"`,
        `"${issue.assignees.map(a => a.login).join('; ')}"`,
        issue.creator.login,
        issue.created_at,
        issue.updated_at,
        issue.url
      ];
      csv += row.join(',') + '\n';
    });
    
    fs.writeFileSync(filename, csv);
    return filename;
  }
}

// Usage example
async function main() {
  const fetcher = new GitHubIssueFetcher(process.env.GITHUB_TOKEN);
  
  // Example 1: Search with query
  const searchParams = fetcher.buildSearchQuery({
    repo: 'facebook/react',
    state: 'open',
    labels: ['bug', 'good first issue'],
    sort: 'updated',
    direction: 'desc'
  });
  
  const searchResults = await fetcher.searchIssues(searchParams);
  console.log(`Found ${searchResults.total_count} issues`);
  
  // Example 2: Get repo issues directly
  const issues = await fetcher.getRepoIssues('facebook/react', {
    state: 'open',
    labels: ['bug'],
    perPage: 10
  });
  
  // Example 3: Get all issues with pagination
  const allIssues = await fetcher.getAllIssues('facebook/react', {
    state: 'open',
    labels: ['bug']
  }, 5); // Max 5 pages
  
  // Export examples
  fetcher.exportToJSON(allIssues, 'react-bugs.json');
  fetcher.exportToCSV(allIssues, 'react-bugs.csv');
  
  console.log(`Exported ${allIssues.length} issues`);
}

// Run if called directly
if (require.main === module) {
  main().catch(console.error);
}

module.exports = GitHubIssueFetcher;
