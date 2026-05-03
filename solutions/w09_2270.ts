// components/GitHubIssueSearch.js
import React, { useState, useEffect } from 'react';
import { Octokit } from '@octokit/rest';

const octokit = new Octokit({
  auth: process.env.GITHUB_TOKEN
});

export default function GitHubIssueSearch() {
  const [searchFilter, setSearchFilter] = useState('');
  const [issues, setIssues] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [filters, setFilters] = useState({
    state: 'open',
    labels: '',
    assignee: '',
    author: '',
    involved: '',
    milestone: '',
    sort: 'created',
    direction: 'desc',
    per_page: 30,
    page: 1
  });

  const [repo, setRepo] = useState({ owner: '', repo: '' });

  useEffect(() => {
    if (searchFilter) {
      const match = searchFilter.match(/repo:([^/]+)\/([^\s]+)/);
      if (match) {
        setRepo({ owner: match[1], repo: match[2] });
      }
    }
  }, [searchFilter]);

  const buildQuery = () => {
    let query = searchFilter || '';
    
    if (repo.owner && repo.repo) {
      query = `repo:${repo.owner}/${repo.repo}`;
    }
    
    if (filters.state) query += ` state:${filters.state}`;
    if (filters.labels) query += ` label:${filters.labels}`;
    if (filters.assignee) query += ` assignee:${filters.assignee}`;
    if (filters.author) query += ` author:${filters.author}`;
    if (filters.involved) query += ` involves:${filters.involved}`;
    if (filters.milestone) query += ` milestone:${filters.milestone}`;
    
    return query.trim();
  };

  const searchIssues = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const query = buildQuery();
      if (!query) {
        setError('Please provide a search filter or repository');
        setLoading(false);
        return;
      }

      const response = await octokit.rest.search.issuesAndPullRequests({
        q: query,
        sort: filters.sort,
        order: filters.direction,
        per_page: filters.per_page,
        page: filters.page
      });

      setIssues(response.data.items.filter(item => !item.pull_request));
    } catch (err) {
      setError(err.message || 'Failed to fetch issues');
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (field, value) => {
    setFilters(prev => ({ ...prev, [field]: value }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    setFilters(prev => ({ ...prev, page: 1 }));
    searchIssues();
  };

  const handleExport = (format) => {
    if (format === 'json') {
      const blob = new Blob([JSON.stringify(issues, null, 2)], { type: 'application/json' });
      downloadBlob(blob, 'issues.json');
    } else if (format === 'csv') {
      const headers = ['Number', 'Title', 'State', 'Labels', 'Assignee', 'Created', 'Updated'];
      const rows = issues.map(issue => [
        issue.number,
        `"${issue.title.replace(/"/g, '""')}"`,
        issue.state,
        `"${issue.labels.map(l => l.name).join(', ')}"`,
        issue.assignee?.login || '',
        issue.created_at,
        issue.updated_at
      ]);
      const csv = [headers.join(','), ...rows.map(r => r.join(','))].join('\n');
      const blob = new Blob([csv], { type: 'text/csv' });
      downloadBlob(blob, 'issues.csv');
    }
  };

  const downloadBlob = (blob, filename) => {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="github-issue-search">
      <h2>GitHub Issue Search</h2>
      
      <form onSubmit={handleSubmit} className="search-form">
        <div className="search-filter">
          <input
            type="text"
            placeholder="Search filter (e.g., repo:owner/repo is:issue)"
            value={searchFilter}
            onChange={(e) => setSearchFilter(e.target.value)}
            className="search-input"
          />
        </div>

        <div className="filters-grid">
          <div className="filter-group">
            <label>State</label>
            <select
              value={filters.state}
              onChange={(e) => handleFilterChange('state', e.target.value)}
            >
              <option value="open">Open</option>
              <option value="closed">Closed</option>
              <option value="all">All</option>
            </select>
          </div>

          <div className="filter-group">
            <label>Labels</label>
            <input
              type="text"
              placeholder="bug,feature,enhancement"
              value={filters.labels}
              onChange={(e) => handleFilterChange('labels', e.target.value)}
            />
          </div>

          <div className="filter-group">
            <label>Assignee</label>
            <input
              type="text"
              placeholder="username"
              value={filters.assignee}
              onChange={(e) => handleFilterChange('assignee', e.target.value)}
            />
          </div>

          <div className="filter-group">
            <label>Author</label>
            <input
              type="text"
              placeholder="username"
              value={filters.author}
              onChange={(e) => handleFilterChange('author', e.target.value)}
            />
          </div>

          <div className="filter-group">
            <label>Involved</label>
            <input
              type="text"
              placeholder="username"
              value={filters.involved}
              onChange={(e) => handleFilterChange('involved', e.target.value)}
            />
          </div>

          <div className="filter-group">
            <label>Milestone</label>
            <input
              type="text"
              placeholder="milestone name"
              value={filters.milestone}
              onChange={(e) => handleFilterChange('milestone', e.target.value)}
            />
          </div>

          <div className="filter-group">
            <label>Sort</label>
            <select
              value={filters.sort}
              onChange={(e) => handleFilterChange('sort', e.target.value)}
            >
              <option value="created">Created</option>
              <option value="updated">Updated</option>
              <option value="comments">Comments</option>
              <option value="reactions">Reactions</option>
            </select>
          </div>

          <div className="filter-group">
            <label>Direction</label>
            <select
              value={filters.direction}
              onChange={(e) => handleFilterChange('direction', e.target.value)}
            >
              <option value="desc">Descending</option>
              <option value="asc">Ascending</option>
            </select>
          </div>

          <div className="filter-group">
            <label>Per Page</label>
            <select
              value={filters.per_page}
              onChange={(e) => handleFilterChange('per_page', parseInt(e.target.value))}
            >
              <option value="30">30</option>
              <option value="50">50</option>
              <option value="100">100</option>
            </select>
          </div>
        </div>

        <div className="actions">
          <button type="submit" disabled={loading} className="btn-primary">
            {loading ? 'Searching...' : 'Search Issues'}
          </button>
          
          {issues.length > 0 && (
            <div className="export-buttons">
              <button type="button" onClick={() => handleExport('json')} className="btn-secondary">
                Export JSON
              </button>
              <button type="button" onClick={() => handleExport('csv')} className="btn-secondary">
                Export CSV
              </button>
            </div>
          )}
        </div>
      </form>

      {error && <div className="error-message">{error}</div>}

      {loading && <div className="loading">Loading issues...</div>}

      {!loading && issues.length > 0 && (
        <div className="issues-list">
          <div className="issues-header">
            <span>Found {issues.length} issues</span>
            <div className="pagination">
              <button
                onClick={() => {
                  setFilters(prev => ({ ...prev, page: Math.max(1, prev.page - 1) }));
                  searchIssues();
                }}
                disabled={filters.page === 1}
              >
                Previous
              </button>
              <span>Page {filters.page}</span>
              <button
                onClick={() => {
                  setFilters(prev => ({ ...prev, page: prev.page + 1 }));
                  searchIssues();
                }}
                disabled={issues.length < filters.per_page}
              >
                Next
              </button>
            </div>
          </div>

          {issues.map(issue => (
            <div key={issue.id} className="issue-item">
              <div className="issue-title">
                <a href={issue.html_url} target="_blank" rel="noopener noreferrer">
                  #{issue.number} - {issue.title}
                </a>
              </div>
              <div className="issue-meta">
                <span className={`state-badge ${issue.state}`}>{issue.state}</span>
                {issue.labels.map(label => (
                  <span
                    key={label.id}
                    className="label-badge"
                    style={{ backgroundColor: `#${label.color}` }}
                  >
                    {label.name}
                  </span>
                ))}
                <span className="author">by {issue.user.login}</span>
                {issue.assignee && <span className="assignee">assigned to {issue.assignee.login}</span>}
                <span className="date">created {new Date(issue.created_at).toLocaleDateString()}</span>
                <span className="comments">{issue.comments} comments</span>
              </div>
            </div>
          ))}
        </div>
      )}

      {!loading && issues.length === 0 && !error && (
        <div className="no-results">No issues found. Try adjusting your search filters.</div>
      )}

      <style jsx>{`
        .github-issue-search {
          max-width: 1200px;
          margin: 0 auto;
          padding: 20px;
          font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
        }

        h2 {
          color: #24292e;
          margin-bottom: 20px;
        }

        .search-form {
          background: #f6f8fa;
          border: 1px solid #e1e4e8;
          border-radius: 6px;
          padding: 20px;
          margin-bottom: 20px;
        }

        .search-filter {
          margin-bottom: 15px;
        }

        .search-input {
          width: 100%;
          padding: 10px;
          border: 1px solid #e1e4e8;
          border-radius: 6px;
          font-size: 14px;
        }

        .filters-grid {
          display: grid;
          grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
          gap: 15px;
          margin-bottom: 20px;
        }

        .filter-group {
          display: flex;
          flex-direction: column;
        }

        .filter-group label {
          font-size: 12px;
          font-weight: 600;
          color: #586069;
          margin-bottom: 5px;
        }

        .filter-group input,
        .filter-group select {
          padding: 8px;
          border: 1px solid #e1e4e8;
          border-radius: 4px;
          font-size: 13px;
        }

        .actions {
          display: flex;
          gap: 10px;
          align-items: center;
        }

        .btn-primary {
          background: #2ea44f;
          color: white;
          border: none;
          padding: 10px 20px;
          border-radius: 6px;
          cursor: pointer;
          font-size: 14px;
          font-weight: 600;
        }

        .btn-primary:disabled {
          opacity: 0.6;
          cursor: not-allowed;
        }

        .btn-secondary {
          background: #f6f8fa;
          color: #24292e;
          border: 1px solid #e1e4e8;
          padding: 10px 20px;
          border-radius: 6px;
          cursor: pointer;
          font-size: 14px;
        }

        .export-buttons {
          display: flex;
          gap: 10px;
        }

        .error-message {
          background: #ffeef0;
          border: 1px solid #f97583;
          color: #cb2431;
          padding: 10px;
          border-radius: 6px;
          margin-bottom: 20px;
        }

        .loading {
          text-align: center;
          padding: 40px;
          color: #586069;
        }

        .issues-list {
          background: white;
          border: 1px solid #e1e4e8;
          border-radius: 6px;
        }

        .issues-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 15px;
          background: #f6f8fa;
          border-bottom: 1px solid #e1e4e8;
          border-radius: 6px 6px 0 0;
        }

        .pagination {
          display: flex;
          gap: 10px;
          align-items: center;
        }

        .pagination button {
          padding: 5px 10px;
          border: 1px solid #e1e4e8;
          border-radius: 4px;
          background: white;
          cursor: pointer;
        }

        .pagination button:disabled {
          opacity: 0.5;
          cursor: not-allowed;
        }

        .issue-item {
          padding: 15px;
          border-bottom: 1px solid #e1e4e8;
        }

        .issue-item:last-child {
          border-bottom: none;
        }

        .issue-title a {
          color: #0366d6;
          text-decoration: none;
          font-weight: 600;
          font-size: 16px;
        }

        .issue-title a:hover {
          text-decoration: underline;
        }

        .issue-meta {
          display: flex;
          gap: 8px;
          align-items: center;
          margin-top: 8px;
          font-size: 12px;
          color: #586069;
          flex-wrap: wrap;
        }

        .state-badge {
          padding: 2px 8px;
          border-radius: 12px;
          font-weight: 600;
          font-size: 11px;
        }

        .state-badge.open {
          background: #dcffe4;
          color: #22863a;
        }

        .state-badge.closed {
          background: #ffeef0;
          color: #cb2431;
        }

        .label-badge {
          padding: 2px 6px;
          border-radius: 12px;
          font-size: 11px;
          color: white;
        }

        .no-results {
          text-align: center;
          padding: 40px;
          color: #586069;
        }
      `}</style>
    </div>
  );
}
