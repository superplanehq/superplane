import { useState, type FormEvent } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { recordLastUsedLoginMethod } from "@/lib/lastUsedLoginMethod";

import { discoverSso, ssoStartHref, type SsoDiscoveryResult } from "./discover";

const DOMAIN_UNAVAILABLE = "SSO is not set up for this email domain.";
const DISCOVERY_FAILED = "Single Sign-On could not be checked. Try again.";

export function ContinueWithSso({
  redirect,
  lastUsed,
  onLastUsedHint,
  startOpen = false,
  initialError = null,
  discover = discoverSso,
}: {
  redirect: string;
  lastUsed: boolean;
  onLastUsedHint?: string;
  startOpen?: boolean;
  initialError?: string | null;
  discover?: (email: string) => Promise<SsoDiscoveryResult>;
}) {
  const [open, setOpen] = useState(startOpen);
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(initialError);
  const [loading, setLoading] = useState(false);

  const submit = async (event: FormEvent) => {
    event.preventDefault();
    setError(null);
    if (!email.trim()) {
      setError("Email is required.");
      return;
    }
    setLoading(true);
    const discovery = await discover(email.trim());
    setLoading(false);
    if (discovery.status === "error") {
      setError(DISCOVERY_FAILED);
      return;
    }
    if (discovery.status !== "available") {
      setError(DOMAIN_UNAVAILABLE);
      return;
    }
    recordLastUsedLoginMethod("sso");
    window.location.href = ssoStartHref(discovery.startUrl, email.trim(), redirect);
  };

  if (!open) {
    return (
      <div>
        <Button type="button" variant="outline" className="w-full justify-center" onClick={() => setOpen(true)}>
          Continue with SSO
        </Button>
        {lastUsed && onLastUsedHint ? <p className="mt-2 text-center text-xs text-gray-500">{onLastUsedHint}</p> : null}
      </div>
    );
  }

  return (
    <form className="space-y-3" onSubmit={submit} data-testid="sso-email-step">
      <div className="space-y-2">
        <Label htmlFor="sso-email">Work email</Label>
        <Input
          id="sso-email"
          type="email"
          autoComplete="username"
          value={email}
          onChange={(event) => setEmail(event.target.value)}
        />
      </div>
      {error ? (
        <p className="text-sm text-red-600" role="alert">
          {error}
        </p>
      ) : null}
      <Button type="submit" className="w-full" disabled={loading}>
        Continue with SSO
      </Button>
    </form>
  );
}

export function SsoOrganizationPrompt({
  required,
  onContinue,
  onUseAnother,
}: {
  required: boolean;
  onContinue: () => void;
  onUseAnother: () => void;
}) {
  return (
    <div className="space-y-3 rounded-md border p-4" data-testid="sso-organization-prompt">
      <p className="text-sm">{required ? "Your organization requires SSO." : "Your organization uses SSO."}</p>
      <Button type="button" className="w-full" onClick={onContinue}>
        Continue with SSO
      </Button>
      <Button type="button" variant="outline" className="w-full" onClick={onUseAnother}>
        Use another method
      </Button>
    </div>
  );
}
