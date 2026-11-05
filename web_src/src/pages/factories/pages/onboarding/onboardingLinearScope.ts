import { EMPTY_LINEAR_INTAKE_SCOPE, type LinearIntakeScope } from "../linearIntakeScope";

const STORAGE_PREFIX = "superplane:onboarding-linear-scope";

function storageKey(factoryId: string, integrationId: string): string {
  return `${STORAGE_PREFIX}:${factoryId}:${integrationId}`;
}

function stringList(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((entry): entry is string => typeof entry === "string" && entry.length > 0);
}

export function readOnboardingLinearScope(factoryId: string, integrationId: string): LinearIntakeScope {
  if (!factoryId || !integrationId) return EMPTY_LINEAR_INTAKE_SCOPE;
  const raw = localStorage.getItem(storageKey(factoryId, integrationId));
  if (!raw) return EMPTY_LINEAR_INTAKE_SCOPE;
  try {
    const parsed = JSON.parse(raw) as Partial<LinearIntakeScope> | null;
    return {
      teamId: typeof parsed?.teamId === "string" ? parsed.teamId : "",
      projectIds: stringList(parsed?.projectIds),
      includeIssuesWithoutProject:
        typeof parsed?.includeIssuesWithoutProject === "boolean" ? parsed.includeIssuesWithoutProject : true,
      labels: stringList(parsed?.labels),
    };
  } catch {
    return EMPTY_LINEAR_INTAKE_SCOPE;
  }
}

export function writeOnboardingLinearScope(factoryId: string, integrationId: string, scope: LinearIntakeScope): void {
  if (!factoryId || !integrationId) return;
  if (!scope.teamId) {
    localStorage.removeItem(storageKey(factoryId, integrationId));
    return;
  }
  localStorage.setItem(storageKey(factoryId, integrationId), JSON.stringify(scope));
}
