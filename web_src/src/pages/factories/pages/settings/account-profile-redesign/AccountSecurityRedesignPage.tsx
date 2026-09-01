import { useState } from "react";

import { Icon } from "@/components/Icon";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { showSuccessToast } from "@/lib/toast";
import { Plus } from "lucide-react";

import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";
import type { AccountRedesignSsoAccount, AccountRedesignToken } from "./accountProfileRedesignMocks";
import { SettingsActionRow } from "./accountProfileRedesignParts";

const SSO_PROVIDERS = [
  { provider: "github" as const, label: "GitHub" },
  { provider: "google" as const, label: "Google" },
];

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
}: {
  passwordSet: boolean;
  tokens: AccountRedesignToken[];
  ssoAccounts: AccountRedesignSsoAccount[];
  onChangePassword: () => void;
  onConnectSso: (provider: AccountRedesignSsoAccount["provider"]) => void;
  onDisconnectSso: (provider: AccountRedesignSsoAccount["provider"]) => void;
  onCreateToken: (name: string) => string;
  onRevokeToken: (id: string) => void;
  hideMockDialogs?: boolean;
}) {
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [createTokenOpen, setCreateTokenOpen] = useState(false);
  const [createdSecret, setCreatedSecret] = useState<string | null>(null);
  const [disconnectProvider, setDisconnectProvider] = useState<(typeof SSO_PROVIDERS)[number] | null>(null);

  const connectedCount = ssoAccounts.filter((account) => account.identity).length;
  const canDisconnectSso = passwordSet || connectedCount > 1;

  return (
    <>
      <FactorySettingsPageFrame title="Security" subtitle="Manage sign-in methods and personal tokens.">
        {passwordSet ? (
          <PasswordCard
            onChangePassword={() => {
              if (hideMockDialogs) {
                onChangePassword();
                return;
              }
              setPasswordOpen(true);
            }}
          />
        ) : null}
        <SignInMethodsCard
          ssoAccounts={ssoAccounts}
          canDisconnectSso={canDisconnectSso}
          onConnectSso={onConnectSso}
          onDisconnect={(item) => setDisconnectProvider(item)}
        />
        <PersonalTokensCard
          tokens={tokens}
          onCreate={() => {
            if (hideMockDialogs) {
              onCreateToken("");
              return;
            }
            setCreateTokenOpen(true);
          }}
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
          if (!open) {
            setDisconnectProvider(null);
          }
        }}
        onConfirm={() => {
          if (!disconnectProvider) {
            return;
          }
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
            if (!open) {
              setCreatedSecret(null);
            }
          }}
          onCreate={(name) => {
            setCreatedSecret(onCreateToken(name));
          }}
        />
      )}
    </>
  );
}

function PasswordCard({ onChangePassword }: { onChangePassword: () => void }) {
  return (
    <FactorySettingsCard title="Password" data-testid="account-redesign-password-card">
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
    </FactorySettingsCard>
  );
}

function SignInMethodsCard({
  ssoAccounts,
  canDisconnectSso,
  onConnectSso,
  onDisconnect,
}: {
  ssoAccounts: AccountRedesignSsoAccount[];
  canDisconnectSso: boolean;
  onConnectSso: (provider: AccountRedesignSsoAccount["provider"]) => void;
  onDisconnect: (item: (typeof SSO_PROVIDERS)[number]) => void;
}) {
  return (
    <FactorySettingsCard title="Sign-in methods" data-testid="account-redesign-sso">
      <p className="text-[12px] text-muted-foreground">
        Connect GitHub or Google to sign in to this SuperPlane account. Each method signs you in to the same account.
      </p>
      <ul className="mt-4 space-y-4">
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
                      Connect {item.label}
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

function PersonalTokensCard({
  tokens,
  onCreate,
  onRevokeToken,
}: {
  tokens: AccountRedesignToken[];
  onCreate: () => void;
  onRevokeToken: (id: string) => void;
}) {
  return (
    <FactorySettingsCard
      title="Personal tokens"
      action={
        <Button size="sm" onClick={onCreate} data-testid="account-redesign-create-token">
          <Plus className="size-3.5" aria-hidden />
          Create token
        </Button>
      }
    >
      <p className="text-[12px] text-muted-foreground">
        This token acts as you. Organization API keys act as the organization.
      </p>
      {tokens.length === 0 ? (
        <div
          className="mt-4 flex flex-col items-center rounded-lg border border-dashed border-border px-6 py-10 text-center"
          data-testid="account-redesign-token-empty"
        >
          <Icon name="key-round" size="lg" className="text-muted-foreground" />
          <p className="mt-2 text-sm font-medium text-foreground">No personal tokens</p>
          <p className="mt-1 max-w-sm text-xs text-muted-foreground">
            Create a token to call the SuperPlane API from a script or the CLI.
          </p>
        </div>
      ) : (
        <table className="mt-4 w-full text-left" data-testid="account-redesign-token-list">
          <thead>
            <tr>
              <th className="border-b border-border pb-2 text-xs font-medium text-muted-foreground">Name</th>
              <th className="border-b border-border pb-2 text-xs font-medium text-muted-foreground">Created</th>
              <th className="border-b border-border pb-2 text-xs font-medium text-muted-foreground">Last used</th>
              <th className="border-b border-border pb-2" />
            </tr>
          </thead>
          <tbody>
            {tokens.map((token) => (
              <tr key={token.id}>
                <td className="py-2.5 text-[13px] text-foreground">{token.name}</td>
                <td className="py-2.5 text-[13px] text-muted-foreground">{token.createdAt}</td>
                <td className="py-2.5 text-[13px] text-muted-foreground">{token.lastUsedAt ?? "Never"}</td>
                <td className="py-2.5 text-right">
                  <Button type="button" size="sm" variant="ghost" onClick={() => onRevokeToken(token.id)}>
                    Revoke
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
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

function DisconnectSsoDialog({
  provider,
  onOpenChange,
  onConfirm,
}: {
  provider: (typeof SSO_PROVIDERS)[number] | null;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  if (!provider) {
    return null;
  }

  return (
    <Dialog open onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Disconnect {provider.label}</DialogTitle>
          <DialogDescription>You cannot sign in with {provider.label} until you connect it again.</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Keep {provider.label}
          </Button>
          <Button type="button" variant="destructive" onClick={onConfirm}>
            Disconnect {provider.label}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function PasswordDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  const [currentPassword, setCurrentPassword] = useState("");
  const [nextPassword, setNextPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const valid = currentPassword.length > 0 && nextPassword.length >= 8 && nextPassword === confirmPassword;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change password</DialogTitle>
          <DialogDescription>Use at least 8 characters for the new password.</DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <PasswordField
            id="mock-current-password"
            label="Current password"
            value={currentPassword}
            onChange={setCurrentPassword}
          />
          <PasswordField id="mock-new-password" label="New password" value={nextPassword} onChange={setNextPassword} />
          <PasswordField
            id="mock-confirm-password"
            label="Confirm new password"
            value={confirmPassword}
            onChange={setConfirmPassword}
          />
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" disabled={!valid} onClick={onConfirm}>
            Update password
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function PasswordField({
  id,
  label,
  value,
  onChange,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <div className="space-y-2">
      <Label htmlFor={id}>{label}</Label>
      <Input id={id} type="password" value={value} onChange={(event) => onChange(event.target.value)} />
    </div>
  );
}

function CreateTokenDialog({
  open,
  secret,
  onOpenChange,
  onCreate,
}: {
  open: boolean;
  secret: string | null;
  onOpenChange: (open: boolean) => void;
  onCreate: (name: string) => void;
}) {
  const [name, setName] = useState("");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{secret ? "Token created" : "Create personal token"}</DialogTitle>
          <DialogDescription>
            {secret ? "Copy this secret now. You cannot view it again." : "Name the token so you can revoke it later."}
          </DialogDescription>
        </DialogHeader>
        {secret ? (
          <div className="rounded-md border border-border bg-muted/40 px-3 py-2 font-mono text-[13px]">{secret}</div>
        ) : (
          <div className="space-y-2">
            <Label htmlFor="account-redesign-token-name">Token name</Label>
            <Input
              id="account-redesign-token-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              data-testid="account-redesign-token-name"
            />
          </div>
        )}
        <DialogFooter>
          {secret ? (
            <Button type="button" onClick={() => onOpenChange(false)}>
              Done
            </Button>
          ) : (
            <>
              <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button type="button" disabled={!name.trim()} onClick={() => onCreate(name.trim())}>
                Create token
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
