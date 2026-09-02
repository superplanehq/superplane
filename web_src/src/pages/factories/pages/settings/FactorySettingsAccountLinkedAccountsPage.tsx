import { useEffect } from "react";
import { useLocation, useNavigate } from "react-router";

import { useAccount } from "@/contexts/useAccount";
import { disconnectAccountProvider, ssoLinkHref } from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";

import { AccountLinkedAccountsRedesignPage } from "./account-profile-redesign/AccountLinkedAccountsRedesignPage";

function linkedAccountsFromProviders(
  providers: Array<{ provider: string; email?: string; username?: string }> | undefined,
) {
  const connected = new Map(
    (providers ?? []).map((provider) => [provider.provider, provider.username || provider.email || provider.provider]),
  );
  return [
    { provider: "github" as const, identity: connected.get("github") ?? null },
    { provider: "google" as const, identity: connected.get("google") ?? null },
  ];
}

function providerLabel(provider: string | null): string {
  return provider === "google" ? "Google" : "GitHub";
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

    const provider = providerLabel(params.get("provider"));
    if (error === "signin_method_in_use") {
      showErrorToast(
        `This ${provider} account already belongs to another SuperPlane account. Delete that account first.`,
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

export function FactorySettingsAccountLinkedAccountsPage() {
  const { account, refreshAccount } = useAccount();
  const location = useConsumeAuthLinkResult(refreshAccount);

  if (!account) {
    return <p className="text-[13px] text-muted-foreground">Loading linked accounts…</p>;
  }

  return (
    <AccountLinkedAccountsRedesignPage
      ssoAccounts={linkedAccountsFromProviders(account.providers)}
      passwordSet={account.has_password}
      onConnect={(provider) => {
        window.location.assign(ssoLinkHref(provider, `${location.pathname}${location.search}`));
      }}
      onDisconnect={(provider) => {
        void disconnectAccountProvider(provider)
          .then(async () => {
            await refreshAccount();
            showSuccessToast(`${providerLabel(provider)} disconnected.`);
          })
          .catch((error) => {
            showErrorToast(getApiErrorMessage(error, "Failed to disconnect the account."));
          });
      }}
    />
  );
}
