import { useState } from "react";

import { LoadingButton } from "@/components/ui/loading-button";
import { NOTIFICATION_TYPE_OPTIONS, eventTypesFromToggles } from "@/lib/notificationSettings";
import { showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { BellOff } from "lucide-react";

import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";
import { FactorySettingsNotificationWorkspacePicker } from "../FactorySettingsNotificationWorkspacePicker";
import type { AccountRedesignNotifications } from "./accountProfileRedesignMocks";
import { SettingsToggleRow } from "./accountProfileRedesignParts";

export function AccountNotificationsRedesignPage({
  email,
  workspaces,
  notifications,
  onChange,
  onSave,
}: {
  email: string;
  workspaces: Array<{ id: string; name: string }>;
  notifications: AccountRedesignNotifications;
  onChange: (notifications: AccountRedesignNotifications) => void;
  onSave: () => void | Promise<void>;
}) {
  const [saved, setSaved] = useState(notifications);
  const [workspaceError, setWorkspaceError] = useState("");
  const [eventError, setEventError] = useState("");

  const isDirty = JSON.stringify(notifications) !== JSON.stringify(saved);
  const saveError = notificationSaveError(notifications);

  const handleSave = async () => {
    if (saveError.workspace) {
      setWorkspaceError(saveError.workspace);
      return;
    }
    if (saveError.events) {
      setEventError(saveError.events);
      return;
    }
    await onSave();
    setSaved(notifications);
    showSuccessToast("Notification settings saved.");
  };

  return (
    <FactorySettingsPageFrame
      title="Notifications"
      subtitle="Choose which task emails SuperPlane sends you."
      actions={
        <LoadingButton
          disabled={!isDirty}
          onClick={() => void handleSave()}
          data-testid="account-redesign-notifications-save"
        >
          Save
        </LoadingButton>
      }
    >
      <div className="space-y-6" data-testid="account-redesign-notifications">
        <EmailCard
          email={email}
          notifications={notifications}
          onChange={(next) => {
            setWorkspaceError("");
            setEventError("");
            onChange(next);
          }}
        />
        {notifications.emailEnabled ? (
          <EnabledNotificationSections
            workspaces={workspaces}
            notifications={notifications}
            eventError={eventError}
            workspaceError={workspaceError}
            onChange={onChange}
            onEventChange={() => setEventError("")}
            onWorkspaceChange={() => setWorkspaceError("")}
          />
        ) : null}
      </div>
    </FactorySettingsPageFrame>
  );
}

function EmailCard({
  email,
  notifications,
  onChange,
}: {
  email: string;
  notifications: AccountRedesignNotifications;
  onChange: (notifications: AccountRedesignNotifications) => void;
}) {
  return (
    <FactorySettingsCard title="Email" data-testid="account-redesign-notifications-email">
      <SettingsToggleRow
        title="Send task emails"
        description={`SuperPlane sends mail to ${email}. SuperPlane does not send mail for your own actions.`}
        checked={notifications.emailEnabled}
        onCheckedChange={(emailEnabled) => onChange({ ...notifications, emailEnabled })}
        testId="account-redesign-notifications-email-toggle"
      />
      {!notifications.emailEnabled ? (
        <div
          className="mt-1 flex items-center gap-2 rounded-md border border-border bg-muted/40 px-3 py-2 text-[13px] text-muted-foreground"
          data-testid="account-redesign-notifications-off"
        >
          <BellOff className="size-4 shrink-0" aria-hidden />
          <span>Task emails are off.</span>
        </div>
      ) : null}
    </FactorySettingsCard>
  );
}

function EnabledNotificationSections({
  workspaces,
  notifications,
  eventError,
  workspaceError,
  onChange,
  onEventChange,
  onWorkspaceChange,
}: {
  workspaces: Array<{ id: string; name: string }>;
  notifications: AccountRedesignNotifications;
  eventError: string;
  workspaceError: string;
  onChange: (notifications: AccountRedesignNotifications) => void;
  onEventChange: () => void;
  onWorkspaceChange: () => void;
}) {
  return (
    <>
      <FactorySettingsCard title="Events" data-testid="account-redesign-notifications-events">
        <p className="text-[12px] text-muted-foreground">Choose which events send an email.</p>
        <div className="mt-1 divide-y divide-border">
          {NOTIFICATION_TYPE_OPTIONS.map((option) => (
            <SettingsToggleRow
              key={option.key}
              title={option.label}
              description={option.description}
              checked={notifications.events[option.key]}
              onCheckedChange={(checked) => {
                onEventChange();
                onChange({
                  ...notifications,
                  events: { ...notifications.events, [option.key]: checked },
                });
              }}
              testId={`account-redesign-notifications-event-${option.key}`}
            />
          ))}
        </div>
        {eventError ? <p className="mt-2 text-[11px] text-destructive">{eventError}</p> : null}
      </FactorySettingsCard>

      <FactorySettingsCard title="Workspaces" data-testid="account-redesign-notifications-workspaces">
        <p className="text-[12px] text-muted-foreground">Choose which workspaces can send you email.</p>
        <div className="mt-3 space-y-2" role="radiogroup" aria-label="Workspaces">
          <WorkspaceScopeChoice
            id="account-redesign-notifications-scope-all"
            label="All workspaces"
            description="Every workspace you can access."
            checked={notifications.workspaceScope === "all"}
            onSelect={() => {
              onWorkspaceChange();
              onChange({ ...notifications, workspaceScope: "all" });
            }}
          />
          <WorkspaceScopeChoice
            id="account-redesign-notifications-scope-selected"
            label="Selected workspaces"
            description="Only the workspaces you pick."
            checked={notifications.workspaceScope === "selected"}
            onSelect={() => {
              onWorkspaceChange();
              onChange({ ...notifications, workspaceScope: "selected" });
            }}
          />
        </div>
        {notifications.workspaceScope === "selected" ? (
          <div className="mt-4">
            <FactorySettingsNotificationWorkspacePicker
              factories={workspaces}
              selectedFactoryIds={notifications.workspaceIds}
              onAdd={(workspaceId) => {
                onWorkspaceChange();
                onChange({
                  ...notifications,
                  workspaceIds: [...notifications.workspaceIds, workspaceId],
                });
              }}
              onRemove={(workspaceId) => {
                onWorkspaceChange();
                onChange({
                  ...notifications,
                  workspaceIds: notifications.workspaceIds.filter((id) => id !== workspaceId),
                });
              }}
            />
          </div>
        ) : null}
        {workspaceError ? <p className="mt-2 text-[11px] text-destructive">{workspaceError}</p> : null}
      </FactorySettingsCard>
    </>
  );
}

function notificationSaveError(notifications: AccountRedesignNotifications): {
  workspace?: string;
  events?: string;
} {
  if (!notifications.emailEnabled) {
    return {};
  }
  if (notifications.workspaceScope === "selected" && notifications.workspaceIds.length === 0) {
    return { workspace: "Select at least one workspace." };
  }
  if (eventTypesFromToggles(notifications.events).length === 0) {
    return { events: "Select at least one event, or turn task emails off." };
  }
  return {};
}

function WorkspaceScopeChoice({
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
        "flex w-full items-start gap-3 rounded-md border px-3 py-2.5 text-left",
        checked ? "border-foreground/25 bg-muted/40" : "border-border hover:bg-muted/20",
      )}
    >
      <span
        className={cn(
          "mt-0.5 flex size-3.5 shrink-0 items-center justify-center rounded-full border",
          checked ? "border-foreground" : "border-muted-foreground/40",
        )}
        aria-hidden
      >
        {checked ? <span className="size-1.5 rounded-full bg-foreground" /> : null}
      </span>
      <span>
        <span className="block text-[13px] font-medium text-foreground">{label}</span>
        <span className="mt-0.5 block text-[12px] text-muted-foreground">{description}</span>
      </span>
    </button>
  );
}
