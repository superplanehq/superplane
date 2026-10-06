export const INTEGRATION_LIST_NOTICE_TYPE = "list-notice";
export const DATADOG_ISSUE_SEARCH_FAILED_NOTICE_ID = "issue-search-failed";

export type IntegrationListResource = {
  id?: string;
  name?: string;
  type?: string;
};

export function isIntegrationListNotice(resource: IntegrationListResource): boolean {
  return resource.type === INTEGRATION_LIST_NOTICE_TYPE;
}

export function integrationListNotices(resources: IntegrationListResource[]): IntegrationListResource[] {
  return resources.filter(isIntegrationListNotice);
}

export function withoutIntegrationListNotices<T extends IntegrationListResource>(resources: T[]): T[] {
  return resources.filter((resource) => !isIntegrationListNotice(resource));
}

export function hasIssueSearchFailedNotice(resources: IntegrationListResource[]): boolean {
  return resources.some(
    (resource) => isIntegrationListNotice(resource) && resource.id === DATADOG_ISSUE_SEARCH_FAILED_NOTICE_ID,
  );
}
