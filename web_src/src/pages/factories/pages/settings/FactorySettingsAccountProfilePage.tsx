import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";

import { PersonalApiTokenDialogs } from "@/components/PersonalApiTokens";
import { useAccount } from "@/contexts/useAccount";
import { meKeys } from "@/hooks/useMe";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { usePersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import {
  accountEmailOptions,
  disconnectAccountProvider,
  disconnectLinkedAccount,
  linkedAccountConnectHref,
  ssoLinkHref,
  updateAccountEmail,
  updateAccountName,
} from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ChangePasswordDialog } from "@/pages/organization/settings/components/ChangePasswordDialog";

import { AccountProfileRedesignPage } from "./account-profile-redesign/AccountProfileRedesignPage";
import { AccountSecurityRedesignPage } from "./account-profile-redesign/AccountSecurityRedesignPage";
import { DeleteAccountDangerZone } from "./DeleteAccountDangerZone";
import { useAccountSettingsAuthResults } from "./useAccountSettingsAuthResults";

function ssoAccountsFromAccount(providers: Array<{ provider: string; email?: string; username?: string }> | undefined) {
  const connected = new Map(
    (providers ?? []).map((provider) => [provider.provider, provider.username || provider.email || provider.provider]),
  );
  return [
    { provider: "github" as const, identity: connected.get("github") ?? null },
    { provider: "google" as const, identity: connected.get("google") ?? null },
  ];
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

  const velocityGithub = (account.linked_accounts ?? []).find((linked) => linked.provider === "github");
  const tokens = tokensPanel.tokens.map((token) => ({
    id: token.id || "",
    name: token.name || "Unnamed",
    createdAt: token.createdAt ? new Date(token.createdAt).toLocaleDateString() : "Unknown",
    lastUsedAt: token.lastUsedAt ? new Date(token.lastUsedAt).toLocaleDateString() : undefined,
  }));

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
        security={
          <AccountSecurityRedesignPage
            passwordSet={account.has_password}
            tokens={tokens}
            ssoAccounts={ssoAccountsFromAccount(account.providers)}
            hideMockDialogs
            embedded
            onChangePassword={() => setPasswordOpen(true)}
            onConnectSso={(provider) => {
              window.location.assign(ssoLinkHref(provider, `${location.pathname}${location.search}`));
            }}
            onDisconnectSso={(provider) => {
              void disconnectAccountProvider(provider)
                .then(async () => {
                  await refreshAccount();
                  showSuccessToast(provider === "github" ? "GitHub disconnected." : "Google disconnected.");
                })
                .catch((error) => {
                  showErrorToast(getApiErrorMessage(error, "Failed to disconnect sign-in method."));
                });
            }}
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
        dangerZone={<DeleteAccountDangerZone email={account.email} />}
      />
      {account.has_password ? <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} /> : null}
      <PersonalApiTokenDialogs panel={tokensPanel} />
    </>
  );
}
