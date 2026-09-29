import { useState } from "react";

import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { currentBrowserNotificationPermission } from "@/lib/browserNotifications";
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
  const [browserWorkspaceError, setBrowserWorkspaceError] = useState("");
  const [browserEventError, setBrowserEventError] = useState("");
  const [permission, setPermission] = useState(currentBrowserNotificationPermission);

  const isDirty = JSON.stringify(notifications) !== JSON.stringify(saved);
  const emailSaveError = notificationSaveError(notifications, "email");
  const browserSaveError = notificationSaveError(notifications, "browser");

  const handleSave = async () => {
    if (emailSaveError.workspace) {
      setWorkspaceError(emailSaveError.workspace);
      return;
    }
    if (emailSaveError.events) {
      setEventError(emailSaveError.events);
      return;
    }
    if (browserSaveError.workspace) {
      setBrowserWorkspaceError(browserSaveError.workspace);
      return;
    }
    if (browserSaveError.events) {
      setBrowserEventError(browserSaveError.events);
      return;
    }
    await onSave();
    setSaved(notifications);
    showSuccessToast("Notification settings saved.");
  };

  return (
    <FactorySettingsPageFrame
      title="Notifications"
      subtitle="Choose which task emails and browser alerts SuperPlane sends you."
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
        <BrowserCard
          notifications={notifications}
          permission={permission}
          onChange={async (next) => {
            setBrowserWorkspaceError("");
            setBrowserEventError("");
            if (next.browserEnabled && !notifications.browserEnabled) {
              setPermission(await requestBrowserPermission());
            } else {
              setPermission(currentBrowserNotificationPermission());
            }
            onChange(next);
          }}
          onAllow={async () => {
            setPermission(await requestBrowserPermission());
          }}
        />
        {notifications.browserEnabled ? (
          <EnabledBrowserSections
            workspaces={workspaces}
            notifications={notifications}
            eventError={browserEventError}
            workspaceError={browserWorkspaceError}
            onChange={onChange}
            onEventChange={() => setBrowserEventError("")}
            onWorkspaceChange={() => setBrowserWorkspaceError("")}
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

function BrowserCard({
  notifications,
  permission,
  onChange,
  onAllow,
}: {
  notifications: AccountRedesignNotifications;
  permission: ReturnType<typeof currentBrowserNotificationPermission>;
  onChange: (notifications: AccountRedesignNotifications) => void | Promise<void>;
  onAllow: () => void | Promise<void>;
}) {
  return (
    <FactorySettingsCard title="Browser" data-testid="account-redesign-notifications-browser">
      <SettingsToggleRow
        title="Show browser notifications"
        description="SuperPlane shows an alert in this browser while SuperPlane is open. SuperPlane does not show an alert for your own actions."
        checked={notifications.browserEnabled}
        onCheckedChange={(browserEnabled) => void onChange({ ...notifications, browserEnabled })}
        testId="account-redesign-notifications-browser-toggle"
      />
      {!notifications.browserEnabled ? (
        <div
          className="mt-1 flex items-center gap-2 rounded-md border border-border bg-muted/40 px-3 py-2 text-[13px] text-muted-foreground"
          data-testid="account-redesign-notifications-browser-off"
        >
          <BellOff className="size-4 shrink-0" aria-hidden />
          <span>Browser notifications are off.</span>
        </div>
      ) : null}
      {notifications.browserEnabled && permission === "default" ? (
        <div
          className="mt-2 flex flex-col gap-2 rounded-md border border-border bg-muted/40 px-3 py-2 sm:flex-row sm:items-center sm:justify-between"
          data-testid="account-redesign-notifications-browser-allow"
        >
          <p className="text-[12px] text-muted-foreground">
            This browser needs permission before SuperPlane can show alerts.
          </p>
          <Button type="button" size="sm" variant="outline" onClick={() => void onAllow()}>
            Allow notifications
          </Button>
        </div>
      ) : null}
      {notifications.browserEnabled && permission !== "granted" && permission !== "default" ? (
        <p
          className="mt-2 text-[12px] text-muted-foreground"
          data-testid="account-redesign-notifications-browser-blocked"
        >
          This browser blocked notifications. Allow notifications for this site, then turn this setting on again.
        </p>
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
    <ChannelChoiceSections
      testIdPrefix="account-redesign-notifications"
      eventsHelp="Choose which events send an email."
      workspacesHelp="Choose which workspaces can send you email."
      events={notifications.events}
      workspaceScope={notifications.workspaceScope}
      workspaceIds={notifications.workspaceIds}
      workspaces={workspaces}
      eventError={eventError}
      workspaceError={workspaceError}
      onEventToggle={(key, checked) => {
        onEventChange();
        onChange({
          ...notifications,
          events: { ...notifications.events, [key]: checked },
        });
      }}
      onWorkspaceScopeChange={(workspaceScope) => {
        onWorkspaceChange();
        onChange({ ...notifications, workspaceScope });
      }}
      onAddWorkspace={(workspaceId) => {
        onWorkspaceChange();
        onChange({
          ...notifications,
          workspaceIds: [...notifications.workspaceIds, workspaceId],
        });
      }}
      onRemoveWorkspace={(workspaceId) => {
        onWorkspaceChange();
        onChange({
          ...notifications,
          workspaceIds: notifications.workspaceIds.filter((id) => id !== workspaceId),
        });
      }}
    />
  );
}

function EnabledBrowserSections({
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
      <ChannelChoiceSections
        testIdPrefix="account-redesign-notifications-browser"
        eventsHelp="Choose which events show a browser alert."
        workspacesHelp="Choose which workspaces can show a browser alert."
        events={notifications.browserEvents}
        workspaceScope={notifications.browserWorkspaceScope}
        workspaceIds={notifications.browserWorkspaceIds}
        workspaces={workspaces}
        eventError={eventError}
        workspaceError={workspaceError}
        onEventToggle={(key, checked) => {
          onEventChange();
          onChange({
            ...notifications,
            browserEvents: { ...notifications.browserEvents, [key]: checked },
          });
        }}
        onWorkspaceScopeChange={(workspaceScope) => {
          onWorkspaceChange();
          onChange({ ...notifications, browserWorkspaceScope: workspaceScope });
        }}
        onAddWorkspace={(workspaceId) => {
          onWorkspaceChange();
          onChange({
            ...notifications,
            browserWorkspaceIds: [...notifications.browserWorkspaceIds, workspaceId],
          });
        }}
        onRemoveWorkspace={(workspaceId) => {
          onWorkspaceChange();
          onChange({
            ...notifications,
            browserWorkspaceIds: notifications.browserWorkspaceIds.filter((id) => id !== workspaceId),
          });
        }}
      />
      <FactorySettingsCard title="Source board" data-testid="account-redesign-notifications-browser-board">
        <SettingsToggleRow
          title="Show alerts on the source board"
          description="Show an alert when you look at the board that the event came from."
          checked={notifications.browserShowWhileViewing}
          onCheckedChange={(browserShowWhileViewing) => onChange({ ...notifications, browserShowWhileViewing })}
          testId="account-redesign-notifications-browser-while-viewing"
        />
      </FactorySettingsCard>
    </>
  );
}

function ChannelChoiceSections({
  testIdPrefix,
  eventsHelp,
  workspacesHelp,
  events,
  workspaceScope,
  workspaceIds,
  workspaces,
  eventError,
  workspaceError,
  onEventToggle,
  onWorkspaceScopeChange,
  onAddWorkspace,
  onRemoveWorkspace,
}: {
  testIdPrefix: string;
  eventsHelp: string;
  workspacesHelp: string;
  events: AccountRedesignNotifications["events"];
  workspaceScope: AccountRedesignNotifications["workspaceScope"];
  workspaceIds: string[];
  workspaces: Array<{ id: string; name: string }>;
  eventError: string;
  workspaceError: string;
  onEventToggle: (key: keyof AccountRedesignNotifications["events"], checked: boolean) => void;
  onWorkspaceScopeChange: (scope: AccountRedesignNotifications["workspaceScope"]) => void;
  onAddWorkspace: (workspaceId: string) => void;
  onRemoveWorkspace: (workspaceId: string) => void;
}) {
  return (
    <>
      <FactorySettingsCard title="Events" data-testid={`${testIdPrefix}-events`}>
        <p className="text-[12px] text-muted-foreground">{eventsHelp}</p>
        <div className="mt-1 divide-y divide-border">
          {NOTIFICATION_TYPE_OPTIONS.map((option) => (
            <SettingsToggleRow
              key={option.key}
              title={option.label}
              description={option.description}
              checked={events[option.key]}
              onCheckedChange={(checked) => onEventToggle(option.key, checked)}
              testId={`${testIdPrefix}-event-${option.key}`}
            />
          ))}
        </div>
        {eventError ? <p className="mt-2 text-[11px] text-destructive">{eventError}</p> : null}
      </FactorySettingsCard>

      <FactorySettingsCard title="Workspaces" data-testid={`${testIdPrefix}-workspaces`}>
        <p className="text-[12px] text-muted-foreground">{workspacesHelp}</p>
        <div className="mt-3 space-y-2" role="radiogroup" aria-label="Workspaces">
          <WorkspaceScopeChoice
            id={`${testIdPrefix}-scope-all`}
            label="All workspaces"
            description="Every workspace you can access."
            checked={workspaceScope === "all"}
            onSelect={() => onWorkspaceScopeChange("all")}
          />
          <WorkspaceScopeChoice
            id={`${testIdPrefix}-scope-selected`}
            label="Selected workspaces"
            description="Only the workspaces you pick."
            checked={workspaceScope === "selected"}
            onSelect={() => onWorkspaceScopeChange("selected")}
          />
        </div>
        {workspaceScope === "selected" ? (
          <div className="mt-4">
            <FactorySettingsNotificationWorkspacePicker
              factories={workspaces}
              selectedFactoryIds={workspaceIds}
              onAdd={onAddWorkspace}
              onRemove={onRemoveWorkspace}
            />
          </div>
        ) : null}
        {workspaceError ? <p className="mt-2 text-[11px] text-destructive">{workspaceError}</p> : null}
      </FactorySettingsCard>
    </>
  );
}

function notificationSaveError(
  notifications: AccountRedesignNotifications,
  channel: "email" | "browser",
): {
  workspace?: string;
  events?: string;
} {
  const enabled = channel === "email" ? notifications.emailEnabled : notifications.browserEnabled;
  const workspaceScope = channel === "email" ? notifications.workspaceScope : notifications.browserWorkspaceScope;
  const workspaceIds = channel === "email" ? notifications.workspaceIds : notifications.browserWorkspaceIds;
  const events = channel === "email" ? notifications.events : notifications.browserEvents;
  if (!enabled) {
    return {};
  }
  if (workspaceScope === "selected" && workspaceIds.length === 0) {
    return { workspace: "Select at least one workspace." };
  }
  if (eventTypesFromToggles(events).length === 0) {
    return {
      events:
        channel === "email"
          ? "Select at least one event, or turn task emails off."
          : "Select at least one event, or turn browser notifications off.",
    };
  }
  return {};
}

async function requestBrowserPermission(): Promise<ReturnType<typeof currentBrowserNotificationPermission>> {
  if (typeof Notification === "undefined") {
    return "unsupported";
  }
  await Notification.requestPermission();
  return currentBrowserNotificationPermission();
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
