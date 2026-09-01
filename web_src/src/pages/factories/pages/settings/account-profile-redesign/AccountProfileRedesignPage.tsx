import { useRef, useState } from "react";

import { Avatar } from "@/components/Avatar/avatar";
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
import { LoadingButton } from "@/components/ui/loading-button";
import { showSuccessToast } from "@/lib/toast";
import { Copy, LogOut, Trash2 } from "lucide-react";

import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";
import { initialsFor, SettingsActionRow } from "./accountProfileRedesignParts";

const MAX_NAME_LENGTH = 80;

export function AccountProfileRedesignPage({
  name,
  email,
  userId,
  avatarUrl,
  onNameChange,
  onAvatarChange,
  onEmailChange,
  onSave,
  onLeaveWorkspace,
  onDeleteAccount,
}: {
  name: string;
  email: string;
  userId: string;
  avatarUrl: string | null;
  onNameChange: (name: string) => void;
  onAvatarChange: (avatarUrl: string | null) => void;
  onEmailChange: (email: string) => void;
  onSave: () => void;
  onLeaveWorkspace: () => void;
  onDeleteAccount: () => void;
}) {
  const [savedName, setSavedName] = useState(name);
  const [savedAvatar, setSavedAvatar] = useState(avatarUrl);
  const [copied, setCopied] = useState(false);
  const [emailOpen, setEmailOpen] = useState(false);
  const [leaveOpen, setLeaveOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const isDirty = name.trim() !== savedName || avatarUrl !== savedAvatar;
  const nameError = name.trim() ? "" : "Name is required.";

  const handleSave = () => {
    if (nameError) {
      return;
    }
    onSave();
    setSavedName(name.trim());
    setSavedAvatar(avatarUrl);
    showSuccessToast("Profile saved.");
  };

  const handleCopyId = async () => {
    await navigator.clipboard.writeText(userId);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  const handlePhoto = (file: File | undefined) => {
    if (!file) {
      return;
    }
    const reader = new FileReader();
    reader.onload = () => {
      if (typeof reader.result === "string") {
        onAvatarChange(reader.result);
      }
    };
    reader.readAsDataURL(file);
  };

  return (
    <>
      <FactorySettingsPageFrame title="Profile" subtitle="Manage how your name and email appear in SuperPlane.">
        <FactorySettingsCard title="Identity" data-testid="account-redesign-identity">
          <div className="space-y-6">
            <div className="flex items-center gap-4">
              <button
                type="button"
                className="group relative size-16 shrink-0 rounded-full"
                onClick={() => fileInputRef.current?.click()}
                data-testid="account-redesign-avatar"
                aria-label="Change photo"
              >
                <Avatar
                  src={avatarUrl || undefined}
                  initials={avatarUrl ? undefined : initialsFor(name || email)}
                  alt={name || email}
                  className="size-16"
                />
                <span className="absolute inset-0 flex items-center justify-center rounded-full bg-black/50 text-[11px] font-medium text-white opacity-0 group-hover:opacity-100">
                  Change
                </span>
              </button>
              <div className="space-y-1">
                <Button type="button" size="sm" variant="outline" onClick={() => fileInputRef.current?.click()}>
                  Change photo
                </Button>
                {avatarUrl ? (
                  <Button type="button" size="sm" variant="ghost" onClick={() => onAvatarChange(null)}>
                    Remove photo
                  </Button>
                ) : null}
                <p className="text-[12px] text-muted-foreground">PNG or JPG. Used on tasks and comments.</p>
              </div>
              <input
                ref={fileInputRef}
                type="file"
                accept="image/png,image/jpeg"
                className="sr-only"
                onChange={(event) => handlePhoto(event.target.files?.[0])}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="account-redesign-name">Name</Label>
              <Input
                id="account-redesign-name"
                data-testid="account-redesign-name"
                value={name}
                maxLength={MAX_NAME_LENGTH}
                onChange={(event) => onNameChange(event.target.value)}
              />
              <p className="text-[12px] text-muted-foreground">This name appears on tasks, comments, and mentions.</p>
              {nameError ? <p className="text-[11px] text-destructive">{nameError}</p> : null}
            </div>

            <SettingsActionRow
              title="Email"
              description={email}
              testId="account-redesign-email"
              action={
                <Button type="button" size="sm" variant="outline" onClick={() => setEmailOpen(true)}>
                  Change email
                </Button>
              }
            />

            <SettingsActionRow
              title="User ID"
              description={<span className="font-mono text-[12px]">{userId}</span>}
              testId="account-redesign-user-id"
              action={
                <Button type="button" size="sm" variant="outline" onClick={() => void handleCopyId()}>
                  <Copy className="size-3.5" aria-hidden />
                  {copied ? "Copied" : "Copy"}
                </Button>
              }
            />

            <LoadingButton
              disabled={!isDirty || Boolean(nameError)}
              onClick={handleSave}
              data-testid="account-redesign-save"
            >
              Save
            </LoadingButton>
          </div>
        </FactorySettingsCard>

        <FactorySettingsCard title="Workspace access" data-testid="account-redesign-workspace-access">
          <SettingsActionRow
            title="Leave workspace"
            description="Remove yourself from Instabot. Tasks you own stay in the workspace."
            action={
              <Button type="button" size="sm" variant="ghost" onClick={() => setLeaveOpen(true)}>
                <LogOut className="size-3.5" aria-hidden />
                Leave workspace
              </Button>
            }
          />
        </FactorySettingsCard>

        <FactorySettingsCard
          title="Danger zone"
          titleClassName="text-destructive"
          className="border-destructive/40"
          data-testid="account-redesign-danger"
        >
          <SettingsActionRow
            title="Delete account"
            description="Permanently delete your SuperPlane account. This cannot be undone."
            action={
              <Button type="button" size="sm" variant="destructive" onClick={() => setDeleteOpen(true)}>
                <Trash2 className="size-3.5" aria-hidden />
                Delete account
              </Button>
            }
          />
        </FactorySettingsCard>
      </FactorySettingsPageFrame>

      <ChangeEmailDialog
        open={emailOpen}
        currentEmail={email}
        onOpenChange={setEmailOpen}
        onConfirm={(next) => {
          onEmailChange(next);
          setEmailOpen(false);
          showSuccessToast("Confirmation sent to the new address.");
        }}
      />
      <ConfirmDialog
        open={leaveOpen}
        title="Leave workspace"
        description="You lose access to Instabot. Tasks you own stay assigned to you."
        confirmLabel="Leave workspace"
        cancelLabel="Stay in workspace"
        onOpenChange={setLeaveOpen}
        onConfirm={() => {
          onLeaveWorkspace();
          setLeaveOpen(false);
          showSuccessToast("Mockup: you left the workspace.");
        }}
      />
      <ConfirmDialog
        open={deleteOpen}
        title="Delete account"
        description="This permanently deletes your SuperPlane account and personal tokens."
        confirmLabel="Delete account"
        cancelLabel="Keep account"
        destructive
        onOpenChange={setDeleteOpen}
        onConfirm={() => {
          onDeleteAccount();
          setDeleteOpen(false);
          showSuccessToast("Mockup: account not deleted.");
        }}
      />
    </>
  );
}

function ChangeEmailDialog({
  open,
  currentEmail,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  currentEmail: string;
  onOpenChange: (open: boolean) => void;
  onConfirm: (email: string) => void;
}) {
  const [nextEmail, setNextEmail] = useState("");
  const valid = nextEmail.includes("@") && nextEmail !== currentEmail;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Change email</DialogTitle>
          <DialogDescription>We send a confirmation link to the new address.</DialogDescription>
        </DialogHeader>
        <div className="space-y-3">
          <div>
            <p className="text-[12px] text-muted-foreground">Current email</p>
            <p className="text-[13px] text-foreground">{currentEmail}</p>
          </div>
          <div className="space-y-2">
            <Label htmlFor="account-redesign-new-email">New email</Label>
            <Input
              id="account-redesign-new-email"
              type="email"
              value={nextEmail}
              onChange={(event) => setNextEmail(event.target.value)}
              data-testid="account-redesign-new-email"
            />
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" disabled={!valid} onClick={() => onConfirm(nextEmail.trim())}>
            Send confirmation
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function ConfirmDialog({
  open,
  title,
  description,
  confirmLabel,
  cancelLabel,
  destructive,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  title: string;
  description: string;
  confirmLabel: string;
  cancelLabel: string;
  destructive?: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            {cancelLabel}
          </Button>
          <Button type="button" variant={destructive ? "destructive" : "default"} onClick={onConfirm}>
            {confirmLabel}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
