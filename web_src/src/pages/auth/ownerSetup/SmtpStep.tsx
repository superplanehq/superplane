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
        <ErrorBanner message={error} />
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <Label className={fieldLabelClass}>{copy.host}</Label>
            <InputGroup>
              <Input
                value={form.host}
                onChange={(event) => setField("host", event.target.value)}
                placeholder="smtp.example.com"
                data-testid="owner-setup-smtp-host"
              />
            </InputGroup>
          </div>
          <div>
            <Label className={fieldLabelClass}>{copy.port}</Label>
            <InputGroup>
              <Input
                value={form.port}
                onChange={(event) => setField("port", event.target.value)}
                placeholder="587"
                data-testid="owner-setup-smtp-port"
              />
            </InputGroup>
          </div>
          <div>
            <Label className={fieldLabelClass}>{copy.username}</Label>
            <InputGroup>
              <Input
                value={form.username}
                onChange={(event) => setField("username", event.target.value)}
                placeholder="smtp-user"
              />
            </InputGroup>
          </div>
          <div>
            <Label className={fieldLabelClass}>{copy.password}</Label>
            <InputGroup>
              <Input
                type="password"
                value={form.password}
                onChange={(event) => setField("password", event.target.value)}
                placeholder={copy.password}
                className="ph-no-capture"
              />
            </InputGroup>
          </div>
          <div>
            <Label className={fieldLabelClass}>{copy.fromName}</Label>
            <InputGroup>
              <Input value={form.fromName} onChange={(event) => setField("fromName", event.target.value)} />
            </InputGroup>
          </div>
          <div>
            <Label className={fieldLabelClass}>{copy.fromEmail}</Label>
            <InputGroup>
              <Input
                type="email"
                value={form.fromEmail}
                onChange={(event) => setField("fromEmail", event.target.value)}
                placeholder="noreply@example.com"
                data-testid="owner-setup-smtp-from-email"
              />
            </InputGroup>
          </div>
        </div>
        <div className="flex items-center justify-between gap-4 pt-2">
          <div>
            <p className="text-[13px] font-medium">{copy.useTls}</p>
            <p className="mt-1 text-[12px] text-muted-foreground">{copy.useTlsHelp}</p>
          </div>
          <Switch checked={form.useTLS} onCheckedChange={(checked) => setField("useTLS", checked)} />
        </div>
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
