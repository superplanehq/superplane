import { useState, type ReactNode } from "react";

import { ThemePreferenceControl } from "@/components/ThemePreferenceControl";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger } from "@/components/ui/select";
import { accountEmailSourceLabel, type AccountEmailOption } from "@/lib/accountSettings";
import { showSuccessToast } from "@/lib/toast";

import { FactorySettingsCard, FactorySettingsPageFrame } from "../FactorySettingsCard";
import { SettingsIdentityField } from "../settingsIdentityField";
import { SettingsActionRow } from "./accountProfileRedesignParts";
import { AccountProfileVelocityGithubCard } from "./AccountProfileVelocityGithubCard";

const MAX_NAME_LENGTH = 80;

export function AccountProfileRedesignPage({
  name,
  email,
  emailOptions = [],
  onNameChange,
  onEmailChange,
  onSave,
  velocityGithubUsername = null,
  onLinkVelocityGithub,
  onRemoveVelocityGithub,
  security,
  dangerZone,
}: {
  name: string;
  email: string;
  emailOptions?: AccountEmailOption[];
  onNameChange: (name: string) => void;
  onEmailChange?: (email: string) => void | Promise<void>;
  onSave: () => void | Promise<void>;
  velocityGithubUsername?: string | null;
  onLinkVelocityGithub?: () => void;
  onRemoveVelocityGithub?: () => void;
  security?: ReactNode;
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
    <FactorySettingsPageFrame
      title="Account"
      subtitle="Preferences, profile information, and security for your SuperPlane account."
    >
      <FactorySettingsCard title="Preferences" data-testid="account-redesign-appearance">
        <SettingsActionRow
          title="Theme"
          description="Choose light, dark, or match this device."
          testId="account-redesign-appearance-theme"
          action={<ThemePreferenceControl variant="settings" />}
        />
      </FactorySettingsCard>

      <FactorySettingsCard title="Profile information" data-testid="account-redesign-identity">
        <div className="space-y-6">
          <SettingsIdentityField
            name={name}
            nameId="account-redesign-name"
            nameTestId="account-redesign-name"
            initialsFrom={name || email}
            maxLength={MAX_NAME_LENGTH}
            helperText="This name appears on tasks, comments, and mentions."
            error={nameError}
            onNameChange={onNameChange}
          />

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

      {onLinkVelocityGithub && onRemoveVelocityGithub ? (
        <AccountProfileVelocityGithubCard
          username={velocityGithubUsername}
          onLink={onLinkVelocityGithub}
          onRemove={onRemoveVelocityGithub}
        />
      ) : null}

      {security}

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
          "SuperPlane uses this email to sign you in. Change this in Security & access below."
        )}
      </p>
    </div>
  );
}
