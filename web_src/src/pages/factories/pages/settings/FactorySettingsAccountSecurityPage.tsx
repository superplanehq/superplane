import { useState } from "react";

import { PersonalApiTokenDialogs } from "@/components/PersonalApiTokens";
import { useAccount } from "@/contexts/useAccount";
import { usePersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import { disconnectAccountProvider, ssoLinkHref } from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ChangePasswordDialog } from "@/pages/organization/settings/components/ChangePasswordDialog";

import { AccountSecurityRedesignPage } from "./account-profile-redesign/AccountSecurityRedesignPage";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";
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

export function FactorySettingsAccountSecurityPage() {
  const { organizationId } = useFactorySettingsLayout();
  const { account, refreshAccount } = useAccount();
  const tokensPanel = usePersonalTokensPanel(organizationId);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const location = useAccountSettingsAuthResults(refreshAccount);

  if (!account) {
    return <p className="text-[13px] text-muted-foreground">Loading security…</p>;
  }

  const tokens = tokensPanel.tokens.map((token) => ({
    id: token.id || "",
    name: token.name || "Unnamed",
    createdAt: token.createdAt ? new Date(token.createdAt).toLocaleDateString() : "Unknown",
    lastUsedAt: token.lastUsedAt ? new Date(token.lastUsedAt).toLocaleDateString() : undefined,
  }));

  return (
    <>
      <AccountSecurityRedesignPage
        passwordSet={account.has_password}
        tokens={tokens}
        ssoAccounts={ssoAccountsFromAccount(account.providers)}
        hideMockDialogs
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
      {account.has_password ? <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} /> : null}
      <PersonalApiTokenDialogs panel={tokensPanel} />
    </>
  );
}
