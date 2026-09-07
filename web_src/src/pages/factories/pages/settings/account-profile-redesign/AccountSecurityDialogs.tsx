import { useState } from "react";

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

import type { SsoProviderItem } from "./accountSecuritySso";

export function DisconnectSsoDialog({
  provider,
  onOpenChange,
  onConfirm,
}: {
  provider: SsoProviderItem | null;
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

export function PasswordDialog({
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

export function CreateTokenDialog({
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
