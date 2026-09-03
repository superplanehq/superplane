import { useState, type ReactNode } from "react";
import { Link } from "react-router";

import { Avatar } from "@/components/Avatar/avatar";
import { ThemePreferenceControl } from "@/components/ThemePreferenceControl";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select";
import { accountEmailSourceLabel, type AccountEmailOption } from "@/lib/accountSettings";
import { getNameInitials } from "@/lib/nameInitials";
import { showSuccessToast } from "@/lib/toast";

import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";
import { SettingsActionRow } from "./accountProfileRedesignParts";
import { AccountProfileVelocityGithubCard } from "./AccountProfileVelocityGithubCard";

const MAX_NAME_LENGTH = 80;

export function AccountProfileRedesignPage({
  name,
  email,
  emailOptions = [],
  securityHref,
  onNameChange,
  onEmailChange,
  onSave,
  velocityGithubUsername = null,
  onLinkVelocityGithub,
  onRemoveVelocityGithub,
  dangerZone,
}: {
  name: string;
  email: string;
  emailOptions?: AccountEmailOption[];
  securityHref?: string;
  onNameChange: (name: string) => void;
  onEmailChange?: (email: string) => void | Promise<void>;
  onSave: () => void | Promise<void>;
  velocityGithubUsername?: string | null;
  onLinkVelocityGithub?: () => void;
  onRemoveVelocityGithub?: () => void;
  dangerZone?: ReactNode;
}) {
  const [savedName, setSavedName] = useState(name);

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

  return (
    <FactorySettingsPageFrame title="Profile" subtitle="Your name, appearance, and GitHub identity.">
      <FactorySettingsCard title="Identity" data-testid="account-redesign-identity">
        <div className="space-y-6">
          <div className="flex items-start gap-4">
            <Avatar initials={getNameInitials(name || email) || "?"} alt={name || email} className="size-16 shrink-0" />
            <div className="min-w-0 flex-1 space-y-2">
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
          </div>

          <ProfileEmailField
            email={email}
            options={emailOptions}
            securityHref={securityHref}
            onEmailChange={onEmailChange}
          />

          <LoadingButton
            disabled={!isDirty || Boolean(nameError)}
            onClick={() => void handleSave()}
            data-testid="account-redesign-save"
          >
            Save
          </LoadingButton>
        </div>
      </FactorySettingsCard>

      {onLinkVelocityGithub && onRemoveVelocityGithub ? (
        <AccountProfileVelocityGithubCard
          username={velocityGithubUsername}
          onLink={onLinkVelocityGithub}
          onRemove={onRemoveVelocityGithub}
        />
      ) : null}

      <FactorySettingsCard title="Appearance" data-testid="account-redesign-appearance">
        <SettingsActionRow
          title="Theme"
          description="Choose light, dark, or match this device."
          testId="account-redesign-appearance-theme"
          action={<ThemePreferenceControl variant="settings" />}
        />
      </FactorySettingsCard>

      {dangerZone}
    </FactorySettingsPageFrame>
  );
}

function ProfileEmailField({
  email,
  options,
  securityHref,
  onEmailChange,
}: {
  email: string;
  options: AccountEmailOption[];
  securityHref?: string;
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
        <p id="account-redesign-email" className="text-[13px] text-foreground" data-testid="account-redesign-email">
          {email}
        </p>
      )}
      <p className="text-[12px] text-muted-foreground">
        {canSwitch ? (
          <>
            Choose an email from a connected sign-in method. SuperPlane uses this email to sign you in and send
            notifications.
          </>
        ) : (
          <>
            SuperPlane uses this email to sign you in.{" "}
            {securityHref ? (
              <Link to={securityHref} className="text-foreground underline-offset-2 hover:underline">
                Change this on Security
              </Link>
            ) : (
              "Change this on Security."
            )}
          </>
        )}
      </p>
    </div>
  );
}
