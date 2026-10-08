import { Input, InputGroup } from "@/components/Input/input";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { FirstRunHeading } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import { useState } from "react";

import { ErrorBanner } from "./ErrorBanner";
import { OWNER_SETUP_COPY } from "./ownerSetupCopy";

const copy = OWNER_SETUP_COPY.smtp;
const fieldLabelClass = "mb-2 block text-left text-[13px] font-medium";

type SMTPForm = {
  host: string;
  port: string;
  username: string;
  password: string;
  fromName: string;
  fromEmail: string;
  useTLS: boolean;
};

const emptyForm: SMTPForm = {
  host: "",
  port: "587",
  username: "",
  password: "",
  fromName: "SuperPlane",
  fromEmail: "",
  useTLS: true,
};

function smtpRequestBody(form: SMTPForm) {
  const port = Number(form.port.trim());
  return {
    smtp_enabled: true,
    smtp_host: form.host.trim(),
    smtp_port: Number.isFinite(port) ? port : 0,
    smtp_username: form.username.trim(),
    smtp_from_name: form.fromName.trim(),
    smtp_from_email: form.fromEmail.trim(),
    smtp_use_tls: form.useTLS,
    ...(form.password !== "" ? { smtp_password: form.password } : {}),
  };
}

function smtpFormValid(form: SMTPForm) {
  return form.host.trim() !== "" && form.port.trim() !== "" && form.fromEmail.trim() !== "";
}

function SmtpField({
  label,
  value,
  onChange,
  type,
  placeholder,
  testId,
  className,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  type?: string;
  placeholder?: string;
  testId?: string;
  className?: string;
}) {
  return (
    <div>
      <Label className={fieldLabelClass}>{label}</Label>
      <InputGroup>
        <Input
          type={type}
          value={value}
          onChange={(event) => onChange(event.target.value)}
          placeholder={placeholder}
          data-testid={testId}
          className={className}
        />
      </InputGroup>
    </div>
  );
}

function SmtpFormFields({
  form,
  error,
  onChange,
}: {
  form: SMTPForm;
  error: string | null;
  onChange: (field: keyof SMTPForm, value: boolean | string) => void;
}) {
  return (
    <>
      <ErrorBanner message={error} />
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <SmtpField
          label={copy.host}
          value={form.host}
          onChange={(value) => onChange("host", value)}
          placeholder="smtp.example.com"
          testId="owner-setup-smtp-host"
        />
        <SmtpField
          label={copy.port}
          value={form.port}
          onChange={(value) => onChange("port", value)}
          placeholder="587"
          testId="owner-setup-smtp-port"
        />
        <SmtpField
          label={copy.username}
          value={form.username}
          onChange={(value) => onChange("username", value)}
          placeholder="smtp-user"
        />
        <SmtpField
          label={copy.password}
          value={form.password}
          onChange={(value) => onChange("password", value)}
          type="password"
          placeholder={copy.password}
          className="ph-no-capture"
        />
        <SmtpField label={copy.fromName} value={form.fromName} onChange={(value) => onChange("fromName", value)} />
        <SmtpField
          label={copy.fromEmail}
          value={form.fromEmail}
          onChange={(value) => onChange("fromEmail", value)}
          type="email"
          placeholder="noreply@example.com"
          testId="owner-setup-smtp-from-email"
        />
      </div>
      <div className="flex items-center justify-between gap-4 pt-2">
        <div>
          <p className="text-[13px] font-medium">{copy.useTls}</p>
          <p className="mt-1 text-[12px] text-muted-foreground">{copy.useTlsHelp}</p>
        </div>
        <Switch checked={form.useTLS} onCheckedChange={(checked) => onChange("useTLS", checked)} />
      </div>
    </>
  );
}

export function SmtpStep({ onContinue, onSkip }: { onContinue: () => void; onSkip: () => void }) {
  const [form, setForm] = useState<SMTPForm>(emptyForm);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const setField = (field: keyof SMTPForm, value: boolean | string) => {
    setForm((current) => ({ ...current, [field]: value }));
  };

  const save = async () => {
    if (!smtpFormValid(form)) {
      setError(copy.required);
      return;
    }
    setError(null);
    setSaving(true);
    try {
      const response = await fetch("/admin/api/installation/network-settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify(smtpRequestBody(form)),
      });
      if (!response.ok) {
        throw new Error(copy.error);
      }
      onContinue();
    } catch {
      setError(copy.error);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div data-testid="owner-setup-smtp">
      <FirstRunHeading headline={copy.headline} size="display">
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <form
        className="mt-8 space-y-4 text-left"
        onSubmit={(event) => {
          event.preventDefault();
          void save();
        }}
      >
        <SmtpFormFields form={form} error={error} onChange={setField} />
        <div className="mt-4 flex flex-wrap items-center gap-3">
          <Button type="submit" className="min-w-40" disabled={saving}>
            {saving ? copy.saving : copy.save}
          </Button>
          <Button type="button" variant="ghost" data-testid="owner-setup-smtp-skip" disabled={saving} onClick={onSkip}>
            {copy.skip}
          </Button>
        </div>
      </form>
    </div>
  );
}
