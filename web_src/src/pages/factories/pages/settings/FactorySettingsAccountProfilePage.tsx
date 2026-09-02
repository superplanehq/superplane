import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { useAccount } from "@/contexts/useAccount";
import { meKeys } from "@/hooks/useMe";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { accountEmailOptions, updateAccountEmail, updateAccountName } from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { AccountProfileRedesignPage } from "./account-profile-redesign/AccountProfileRedesignPage";
import { DeleteAccountDangerZone } from "./DeleteAccountDangerZone";

export function FactorySettingsAccountProfilePage() {
  const { account, refreshAccount } = useAccount();
  const organizationId = useOrganizationId();
  const queryClient = useQueryClient();
  const [name, setName] = useState(account?.name ?? "");

  useEffect(() => {
    if (account?.name) {
      setName(account.name);
    }
  }, [account?.name]);

  if (!account) {
    return <p className="text-[13px] text-muted-foreground">Loading profile…</p>;
  }

  return (
    <AccountProfileRedesignPage
      name={name}
      email={account.email}
      emailOptions={accountEmailOptions({
        email: account.email,
        hasPassword: account.has_password,
        providers: account.providers,
      })}
      userId={account.id}
      onNameChange={setName}
      onEmailChange={async (email) => {
        try {
          await updateAccountEmail(email);
          await refreshAccount();
          showSuccessToast("Email updated.");
          if (organizationId) {
            await queryClient.invalidateQueries({ queryKey: meKeys.me(organizationId) });
          }
        } catch (error) {
          showErrorToast(getApiErrorMessage(error, "Failed to update email."));
        }
      }}
      onSave={async () => {
        try {
          await updateAccountName(name.trim());
          await refreshAccount();
          if (organizationId) {
            await queryClient.invalidateQueries({ queryKey: meKeys.me(organizationId) });
          }
        } catch (error) {
          showErrorToast(getApiErrorMessage(error, "Failed to save profile."));
          throw error;
        }
      }}
      dangerZone={<DeleteAccountDangerZone email={account.email} />}
    />
  );
}
