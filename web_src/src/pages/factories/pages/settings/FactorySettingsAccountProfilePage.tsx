import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { useParams } from "react-router";

import { useAccount } from "@/contexts/useAccount";
import { meKeys } from "@/hooks/useMe";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import {
  accountEmailOptions,
  disconnectLinkedAccount,
  linkedAccountConnectHref,
  updateAccountEmail,
  updateAccountName,
} from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { AccountProfileRedesignPage } from "./account-profile-redesign/AccountProfileRedesignPage";
import { DeleteAccountDangerZone } from "./DeleteAccountDangerZone";
import { useAccountSettingsAuthResults } from "./useAccountSettingsAuthResults";

export function FactorySettingsAccountProfilePage() {
  const { account, refreshAccount } = useAccount();
  const organizationId = useOrganizationId();
  const { factoryKey } = useParams<{ factoryKey: string }>();
  const queryClient = useQueryClient();
  const [name, setName] = useState(account?.name ?? "");
  const location = useAccountSettingsAuthResults(refreshAccount);

  useEffect(() => {
    if (account?.name) {
      setName(account.name);
    }
  }, [account?.name]);

  if (!account) {
    return <p className="text-[13px] text-muted-foreground">Loading profile…</p>;
  }

  const securityHref =
    organizationId && factoryKey
      ? factorySettingsSectionPath(organizationId, factoryKey, "account", "security")
      : undefined;
  const velocityGithub = (account.linked_accounts ?? []).find((linked) => linked.provider === "github");

  return (
    <AccountProfileRedesignPage
      name={name}
      email={account.email}
      emailOptions={accountEmailOptions({
        email: account.email,
        hasPassword: account.has_password,
        providers: account.providers,
      })}
      securityHref={securityHref}
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
      velocityGithubUsername={velocityGithub?.username ?? null}
      onLinkVelocityGithub={() => {
        window.location.assign(linkedAccountConnectHref("github", `${location.pathname}${location.search}`));
      }}
      onRemoveVelocityGithub={() => {
        void disconnectLinkedAccount("github")
          .then(async () => {
            await refreshAccount();
            showSuccessToast("GitHub link removed.");
          })
          .catch((error) => {
            showErrorToast(getApiErrorMessage(error, "Failed to remove the linked account."));
          });
      }}
      dangerZone={<DeleteAccountDangerZone email={account.email} />}
    />
  );
}
