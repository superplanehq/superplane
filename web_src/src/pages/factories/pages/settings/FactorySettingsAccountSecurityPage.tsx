import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { PersonalApiTokenDialogs } from "@/components/PersonalApiTokens";
import { useAccount } from "@/contexts/useAccount";
import { usePersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import { disconnectAccountProvider, ssoLinkHref } from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { ChangePasswordDialog } from "@/pages/organization/settings/components/ChangePasswordDialog";

import { AccountSecurityRedesignPage } from "./account-profile-redesign/AccountSecurityRedesignPage";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

function ssoAccountsFromAccount(providers: Array<{ provider: string; email?: string; username?: string }> | undefined) {
  const connected = new Map(
    (providers ?? []).map((provider) => [provider.provider, provider.username || provider.email || provider.provider]),
  );
  return [
    { provider: "github" as const, identity: connected.get("github") ?? null },
    { provider: "google" as const, identity: connected.get("google") ?? null },
  ];
}

function useConsumeAuthLinkResult(refreshAccount: () => Promise<void>) {
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const error = params.get("auth_error");
    const result = params.get("auth_link_result");
    if (!error && !result) {
      return;
    }
    const provider = params.get("provider") === "google" ? "Google" : "GitHub";
    if (error === "signin_method_in_use") {
      showErrorToast(
        `This ${provider} identity already belongs to another SuperPlane account. Delete that account first.`,
      );
    }
    if (result === "connected") {
      showSuccessToast(`${provider} connected.`);
      void refreshAccount();
    }
    params.delete("auth_error");
    params.delete("auth_link_result");
    params.delete("provider");
    const search = params.toString();
    void navigate({ pathname: location.pathname, search: search ? `?${search}` : "" }, { replace: true });
  }, [location.pathname, location.search, navigate, refreshAccount]);

  return location;
}

export function FactorySettingsAccountSecurityPage() {
  const { organizationId } = useFactorySettingsLayout();
  const { account, refreshAccount } = useAccount();
  const tokensPanel = usePersonalTokensPanel(organizationId);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const location = useConsumeAuthLinkResult(refreshAccount);

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
