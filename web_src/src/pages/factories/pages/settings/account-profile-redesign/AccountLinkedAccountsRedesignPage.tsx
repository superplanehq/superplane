import { FactorySettingsPageFrame } from "../FactorySettingsCard";
import type { AccountRedesignSsoAccount } from "./accountProfileRedesignMocks";
import { LinkedAccountsCard } from "./LinkedAccountsCard";

export function AccountLinkedAccountsRedesignPage({
  ssoAccounts,
  passwordSet,
  onConnect,
  onDisconnect,
}: {
  ssoAccounts: AccountRedesignSsoAccount[];
  passwordSet: boolean;
  onConnect: (provider: AccountRedesignSsoAccount["provider"]) => void;
  onDisconnect: (provider: AccountRedesignSsoAccount["provider"]) => void;
}) {
  return (
    <FactorySettingsPageFrame
      title="Linked accounts"
      subtitle="Link the accounts you use on other services to this SuperPlane account."
    >
      <LinkedAccountsCard
        ssoAccounts={ssoAccounts}
        passwordSet={passwordSet}
        onConnect={onConnect}
        onDisconnect={onDisconnect}
      />
    </FactorySettingsPageFrame>
  );
}
