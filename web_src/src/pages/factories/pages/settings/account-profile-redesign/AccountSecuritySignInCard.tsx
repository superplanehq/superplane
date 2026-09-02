import { Button } from "@/components/ui/button";

import { FactorySettingsCard } from "../FactorySettingsCard";
import type { AccountRedesignSsoAccount } from "./accountProfileRedesignMocks";
import { SettingsActionRow } from "./accountProfileRedesignParts";
import { SSO_PROVIDERS, type SsoProviderItem } from "./accountSecuritySso";

export function AccountSecuritySignInCard({
  passwordSet,
  ssoAccounts,
  canDisconnectSso,
  onChangePassword,
  onConnectSso,
  onDisconnect,
}: {
  passwordSet: boolean;
  ssoAccounts: AccountRedesignSsoAccount[];
  canDisconnectSso: boolean;
  onChangePassword: () => void;
  onConnectSso: (provider: AccountRedesignSsoAccount["provider"]) => void;
  onDisconnect: (item: SsoProviderItem) => void;
}) {
  return (
    <FactorySettingsCard title="Sign in methods" data-testid="account-redesign-signin">
      <p className="text-[12px] text-muted-foreground">
        Use a password or a connected provider to sign in to this SuperPlane account.
      </p>
      <ul className="mt-4 space-y-4">
        {passwordSet ? (
          <li>
            <SettingsActionRow
              title="Password"
              description="Password is set."
              testId="account-redesign-password"
              action={
                <Button type="button" size="sm" variant="outline" onClick={onChangePassword}>
                  Change password
                </Button>
              }
            />
          </li>
        ) : null}
        {SSO_PROVIDERS.map((item) => {
          const account = ssoAccounts.find((entry) => entry.provider === item.provider);
          const identity = account?.identity ?? null;
          return (
            <li key={item.provider}>
              <SettingsActionRow
                title={
                  <span className="inline-flex items-center gap-2">
                    <SsoProviderIcon provider={item.provider} />
                    {item.label}
                  </span>
                }
                description={identity ? `Connected as ${identity}` : "Not connected"}
                testId={`account-redesign-sso-${item.provider}`}
                action={
                  identity ? (
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      disabled={!canDisconnectSso}
                      onClick={() => onDisconnect(item)}
                    >
                      Disconnect
                    </Button>
                  ) : (
                    <Button type="button" size="sm" variant="outline" onClick={() => onConnectSso(item.provider)}>
                      Sign in with {item.label}
                    </Button>
                  )
                }
              />
            </li>
          );
        })}
      </ul>
      {!canDisconnectSso ? (
        <p className="mt-3 text-[12px] text-muted-foreground">Keep at least one sign-in method.</p>
      ) : null}
    </FactorySettingsCard>
  );
}

function SsoProviderIcon({ provider }: { provider: AccountRedesignSsoAccount["provider"] }) {
  if (provider === "github") {
    return (
      <svg className="size-4" viewBox="0 0 24 24" fill="currentColor" aria-hidden>
        <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z" />
      </svg>
    );
  }

  return (
    <svg className="size-4" viewBox="0 0 24 24" aria-hidden>
      <path
        fill="#4285F4"
        d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
      />
      <path
        fill="#34A853"
        d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
      />
      <path
        fill="#FBBC05"
        d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
      />
      <path
        fill="#EA4335"
        d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
      />
    </svg>
  );
}
