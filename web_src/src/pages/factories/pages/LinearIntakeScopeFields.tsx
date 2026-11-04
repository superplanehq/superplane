import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { useIntegrationResources } from "@/hooks/useIntegrations";
import { useId, useState } from "react";

import { IntakeSettingsRadioOption } from "./IntakeSettingsRadioOption";
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
  showLabels = true,
  settingsLayout = false,
  onLabelFilterChange,
  onProjectFilterChange,
}: {
  organizationId: string;
  integrationId: string;
  scope: LinearIntakeScope;
  onChange: (scope: LinearIntakeScope) => void;
  showLabels?: boolean;
  settingsLayout?: boolean;
  onLabelFilterChange?: (enabled: boolean) => void;
  onProjectFilterChange?: (enabled: boolean) => void;
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
  // A saved project that Linear no longer returns stays selected and visible,
  // so the user removes it or picks a replacement. Dropping it silently could
  // widen the intake: an empty project list accepts every ticket in the team.
  const missingProjectIds = missingLinearProjectIds(
    scope.projectIds,
    projectsQuery.data ?? [],
    linearProjectsSettled(connected, showFilters, projectsQuery.isPending, projectsQuery.isError),
  );

  return (
    <div className="flex flex-col gap-6" data-testid="linear-intake-filters">
      {settingsLayout ? <h3 className="workspace-section-title">Which issues to include</h3> : null}
      <LinearScopeTeam
        settingsLayout={settingsLayout}
        teams={teamsQuery.data ?? []}
        selectedId={scope.teamId}
        loading={teamsQuery.isLoading}
        error={teamsQuery.isError}
        onSelect={(teamId) => onChange(selectLinearTeam(scope, teamId))}
        onRetry={() => void teamsQuery.refetch()}
      />
      {showFilters ? (
        <>
          <LinearScopeProjects
            scope={scope}
            onChange={onChange}
            settingsLayout={settingsLayout}
            onProjectFilterChange={onProjectFilterChange}
            projects={projectsQuery.data ?? []}
            missingProjectIds={missingProjectIds}
            loading={projectsQuery.isLoading}
            error={projectsQuery.isError}
            onRetry={() => void projectsQuery.refetch()}
          />
          {showLabels ? (
            <LinearScopeLabels
              organizationId={organizationId}
              integrationId={integrationId}
              scope={scope}
              onChange={onChange}
              settingsLayout={settingsLayout}
              onLabelFilterChange={onLabelFilterChange}
            />
          ) : null}
        </>
      ) : null}
    </div>
  );
}

function missingLinearProjectIds(
  selectedIds: string[],
  projects: Array<{ id?: string }>,
  projectsSettled: boolean,
): string[] {
  if (!projectsSettled) return [];
  const fetchedIds = new Set(projects.map((project) => project.id).filter((id): id is string => Boolean(id)));
  return selectedIds.filter((id) => !fetchedIds.has(id));
}

function linearProjectsSettled(connected: boolean, showFilters: boolean, pending: boolean, error: boolean): boolean {
  return connected && showFilters && !pending && !error;
}

function LinearScopeTeam({
  settingsLayout,
  teams,
  selectedId,
  loading,
  error,
  onSelect,
  onRetry,
}: {
  settingsLayout: boolean;
  teams: Array<{ id?: string; name?: string }>;
  selectedId: string;
  loading: boolean;
  error: boolean;
  onSelect: (id: string) => void;
  onRetry: () => void;
}) {
  return (
    <fieldset className="min-w-0">
      <legend className={settingsLayout ? "text-sm font-medium" : "workspace-section-title"}>
        {LINEAR_INTAKE_SETUP_COPY.teamLabel}
      </legend>
      <div className="mt-2">
        <LinearTeamPicker
          teams={teams}
          selectedId={selectedId}
          loading={loading}
          error={error}
          onSelect={onSelect}
          onRetry={onRetry}
        />
      </div>
    </fieldset>
  );
}

function LinearScopeLabels({
  organizationId,
  integrationId,
  scope,
  onChange,
  settingsLayout,
  onLabelFilterChange,
}: {
  organizationId: string;
  integrationId: string;
  scope: LinearIntakeScope;
  onChange: (scope: LinearIntakeScope) => void;
  settingsLayout: boolean;
  onLabelFilterChange?: (enabled: boolean) => void;
}) {
  return (
    <fieldset className="min-w-0">
      <legend className={settingsLayout ? "text-sm font-medium" : "workspace-section-title"}>
        {LINEAR_INTAKE_SETUP_COPY.labelsLabel}
      </legend>
      {settingsLayout ? (
        <LinearSettingsLabels
          organizationId={organizationId}
          integrationId={integrationId}
          scope={scope}
          onChange={onChange}
          onLabelFilterChange={onLabelFilterChange}
        />
      ) : (
        <>
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
        </>
      )}
    </fieldset>
  );
}

function LinearScopeProjects({
  scope,
  onChange,
  settingsLayout,
  onProjectFilterChange,
  projects,
  missingProjectIds,
  loading,
  error,
  onRetry,
}: {
  scope: LinearIntakeScope;
  onChange: (scope: LinearIntakeScope) => void;
  settingsLayout: boolean;
  onProjectFilterChange?: (enabled: boolean) => void;
  projects: Array<{ id?: string; name?: string }>;
  missingProjectIds: string[];
  loading: boolean;
  error: boolean;
  onRetry: () => void;
}) {
  const choiceName = useId();
  const [filterByProject, setFilterByProject] = useState(scope.projectIds.length > 0);
  const selectedProjects = filterByProject || scope.projectIds.length > 0;

  return (
    <fieldset className="min-w-0">
      <legend className={settingsLayout ? "text-sm font-medium" : "workspace-section-title"}>
        {LINEAR_INTAKE_SETUP_COPY.projectsLabel}
      </legend>
      {settingsLayout ? (
        <div className="mt-3 flex flex-col gap-3">
          <IntakeSettingsRadioOption
            variant="plain"
            name={choiceName}
            value="all"
            title="All projects"
            checked={!selectedProjects}
            onChange={() => {
              setFilterByProject(false);
              onProjectFilterChange?.(false);
              onChange({ ...scope, projectIds: [] });
            }}
          />
          <IntakeSettingsRadioOption
            variant="plain"
            name={choiceName}
            value="selected"
            title="Only issues in selected projects"
            checked={selectedProjects}
            onChange={() => {
              setFilterByProject(true);
              onProjectFilterChange?.(true);
            }}
          />
          {selectedProjects ? (
            <div className="ml-7 flex min-w-0 flex-col gap-2">
              <LinearProjectChoices
                scope={scope}
                onChange={onChange}
                projects={projects}
                missingProjectIds={missingProjectIds}
                loading={loading}
                error={error}
                onRetry={onRetry}
              />
              {scope.projectIds.length === 0 ? (
                <p className="text-xs leading-5 text-muted-foreground">
                  Select at least one project, or choose All projects.
                </p>
              ) : null}
            </div>
          ) : null}
        </div>
      ) : (
        <>
          <p className="workspace-body-text mt-1 text-muted-foreground">{LINEAR_INTAKE_SETUP_COPY.projectsHelper}</p>
          <div className="mt-2">
            <LinearProjectChoices
              scope={scope}
              onChange={onChange}
              projects={projects}
              missingProjectIds={missingProjectIds}
              loading={loading}
              error={error}
              onRetry={onRetry}
            />
          </div>
        </>
      )}
    </fieldset>
  );
}

function LinearProjectChoices({
  scope,
  onChange,
  projects,
  missingProjectIds,
  loading,
  error,
  onRetry,
}: {
  scope: LinearIntakeScope;
  onChange: (scope: LinearIntakeScope) => void;
  projects: Array<{ id?: string; name?: string }>;
  missingProjectIds: string[];
  loading: boolean;
  error: boolean;
  onRetry: () => void;
}) {
  return (
    <>
      <LinearProjectPicker
        projects={projects}
        selectedIds={scope.projectIds}
        missingSelectedIds={missingProjectIds}
        loading={loading}
        error={error}
        onToggle={(projectId) => onChange(toggleLinearProject(scope, projectId))}
        onRetry={onRetry}
      />
      {scope.teamId && scope.projectIds.length > 0 ? (
        <Label
          htmlFor="linear-include-no-project-input"
          className="mt-3 cursor-pointer gap-2 text-[13px] leading-[19.5px] font-normal"
        >
          <Checkbox
            id="linear-include-no-project-input"
            checked={scope.includeIssuesWithoutProject}
            onChange={(event) => onChange({ ...scope, includeIssuesWithoutProject: event.target.checked })}
            data-testid="linear-include-no-project"
          />
          {LINEAR_INTAKE_SETUP_COPY.includeNoProjectLabel}
        </Label>
      ) : null}
    </>
  );
}

function LinearSettingsLabels({
  organizationId,
  integrationId,
  scope,
  onChange,
  onLabelFilterChange,
}: {
  organizationId: string;
  integrationId: string;
  scope: LinearIntakeScope;
  onChange: (scope: LinearIntakeScope) => void;
  onLabelFilterChange?: (enabled: boolean) => void;
}) {
  const labelChoiceName = useId();
  const [filterByLabel, setFilterByLabel] = useState(scope.labels.length > 0);
  const selectedLabels = filterByLabel || scope.labels.length > 0;

  return (
    <div className="mt-3 flex flex-col gap-3">
      <IntakeSettingsRadioOption
        variant="plain"
        name={labelChoiceName}
        value="all"
        title="All issues"
        checked={!selectedLabels}
        onChange={() => {
          setFilterByLabel(false);
          onLabelFilterChange?.(false);
          onChange({ ...scope, labels: [] });
        }}
      />
      <IntakeSettingsRadioOption
        variant="plain"
        name={labelChoiceName}
        value="selected"
        title="Only issues with selected labels"
        checked={selectedLabels}
        onChange={() => {
          setFilterByLabel(true);
          onLabelFilterChange?.(true);
        }}
      />
      {selectedLabels ? (
        <div className="ml-7 flex min-w-0 flex-col gap-2">
          <LinearLabelField
            organizationId={organizationId}
            integrationId={integrationId}
            teamId={scope.teamId}
            labels={scope.labels}
            onChange={(labels) => onChange({ ...scope, labels })}
          />
          {scope.labels.length === 0 ? (
            <p className="text-xs leading-5 text-muted-foreground">Select at least one label, or choose All issues.</p>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
