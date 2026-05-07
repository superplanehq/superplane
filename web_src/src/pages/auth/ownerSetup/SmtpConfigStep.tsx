import React from "react";
import { Input, InputGroup } from "@/components/Input/input";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/ui/checkbox";
import { ErrorBanner } from "./ErrorBanner";

type SmtpConfigStepProps = {
  smtpHost: string;
  smtpPort: string;
  smtpUsername: string;
  smtpPassword: string;
  smtpFromName: string;
  smtpFromEmail: string;
  smtpUseTLS: boolean;
  loading: boolean;
  error: string | null;
  fieldErrors: Record<string, string>;
  onSmtpHostChange: (value: string) => void;
  onSmtpPortChange: (value: string) => void;
  onSmtpUsernameChange: (value: string) => void;
  onSmtpPasswordChange: (value: string) => void;
  onSmtpFromNameChange: (value: string) => void;
  onSmtpFromEmailChange: (value: string) => void;
  onSmtpUseTLSChange: (value: boolean) => void;
  onBack: () => void;
  onSkipSMTP: () => void;
  onSubmit: (event: React.FormEvent) => void;
};

type SmtpConfigFieldsProps = Pick<
  SmtpConfigStepProps,
  | "smtpHost"
  | "smtpPort"
  | "smtpUsername"
  | "smtpPassword"
  | "smtpFromName"
  | "smtpFromEmail"
  | "smtpUseTLS"
  | "fieldErrors"
  | "onSmtpHostChange"
  | "onSmtpPortChange"
  | "onSmtpUsernameChange"
  | "onSmtpPasswordChange"
  | "onSmtpFromNameChange"
  | "onSmtpFromEmailChange"
  | "onSmtpUseTLSChange"
>;

const SmtpConfigFields: React.FC<SmtpConfigFieldsProps> = ({
  smtpHost,
  smtpPort,
  smtpUsername,
  smtpPassword,
  smtpFromName,
  smtpFromEmail,
  smtpUseTLS,
  fieldErrors,
  onSmtpHostChange,
  onSmtpPortChange,
  onSmtpUsernameChange,
  onSmtpPasswordChange,
  onSmtpFromNameChange,
  onSmtpFromEmailChange,
  onSmtpUseTLSChange,
}) => (
  <>
    <div>
      <Label className="mb-2 block text-left">
        SMTP Host <span className="text-gray-800">*</span>
      </Label>
      <InputGroup>
        <Input
          type="text"
          value={smtpHost}
          onChange={(event) => onSmtpHostChange(event.target.value)}
          placeholder="smtp.example.com"
          className={fieldErrors.smtpHost ? "border-red-500" : ""}
        />
      </InputGroup>
      {fieldErrors.smtpHost && <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.smtpHost}</p>}
    </div>

    <div>
      <Label className="mb-2 block text-left">
        SMTP Port <span className="text-gray-800">*</span>
      </Label>
      <InputGroup>
        <Input
          type="text"
          value={smtpPort}
          onChange={(event) => onSmtpPortChange(event.target.value)}
          placeholder="587"
          className={fieldErrors.smtpPort ? "border-red-500" : ""}
        />
      </InputGroup>
      {fieldErrors.smtpPort && <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.smtpPort}</p>}
    </div>

    <div>
      <Label className="mb-2 block text-left">SMTP Username</Label>
      <InputGroup>
        <Input
          type="text"
          value={smtpUsername}
          onChange={(event) => onSmtpUsernameChange(event.target.value)}
          placeholder="smtp-user"
        />
      </InputGroup>
    </div>

    <div>
      <Label className="mb-2 block text-left">SMTP Password</Label>
      <InputGroup>
        <Input
          type="password"
          value={smtpPassword}
          onChange={(event) => onSmtpPasswordChange(event.target.value)}
          placeholder="SMTP password"
          className={fieldErrors.smtpPassword ? "border-red-500 ph-no-capture" : "ph-no-capture"}
        />
      </InputGroup>
      {fieldErrors.smtpPassword && (
        <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.smtpPassword}</p>
      )}
    </div>

    <div>
      <Label className="mb-2 block text-left">From Name</Label>
      <InputGroup>
        <Input
          type="text"
          value={smtpFromName}
          onChange={(event) => onSmtpFromNameChange(event.target.value)}
          placeholder="SuperPlane"
        />
      </InputGroup>
    </div>

    <div>
      <Label className="mb-2 block text-left">
        From Email <span className="text-gray-800">*</span>
      </Label>
      <InputGroup>
        <Input
          type="email"
          value={smtpFromEmail}
          onChange={(event) => onSmtpFromEmailChange(event.target.value)}
          placeholder="noreply@example.com"
          className={fieldErrors.smtpFromEmail ? "border-red-500" : ""}
        />
      </InputGroup>
      {fieldErrors.smtpFromEmail && (
        <p className="mt-1 text-xs text-red-600 dark:text-red-400">{fieldErrors.smtpFromEmail}</p>
      )}
    </div>

    <div className="flex items-center gap-2">
      <Checkbox
        id="owner-setup-smtp-use-tls"
        checked={smtpUseTLS}
        onCheckedChange={(checked) => onSmtpUseTLSChange(checked === true)}
      />
      <Label htmlFor="owner-setup-smtp-use-tls">Use TLS (STARTTLS)</Label>
    </div>
  </>
);

export const SmtpConfigStep: React.FC<SmtpConfigStepProps> = ({
  smtpHost,
  smtpPort,
  smtpUsername,
  smtpPassword,
  smtpFromName,
  smtpFromEmail,
  smtpUseTLS,
  loading,
  error,
  fieldErrors,
  onSmtpHostChange,
  onSmtpPortChange,
  onSmtpUsernameChange,
  onSmtpPasswordChange,
  onSmtpFromNameChange,
  onSmtpFromEmailChange,
  onSmtpUseTLSChange,
  onBack,
  onSkipSMTP,
  onSubmit,
}) => (
  <form onSubmit={onSubmit} className="space-y-6">
    <ErrorBanner message={error} />
    <div className="text-left">
      <h4 className="mb-2 text-lg font-medium text-gray-800 dark:text-white">SMTP configuration</h4>
      <Text className="text-gray-800 dark:text-gray-300">Configure email delivery for this instance.</Text>
    </div>
    <SmtpConfigFields
      smtpHost={smtpHost}
      smtpPort={smtpPort}
      smtpUsername={smtpUsername}
      smtpPassword={smtpPassword}
      smtpFromName={smtpFromName}
      smtpFromEmail={smtpFromEmail}
      smtpUseTLS={smtpUseTLS}
      fieldErrors={fieldErrors}
      onSmtpHostChange={onSmtpHostChange}
      onSmtpPortChange={onSmtpPortChange}
      onSmtpUsernameChange={onSmtpUsernameChange}
      onSmtpPasswordChange={onSmtpPasswordChange}
      onSmtpFromNameChange={onSmtpFromNameChange}
      onSmtpFromEmailChange={onSmtpFromEmailChange}
      onSmtpUseTLSChange={onSmtpUseTLSChange}
    />

    <div className="flex flex-wrap items-center gap-3">
      <Button type="button" variant="outline" disabled={loading} onClick={onBack}>
        Back
      </Button>
      <div className="ml-auto flex flex-wrap justify-end gap-3">
        <Button type="button" variant="outline" disabled={loading} onClick={onSkipSMTP}>
          Do this later
        </Button>
        <Button type="submit" disabled={loading}>
          {loading ? "Saving..." : "Finish setup"}
        </Button>
      </div>
    </div>
  </form>
);
