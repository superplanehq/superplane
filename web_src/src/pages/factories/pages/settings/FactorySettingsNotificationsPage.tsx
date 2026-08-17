import type { FactoriesNotificationSettings, NotificationSettingsWorkspaceScope } from "@/api-client";
import { Checkbox } from "@/components/ui/checkbox";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "@/lib/utils";
import { useFactories } from "@/hooks/useFactoryData";
import { useNotificationSettings, useUpdateNotificationSettings } from "@/hooks/useNotificationSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { useEffect, useState } from "react";
import { WorkspacePageHeader } from "../../layout/WorkspacePageHeader";
import { factoryCardClassName, factoryContentBodyClassName } from "../factoryPageLayoutStyles";
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
}

const NOTIFICATION_TYPE_OPTIONS: NotificationTypeOption[] = [
  {
    key: "workOrderAssigned",
    label: "Added as owner",
    description: "You become an owner of a work order.",
  },
  {
    key: "workOrderCommentOwned",
    label: "Comments on work orders you own",
    description: "Someone comments on a work order you own.",
  },
  {
    key: "workOrderCommentCreated",
    label: "Comments on work orders you created",
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
];

interface NotificationsFormState {
  enabled: boolean;
  scope: "all" | "selected";
  selectedFactoryIds: string[];
  toggles: NotificationTypeToggles;
}

function formStateFromSettings(settings: FactoriesNotificationSettings | undefined): NotificationsFormState {
  return {
    enabled: settings?.enabled ?? false,
    scope: settings?.workspaceScope === "WORKSPACE_SCOPE_SELECTED" ? "selected" : "all",
    selectedFactoryIds: settings?.factoryIds ?? [],
    toggles: {
      workOrderAssigned: settings?.workOrderAssigned ?? true,
      workOrderCommentOwned: settings?.workOrderCommentOwned ?? true,
      workOrderCommentCreated: settings?.workOrderCommentCreated ?? true,
      workOrderStatusOwned: settings?.workOrderStatusOwned ?? true,
      workOrderArtifactOwned: settings?.workOrderArtifactOwned ?? true,
    },
  };
}

function settingsFromFormState(state: NotificationsFormState): FactoriesNotificationSettings {
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
  const { data: settings, isLoading } = useNotificationSettings(organizationId);
  const { data: factories = [] } = useFactories(organizationId);
  const updateSettings = useUpdateNotificationSettings(organizationId);

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
    <>
      <WorkspacePageHeader
        title="Notifications"
        subtitle="Choose which work order emails you receive. These settings apply to you only."
      />

      <div className={factoryContentBodyClassName}>
        <div className="max-w-2xl space-y-6">
          <section className={cn("p-6", factoryCardClassName)} data-testid="factory-settings-notifications-form">
            <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Email notifications</h2>
            <div className="mt-4 space-y-4">
              <label className="flex items-start gap-3" htmlFor="notifications-enabled">
                <Checkbox
                  id="notifications-enabled"
                  data-testid="notifications-enabled"
                  checked={form.enabled}
                  disabled={isLoading}
                  onChange={(event) => {
                    const enabled = event.target.checked;
                    setForm((prev) => ({ ...prev, enabled }));
                  }}
                />
                <span className="space-y-0.5">
                  <span className="block text-[13px] font-medium text-foreground">Send me email notifications</span>
                  <span className="block text-[12px] text-muted-foreground">
                    Get an email when work order activity involves you. You never get an email about your own actions.
                  </span>
                </span>
              </label>
            </div>

            <div className={cn("mt-6 space-y-6", !form.enabled && "opacity-50")}>
              <WorkspaceScopeSection
                scope={form.scope}
                selectedFactoryIds={form.selectedFactoryIds}
                factories={factories}
                scopeError={scopeError}
                onScopeChange={(scope) => {
                  setScopeError("");
                  setForm((prev) => ({ ...prev, scope }));
                }}
                onToggleFactory={(factoryId, selected) => {
                  setScopeError("");
                  setForm((prev) => ({
                    ...prev,
                    selectedFactoryIds: selected
                      ? [...prev.selectedFactoryIds, factoryId]
                      : prev.selectedFactoryIds.filter((id) => id !== factoryId),
                  }));
                }}
              />

              <NotificationTypesSection
                toggles={form.toggles}
                onToggle={(key, value) => setForm((prev) => ({ ...prev, toggles: { ...prev.toggles, [key]: value } }))}
              />
            </div>

            <div className="mt-6 flex justify-end">
              <LoadingButton
                disabled={isLoading || !isDirty}
                loading={updateSettings.isPending}
                loadingText="Saving..."
                onClick={() => void handleSave()}
                data-testid="notifications-save"
              >
                Save
              </LoadingButton>
            </div>
          </section>
        </div>
      </div>
    </>
  );
}

interface WorkspaceScopeSectionProps {
  scope: "all" | "selected";
  selectedFactoryIds: string[];
  factories: { id?: string; name?: string }[];
  scopeError: string;
  onScopeChange: (scope: "all" | "selected") => void;
  onToggleFactory: (factoryId: string, selected: boolean) => void;
}

function WorkspaceScopeSection({
  scope,
  selectedFactoryIds,
  factories,
  scopeError,
  onScopeChange,
  onToggleFactory,
}: WorkspaceScopeSectionProps) {
  return (
    <div className="space-y-2">
      <Label>Workspaces</Label>
      <p className="text-[12px] text-muted-foreground">Choose which workspaces send you notifications.</p>
      <div className="space-y-2" role="radiogroup" aria-label="Workspace scope">
        <ScopeRadio
          id="notifications-scope-all"
          label="All workspaces"
          checked={scope === "all"}
          onSelect={() => onScopeChange("all")}
        />
        <ScopeRadio
          id="notifications-scope-selected"
          label="Selected workspaces"
          checked={scope === "selected"}
          onSelect={() => onScopeChange("selected")}
        />
      </div>
      {scope === "selected" ? (
        <div className="ml-6 mt-2 space-y-2 rounded-md border border-border bg-background p-3">
          {factories.length === 0 ? (
            <p className="text-[12px] text-muted-foreground">No workspaces available.</p>
          ) : (
            factories.map((factory) => {
              const factoryId = factory.id ?? "";
              return (
                <label
                  key={factoryId}
                  className="flex items-center gap-2"
                  htmlFor={`notifications-factory-${factoryId}`}
                >
                  <Checkbox
                    id={`notifications-factory-${factoryId}`}
                    data-testid={`notifications-factory-${factoryId}`}
                    checked={selectedFactoryIds.includes(factoryId)}
                    onChange={(event) => onToggleFactory(factoryId, event.target.checked)}
                  />
                  <span className="text-[13px] text-foreground">{factory.name}</span>
                </label>
              );
            })
          )}
        </div>
      ) : null}
      {scopeError ? <p className="text-[11px] text-destructive">{scopeError}</p> : null}
    </div>
  );
}

interface ScopeRadioProps {
  id: string;
  label: string;
  checked: boolean;
  onSelect: () => void;
}

function ScopeRadio({ id, label, checked, onSelect }: ScopeRadioProps) {
  return (
    <label className="flex cursor-pointer items-center gap-2" htmlFor={id}>
      <input
        type="radio"
        id={id}
        data-testid={id}
        name="notifications-scope"
        className="size-3.5 accent-foreground"
        checked={checked}
        onChange={onSelect}
      />
      <span className="text-[13px] text-foreground">{label}</span>
    </label>
  );
}

interface NotificationTypesSectionProps {
  toggles: NotificationTypeToggles;
  onToggle: (key: keyof NotificationTypeToggles, value: boolean) => void;
}

function NotificationTypesSection({ toggles, onToggle }: NotificationTypesSectionProps) {
  return (
    <div className="space-y-2">
      <Label>Notify me about</Label>
      <div className="space-y-3">
        {NOTIFICATION_TYPE_OPTIONS.map((option) => (
          <label key={option.key} className="flex items-start gap-3" htmlFor={`notifications-type-${option.key}`}>
            <Checkbox
              id={`notifications-type-${option.key}`}
              data-testid={`notifications-type-${option.key}`}
              checked={toggles[option.key]}
              onChange={(event) => onToggle(option.key, event.target.checked)}
            />
            <span className="space-y-0.5">
              <span className="block text-[13px] font-medium text-foreground">{option.label}</span>
              <span className="block text-[12px] text-muted-foreground">{option.description}</span>
            </span>
          </label>
        ))}
      </div>
    </div>
  );
}
