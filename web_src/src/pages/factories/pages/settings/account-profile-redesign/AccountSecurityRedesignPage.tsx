import { useState } from "react";

import { showSuccessToast } from "@/lib/toast";

import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import type { AccountRedesignToken } from "./accountProfileRedesignMocks";
import { CreateTokenDialog, PasswordDialog } from "./AccountSecurityDialogs";
import { AccountSecuritySignInCard } from "./AccountSecuritySignInCard";
import { AccountSecurityTokensCard } from "./AccountSecurityTokensCard";

type AccountSecurityRedesignPageProps = {
  passwordSet: boolean;
  tokens: AccountRedesignToken[];
  onChangePassword: () => void;
  onCreateToken: (name: string) => string;
  onRevokeToken: (id: string) => void;
  hideMockDialogs?: boolean;
  embedded?: boolean;
};

export function AccountSecurityRedesignPage({
  passwordSet,
  tokens,
  onChangePassword,
  onCreateToken,
  onRevokeToken,
  hideMockDialogs = false,
  embedded = false,
}: AccountSecurityRedesignPageProps) {
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [createTokenOpen, setCreateTokenOpen] = useState(false);
  const [createdSecret, setCreatedSecret] = useState<string | null>(null);

  const cards = (
    <div className="flex flex-col gap-5" data-testid="account-redesign-security">
      <h2 className="text-[15px] font-medium tracking-[-0.01em] text-foreground">Security & access</h2>
      <AccountSecuritySignInCard
        passwordSet={passwordSet}
        onChangePassword={() => (hideMockDialogs ? onChangePassword() : setPasswordOpen(true))}
      />
      <AccountSecurityTokensCard
        tokens={tokens}
        onCreate={() => (hideMockDialogs ? onCreateToken("") : setCreateTokenOpen(true))}
        onRevokeToken={onRevokeToken}
      />
    </div>
  );

  return (
    <>
      {embedded ? (
        cards
      ) : (
        <FactorySettingsPageFrame title="Security & access" subtitle="Manage sign-in methods and personal tokens.">
          {cards}
        </FactorySettingsPageFrame>
      )}

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
