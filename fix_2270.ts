// GitHub Issue Search Module
class GitHubIssueSearch {
  constructor(token) {
    this.token = token;
    this.baseUrl = 'https://api.github.com';
  }

  async searchIssues({
    repo,
    state = 'open',
    labels,
    assignee,
    author,
    mentioned,
    sort = 'created',
    direction = 'desc',
    perPage = 30,
    page = 1
  } = {}) {
    if (!repo) throw new Error('Repository is required (format: owner/repo)');

    let query = `repo:${repo}`;
    
    if (state) query += ` state:${state}`;
    if (labels) query += ` label:${labels}`;
    if (assignee) query += ` assignee:${assignee}`;
    if (author) query += ` author:${author}`;
    if (mentioned) query += ` involves:${mentioned}`;

    const url = `${this.baseUrl}/search/issues?q=${encodeURIComponent(query)}&sort=${sort}&order=${direction}&per_page=${perPage}&page=${page}`;

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
      total: data.total_count,
      issues: data.items.map(item => ({
        id: item.id,
        number: item.number,
        title: item.title,
        state: item.state,
        body: item.body,
        labels: item.labels.map(l => l.name),
        assignees: item.assignees.map(a => a.login),
        author: item.user.login,
        createdAt: item.created_at,
        updatedAt: item.updated_at,
        url: item.html_url
      }))
    };
  }

  async getRepositoryIssues(repo, options = {}) {
    return this.searchIssues({ repo, ...options });
  }
}

// UI Component for Issue Search
class IssueSearchUI {
  constructor(containerId, token) {
    this.container = document.getElementById(containerId);
    this.api = new GitHubIssueSearch(token);
    this.currentPage = 1;
    this.init();
  }

  init() {
    this.container.innerHTML = `
      <div class="issue-search-container">
        <div class="search-header">
          <input type="text" id="searchFilter" placeholder="Search filter (e.g., is:open is:issue)" />
          <button id="searchButton">Search Issues</button>
        </div>
        <div class="search-filters">
          <div class="filter-row">
            <label>Repository:</label>
            <input type="text" id="repoInput" placeholder="owner/repo" />
          </div>
          <div class="filter-row">
            <label>State:</label>
            <select id="stateSelect">
              <option value="open">Open</option>
              <option value="closed">Closed</option>
              <option value="all">All</option>
            </select>
          </div>
          <div class="filter-row">
            <label>Labels:</label>
            <input type="text" id="labelsInput" placeholder="bug,feature,enhancement" />
          </div>
          <div class="filter-row">
            <label>Assignee:</label>
            <input type="text" id="assigneeInput" placeholder="username" />
          </div>
          <div class="filter-row">
            <label>Author:</label>
            <input type="text" id="authorInput" placeholder="username" />
          </div>
          <div class="filter-row">
            <label>Mentioned:</label>
            <input type="text" id="mentionedInput" placeholder="username" />
          </div>
          <div class="filter-row">
            <label>Sort by:</label>
            <select id="sortSelect">
              <option value="created">Created</option>
              <option value="updated">Updated</option>
              <option value="comments">Comments</option>
            </select>
          </div>
          <div class="filter-row">
            <label>Direction:</label>
            <select id="directionSelect">
              <option value="desc">Descending</option>
              <option value="asc">Ascending</option>
            </select>
          </div>
        </div>
        <div id="resultsContainer" class="results-container"></div>
        <div id="paginationContainer" class="pagination-container"></div>
      </div>
    `;

    document.getElementById('searchButton').addEventListener('click', () => this.search());
    document.getElementById('searchFilter').addEventListener('keypress', (e) => {
      if (e.key === 'Enter') this.search();
    });
  }

  async search(page = 1) {
    const searchFilter = document.getElementById('searchFilter').value;
    const repo = document.getElementById('repoInput').value || searchFilter.match(/repo:(\S+)/)?.[1];
    const state = document.getElementById('stateSelect').value;
    const labels = document.getElementById('labelsInput').value || searchFilter.match(/label:(\S+)/)?.[1];
    const assignee = document.getElementById('assigneeInput').value || searchFilter.match(/assignee:(\S+)/)?.[1];
    const author = document.getElementById('authorInput').value || searchFilter.match(/author:(\S+)/)?.[1];
    const mentioned = document.getElementById('mentionedInput').value || searchFilter.match(/involves:(\S+)/)?.[1];
    const sort = document.getElementById('sortSelect').value;
    const direction = document.getElementById('directionSelect').value;

    if (!repo) {
      alert('Please enter a repository (owner/repo)');
      return;
    }

    try {
      const result = await this.api.searchIssues({
        repo,
        state,
        labels,
        assignee,
        author,
        mentioned,
        sort,
        direction,
        page
      });

      this.displayResults(result);
      this.displayPagination(result.total, page);
    } catch (error) {
      document.getElementById('resultsContainer').innerHTML = `
        <div class="error">Error: ${error.message}</div>
      `;
    }
  }

  displayResults(result) {
    const container = document.getElementById('resultsContainer');
    
    if (result.issues.length === 0) {
      container.innerHTML = '<div class="no-results">No issues found</div>';
      return;
    }

    container.innerHTML = `
      <div class="results-summary">Found ${result.total} issues</div>
      <div class="issues-list">
        ${result.issues.map(issue => `
          <div class="issue-item">
            <div class="issue-header">
              <span class="issue-number">#${issue.number}</span>
              <span class="issue-state ${issue.state}">${issue.state}</span>
              <a href="${issue.url}" target="_blank" class="issue-title">${issue.title}</a>
            </div>
            <div class="issue-meta">
              <span>Author: ${issue.author}</span>
              <span>Created: ${new Date(issue.createdAt).toLocaleDateString()}</span>
              ${issue.labels.length > 0 ? `<span>Labels: ${issue.labels.join(', ')}</span>` : ''}
              ${issue.assignees.length > 0 ? `<span>Assignees: ${issue.assignees.join(', ')}</span>` : ''}
            </div>
          </div>
        `).join('')}
      </div>
    `;
  }

  displayPagination(total, currentPage) {
    const container = document.getElementById('paginationContainer');
    const totalPages = Math.ceil(total / 30);
    
    if (totalPages <= 1) {
      container.innerHTML = '';
      return;
    }

    let paginationHTML = '<div class="pagination">';
    
    if (currentPage > 1) {
      paginationHTML += `<button onclick="issueSearch.search(${currentPage - 1})">Previous</button>`;
    }
    
    paginationHTML += `<span>Page ${currentPage} of ${totalPages}</span>`;
    
    if (currentPage < totalPages) {
      paginationHTML += `<button onclick="issueSearch.search(${currentPage + 1})">Next</button>`;
    }
    
    paginationHTML += '</div>';
    container.innerHTML = paginationHTML;
  }
}

// Export for use
if (typeof module !== 'undefined' && module.exports) {
  module.exports = { GitHubIssueSearch, IssueSearchUI };
}
