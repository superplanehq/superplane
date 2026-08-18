import type { MeNotificationSettings, NotificationSettingsWorkspaceScope } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactories } from "@/hooks/useFactoryData";
import { useNotificationSettings, useUpdateNotificationSettings } from "@/hooks/useNotificationSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { Switch } from "@/ui/switch";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/ui/tooltip";
import { Info } from "lucide-react";
import { useEffect, useState } from "react";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { FactorySettingsNotificationWorkspacePicker } from "./FactorySettingsNotificationWorkspacePicker";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

interface NotificationTypeOption {
  key: keyof NotificationTypeToggles;
  label: string;
  description: string;
}

interface NotificationTypeToggles {
  workOrderAssigned: boolean;
  workOrderCommentOwned: boolean;
  workOrderCommentCreated: boolean;
  workOrderStatusOwned: boolean;
  workOrderArtifactOwned: boolean;
  workOrderMentioned: boolean;
}

const NOTIFICATION_TYPE_OPTIONS: NotificationTypeOption[] = [
  {
    key: "workOrderAssigned",
    label: "Added as owner",
    description: "You become an owner of a work order.",
  },
  {
    key: "workOrderCommentOwned",
    label: "Comments you own",
    description: "Someone comments on a work order you own.",
  },
  {
    key: "workOrderCommentCreated",
    label: "Comments you created",
    description: "Someone comments on a work order you created.",
  },
  {
    key: "workOrderStatusOwned",
    label: "Status changes",
    description: "A work order you own or created opens, closes, or moves back to draft.",
  },
  {
    key: "workOrderArtifactOwned",
    label: "New artifacts",
    description: "An artifact is added to a work order you own.",
  },
  {
    key: "workOrderMentioned",
    label: "Mentions",
    description: "Someone mentions you in a work order comment.",
  },
];

interface NotificationsFormState {
  enabled: boolean;
  scope: "all" | "selected";
  selectedFactoryIds: string[];
  toggles: NotificationTypeToggles;
}

function formStateFromSettings(settings: MeNotificationSettings | undefined): NotificationsFormState {
  return {
    enabled: settings?.enabled ?? true,
    scope: settings?.workspaceScope === "WORKSPACE_SCOPE_SELECTED" ? "selected" : "all",
    selectedFactoryIds: settings?.factoryIds ?? [],
    toggles: {
      workOrderAssigned: settings?.workOrderAssigned ?? true,
      workOrderCommentOwned: settings?.workOrderCommentOwned ?? true,
      workOrderCommentCreated: settings?.workOrderCommentCreated ?? true,
      workOrderStatusOwned: settings?.workOrderStatusOwned ?? true,
      workOrderArtifactOwned: settings?.workOrderArtifactOwned ?? true,
      workOrderMentioned: settings?.workOrderMentioned ?? true,
    },
  };
}

function settingsFromFormState(state: NotificationsFormState): MeNotificationSettings {
  const scope: NotificationSettingsWorkspaceScope =
    state.scope === "selected" ? "WORKSPACE_SCOPE_SELECTED" : "WORKSPACE_SCOPE_ALL";

  return {
    enabled: state.enabled,
    workspaceScope: scope,
    factoryIds: state.scope === "selected" ? state.selectedFactoryIds : [],
    ...state.toggles,
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

  useEffect(() => {
    const next = formStateFromSettings(settings);
    setForm(next);
    setSavedForm(next);
    setScopeError("");
  }, [settings]);

  const isDirty = isDirtyState(form, savedForm);

  const handleSave = async () => {
    if (!canUpdate) {
      return;
    }
    if (form.enabled && form.scope === "selected" && form.selectedFactoryIds.length === 0) {
      setScopeError("Select at least one workspace, or use all workspaces.");
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
          <div className="flex items-center gap-2.5">
            <Switch
              id="notifications-enabled"
              checked={form.enabled}
              disabled={isLoading || !canUpdate}
              aria-label="Email notifications"
              data-testid="notifications-enabled"
              onCheckedChange={(enabled) => setForm((prev) => ({ ...prev, enabled }))}
            />
            <Label htmlFor="notifications-enabled" className="text-[13px] font-medium text-foreground">
              Email notifications
            </Label>
          </div>
          <div className={cn("mt-6 space-y-6", (!form.enabled || !canUpdate) && "pointer-events-none opacity-50")}>
            <WorkspaceScopeSection
              scope={form.scope}
              selectedFactoryIds={form.selectedFactoryIds}
              factories={factories}
              scopeError={scopeError}
              onScopeChange={(scope) => {
                setScopeError("");
                setForm((prev) => ({ ...prev, scope }));
              }}
              onAddFactory={(factoryId) => {
                setScopeError("");
                setForm((prev) =>
                  prev.selectedFactoryIds.includes(factoryId)
                    ? prev
                    : { ...prev, selectedFactoryIds: [...prev.selectedFactoryIds, factoryId] },
                );
              }}
              onRemoveFactory={(factoryId) => {
                setScopeError("");
                setForm((prev) => ({
                  ...prev,
                  selectedFactoryIds: prev.selectedFactoryIds.filter((id) => id !== factoryId),
                }));
              }}
            />
            <NotificationTypesSection
              toggles={form.toggles}
              onToggle={(key, value) => setForm((prev) => ({ ...prev, toggles: { ...prev.toggles, [key]: value } }))}
            />
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
  scope: "all" | "selected";
  selectedFactoryIds: string[];
  factories: { id?: string; name?: string }[];
  scopeError: string;
  onScopeChange: (scope: "all" | "selected") => void;
  onAddFactory: (factoryId: string) => void;
  onRemoveFactory: (factoryId: string) => void;
}

function WorkspaceScopeSection({
  scope,
  selectedFactoryIds,
  factories,
  scopeError,
  onScopeChange,
  onAddFactory,
  onRemoveFactory,
}: WorkspaceScopeSectionProps) {
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
          id="notifications-scope-selected"
          label="Selected workspaces"
          checked={scope === "selected"}
          onSelect={() => onScopeChange("selected")}
        />
      </div>
      {scope === "selected" ? (
        <FactorySettingsNotificationWorkspacePicker
          factories={factories}
          selectedFactoryIds={selectedFactoryIds}
          onAdd={onAddFactory}
          onRemove={onRemoveFactory}
        />
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

function NotificationTypesSection({
  toggles,
  onToggle,
}: {
  toggles: NotificationTypeToggles;
  onToggle: (key: keyof NotificationTypeToggles, value: boolean) => void;
}) {
  return (
    <div className="space-y-3">
      <Label>Notify me about</Label>
      <div className="grid grid-cols-2 gap-x-4 gap-y-3">
        {NOTIFICATION_TYPE_OPTIONS.map((option) => (
          <NotificationTypeRow
            key={option.key}
            option={option}
            checked={toggles[option.key]}
            onToggle={(value) => onToggle(option.key, value)}
          />
        ))}
      </div>
    </div>
  );
}

function NotificationTypeRow({
  option,
  checked,
  onToggle,
}: {
  option: NotificationTypeOption;
  checked: boolean;
  onToggle: (value: boolean) => void;
}) {
  const checkboxId = `notifications-type-${option.key}`;

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
