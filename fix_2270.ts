// GitHub Issue Search Module
class GitHubIssueSearch {
  constructor(token) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
  }

  /**
   * Search for issues in a repository
   * @param {string} owner - Repository owner
   * @param {string} repo - Repository name
   * @param {Object} filters - Search filters
   * @returns {Promise<Array>} - Array of issues
   */
  async searchIssues(owner, repo, filters = {}) {
    const query = this.buildQuery(owner, repo, filters);
    const url = `${this.baseUrl}/search/issues?q=${encodeURIComponent(query)}`;
    
    const response = await fetch(url, {
      headers: {
        'Authorization': `token ${this.token}`,
        'Accept': 'application/vnd.github.v3+json'
      }
    });

    if (!response.ok) {
      throw new Error(`GitHub API error: ${response.status}`);
    }

    const data = await response.json();
    return data.items;
  }

  /**
   * Build search query from filters
   */
  buildQuery(owner, repo, filters) {
    const parts = [`repo:${owner}/${repo}`];

    if (filters.state) parts.push(`state:${filters.state}`);
    if (filters.labels) parts.push(`label:${filters.labels}`);
    if (filters.assignee) parts.push(`assignee:${filters.assignee}`);
    if (filters.author) parts.push(`author:${filters.author}`);
    if (filters.involves) parts.push(`involves:${filters.involves}`);
    if (filters.milestone) parts.push(`milestone:${filters.milestone}`);
    if (filters.is) parts.push(`is:${filters.is}`);
    if (filters.created) parts.push(`created:${filters.created}`);
    if (filters.updated) parts.push(`updated:${filters.updated}`);
    if (filters.comments) parts.push(`comments:${filters.comments}`);
    if (filters.search) parts.push(filters.search);

    return parts.join(' ');
  }

  /**
   * Get all issues with pagination
   */
  async getAllIssues(owner, repo, filters = {}) {
    let allIssues = [];
    let page = 1;
    let hasMore = true;

    while (hasMore) {
      const query = this.buildQuery(owner, repo, filters);
      const url = `${this.baseUrl}/search/issues?q=${encodeURIComponent(query)}&page=${page}&per_page=100`;
      
      const response = await fetch(url, {
        headers: {
          'Authorization': `token ${this.token}`,
          'Accept': 'application/vnd.github.v3+json'
        }
      });

      if (!response.ok) {
        throw new Error(`GitHub API error: ${response.status}`);
      }

      const data = await response.json();
      allIssues = allIssues.concat(data.items);
      
      hasMore = data.items.length === 100;
      page++;
    }

    return allIssues;
  }
}

// UI Component for Issue Search
class IssueSearchUI {
  constructor(containerId, githubToken) {
    this.container = document.getElementById(containerId);
    this.search = new GitHubIssueSearch(githubToken);
    this.init();
  }

  init() {
    this.container.innerHTML = `
      <div class="issue-search-container">
        <div class="search-filter-top">
          <input type="text" id="searchFilter" placeholder="Search issues..." class="search-input">
        </div>
        <div class="filter-fields">
          <div class="filter-row">
            <label>Repository:</label>
            <input type="text" id="owner" placeholder="Owner" class="filter-input">
            <input type="text" id="repo" placeholder="Repository" class="filter-input">
          </div>
          <div class="filter-row">
            <label>State:</label>
            <select id="state" class="filter-select">
              <option value="">Any</option>
              <option value="open">Open</option>
              <option value="closed">Closed</option>
            </select>
            <label>Labels:</label>
            <input type="text" id="labels" placeholder="bug,feature" class="filter-input">
          </div>
          <div class="filter-row">
            <label>Assignee:</label>
            <input type="text" id="assignee" placeholder="username" class="filter-input">
            <label>Author:</label>
            <input type="text" id="author" placeholder="username" class="filter-input">
          </div>
          <div class="filter-row">
            <label>Involves:</label>
            <input type="text" id="involves" placeholder="username" class="filter-input">
            <label>Milestone:</label>
            <input type="text" id="milestone" placeholder="milestone name" class="filter-input">
          </div>
          <div class="filter-row">
            <label>Created:</label>
            <input type="text" id="created" placeholder=">2023-01-01" class="filter-input">
            <label>Updated:</label>
            <input type="text" id="updated" placeholder="<2023-12-31" class="filter-input">
          </div>
          <div class="filter-row">
            <label>Comments:</label>
            <input type="text" id="comments" placeholder=">5" class="filter-input">
            <label>Type:</label>
            <select id="is" class="filter-select">
              <option value="">Any</option>
              <option value="issue">Issue</option>
              <option value="pr">Pull Request</option>
            </select>
          </div>
        </div>
        <button id="searchButton" class="search-button">Search Issues</button>
        <div id="results" class="results-container"></div>
      </div>
    `;

    this.bindEvents();
  }

  bindEvents() {
    document.getElementById('searchButton').addEventListener('click', () => this.performSearch());
    document.getElementById('searchFilter').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') this.performSearch();
    });
  }

  async performSearch() {
    const filters = {
      search: document.getElementById('searchFilter').value,
      state: document.getElementById('state').value,
      labels: document.getElementById('labels').value,
      assignee: document.getElementById('assignee').value,
      author: document.getElementById('author').value,
      involves: document.getElementById('involves').value,
      milestone: document.getElementById('milestone').value,
      created: document.getElementById('created').value,
      updated: document.getElementById('updated').value,
      comments: document.getElementById('comments').value,
      is: document.getElementById('is').value
    };

    const owner = document.getElementById('owner').value;
    const repo = document.getElementById('repo').value;

    if (!owner || !repo) {
      alert('Please enter both owner and repository name');
      return;
    }

    try {
      const issues = await this.search.searchIssues(owner, repo, filters);
      this.displayResults(issues);
    } catch (error) {
      console.error('Search failed:', error);
      document.getElementById('results').innerHTML = `<div class="error">Error: ${error.message}</div>`;
    }
  }

  displayResults(issues) {
    const resultsDiv = document.getElementById('results');
    
    if (issues.length === 0) {
      resultsDiv.innerHTML = '<div class="no-results">No issues found</div>';
      return;
    }

    let html = `<div class="issue-count">Found ${issues.length} issues</div>`;
    html += '<div class="issue-list">';
    
    issues.forEach(issue => {
      html += `
        <div class="issue-item">
          <div class="issue-title">
            <a href="${issue.html_url}" target="_blank">${issue.title}</a>
            <span class="issue-number">#${issue.number}</span>
          </div>
          <div class="issue-meta">
            <span class="issue-state ${issue.state}">${issue.state}</span>
            <span class="issue-labels">${issue.labels.map(l => `<span class="label" style="background-color:#${l.color}">${l.name}</span>`).join('')}</span>
            <span class="issue-assignee">${issue.assignee ? `Assigned to: ${issue.assignee.login}` : ''}</span>
            <span class="issue-date">Created: ${new Date(issue.created_at).toLocaleDateString()}</span>
            <span class="issue-comments">Comments: ${issue.comments}</span>
          </div>
        </div>
      `;
    });

    html += '</div>';
    resultsDiv.innerHTML = html;
  }
}

// Export for use
export { GitHubIssueSearch, IssueSearchUI };
