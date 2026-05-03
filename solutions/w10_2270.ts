<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GitHub Issue Explorer · SuperPlane</title>
    <!-- Minimal styling, no external dependencies -->
    <style>
        * {
            box-sizing: border-box;
            margin: 0;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, sans-serif;
        }
        body {
            background: #f6f8fa;
            display: flex;
            justify-content: center;
            padding: 2rem 1rem;
        }
        .container {
            max-width: 1100px;
            width: 100%;
            background: white;
            border-radius: 12px;
            box-shadow: 0 2px 12px rgba(0,0,0,0.08);
            padding: 2rem 2rem 2.5rem;
        }
        h1 {
            font-size: 1.8rem;
            font-weight: 600;
            margin-bottom: 0.25rem;
            color: #0d1117;
        }
        .subhead {
            color: #57606a;
            margin-bottom: 1.5rem;
            font-size: 0.95rem;
            border-bottom: 1px solid #e1e4e8;
            padding-bottom: 0.75rem;
        }
        .search-top {
            display: flex;
            gap: 0.5rem;
            margin-bottom: 1.5rem;
            flex-wrap: wrap;
        }
        .search-top input {
            flex: 1;
            min-width: 200px;
            padding: 0.65rem 1rem;
            border: 1px solid #d0d7de;
            border-radius: 8px;
            font-size: 0.95rem;
            background: #f6f8fa;
            transition: 0.2s;
        }
        .search-top input:focus {
            outline: none;
            border-color: #0969da;
            box-shadow: 0 0 0 3px rgba(9,105,218,0.15);
            background: white;
        }
        .search-top button {
            background: #2da44e;
            color: white;
            border: none;
            padding: 0.65rem 1.5rem;
            border-radius: 8px;
            font-weight: 600;
            font-size: 0.95rem;
            cursor: pointer;
            transition: 0.15s;
            white-space: nowrap;
        }
        .search-top button:hover {
            background: #218838;
        }
        .filter-grid {
            display: flex;
            flex-wrap: wrap;
            gap: 0.75rem 1rem;
            background: #f6f8fa;
            padding: 1rem 1.25rem;
            border-radius: 10px;
            margin-bottom: 1.75rem;
            border: 1px solid #e1e4e8;
            align-items: center;
        }
        .filter-group {
            display: flex;
            align-items: center;
            gap: 0.3rem 0.6rem;
            flex-wrap: wrap;
        }
        .filter-group label {
            font-size: 0.8rem;
            font-weight: 600;
            color: #24292f;
            text-transform: uppercase;
            letter-spacing: 0.02em;
        }
        .filter-group select, .filter-group input {
            padding: 0.35rem 0.6rem;
            border: 1px solid #d0d7de;
            border-radius: 6px;
            background: white;
            font-size: 0.85rem;
            min-width: 110px;
        }
        .filter-group input {
            min-width: 130px;
        }
        .filter-group select {
            background: white;
        }
        .status-badge {
            display: inline-block;
            background: #ddf4ff;
            color: #0969da;
            padding: 0.2rem 0.7rem;
            border-radius: 20px;
            font-size: 0.75rem;
            font-weight: 600;
            margin-left: 0.25rem;
        }
        .results-meta {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 0.75rem;
            flex-wrap: wrap;
        }
        .results-count {
            font-weight: 500;
            color: #24292f;
        }
        .export-btn {
            background: transparent;
            border: 1px solid #d0d7de;
            padding: 0.3rem 1rem;
            border-radius: 6px;
            font-size: 0.8rem;
            cursor: pointer;
            color: #0969da;
            font-weight: 500;
        }
        .export-btn:hover {
            background: #e8f0fe;
        }
        .issue-list {
            list-style: none;
            padding: 0;
            margin: 0;
        }
        .issue-item {
            display: flex;
            align-items: flex-start;
            gap: 0.75rem;
            padding: 0.9rem 0.5rem;
            border-bottom: 1px solid #e1e4e8;
            transition: background 0.1s;
        }
        .issue-item:hover {
            background: #f6f8fa;
        }
        .issue-icon {
            font-size: 1.2rem;
            margin-top: 0.1rem;
            width: 1.6rem;
            text-align: center;
            flex-shrink: 0;
        }
        .issue-icon.open {
            color: #2da44e;
        }
        .issue-icon.closed {
            color: #8250df;
        }
        .issue-content {
            flex: 1;
            min-width: 0;
        }
        .issue-title {
            font-weight: 600;
            color: #0d1117;
            text-decoration: none;
            font-size: 0.95rem;
        }
        .issue-title:hover {
            color: #0969da;
        }
        .issue-meta {
            font-size: 0.78rem;
            color: #57606a;
            margin-top: 0.2rem;
            display: flex;
            flex-wrap: wrap;
            gap: 0.4rem 1rem;
        }
        .issue-labels {
            display: flex;
            flex-wrap: wrap;
            gap: 0.3rem;
            margin-top: 0.25rem;
        }
        .label {
            display: inline-block;
            padding: 0.1rem 0.6rem;
            border-radius: 12px;
            font-size: 0.7rem;
            font-weight: 500;
            background: #e1e4e8;
            color: #24292f;
        }
        .pagination {
            display: flex;
            justify-content: center;
            gap: 0.5rem;
            margin-top: 1.5rem;
            flex-wrap: wrap;
        }
        .pagination button {
            background: white;
            border: 1px solid #d0d7de;
            padding: 0.4rem 1rem;
            border-radius: 6px;
            cursor: pointer;
            font-size: 0.85rem;
        }
        .pagination button:disabled {
            opacity: 0.5;
            cursor: default;
        }
        .pagination button:hover:not(:disabled) {
            background: #e8f0fe;
        }
        .empty-state {
            text-align: center;
            padding: 2.5rem 1rem;
            color: #57606a;
        }
        .error-message {
            background: #ffebe9;
            border: 1px solid #f85149;
            color: #b4231e;
            padding: 0.6rem 1rem;
            border-radius: 8px;
            margin-bottom: 1rem;
            font-size: 0.9rem;
        }
        .loading {
            text-align: center;
            padding: 2rem;
            color: #57606a;
        }
        .spinner {
            display: inline-block;
            width: 1.2rem;
            height: 1.2rem;
            border: 2px solid #d0d7de;
            border-top-color: #0969da;
            border-radius: 50%;
            animation: spin 0.6s linear infinite;
            margin-right: 0.5rem;
            vertical-align: middle;
        }
        @keyframes spin { to { transform: rotate(360deg); } }
        .footer-note {
            margin-top: 1.5rem;
            font-size: 0.75rem;
            color: #8b949e;
            text-align: center;
            border-top: 1px solid #e1e4e8;
            padding-top: 1rem;
        }
        .wallet {
            background: #f0f6fc;
            padding: 0.2rem 0.8rem;
            border-radius: 30px;
            font-family: monospace;
            font-size: 0.7rem;
        }
    </style>
</head>
<body>
<div class="container" id="app">
    <h1>🔍 GitHub Issue Explorer</h1>
    <div class="subhead">Search & filter repository issues · SuperPlane · <span class="wallet">TU8NBT5iGyMNkLwWmWmgy7tFMbKnafLHcu</span></div>

    <!-- TOP SEARCH FILTER (prefills form) -->
    <div class="search-top">
        <input type="text" id="searchTop" placeholder="e.g. is:open label:bug repo:facebook/react" value="repo:facebook/react is:open">
        <button id="searchBtn">🔎 Search Issues</button>
    </div>

    <!-- INDIVIDUAL FILTER FIELDS (qualifiers) -->
    <div class="filter-grid">
        <div class="filter-group">
            <label>State</label>
            <select id="filterState">
                <option value="open">Open</option>
                <option value="closed">Closed</option>
                <option value="">Any</option>
            </select>
        </div>
        <div class="filter-group">
            <label>Label</label>
            <input type="text" id="filterLabel" placeholder="bug, enhancement, help wanted" value="">
        </div>
        <div class="filter-group">
            <label>Assignee</label>
            <input type="text" id="filterAssignee" placeholder="username" value="">
        </div>
        <div class="filter-group">
            <label>Author</label>
            <input type="text" id="filterAuthor" placeholder="author" value="">
        </div>
        <div class="filter-group">
            <label>Involves</label>
            <input type="text" id="filterInvolves" placeholder="@me or user" value="">
        </div>
        <div class="filter-group">
            <label>Repo</label>
            <input type="text" id="filterRepo" placeholder="owner/repo" value="facebook/react">
        </div>
    </div>

    <!-- RESULTS HEADER -->
    <div class="results-meta">
        <span class="results-count" id="resultsCount">Loading issues...</span>
        <button class="export-btn" id="exportCSV">📥 Export CSV (for Jira/Slack)</button>
    </div>

    <!-- ERROR / LOADING / LIST -->
    <div id="errorContainer"></div>
    <div id="loadingContainer"></div>
    <ul class="issue-list" id="issueList"></ul>
    <div id="emptyState" class="empty-state" style="display:none;">✨ No issues match your filters. Try adjusting.</div>

    <!-- PAGINATION -->
    <div class="pagination" id="pagination">
        <button id="prevPage" disabled>← Previous</button>
        <span id="pageIndicator" style="align-self:center; font-size:0.85rem;">Page 1</span>
        <button id="nextPage" disabled>Next →</button>
    </div>

    <div class="footer-note">
        ⚡ GitHub Issue Search · data from public repos · wallet: TU8NBT5iGyMNkLwWmWmgy7tFMbKnafLHcu
    </div>
</div>

<script>
    (function() {
        'use strict';

        // ---------- state ----------
        let currentPage = 1;
        let totalCount = 0;
        let issuesData = [];
        const PER_PAGE = 10;
        let currentQuery = 'repo:facebook/react is:open'; // default from top search

        // DOM refs
        const searchTopInput = document.getElementById('searchTop');
        const searchBtn = document.getElementById('searchBtn');
        const filterState = document.getElementById('filterState');
        const filterLabel = document.getElementById('filterLabel');
        const filterAssignee = document.getElementById('filterAssignee');
        const filterAuthor = document.getElementById('filterAuthor');
        const filterInvolves = document.getElementById('filterInvolves');
        const filterRepo = document.getElementById('filterRepo');
        const issueListEl = document.getElementById('issueList');
        const resultsCountEl = document.getElementById('resultsCount');
        const emptyStateEl = document.getElementById('emptyState');
        const errorContainer = document.getElementById('errorContainer');
        const loadingContainer = document.getElementById('loadingContainer');
        const prevBtn = document.getElementById('prevPage');
        const nextBtn = document.getElementById('nextPage');
        const pageIndicator = document.getElementById('pageIndicator');
        const exportBtn = document.getElementById('exportCSV');

        // ---------- helpers ----------
        function buildQueryFromFilters() {
            const parts = [];

            // repo (mandatory for meaningful search)
            const repo = filterRepo.value.trim();
            if (repo) {
                // support owner/repo or just repo
                if (repo.includes('/')) {
                    parts.push(`repo:${repo}`);
                } else {
                    parts.push(`repo:${repo}`);
                }
            } else {
                // fallback to avoid empty
                parts.push('repo:facebook/react');
            }

            // state
            const state = filterState.value;
            if (state === 'open') parts.push('is:open');
            else if (state === 'closed') parts.push('is:closed');

            // label (comma separated -> multiple label: qualifiers)
            const labelRaw = filterLabel.value.trim();
            if (labelRaw) {
                const labels = labelRaw.split(',').map(l => l.trim()).filter(Boolean);
                labels.forEach(l => parts.push(`label:${l}`));
            }

            // assignee
            const assignee = filterAssignee.value.trim();
            if (assignee) parts.push(`assignee:${assignee}`);

            // author
            const author = filterAuthor.value.trim();
            if (author) parts.push(`author:${author}`);

            // involves
            const involves = filterInvolves.value.trim();
            if (involves) parts.push(`involves:${involves}`);

            return parts.join(' ');
        }

        // sync top search -> individual fields (prefill)
        function parseTopSearchIntoFields(query) {
            if (!query) return;
            // reset fields first (except repo, we keep)
            filterState.value = '';
            filterLabel.value = '';
            filterAssignee.value = '';
            filterAuthor.value = '';
            filterInvolves.value = '';
            // extract known qualifiers
            const tokens = query.match(/(\w+:\S+)/g) || [];
            tokens.forEach(token => {
                const [key, ...valArr] = token.split(':');
                const val = valArr.join(':'); // in case value contains ':'
                switch(key) {
                    case 'is':
                        if (val === 'open') filterState.value = 'open';
                        else if (val === 'closed') filterState.value = 'closed';
                        break;
                    case 'label':
                        if (filterLabel.value) filterLabel.value += ', ';
                        filterLabel.value += val;
                        break;
                    case 'assignee':
                        filterAssignee.value = val;
                        break;
                    case 'author':
                        filterAuthor.value = val;
                        break;
                    case 'involves':
                        filterInvolves.value = val;
                        break;
                    case 'repo':
                        filterRepo.value = val;
                        break;
                    default: break;
                }
            });
        }

        // update top search from individual fields
        function updateTopSearchFromFields() {
            const q = buildQueryFromFilters();
            searchTopInput.value = q;
            return q;
        }

        // ---------- API call (GitHub issue search) ----------
        async function fetchIssues(query, page = 1) {
            const perPage = PER_PAGE;
            const url = `https://api.github.com/search/issues?q=${encodeURIComponent(query)}&per_page=${perPage}&page=${page}`;
            const response = await fetch(url, {
                headers: { 'Accept': 'application/vnd.github.v3+json' }
            });
            if (!response.ok) {
                let errMsg = `GitHub API error: ${response.status}`;
                try {
                    const errBody = await response.json();
                    if (errBody.message) errMsg += ` — ${errBody.message}`;
                } catch (_) {}
                throw new Error(errMsg);
            }
            const data = await response.json();
            return {
                items: data.items || [],
                totalCount: data.total_count || 0
            };
        }

        // ---------- render ----------
        function renderIssues(issues, total) {
            issueListEl.innerHTML = '';
            if (!issues || issues.length === 0) {
                emptyStateEl.style.display = 'block';
                resultsCountEl.textContent = `0 issues`;
                return;
            }
            emptyStateEl.style.display = 'none';
            resultsCountEl.textContent = `${total} issue${total !== 1 ? 's' : ''} · showing page ${currentPage}`;

            issues.forEach(issue => {
                const li = document.createElement('li');
                li.className = 'issue-item';

                const iconSpan = document.createElement('span');
                iconSpan.className = `issue-icon ${issue.state === 'open'