import { useState } from "react";

import { showSuccessToast } from "@/lib/toast";

import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import type { AccountRedesignSsoAccount, AccountRedesignToken } from "./accountProfileRedesignMocks";
import { CreateTokenDialog, DisconnectSsoDialog, PasswordDialog } from "./AccountSecurityDialogs";
import { AccountSecuritySignInCard } from "./AccountSecuritySignInCard";
import { AccountSecurityTokensCard } from "./AccountSecurityTokensCard";
import type { SsoProviderItem } from "./accountSecuritySso";

type AccountSecurityRedesignPageProps = {
  passwordSet: boolean;
  tokens: AccountRedesignToken[];
  ssoAccounts: AccountRedesignSsoAccount[];
  onChangePassword: () => void;
  onConnectSso: (provider: AccountRedesignSsoAccount["provider"]) => void;
  onDisconnectSso: (provider: AccountRedesignSsoAccount["provider"]) => void;
  onCreateToken: (name: string) => string;
  onRevokeToken: (id: string) => void;
  hideMockDialogs?: boolean;
};

export function AccountSecurityRedesignPage({
  passwordSet,
  tokens,
  ssoAccounts,
  onChangePassword,
  onConnectSso,
  onDisconnectSso,
  onCreateToken,
  onRevokeToken,
  hideMockDialogs = false,
}: AccountSecurityRedesignPageProps) {
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [createTokenOpen, setCreateTokenOpen] = useState(false);
  const [createdSecret, setCreatedSecret] = useState<string | null>(null);
  const [disconnectProvider, setDisconnectProvider] = useState<SsoProviderItem | null>(null);

  const connectedCount = ssoAccounts.filter((account) => account.identity).length;
  const canDisconnectSso = passwordSet || connectedCount > 1;

  return (
    <>
      <FactorySettingsPageFrame title="Security" subtitle="Manage sign-in methods and personal tokens.">
        <AccountSecuritySignInCard
          passwordSet={passwordSet}
          ssoAccounts={ssoAccounts}
          canDisconnectSso={canDisconnectSso}
          onChangePassword={() => (hideMockDialogs ? onChangePassword() : setPasswordOpen(true))}
          onConnectSso={onConnectSso}
          onDisconnect={setDisconnectProvider}
        />
        <AccountSecurityTokensCard
          tokens={tokens}
          onCreate={() => (hideMockDialogs ? onCreateToken("") : setCreateTokenOpen(true))}
          onRevokeToken={onRevokeToken}
        />
      </FactorySettingsPageFrame>

      {hideMockDialogs ? null : (
        <PasswordDialog
          open={passwordOpen}
          onOpenChange={setPasswordOpen}
          onConfirm={() => {
            onChangePassword();
            setPasswordOpen(false);
            showSuccessToast("Password updated.");
          }}
        />
      )}
      <DisconnectSsoDialog
        provider={disconnectProvider}
        onOpenChange={(open) => {
          if (!open) setDisconnectProvider(null);
        }}
        onConfirm={() => {
          if (!disconnectProvider) return;
          onDisconnectSso(disconnectProvider.provider);
          setDisconnectProvider(null);
        }}
      />
      {hideMockDialogs ? null : (
        <CreateTokenDialog
          open={createTokenOpen}
          secret={createdSecret}
          onOpenChange={(open) => {
            setCreateTokenOpen(open);
            if (!open) setCreatedSecret(null);
          }}
          onCreate={(name) => setCreatedSecret(onCreateToken(name))}
        />
      )}
    </>
  );
}
