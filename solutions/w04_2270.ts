// components/GitHubIssueSearch.js
import React, { useState, useEffect } from 'react';
import axios from 'axios';

const GitHubIssueSearch = () => {
  const [searchFilter, setSearchFilter] = useState('');
  const [issues, setIssues] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [filters, setFilters] = useState({
    state: '',
    labels: '',
    assignee: '',
    author: '',
    involved: '',
    sort: 'created',
    direction: 'desc',
    per_page: 30,
    page: 1
  });

  const [repo, setRepo] = useState('facebook/react');

  const fetchIssues = async () => {
    setLoading(true);
    setError(null);
    
    try {
      let query = `repo:${repo}`;
      
      if (searchFilter) {
        query = `${searchFilter} ${query}`;
      }
      
      if (filters.state) query += ` state:${filters.state}`;
      if (filters.labels) query += ` label:${filters.labels}`;
      if (filters.assignee) query += ` assignee:${filters.assignee}`;
      if (filters.author) query += ` author:${filters.author}`;
      if (filters.involved) query += ` involves:${filters.involved}`;

      const response = await axios.get('https://api.github.com/search/issues', {
        params: {
          q: query,
          sort: filters.sort,
          order: filters.direction,
          per_page: filters.per_page,
          page: filters.page
        },
        headers: {
          'Accept': 'application/vnd.github.v3+json'
        }
      });

      setIssues(response.data.items);
    } catch (err) {
      setError(err.response?.data?.message || 'Error fetching issues');
      setIssues([]);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchIssues();
  }, [filters.page]);

  const handleFilterChange = (field, value) => {
    setFilters(prev => ({
      ...prev,
      [field]: value,
      page: 1
    }));
  };

  const handleSearch = (e) => {
    e.preventDefault();
    setFilters(prev => ({ ...prev, page: 1 }));
    fetchIssues();
  };

  return (
    <div className="github-issue-search">
      <h2>GitHub Issue Search</h2>
      
      <form onSubmit={handleSearch} className="search-form">
        <div className="search-filter">
          <input
            type="text"
            placeholder="Search filter (e.g., is:open is:issue)"
            value={searchFilter}
            onChange={(e) => setSearchFilter(e.target.value)}
            className="search-input"
          />
          <button type="submit" className="search-button">Search</button>
        </div>

        <div className="filters-grid">
          <div className="filter-group">
            <label>Repository</label>
            <input
              type="text"
              value={repo}
              onChange={(e) => setRepo(e.target.value)}
              placeholder="owner/repo"
            />
          </div>

          <div className="filter-group">
            <label>State</label>
            <select
              value={filters.state}
              onChange={(e) => handleFilterChange('state', e.target.value)}
            >
              <option value="">All</option>
              <option value="open">Open</option>
              <option value="closed">Closed</option>
            </select>
          </div>

          <div className="filter-group">
            <label>Labels</label>
            <input
              type="text"
              value={filters.labels}
              onChange={(e) => handleFilterChange('labels', e.target.value)}
              placeholder="bug,feature"
            />
          </div>

          <div className="filter-group">
            <label>Assignee</label>
            <input
              type="text"
              value={filters.assignee}
              onChange={(e) => handleFilterChange('assignee', e.target.value)}
              placeholder="username"
            />
          </div>

          <div className="filter-group">
            <label>Author</label>
            <input
              type="text"
              value={filters.author}
              onChange={(e) => handleFilterChange('author', e.target.value)}
              placeholder="username"
            />
          </div>

          <div className="filter-group">
            <label>Involved</label>
            <input
              type="text"
              value={filters.involved}
              onChange={(e) => handleFilterChange('involved', e.target.value)}
              placeholder="username"
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
            <input
              type="number"
              min="1"
              max="100"
              value={filters.per_page}
              onChange={(e) => handleFilterChange('per_page', e.target.value)}
            />
          </div>
        </div>
      </form>

      {loading && <div className="loading">Loading issues...</div>}
      
      {error && <div className="error">{error}</div>}

      <div className="issues-list">
        {issues.map(issue => (
          <div key={issue.id} className="issue-card">
            <div className="issue-header">
              <span className={`issue-state ${issue.state}`}>{issue.state}</span>
              <a href={issue.html_url} target="_blank" rel="noopener noreferrer" className="issue-title">
                {issue.title}
              </a>
            </div>
            <div className="issue-meta">
              <span>#{issue.number}</span>
              <span>opened by {issue.user?.login}</span>
              <span>{new Date(issue.created_at).toLocaleDateString()}</span>
              {issue.labels?.length > 0 && (
                <div className="issue-labels">
                  {issue.labels.map(label => (
                    <span key={label.id} className="label" style={{backgroundColor: `#${label.color}`}}>
                      {label.name}
                    </span>
                  ))}
                </div>
              )}
            </div>
            {issue.body && (
              <p className="issue-body">{issue.body.substring(0, 200)}...</p>
            )}
          </div>
        ))}
      </div>

      {issues.length > 0 && (
        <div className="pagination">
          <button
            onClick={() => setFilters(prev => ({ ...prev, page: prev.page - 1 }))}
            disabled={filters.page === 1}
          >
            Previous
          </button>
          <span>Page {filters.page}</span>
          <button
            onClick={() => setFilters(prev => ({ ...prev, page: prev.page + 1 }))}
            disabled={issues.length < filters.per_page}
          >
            Next
          </button>
        </div>
      )}
    </div>
  );
};

export default GitHubIssueSearch;
