import { useState, type ReactNode } from "react";

import { Avatar } from "@/components/Avatar/avatar";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select";
import { accountEmailSourceLabel, type AccountEmailOption } from "@/lib/accountSettings";
import { showSuccessToast } from "@/lib/toast";
import { Copy } from "lucide-react";

import { getNameInitials } from "@/lib/nameInitials";

import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";

const MAX_NAME_LENGTH = 80;

function shortenedUserId(id: string): string {
  if (id.length <= 12) {
    return id;
  }
  return `${id.slice(0, 8)}…${id.slice(-4)}`;
}

export function AccountProfileRedesignPage({
  name,
  email,
  emailOptions = [],
  userId,
  onNameChange,
  onEmailChange,
  onSave,
  dangerZone,
}: {
  name: string;
  email: string;
  emailOptions?: AccountEmailOption[];
  userId: string;
  onNameChange: (name: string) => void;
  onEmailChange?: (email: string) => void | Promise<void>;
  onSave: () => void | Promise<void>;
  dangerZone?: ReactNode;
}) {
  const [savedName, setSavedName] = useState(name);
  const [copied, setCopied] = useState(false);

  const isDirty = name.trim() !== savedName;
  const nameError = name.trim() ? "" : "Name is required.";

  const handleSave = async () => {
    if (nameError) {
      return;
    }
    await onSave();
    setSavedName(name.trim());
    showSuccessToast("Profile saved.");
  };

  const handleCopyId = async () => {
    await navigator.clipboard.writeText(userId);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1600);
  };

  return (
    <FactorySettingsPageFrame title="Profile" subtitle="Manage how your name appears in SuperPlane.">
      <FactorySettingsCard title="Identity" data-testid="account-redesign-identity">
        <div className="space-y-6">
          <div className="flex items-center gap-4">
            <Avatar initials={getNameInitials(name || email) || "?"} alt={name || email} className="size-16" />
            <div className="min-w-0">
              <p className="truncate text-[13px] font-medium text-foreground">{name || "No name"}</p>
              <p className="truncate text-[12px] text-muted-foreground">{email}</p>
            </div>
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

          <ProfileEmailField email={email} options={emailOptions} onEmailChange={onEmailChange} />

          <LoadingButton
            disabled={!isDirty || Boolean(nameError)}
            onClick={() => void handleSave()}
            data-testid="account-redesign-save"
          >
            Save
          </LoadingButton>
        </div>
      </FactorySettingsCard>

      <div
        className="flex items-center justify-between gap-3 text-muted-foreground"
        data-testid="account-redesign-user-id"
      >
        <p className="text-[12px]">
          User ID{" "}
          <span className="font-mono" title={userId}>
            {shortenedUserId(userId)}
          </span>
        </p>
        <Button
          type="button"
          size="icon-xs"
          variant="ghost"
          aria-label={copied ? "Copied" : "Copy user ID"}
          onClick={() => void handleCopyId()}
        >
          <Copy className="size-3.5" aria-hidden />
        </Button>
      </div>
      {dangerZone}
    </FactorySettingsPageFrame>
  );
}

function ProfileEmailField({
  email,
  options,
  onEmailChange,
}: {
  email: string;
  options: AccountEmailOption[];
  onEmailChange?: (email: string) => void | Promise<void>;
}) {
  const canSwitch = options.length > 1 && Boolean(onEmailChange);
  return (
    <div className="space-y-2">
      <Label htmlFor="account-redesign-email">Primary email</Label>
      {canSwitch ? (
        <Select
          value={email.toLowerCase()}
          onValueChange={(next) => {
            void onEmailChange?.(next);
          }}
        >
          <SelectTrigger id="account-redesign-email" className="h-8 w-full px-3" data-testid="account-redesign-email">
            <span className="min-w-0 flex-1 truncate text-left">{email}</span>
          </SelectTrigger>
          <SelectContent position="popper">
            {options.map((option) => (
              <SelectItem key={option.email} value={option.email} className="h-auto items-start py-2.5">
                <div className="flex min-w-0 flex-col items-start gap-1 whitespace-normal">
                  <span>{option.email}</span>
                  <span className="text-[12px] leading-4 text-muted-foreground">
                    {accountEmailSourceLabel(option.sources)}
                  </span>
                </div>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      ) : (
        <Input id="account-redesign-email" data-testid="account-redesign-email" value={email} disabled />
      )}
      <p className="text-[12px] text-muted-foreground">
        {canSwitch
          ? "Choose an email from a connected sign-in method. SuperPlane uses this email to sign you in and send notifications."
          : "Used to sign in and receive notifications."}
      </p>
    </div>
  );
}
