// src/actions/github-list-issues.ts
import { createAction, Property } from '@activepieces/pieces-framework';
import { githubCommon } from '../common';
import { githubAuth } from '../../';
import { HttpMethod, httpClient } from '@activepieces/pieces-common';

export const githubListIssues = createAction({
  auth: githubAuth,
  name: 'list_issues',
  displayName: 'List Issues',
  description: 'List issues for a GitHub repository with advanced search filters',
  props: {
    repository: githubCommon.repositoryDropdown,
    search_filter: Property.ShortText({
      displayName: 'Search Filter',
      description: 'Top-level search query (e.g., "is:open label:bug")',
      required: false,
    }),
    state: Property.StaticDropdown({
      displayName: 'State',
      description: 'Filter by issue state',
      required: false,
      options: {
        options: [
          { label: 'Open', value: 'open' },
          { label: 'Closed', value: 'closed' },
          { label: 'All', value: 'all' },
        ],
      },
    }),
    labels: Property.Array({
      displayName: 'Labels',
      description: 'Filter by labels (comma-separated)',
      required: false,
    }),
    assignee: Property.ShortText({
      displayName: 'Assignee',
      description: 'Filter by assignee username',
      required: false,
    }),
    author: Property.ShortText({
      displayName: 'Author',
      description: 'Filter by author username',
      required: false,
    }),
    involved: Property.ShortText({
      displayName: 'Involved',
      description: 'Filter by involved user (mentions, comments, etc.)',
      required: false,
    }),
    sort: Property.StaticDropdown({
      displayName: 'Sort',
      description: 'Sort order',
      required: false,
      options: {
        options: [
          { label: 'Created (desc)', value: 'created' },
          { label: 'Updated (desc)', value: 'updated' },
          { label: 'Comments (desc)', value: 'comments' },
        ],
      },
    }),
    direction: Property.StaticDropdown({
      displayName: 'Direction',
      description: 'Sort direction',
      required: false,
      options: {
        options: [
          { label: 'Descending', value: 'desc' },
          { label: 'Ascending', value: 'asc' },
        ],
      },
    }),
    per_page: Property.Number({
      displayName: 'Per Page',
      description: 'Results per page (max 100)',
      required: false,
      defaultValue: 30,
    }),
    page: Property.Number({
      displayName: 'Page',
      description: 'Page number',
      required: false,
      defaultValue: 1,
    }),
  },
  async run(context) {
    const { repository, search_filter, state, labels, assignee, author, involved, sort, direction, per_page, page } = context.propsValue;
    const { owner, repo } = repository!;

    // Build query string
    const queryParts: string[] = [];
    if (search_filter) {
      queryParts.push(search_filter);
    }
    if (state && state !== 'all') {
      queryParts.push(`state:${state}`);
    }
    if (labels && labels.length > 0) {
      queryParts.push(`label:${labels.join(',')}`);
    }
    if (assignee) {
      queryParts.push(`assignee:${assignee}`);
    }
    if (author) {
      queryParts.push(`author:${author}`);
    }
    if (involved) {
      queryParts.push(`involves:${involved}`);
    }

    const query = queryParts.join(' ');

    // Use search API if query exists, otherwise use list issues API
    if (query) {
      const response = await httpClient.sendRequest({
        method: HttpMethod.GET,
        url: 'https://api.github.com/search/issues',
        headers: {
          Authorization: `Bearer ${context.auth.access_token}`,
          Accept: 'application/vnd.github.v3+json',
        },
        queryParams: {
          q: `repo:${owner}/${repo} ${query}`,
          sort: sort || 'created',
          order: direction || 'desc',
          per_page: String(per_page || 30),
          page: String(page || 1),
        },
      });
      return response.body;
    } else {
      // Use list repository issues API
      const response = await httpClient.sendRequest({
        method: HttpMethod.GET,
        url: `https://api.github.com/repos/${owner}/${repo}/issues`,
        headers: {
          Authorization: `Bearer ${context.auth.access_token}`,
          Accept: 'application/vnd.github.v3+json',
        },
        queryParams: {
          state: state || 'open',
          sort: sort || 'created',
          direction: direction || 'desc',
          per_page: String(per_page || 30),
          page: String(page || 1),
          ...(labels && labels.length > 0 ? { labels: labels.join(',') } : {}),
          ...(assignee ? { assignee } : {}),
          ...(author ? { creator: author } : {}),
          ...(involved ? { mentioned: involved } : {}),
        },
      });
      return response.body;
    }
  },
});
