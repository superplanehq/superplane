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
  togglesFromAllScopeEventTypes,
  togglesFromEventTypes,
  workspaceScopeFromSettings,
  type ConfigurableNotificationType,
  type NotificationTypeToggles,
  type WorkspaceScopeForm,
} from "@/lib/notificationSettings";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";
import { Info } from "lucide-react";
import { useEffect, useState } from "react";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { FactorySettingsNotificationWorkspacePicker } from "./FactorySettingsNotificationWorkspacePicker";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

interface NotificationTypeOption {
  key: ConfigurableNotificationType;
  label: string;
  description: string;
}

const NOTIFICATION_TYPE_OPTIONS: NotificationTypeOption[] = [
  {
    key: "TYPE_WORK_ORDER_ASSIGNED",
    label: "Added as owner",
    description: "You become an owner of a work order.",
  },
  {
    key: "TYPE_WORK_ORDER_COMMENT_OWNED",
    label: "Comments you own",
    description: "Someone comments on a work order you own.",
  },
  {
    key: "TYPE_WORK_ORDER_COMMENT_CREATED",
    label: "Comments you created",
    description: "Someone comments on a work order you created.",
  },
  {
    key: "TYPE_WORK_ORDER_STATUS_OWNED",
    label: "Status changes",
    description: "A work order you own or created opens, closes, or moves back to draft.",
  },
  {
    key: "TYPE_WORK_ORDER_ARTIFACT_OWNED",
    label: "New artifacts",
    description: "An artifact is added to a work order you own.",
  },
];

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
    if (form.scope === "filtered" && form.filters.length === 0) {
      setScopeError("Select at least one workspace, or use all workspaces.");
      return;
    }
    if (form.scope === "all" && eventTypesFromToggles(form.toggles).length === 0) {
      setTypeError("Select at least one event type, or choose none.");
      return;
    }
    if (
      form.scope === "filtered" &&
      form.filters.some((filter) => eventTypesFromToggles(filter.toggles).length === 0)
    ) {
      setTypeError("Select at least one event type for each workspace.");
      return;
    }
    try {
      const saved = await updateSettings.mutateAsync(settingsFromFormState(form));
      const next = formStateFromSettings(saved);
      setForm(next);
      setSavedForm(next);
      showSuccessToast("Notification settings saved.");
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "Failed to save notification settings"));
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
                setForm((prev) =>
                  prev.filters.some((filter) => filter.workspaceId === workspaceId)
                    ? prev
                    : {
                        ...prev,
                        filters: [...prev.filters, { workspaceId, toggles: defaultNotificationTypeToggles() }],
                      },
                );
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
                setForm((prev) => ({
                  ...prev,
                  filters: prev.filters.map((filter) =>
                    filter.workspaceId === workspaceId
                      ? { ...filter, toggles: { ...filter.toggles, [key]: value } }
                      : filter,
                  ),
                }));
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
        <Label>Workspaces</Label>
        <p className="mt-0.5 text-[12px] text-muted-foreground">Choose which workspaces send you notifications.</p>
      </div>
      <div className="inline-flex rounded-md border border-border p-0.5" role="radiogroup" aria-label="Workspace scope">
        <ScopeChoice
          id="notifications-scope-all"
          label="All workspaces"
          checked={scope === "all"}
          onSelect={() => onScopeChange("all")}
        />
        <ScopeChoice
          id="notifications-scope-filtered"
          label="Filtered"
          checked={scope === "filtered"}
          onSelect={() => onScopeChange("filtered")}
        />
        <ScopeChoice
          id="notifications-scope-none"
          label="None"
          checked={scope === "none"}
          onSelect={() => onScopeChange("none")}
        />
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
              error={
                typeError && eventTypesFromToggles(filter.toggles).length === 0 ? typeError : undefined
              }
              onToggle={(key, value) => onToggleType(filter.workspaceId, key, value)}
            />
          ))}
        </div>
      ) : null}
      {scopeError ? <p className="text-[11px] text-destructive">{scopeError}</p> : null}
    </div>
  );
}

function ScopeChoice({
  id,
  label,
  checked,
  onSelect,
}: {
  id: string;
  label: string;
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
      className={cn(
        "rounded-[5px] px-3 py-1.5 text-[13px] tracking-[-0.01em] text-muted-foreground hover:text-foreground",
        checked && "bg-accent font-medium text-foreground",
      )}
      onClick={onSelect}
    >
      {label}
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
