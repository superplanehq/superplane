import { useState } from "react";

import { Icon } from "@/components/Icon";
import { Badge } from "@/components/ui/badge";
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
import { Monitor, Plus, Shield } from "lucide-react";

import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";
import type { AccountRedesignSession, AccountRedesignToken } from "./accountProfileRedesignMocks";
import { SettingsActionRow } from "./accountProfileRedesignParts";

export function AccountSecurityRedesignPage({
  passwordSet,
  twoFactorEnabled,
  tokens,
  sessions,
  onChangePassword,
  onEnableTwoFactor,
  onCreateToken,
  onRevokeToken,
  onRevokeSession,
  onRevokeOtherSessions,
}: {
  passwordSet: boolean;
  twoFactorEnabled: boolean;
  tokens: AccountRedesignToken[];
  sessions: AccountRedesignSession[];
  onChangePassword: () => void;
  onEnableTwoFactor: () => void;
  onCreateToken: (name: string) => string;
  onRevokeToken: (id: string) => void;
  onRevokeSession: (id: string) => void;
  onRevokeOtherSessions: () => void;
}) {
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [twoFactorOpen, setTwoFactorOpen] = useState(false);
  const [createTokenOpen, setCreateTokenOpen] = useState(false);
  const [createdSecret, setCreatedSecret] = useState<string | null>(null);

  return (
    <>
      <FactorySettingsPageFrame title="Security" subtitle="Manage sign-in, sessions, and personal tokens.">
        <FactorySettingsCard title="Sign-in" data-testid="account-redesign-signin">
          <div className="space-y-5">
            <SettingsActionRow
              title="Password"
              description={passwordSet ? "Password is set." : "This account has no password."}
              testId="account-redesign-password"
              action={
                <Button
                  type="button"
                  size="sm"
                  variant="outline"
                  disabled={!passwordSet}
                  onClick={() => setPasswordOpen(true)}
                >
                  Change password
                </Button>
              }
            />
            <SettingsActionRow
              title="Two-factor authentication"
              description={
                twoFactorEnabled
                  ? "Authenticator app is enabled."
                  : "Protects your account with a second factor at sign-in."
              }
              testId="account-redesign-2fa"
              action={
                twoFactorEnabled ? (
                  <Badge
                    variant="outline"
                    className="border-transparent bg-emerald-500/15 text-emerald-700 dark:text-emerald-300"
                  >
                    Enabled
                  </Badge>
                ) : (
                  <Button type="button" size="sm" onClick={() => setTwoFactorOpen(true)}>
                    <Shield className="size-3.5" aria-hidden />
                    Enable
                  </Button>
                )
              }
            />
          </div>
        </FactorySettingsCard>

        <FactorySettingsCard
          title="Personal tokens"
          action={
            <Button size="sm" onClick={() => setCreateTokenOpen(true)} data-testid="account-redesign-create-token">
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

        <FactorySettingsCard
          title="Sessions"
          action={
            sessions.some((session) => !session.isCurrent) ? (
              <Button type="button" size="sm" variant="outline" onClick={onRevokeOtherSessions}>
                Revoke other sessions
              </Button>
            ) : null
          }
        >
          <p className="text-[12px] text-muted-foreground">Devices that are signed in to your account.</p>
          <ul className="mt-4 space-y-3" data-testid="account-redesign-sessions">
            {sessions.map((session) => (
              <li key={session.id} className="flex items-start justify-between gap-4">
                <div className="flex items-start gap-3">
                  <Monitor className="mt-0.5 size-4 text-muted-foreground" aria-hidden />
                  <div>
                    <p className="text-[13px] font-medium text-foreground">
                      {session.device}
                      {session.isCurrent ? (
                        <span className="ml-2 text-[11px] font-normal text-emerald-600 dark:text-emerald-400">
                          This device
                        </span>
                      ) : null}
                    </p>
                    <p className="text-[12px] text-muted-foreground">
                      {session.location} · {session.lastSeen}
                    </p>
                  </div>
                </div>
                {session.isCurrent ? null : (
                  <Button type="button" size="sm" variant="ghost" onClick={() => onRevokeSession(session.id)}>
                    Revoke
                  </Button>
                )}
              </li>
            ))}
          </ul>
        </FactorySettingsCard>
      </FactorySettingsPageFrame>

      <PasswordDialog
        open={passwordOpen}
        onOpenChange={setPasswordOpen}
        onConfirm={() => {
          onChangePassword();
          setPasswordOpen(false);
          showSuccessToast("Password updated.");
        }}
      />
      <TwoFactorDialog
        open={twoFactorOpen}
        onOpenChange={setTwoFactorOpen}
        onConfirm={() => {
          onEnableTwoFactor();
          setTwoFactorOpen(false);
          showSuccessToast("Two-factor authentication enabled.");
        }}
      />
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
    </>
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

function TwoFactorDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  const [code, setCode] = useState("");

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Enable two-factor authentication</DialogTitle>
          <DialogDescription>Add this key to an authenticator app, then enter a 6-digit code.</DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div className="rounded-md border border-border bg-muted/40 px-3 py-2 font-mono text-[13px]">
            JBSW Y3DP EHPK 3PXP
          </div>
          <div className="space-y-2">
            <Label htmlFor="account-redesign-2fa-code">Authentication code</Label>
            <Input
              id="account-redesign-2fa-code"
              inputMode="numeric"
              maxLength={6}
              value={code}
              onChange={(event) => setCode(event.target.value.replace(/\D/g, ""))}
              data-testid="account-redesign-2fa-code"
            />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" disabled={code.length !== 6} onClick={onConfirm}>
            Enable
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
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
