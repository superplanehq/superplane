import type { MeNotificationSettings, NotificationSettingsWorkspaceScope } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactories } from "@/hooks/useFactoryData";
import { useNotificationSettings, useUpdateNotificationSettings } from "@/hooks/useNotificationSettings";
import { getApiErrorMessage } from "@/lib/errors";
import {
  defaultNotificationTypeToggles,
  eventTypesFromToggles,
  filtersFromSettings,
  NOTIFICATION_TYPE_OPTIONS,
  togglesFromAllScopeEventTypes,
  togglesFromEventTypes,
  workspaceScopeFromSettings,
  type ConfigurableNotificationType,
  type NotificationTypeOption,
  type NotificationTypeToggles,
  type WorkspaceScopeForm,
} from "@/lib/notificationSettings";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";
import { BellOff, Info } from "lucide-react";
import { useEffect, useState } from "react";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { FactorySettingsNotificationWorkspacePicker } from "./FactorySettingsNotificationWorkspacePicker";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

const WORKSPACE_SCOPE_TO_PROTO: Record<WorkspaceScopeForm, NotificationSettingsWorkspaceScope> = {
  all: "WORKSPACE_SCOPE_ALL",
  filtered: "WORKSPACE_SCOPE_FILTERED",
  none: "WORKSPACE_SCOPE_NONE",
};

interface WorkspaceFilterForm {
  workspaceId: string;
  toggles: NotificationTypeToggles;
}

interface NotificationsFormState {
  scope: WorkspaceScopeForm;
  toggles: NotificationTypeToggles;
  filters: WorkspaceFilterForm[];
}

function formStateFromSettings(settings: MeNotificationSettings | undefined): NotificationsFormState {
  const scope = workspaceScopeFromSettings(settings);
  return {
    scope,
    toggles:
      scope === "all"
        ? togglesFromAllScopeEventTypes(settings?.workspaces?.eventTypes)
        : defaultNotificationTypeToggles(),
    filters: filtersFromSettings(settings).flatMap((filter) => {
      if (!filter.workspaceId) {
        return [];
      }
      return [{ workspaceId: filter.workspaceId, toggles: togglesFromEventTypes(filter.eventTypes) }];
    }),
  };
}

function settingsFromFormState(state: NotificationsFormState): MeNotificationSettings {
  return {
    workspaces: {
      scope: WORKSPACE_SCOPE_TO_PROTO[state.scope],
      eventTypes: state.scope === "all" ? eventTypesFromToggles(state.toggles) : [],
      filters:
        state.scope === "filtered"
          ? state.filters.map((filter) => ({
              workspaceId: filter.workspaceId,
              eventTypes: eventTypesFromToggles(filter.toggles),
            }))
          : [],
    },
  };
}

function isDirtyState(current: NotificationsFormState, saved: NotificationsFormState): boolean {
  return JSON.stringify(current) !== JSON.stringify(saved);
}

function notificationSaveError(form: NotificationsFormState): { scope?: string; type?: string } {
  if (form.scope === "filtered" && form.filters.length === 0) {
    return { scope: "Select at least one workspace, or use all workspaces." };
  }
  if (form.scope === "all" && eventTypesFromToggles(form.toggles).length === 0) {
    return { type: "Select at least one event type, or choose none." };
  }
  if (form.scope === "filtered" && form.filters.some((filter) => eventTypesFromToggles(filter.toggles).length === 0)) {
    return { type: "Select at least one event type for each workspace." };
  }
  return {};
}

function withAddedWorkspace(prev: NotificationsFormState, workspaceId: string): NotificationsFormState {
  if (prev.filters.some((filter) => filter.workspaceId === workspaceId)) {
    return prev;
  }
  return {
    ...prev,
    filters: [...prev.filters, { workspaceId, toggles: defaultNotificationTypeToggles() }],
  };
}

function withWorkspaceType(
  prev: NotificationsFormState,
  workspaceId: string,
  key: ConfigurableNotificationType,
  value: boolean,
): NotificationsFormState {
  return {
    ...prev,
    filters: prev.filters.map((filter) =>
      filter.workspaceId === workspaceId ? { ...filter, toggles: { ...filter.toggles, [key]: value } } : filter,
    ),
  };
}

export function FactorySettingsNotificationsPage() {
  const { organizationId } = useFactorySettingsLayout();
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const { data: settings, isLoading } = useNotificationSettings(organizationId);
  const { data: factories = [] } = useFactories(organizationId);
  const updateSettings = useUpdateNotificationSettings(organizationId);
  const canUpdate = canAct("notifications", "update");

  const [form, setForm] = useState<NotificationsFormState>(() => formStateFromSettings(settings));
  const [savedForm, setSavedForm] = useState<NotificationsFormState>(() => formStateFromSettings(settings));
  const [scopeError, setScopeError] = useState("");
  const [typeError, setTypeError] = useState("");

  useEffect(() => {
    const next = formStateFromSettings(settings);
    setForm(next);
    setSavedForm(next);
    setScopeError("");
    setTypeError("");
  }, [settings]);

  const isDirty = isDirtyState(form, savedForm);
  const formLocked = isLoading || !canUpdate;

  const handleSave = async () => {
    if (!canUpdate) {
      return;
    }
    const error = notificationSaveError(form);
    if (error.scope) {
      setScopeError(error.scope);
      return;
    }
    if (error.type) {
      setTypeError(error.type);
      return;
    }
    try {
      const saved = await updateSettings.mutateAsync(settingsFromFormState(form));
      const next = formStateFromSettings(saved);
      setForm(next);
      setSavedForm(next);
      showSuccessToast("Notification settings saved.");
    } catch (caught) {
      showErrorToast(getApiErrorMessage(caught, "Failed to save notification settings"));
    }
  };

  return (
    <FactorySettingsPageFrame
      title="Notifications"
      subtitle="Choose which work order emails you receive. You never get an email about your own actions."
    >
      <div className="space-y-6" data-testid="factory-settings-notifications-form">
        <FactorySettingsCard>
          <div className={cn("space-y-6", formLocked && "pointer-events-none opacity-50")}>
            <WorkspaceScopeSection
              scope={form.scope}
              filters={form.filters}
              factories={factories}
              scopeError={scopeError}
              typeError={typeError}
              onScopeChange={(scope) => {
                setScopeError("");
                setTypeError("");
                setForm((prev) => ({ ...prev, scope }));
              }}
              onAddWorkspace={(workspaceId) => {
                setScopeError("");
                setForm((prev) => withAddedWorkspace(prev, workspaceId));
              }}
              onRemoveWorkspace={(workspaceId) => {
                setScopeError("");
                setForm((prev) => ({
                  ...prev,
                  filters: prev.filters.filter((filter) => filter.workspaceId !== workspaceId),
                }));
              }}
              onToggleType={(workspaceId, key, value) => {
                setTypeError("");
                setForm((prev) => withWorkspaceType(prev, workspaceId, key, value));
              }}
            />
            {form.scope === "all" ? (
              <EventTypesSection
                title="Notify me about"
                description="Choose which events send an email."
                idPrefix="notifications-type"
                toggles={form.toggles}
                error={typeError}
                onToggle={(key, value) => {
                  setTypeError("");
                  setForm((prev) => ({
                    ...prev,
                    toggles: { ...prev.toggles, [key]: value },
                  }));
                }}
              />
            ) : null}
          </div>
        </FactorySettingsCard>
        <PermissionTooltip
          allowed={canUpdate || permissionsLoading}
          message="You do not have permission to change notification settings."
        >
          <LoadingButton
            disabled={isLoading || !canUpdate || !isDirty}
            loading={updateSettings.isPending}
            loadingText="Saving..."
            onClick={() => void handleSave()}
            data-testid="notifications-save"
          >
            Save
          </LoadingButton>
        </PermissionTooltip>
      </div>
    </FactorySettingsPageFrame>
  );
}

interface WorkspaceScopeOption {
  value: WorkspaceScopeForm;
  id: string;
  label: string;
  description: string;
}

const WORKSPACE_SCOPE_OPTIONS: WorkspaceScopeOption[] = [
  {
    value: "all",
    id: "notifications-scope-all",
    label: "All workspaces",
    description: "Send an email for events in every workspace you can access.",
  },
  {
    value: "filtered",
    id: "notifications-scope-filtered",
    label: "Choose workspaces",
    description: "Pick specific workspaces and set which events send email for each.",
  },
  {
    value: "none",
    id: "notifications-scope-none",
    label: "Off",
    description: "Do not send any work order emails.",
  },
];

interface WorkspaceScopeSectionProps {
  scope: WorkspaceScopeForm;
  filters: WorkspaceFilterForm[];
  factories: { id?: string; name?: string }[];
  scopeError: string;
  typeError: string;
  onScopeChange: (scope: WorkspaceScopeForm) => void;
  onAddWorkspace: (workspaceId: string) => void;
  onRemoveWorkspace: (workspaceId: string) => void;
  onToggleType: (workspaceId: string, key: ConfigurableNotificationType, value: boolean) => void;
}

function WorkspaceScopeSection({
  scope,
  filters,
  factories,
  scopeError,
  typeError,
  onScopeChange,
  onAddWorkspace,
  onRemoveWorkspace,
  onToggleType,
}: WorkspaceScopeSectionProps) {
  const selectedFactoryIds = filters.map((filter) => filter.workspaceId);
  const factoriesById = new Map(factories.flatMap((factory) => (factory.id ? [[factory.id, factory]] : [])));

  return (
    <div className="space-y-3">
      <div>
        <Label>Workspace scope</Label>
        <p className="mt-0.5 text-[12px] text-muted-foreground">
          Choose which workspaces can send you email, or turn workspace emails off.
        </p>
      </div>
      <div className="space-y-2" role="radiogroup" aria-label="Workspace scope">
        {WORKSPACE_SCOPE_OPTIONS.map((option) => (
          <ScopeChoice
            key={option.value}
            id={option.id}
            label={option.label}
            description={option.description}
            checked={scope === option.value}
            onSelect={() => onScopeChange(option.value)}
          />
        ))}
      </div>
      {scope === "filtered" ? (
        <div className="space-y-4">
          <FactorySettingsNotificationWorkspacePicker
            factories={factories}
            selectedFactoryIds={selectedFactoryIds}
            onAdd={onAddWorkspace}
            onRemove={onRemoveWorkspace}
          />
          {filters.map((filter) => (
            <WorkspaceEventTypesSection
              key={filter.workspaceId}
              workspaceId={filter.workspaceId}
              workspaceName={factoriesById.get(filter.workspaceId)?.name || filter.workspaceId}
              toggles={filter.toggles}
              error={typeError && eventTypesFromToggles(filter.toggles).length === 0 ? typeError : undefined}
              onToggle={(key, value) => onToggleType(filter.workspaceId, key, value)}
            />
          ))}
        </div>
      ) : null}
      {scope === "none" ? (
        <div
          className="flex items-center gap-2 rounded-md border border-border bg-muted/40 px-3 py-2 text-[13px] text-muted-foreground"
          data-testid="notifications-scope-off-message"
        >
          <BellOff className="size-4 shrink-0" aria-hidden />
          <span>You will not receive any work order emails.</span>
        </div>
      ) : null}
      {scopeError ? <p className="text-[11px] text-destructive">{scopeError}</p> : null}
    </div>
  );
}

function ScopeChoice({
  id,
  label,
  description,
  checked,
  onSelect,
}: {
  id: string;
  label: string;
  description: string;
  checked: boolean;
  onSelect: () => void;
}) {
  return (
    <button
      type="button"
      id={id}
      role="radio"
      aria-checked={checked}
      data-testid={id}
      onClick={onSelect}
      className={cn(
        "flex w-full flex-col items-start gap-0.5 rounded-md border border-border px-3 py-2 text-left transition-colors",
        "hover:border-accent-foreground/30 hover:bg-accent/50",
        checked && "border-foreground/40 bg-accent",
      )}
    >
      <span className="text-[13px] font-medium tracking-[-0.01em] text-foreground">{label}</span>
      <span className="text-[12px] text-muted-foreground">{description}</span>
    </button>
  );
}

function WorkspaceEventTypesSection({
  workspaceId,
  workspaceName,
  toggles,
  error,
  onToggle,
}: {
  workspaceId: string;
  workspaceName: string;
  toggles: NotificationTypeToggles;
  error?: string;
  onToggle: (key: ConfigurableNotificationType, value: boolean) => void;
}) {
  return (
    <EventTypesSection
      title={workspaceName}
      description="Choose which events send an email for this workspace."
      idPrefix={`notifications-type-${workspaceId}`}
      toggles={toggles}
      error={error}
      onToggle={onToggle}
    />
  );
}

function EventTypesSection({
  title,
  description,
  idPrefix,
  toggles,
  error,
  onToggle,
}: {
  title: string;
  description: string;
  idPrefix: string;
  toggles: NotificationTypeToggles;
  error?: string;
  onToggle: (key: ConfigurableNotificationType, value: boolean) => void;
}) {
  return (
    <div className="space-y-3">
      <div>
        <Label>{title}</Label>
        <p className="mt-0.5 text-[12px] text-muted-foreground">{description}</p>
      </div>
      <div className="grid grid-cols-2 gap-x-4 gap-y-3">
        {NOTIFICATION_TYPE_OPTIONS.map((option) => (
          <NotificationTypeRow
            key={option.key}
            option={option}
            checkboxId={`${idPrefix}-${option.key}`}
            checked={toggles[option.key]}
            onToggle={(value) => onToggle(option.key, value)}
          />
        ))}
      </div>
      {error ? <p className="text-[11px] text-destructive">{error}</p> : null}
    </div>
  );
}

function NotificationTypeRow({
  option,
  checkboxId,
  checked,
  onToggle,
}: {
  option: NotificationTypeOption;
  checkboxId: string;
  checked: boolean;
  onToggle: (value: boolean) => void;
}) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <Checkbox
        id={checkboxId}
        data-testid={checkboxId}
        checked={checked}
        onChange={(event) => onToggle(event.target.checked)}
      />
      <div className="flex min-w-0 items-center gap-1">
        <Label htmlFor={checkboxId} className="text-[13px] font-medium leading-snug text-foreground">
          {option.label}
        </Label>
        <Tooltip>
          <TooltipTrigger asChild>
            <button
              type="button"
              className="shrink-0 rounded-sm text-muted-foreground hover:text-foreground"
              aria-label={`About ${option.label}`}
            >
              <Info className="size-3.5" aria-hidden />
            </button>
          </TooltipTrigger>
          <TooltipContent side="top" className="max-w-xs">
            {option.description}
          </TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}
