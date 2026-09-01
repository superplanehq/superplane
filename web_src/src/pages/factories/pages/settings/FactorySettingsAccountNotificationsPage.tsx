import { useEffect, useState } from "react";

import { useAccount } from "@/contexts/useAccount";
import { useFactories } from "@/hooks/useFactoryData";
import { useNotificationSettings, useUpdateNotificationSettings } from "@/hooks/useNotificationSettings";
import {
  accountNotificationsFromSettings,
  settingsFromAccountNotifications,
  type AccountNotificationForm,
} from "@/lib/notificationSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";

import { AccountNotificationsRedesignPage } from "./account-profile-redesign/AccountNotificationsRedesignPage";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function FactorySettingsAccountNotificationsPage() {
  const { organizationId } = useFactorySettingsLayout();
  const { account } = useAccount();
  const { data: settings, isLoading } = useNotificationSettings(organizationId);
  const { data: factories = [] } = useFactories(organizationId);
  const updateSettings = useUpdateNotificationSettings(organizationId);
  const [form, setForm] = useState<AccountNotificationForm>(() => accountNotificationsFromSettings(settings));

  useEffect(() => {
    setForm(accountNotificationsFromSettings(settings));
  }, [settings]);

  if (!account || isLoading) {
    return <p className="text-[13px] text-muted-foreground">Loading notifications…</p>;
  }

  return (
    <AccountNotificationsRedesignPage
      email={account.email}
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
