// GitHub Issue Search Module
class GitHubIssueSearch {
  constructor(token) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
  }

  async searchIssues({
    repo,
    state = 'open',
    labels = '',
    assignee = '',
    author = '',
    mentioned = '',
    searchFilter = '',
    page = 1,
    perPage = 30
  } = {}) {
    const queryParts = [];
    
    // Build search query
    if (repo) queryParts.push(`repo:${repo}`);
    if (state) queryParts.push(`state:${state}`);
    if (labels) queryParts.push(`label:${labels}`);
    if (assignee) queryParts.push(`assignee:${assignee}`);
    if (author) queryParts.push(`author:${author}`);
    if (mentioned) queryParts.push(`mentions:${mentioned}`);
    if (searchFilter) queryParts.push(searchFilter);

    const query = queryParts.join(' ');
    const url = `${this.baseUrl}/search/issues?q=${encodeURIComponent(query)}&page=${page}&per_page=${perPage}`;

    try {
      const response = await fetch(url, {
        headers: {
          'Authorization': `token ${this.token}`,
          'Accept': 'application/vnd.github.v3+json'
        }
      });

      if (!response.ok) {
        throw new Error(`GitHub API error: ${response.status} ${response.statusText}`);
      }

      const data = await response.json();
      return {
        totalCount: data.total_count,
        items: data.items.map(issue => ({
          id: issue.id,
          number: issue.number,
          title: issue.title,
          state: issue.state,
          body: issue.body,
          labels: issue.labels.map(l => ({ name: l.name, color: l.color })),
          assignees: issue.assignees.map(a => a.login),
          author: issue.user.login,
          createdAt: issue.created_at,
          updatedAt: issue.updated_at,
          closedAt: issue.closed_at,
          url: issue.html_url,
          comments: issue.comments
        })),
        page,
        perPage
      };
    } catch (error) {
      console.error('Search failed:', error);
      throw error;
    }
  }

  // Helper method to get all issues (paginated)
  async getAllIssues(config, maxPages = 10) {
    let allIssues = [];
    let currentPage = 1;
    let hasMore = true;

    while (hasMore && currentPage <= maxPages) {
      const result = await this.searchIssues({ ...config, page: currentPage });
      allIssues = allIssues.concat(result.items);
      hasMore = result.items.length === config.perPage;
      currentPage++;
    }

    return allIssues;
  }
}

// UI Component for Issue Search Form
class IssueSearchForm {
  constructor(containerId, githubToken) {
    this.container = document.getElementById(containerId);
    this.searchEngine = new GitHubIssueSearch(githubToken);
    this.render();
  }

  render() {
    this.container.innerHTML = `
      <div class="issue-search-container">
        <div class="search-filter-top">
          <input type="text" id="searchFilter" placeholder="Search filter (e.g., bug, feature request)" class="search-input">
          <button onclick="searchForm.handleSearch()" class="search-button">Search Issues</button>
        </div>
        
        <div class="search-fields">
          <div class="field-group">
            <label>Repository (owner/repo)</label>
            <input type="text" id="repo" placeholder="e.g., octocat/Hello-World">
          </div>
          
          <div class="field-group">
            <label>State</label>
            <select id="state">
              <option value="open">Open</option>
              <option value="closed">Closed</option>
              <option value="all">All</option>
            </select>
          </div>
          
          <div class="field-group">
            <label>Labels (comma-separated)</label>
            <input type="text" id="labels" placeholder="e.g., bug, enhancement">
          </div>
          
          <div class="field-group">
            <label>Assignee</label>
            <input type="text" id="assignee" placeholder="GitHub username">
          </div>
          
          <div class="field-group">
            <label>Author</label>
            <input type="text" id="author" placeholder="GitHub username">
          </div>
          
          <div class="field-group">
            <label>Mentioned</label>
            <input type="text" id="mentioned" placeholder="GitHub username">
          </div>
        </div>
        
        <div id="results" class="results-container"></div>
      </div>
    `;
  }

  async handleSearch() {
    const config = {
      repo: document.getElementById('repo').value,
      state: document.getElementById('state').value,
      labels: document.getElementById('labels').value,
      assignee: document.getElementById('assignee').value,
      author: document.getElementById('author').value,
      mentioned: document.getElementById('mentioned').value,
      searchFilter: document.getElementById('searchFilter').value
    };

    const resultsContainer = document.getElementById('results');
    resultsContainer.innerHTML = '<div class="loading">Searching...</div>';

    try {
      const results = await this.searchEngine.searchIssues(config);
      this.displayResults(results, resultsContainer);
    } catch (error) {
      resultsContainer.innerHTML = `<div class="error">Error: ${error.message}</div>`;
    }
  }

  displayResults(results, container) {
    if (results.items.length === 0) {
      container.innerHTML = '<div class="no-results">No issues found</div>';
      return;
    }

    let html = `<div class="results-header">Found ${results.totalCount} issues (showing ${results.items.length})</div>`;
    html += '<div class="issues-list">';

    results.items.forEach(issue => {
      html += `
        <div class="issue-item">
          <div class="issue-title">
            <a href="${issue.url}" target="_blank">#${issue.number} - ${issue.title}</a>
          </div>
          <div class="issue-meta">
            <span class="issue-state ${issue.state}">${issue.state}</span>
            <span class="issue-author">by ${issue.author}</span>
            <span class="issue-date">${new Date(issue.createdAt).toLocaleDateString()}</span>
            <span class="issue-comments">💬 ${issue.comments}</span>
          </div>
          <div class="issue-labels">
            ${issue.labels.map(label => `<span class="label" style="background-color: #${label.color}">${label.name}</span>`).join('')}
          </div>
          ${issue.assignees.length > 0 ? `<div class="issue-assignees">Assigned to: ${issue.assignees.join(', ')}</div>` : ''}
        </div>
      `;
    });

    html += '</div>';
    container.innerHTML = html;
  }
}

// Export for use
const searchForm = new IssueSearchForm('app', 'YOUR_GITHUB_TOKEN');
