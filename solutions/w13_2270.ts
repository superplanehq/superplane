// components/IssueSearch.js
import React, { useState, useEffect } from 'react';
import { Octokit } from '@octokit/rest';

const octokit = new Octokit({
  auth: process.env.GITHUB_TOKEN
});

const IssueSearch = () => {
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
    sort: 'created',
    direction: 'desc',
    per_page: 30,
    page: 1
  });

  const [repo, setRepo] = useState({ owner: '', repo: '' });

  useEffect(() => {
    if (searchFilter) {
      parseSearchFilter(searchFilter);
    }
  }, [searchFilter]);

  const parseSearchFilter = (filter) => {
    const parts = filter.split(' ');
    const parsedFilters = { ...filters };
    
    parts.forEach(part => {
      if (part.startsWith('is:')) {
        parsedFilters.state = part.replace('is:', '');
      } else if (part.startsWith('label:')) {
        parsedFilters.labels = part.replace('label:', '');
      } else if (part.startsWith('assignee:')) {
        parsedFilters.assignee = part.replace('assignee:', '');
      } else if (part.startsWith('author:')) {
        parsedFilters.author = part.replace('author:', '');
      } else if (part.startsWith('involves:')) {
        parsedFilters.involved = part.replace('involves:', '');
      } else if (part.includes('/')) {
        const [owner, repo] = part.split('/');
        setRepo({ owner, repo: repo.replace('repo:', '') });
      }
    });
    
    setFilters(parsedFilters);
  };

  const searchIssues = async () => {
    setLoading(true);
    setError(null);
    
    try {
      const queryParts = [];
      
      if (filters.state) queryParts.push(`is:${filters.state}`);
      if (filters.labels) queryParts.push(`label:${filters.labels}`);
      if (filters.assignee) queryParts.push(`assignee:${filters.assignee}`);
      if (filters.author) queryParts.push(`author:${filters.author}`);
      if (filters.involved) queryParts.push(`involves:${filters.involved}`);
      if (repo.owner && repo.repo) queryParts.push(`repo:${repo.owner}/${repo.repo}`);

      const query = queryParts.join(' ');
      
      if (!query) {
        setError('Please provide at least one search filter');
        setLoading(false);
        return;
      }

      const response = await octokit.search.issuesAndPullRequests({
        q: query,
        sort: filters.sort,
        order: filters.direction,
        per_page: filters.per_page,
        page: filters.page
      });

      setIssues(response.data.items);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  };

  const handleFilterChange = (field, value) => {
    setFilters(prev => ({ ...prev, [field]: value }));
  };

  const handleSubmit = (e) => {
    e.preventDefault();
    searchIssues();
  };

  const exportToCSV = () => {
    const headers = ['Number', 'Title', 'State', 'Labels', 'Assignee', 'Created', 'Updated'];
    const csvContent = [
      headers.join(','),
      ...issues.map(issue => [
        issue.number,
        `"${issue.title.replace(/"/g, '""')}"`,
        issue.state,
        `"${issue.labels.map(l => l.name).join('; ')}"`,
        issue.assignee ? issue.assignee.login : '',
        issue.created_at,
        issue.updated_at
      ].join(','))
    ].join('\n');

    const blob = new Blob([csvContent], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `issues-${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="issue-search">
      <h2>GitHub Issue Search</h2>
      
      <form onSubmit={handleSubmit}>
        <div className="search-filter">
          <input
            type="text"
            placeholder="Search filter (e.g., is:open label:bug repo:owner/repo)"
            value={searchFilter}
            onChange={(e) => setSearchFilter(e.target.value)}
            className="search-input"
          />
        </div>

        <div className="filters">
          <div className="filter-group">
            <label>State:</label>
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
            <label>Labels:</label>
            <input
              type="text"
              placeholder="bug, enhancement, help wanted"
              value={filters.labels}
              onChange={(e) => handleFilterChange('labels', e.target.value)}
            />
          </div>

          <div className="filter-group">
            <label>Assignee:</label>
            <input
              type="text"
              placeholder="username"
              value={filters.assignee}
              onChange={(e) => handleFilterChange('assignee', e.target.value)}
            />
          </div>

          <div className="filter-group">
            <label>Author:</label>
            <input
              type="text"
              placeholder="username"
              value={filters.author}
              onChange={(e) => handleFilterChange('author', e.target.value)}
            />
          </div>

          <div className="filter-group">
            <label>Involved:</label>
            <input
              type="text"
              placeholder="username"
              value={filters.involved}
              onChange={(e) => handleFilterChange('involved', e.target.value)}
            />
          </div>

          <div className="filter-group">
            <label>Sort:</label>
            <select 
              value={filters.sort} 
              onChange={(e) => handleFilterChange('sort', e.target.value)}
            >
              <option value="created">Created</option>
              <option value="updated">Updated</option>
              <option value="comments">Comments</option>
            </select>
          </div>

          <div className="filter-group">
            <label>Direction:</label>
            <select 
              value={filters.direction} 
              onChange={(e) => handleFilterChange('direction', e.target.value)}
            >
              <option value="desc">Descending</option>
              <option value="asc">Ascending</option>
            </select>
          </div>
        </div>

        <button type="submit" disabled={loading}>
          {loading ? 'Searching...' : 'Search Issues'}
        </button>
      </form>

      {error && <div className="error">{error}</div>}

      {issues.length > 0 && (
        <div className="results">
          <div className="results-header">
            <h3>Results ({issues.length})</h3>
            <button onClick={exportToCSV}>Export CSV</button>
          </div>
          
          <div className="issues-list">
            {issues.map(issue => (
              <div key={issue.id} className="issue-item">
                <div className="issue-header">
                  <span className={`issue-state ${issue.state}`}>
                    {issue.state === 'open' ? '🟢' : '🔴'} {issue.state}
                  </span>
                  <a href={issue.html_url} target="_blank" rel="noopener noreferrer">
                    #{issue.number}
                  </a>
                  <span className="issue-title">{issue.title}</span>
                </div>
                <div className="issue-meta">
                  {issue.labels.length > 0 && (
                    <div className="issue-labels">
                      {issue.labels.map(label => (
                        <span 
                          key={label.id} 
                          className="label"
                          style={{ backgroundColor: `#${label.color}` }}
                        >
                          {label.name}
                        </span>
                      ))}
                    </div>
                  )}
                  <div className="issue-details">
                    <span>Created: {new Date(issue.created_at).toLocaleDateString()}</span>
                    <span>Updated: {new Date(issue.updated_at).toLocaleDateString()}</span>
                    {issue.assignee && <span>Assignee: {issue.assignee.login}</span>}
                    <span>Comments: {issue.comments}</span>
                  </div>
                </div>
              </div>
            ))}
          </div>

          <div className="pagination">
            <button 
              disabled={filters.page === 1}
              onClick={() => handleFilterChange('page', filters.page - 1)}
            >
              Previous
            </button>
            <span>Page {filters.page}</span>
            <button 
              disabled={issues.length < filters.per_page}
              onClick={() => handleFilterChange('page', filters.page + 1)}
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
};

export default IssueSearch;
