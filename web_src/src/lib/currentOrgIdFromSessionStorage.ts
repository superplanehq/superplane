
const CURRENT_ORGANIZATION_ID_KEY = "superplane:currentOrganizationId";

export function getCurrentOrgIdFromSessionStorage(): string | null {
  if (typeof window === "undefined") {
    return null;
  }
  try {
    const value = window.sessionStorage.getItem(CURRENT_ORGANIZATION_ID_KEY);
    return value && value.length > 0 ? value : null;
  } catch {
    return null;
  }
}

export function setCurrentOrgIdToSessionStorage(organizationId: string): void {
  if (typeof window === "undefined" || !organizationId) {
    return;
  }
  try {
    window.sessionStorage.setItem(CURRENT_ORGANIZATION_ID_KEY, organizationId);
  } catch {
    // Ignore quota / private mode
  }
}

export function clearCurrentOrgIdFromSessionStorage(): void {
  if (typeof window === "undefined") {
    return;
  }
  try {
    window.sessionStorage.removeItem(CURRENT_ORGANIZATION_ID_KEY);
  } catch {
    // Ignore quota / private mode
  }
}
