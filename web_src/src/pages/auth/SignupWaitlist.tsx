import type React from "react";
import { useState } from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { submitSignupWaitlistEmail } from "@/lib/hubspotForms";
import { Text } from "@/components/Text/text";
import { getSignupWaitlistConfig } from "@/lib/signupWaitlistConfig";

type SignupWaitlistStatus = "idle" | "submitting" | "submitted" | "failed";

export const SignupWaitlist: React.FC = () => {
  const hubSpotConfig = getSignupWaitlistConfig();
  const [email, setEmail] = useState("");
  const [status, setStatus] = useState<SignupWaitlistStatus>("idle");

  const handleSubmit = async (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();

    const trimmedEmail = email.trim();
    if (!hubSpotConfig || !trimmedEmail || status === "submitting") {
      return;
    }

    setStatus("submitting");
    try {
      await submitSignupWaitlistEmail(hubSpotConfig, trimmedEmail);
      setEmail("");
      setStatus("submitted");
    } catch {
      setStatus("failed");
    }
  };

  return (
    <div className="space-y-4">
      <Text className="text-left text-sm leading-6 text-gray-600">
        We are opening access gradually while demand is high.
        {hubSpotConfig && " Leave your email and we will send an invite as capacity opens."}
      </Text>

      {hubSpotConfig && (
        <form onSubmit={handleSubmit} className="space-y-3">
          <div className="space-y-2">
            <Label htmlFor="signup-waitlist-email">Email</Label>
            <Input
              id="signup-waitlist-email"
              type="email"
              value={email}
              onChange={(event) => {
                setEmail(event.target.value);
                if (status !== "submitting") {
                  setStatus("idle");
                }
              }}
              placeholder="you@example.com"
              required
              autoComplete="email"
              data-1p-ignore
            />
          </div>

          <LoadingButton
            type="submit"
            className="w-full"
            loading={status === "submitting"}
            loadingText="Saving..."
            disabled={!email.trim()}
          >
            Notify me
          </LoadingButton>

          {status === "submitted" && (
            <p className="text-left text-sm leading-6 text-gray-700" role="status">
              You are on the waitlist. We will email you when access opens.
            </p>
          )}

          {status === "failed" && (
            <p className="text-left text-sm leading-6 text-red-600" role="alert">
              We could not save your email. Please try again.
            </p>
          )}
        </form>
      )}
    </div>
  );
};
