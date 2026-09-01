import { useState } from "react";

import { useAccount } from "@/contexts/useAccount";
import { useFactories } from "@/hooks/useFactoryData";
import { useNotificationSettings, useUpdateNotificationSettings } from "@/hooks/useNotificationSettings";
import { getApiErrorMessage } from "@/lib/errors";
import {
  accountNotificationsFromSettings,
  settingsFromAccountNotifications,
  type AccountNotificationForm,
} from "@/lib/notificationSettings";
import { showErrorToast } from "@/lib/toast";
import type { MeNotificationSettings } from "@/api-client";

import { AccountNotificationsRedesignPage } from "./account-profile-redesign/AccountNotificationsRedesignPage";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function FactorySettingsAccountNotificationsPage() {
  const { organizationId } = useFactorySettingsLayout();
  const { account } = useAccount();
  const { data: settings, isPending, isError } = useNotificationSettings(organizationId);

  if (!account || isPending) {
    return <p className="text-[13px] text-muted-foreground">Loading notifications…</p>;
  }
  if (isError || settings === undefined) {
    return <p className="text-[13px] text-muted-foreground">Failed to load notification settings.</p>;
  }

  return (
    <LoadedAccountNotifications accountEmail={account.email} organizationId={organizationId} settings={settings} />
  );
}

function LoadedAccountNotifications({
  accountEmail,
  organizationId,
  settings,
}: {
  accountEmail: string;
  organizationId: string;
  settings: MeNotificationSettings;
}) {
  const { data: factories = [] } = useFactories(organizationId);
  const updateSettings = useUpdateNotificationSettings(organizationId);
  const [form, setForm] = useState<AccountNotificationForm>(() => accountNotificationsFromSettings(settings));

  return (
    <AccountNotificationsRedesignPage
      email={accountEmail}
      workspaces={factories.flatMap((factory) =>
        factory.id && factory.name ? [{ id: factory.id, name: factory.name }] : [],
      )}
      notifications={form}
      onChange={setForm}
      onSave={async () => {
        try {
          await updateSettings.mutateAsync(settingsFromAccountNotifications(form));
        } catch (error) {
          showErrorToast(getApiErrorMessage(error, "Failed to save notification settings."));
          throw error;
        }
      }}
    />
  );
}
