import { Checkbox } from "@/components/ui/checkbox";
import { useIntegrationResources } from "@/hooks/useIntegrations";

import { LinearLabelField } from "./LinearLabelField";
import { LinearProjectPicker } from "./LinearProjectPicker";
import { LinearTeamPicker } from "./LinearTeamPicker";
import { LINEAR_INTAKE_SETUP_COPY } from "./linearIntakeSetupCopy";
import { selectLinearTeam, toggleLinearProject, type LinearIntakeScope } from "./linearIntakeScope";

/**
 * Team, projects, and labels of a Linear intake. The setup dialog, the intake
 * settings, and the onboarding step all use it.
 *
 * Projects and labels follow the selected team. An older intake has projects
 * and no team. It keeps showing its projects and the workspace labels until
 * the user picks a team.
 */
export function LinearIntakeScopeFields({
  organizationId,
  integrationId,
  scope,
  onChange,
}: {
  organizationId: string;
  integrationId: string;
  scope: LinearIntakeScope;
  onChange: (scope: LinearIntakeScope) => void;
}) {
  const connected = Boolean(organizationId && integrationId);
  const showFilters = Boolean(scope.teamId) || scope.projectIds.length > 0;
  const teamsQuery = useIntegrationResources(organizationId, integrationId, "team", undefined, {
    enabled: connected,
  });
  const projectsQuery = useIntegrationResources(
    organizationId,
    integrationId,
    "project",
    scope.teamId ? { team: scope.teamId } : undefined,
    { enabled: connected && showFilters },
  );

  return (
    <div className="flex flex-col gap-6" data-testid="linear-intake-filters">
      <fieldset className="min-w-0">
        <legend className="workspace-section-title">{LINEAR_INTAKE_SETUP_COPY.teamLabel}</legend>
        <div className="mt-2">
          <LinearTeamPicker
            teams={teamsQuery.data ?? []}
            selectedId={scope.teamId}
            loading={teamsQuery.isLoading}
            error={teamsQuery.isError}
            onSelect={(teamId) => onChange(selectLinearTeam(scope, teamId))}
            onRetry={() => void teamsQuery.refetch()}
          />
        </div>
      </fieldset>
      {showFilters ? (
        <>
          <fieldset className="min-w-0">
            <legend className="workspace-section-title">{LINEAR_INTAKE_SETUP_COPY.projectsLabel}</legend>
            <p className="workspace-body-text mt-1 text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.projectsHelper}</p>
            <div className="mt-2">
              <LinearProjectPicker
                projects={projectsQuery.data ?? []}
                selectedIds={scope.projectIds}
                loading={projectsQuery.isLoading}
                error={projectsQuery.isError}
                onToggle={(projectId) => onChange(toggleLinearProject(scope, projectId))}
                onRetry={() => void projectsQuery.refetch()}
              />
            </div>
            {scope.projectIds.length > 0 ? (
              <label className="workspace-body-text mt-3 flex cursor-pointer items-center gap-2">
                <Checkbox
                  checked={scope.includeIssuesWithoutProject}
                  onChange={(event) => onChange({ ...scope, includeIssuesWithoutProject: event.target.checked })}
                  data-testid="linear-include-no-project"
                />
                {LINEAR_INTAKE_SETUP_COPY.includeNoProjectLabel}
              </label>
            ) : null}
          </fieldset>
          <fieldset className="min-w-0">
            <legend className="workspace-section-title">{LINEAR_INTAKE_SETUP_COPY.labelsLabel}</legend>
            <p className="workspace-body-text mt-1 text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.labelsHelper}</p>
            <div className="mt-2">
              <LinearLabelField
                organizationId={organizationId}
                integrationId={integrationId}
                teamId={scope.teamId}
                labels={scope.labels}
                onChange={(labels) => onChange({ ...scope, labels })}
              />
            </div>
          </fieldset>
        </>
      ) : null}
    </div>
  );
}
