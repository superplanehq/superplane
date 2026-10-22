import type { FactoriesFactoryIntakeSettings } from "@/api-client";
import type { IntakeSourceSettings } from "./intakeSourceSettingsModel";

/**
 * Which Linear issues an intake accepts. A team is the required scope.
 * Projects and labels narrow it. Empty projects mean every project of the team.
 */
export interface LinearIntakeScope {
  teamId: string;
  projectIds: string[];
  /** Keep issues without a project when projects are selected. */
  includeIssuesWithoutProject: boolean;
  labels: string[];
}

export const EMPTY_LINEAR_INTAKE_SCOPE: LinearIntakeScope = {
  teamId: "",
  projectIds: [],
  includeIssuesWithoutProject: true,
  labels: [],
};

/** Projects and labels belong to one team, so a new team clears them. */
export function selectLinearTeam(scope: LinearIntakeScope, teamId: string): LinearIntakeScope {
  if (scope.teamId === teamId) {
    return scope;
  }
  return { ...scope, teamId, projectIds: [], labels: [] };
}

export function toggleLinearProject(scope: LinearIntakeScope, projectId: string): LinearIntakeScope {
  const projectIds = scope.projectIds.includes(projectId)
    ? scope.projectIds.filter((id) => id !== projectId)
    : [...scope.projectIds, projectId];
  return { ...scope, projectIds };
}

export function linearIntakeScopeToApi(
  scope: LinearIntakeScope,
): Pick<
  FactoriesFactoryIntakeSettings,
  "linearTeamId" | "linearProjectIds" | "linearIncludeIssuesWithoutProject" | "linearLabels"
> {
  return {
    linearTeamId: scope.teamId,
    linearProjectIds: scope.projectIds,
    linearIncludeIssuesWithoutProject: scope.includeIssuesWithoutProject,
    linearLabels: scope.labels,
  };
}

type LinearScopeSettings = Pick<
  IntakeSourceSettings,
  "linearTeamId" | "linearProjectIds" | "linearIncludeIssuesWithoutProject" | "linearLabels"
>;

export function linearScopeFromSettings(settings: LinearScopeSettings): LinearIntakeScope {
  return {
    teamId: settings.linearTeamId,
    projectIds: settings.linearProjectIds,
    includeIssuesWithoutProject: settings.linearIncludeIssuesWithoutProject,
    labels: settings.linearLabels,
  };
}

export function linearSettingsFromScope(scope: LinearIntakeScope): LinearScopeSettings {
  return {
    linearTeamId: scope.teamId,
    linearProjectIds: scope.projectIds,
    linearIncludeIssuesWithoutProject: scope.includeIssuesWithoutProject,
    linearLabels: scope.labels,
  };
}
