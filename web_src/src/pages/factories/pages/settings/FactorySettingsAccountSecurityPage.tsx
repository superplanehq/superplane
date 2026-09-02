import { useState } from "react";

import { PersonalApiTokenDialogs } from "@/components/PersonalApiTokens";
import { useAccount } from "@/contexts/useAccount";
import { usePersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import { ChangePasswordDialog } from "@/pages/organization/settings/components/ChangePasswordDialog";

import { AccountSecurityRedesignPage } from "./account-profile-redesign/AccountSecurityRedesignPage";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

export function FactorySettingsAccountSecurityPage() {
  const { organizationId } = useFactorySettingsLayout();
  const { account } = useAccount();
  const tokensPanel = usePersonalTokensPanel(organizationId);
  const [passwordOpen, setPasswordOpen] = useState(false);

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
        hideMockDialogs
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
      {account.has_password ? <ChangePasswordDialog open={passwordOpen} onOpenChange={setPasswordOpen} /> : null}
      <PersonalApiTokenDialogs panel={tokensPanel} />
    </>
  );
}
