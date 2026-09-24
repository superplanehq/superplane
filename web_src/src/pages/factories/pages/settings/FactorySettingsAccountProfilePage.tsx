import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { PersonalApiTokenDialogs } from "@/components/PersonalApiTokens";
import { useAccount } from "@/contexts/useAccount";
import { meKeys } from "@/hooks/useMe";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { usePersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import {
  accountEmailOptions,
  disconnectLinkedAccount,
  linkedAccountConnectHref,
  updateAccountEmail,
  updateAccountName,
} from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ChangePasswordDialog } from "@/pages/organization/settings/components/ChangePasswordDialog";

import { AccountProfileAssociatedAccountsCard } from "./account-profile-redesign/AccountProfileAssociatedAccountsCard";
import { AccountProfileRedesignPage } from "./account-profile-redesign/AccountProfileRedesignPage";
import { AccountSecurityRedesignPage } from "./account-profile-redesign/AccountSecurityRedesignPage";
import { DeleteAccountDangerZone } from "./DeleteAccountDangerZone";
import { useAccountSettingsAuthResults } from "./useAccountSettingsAuthResults";

function linkedGithubUsername(
  linkedAccounts: Array<{ provider: string; username?: string }> | undefined,
): string | null {
  const github = linkedAccounts?.find((account) => account.provider === "github");
  return github?.username?.trim() || null;
}

export function FactorySettingsAccountProfilePage() {
  const { account, refreshAccount } = useAccount();
  const organizationId = useOrganizationId();
  const queryClient = useQueryClient();
  const [name, setName] = useState(account?.name ?? "");
  const [passwordOpen, setPasswordOpen] = useState(false);
  const location = useAccountSettingsAuthResults(refreshAccount);
  const tokensPanel = usePersonalTokensPanel(organizationId);

  useEffect(() => {
    if (account?.name) {
      setName(account.name);
    }
  }, [account?.name]);

  if (!account) {
    return <p className="text-[13px] text-muted-foreground">Loading profile…</p>;
  }

  const tokens = tokensPanel.tokens.map((token) => ({
    id: token.id || "",
    name: token.name || "Unnamed",
    createdAt: token.createdAt ? new Date(token.createdAt).toLocaleDateString() : "Unknown",
    lastUsedAt: token.lastUsedAt ? new Date(token.lastUsedAt).toLocaleDateString() : undefined,
  }));

  const redirectPath = `${location.pathname}${location.search}`;

  return (
    <>
      <AccountProfileRedesignPage
        name={name}
        email={account.email}
        emailOptions={accountEmailOptions({
          email: account.email,
          hasPassword: account.has_password,
          providers: account.providers,
        })}
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
        associatedAccounts={
          <AccountProfileAssociatedAccountsCard
            githubUsername={linkedGithubUsername(account.linked_accounts)}
            onLinkGithub={() => {
              window.location.assign(linkedAccountConnectHref("github", redirectPath));
            }}
            onRemoveGithub={() => {
              void disconnectLinkedAccount("github")
                .then(async () => {
                  await refreshAccount();
                  showSuccessToast("GitHub link removed.");
                })
                .catch((error) => {
                  showErrorToast(getApiErrorMessage(error, "Failed to remove the GitHub link."));
                });
            }}
          />
        }
        security={
          <AccountSecurityRedesignPage
            passwordSet={account.has_password}
            tokens={tokens}
            hideMockDialogs
            embedded
            onChangePassword={() => setPasswordOpen(true)}
            onCreateToken={() => {
              tokensPanel.openCreateDialog();
              return "";
            }}
            onRevokeToken={(id) => {
              const token = tokensPanel.tokens.find((item) => item.id === id);
              if (token) {
                tokensPanel.requestRevoke(token);
              }
            }}
          />
        }
        dangerZone={
          <DeleteAccountDangerZone
            email={account.email}
            organizationsPendingDeletion={account.organizations_pending_deletion}
          />
        }
      />
      {account.has_password ? <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} /> : null}
      <PersonalApiTokenDialogs panel={tokensPanel} />
    </>
  );
}
