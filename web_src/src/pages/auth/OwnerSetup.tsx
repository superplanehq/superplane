import React, { useState } from "react";
import { posthog, isPostHogEnabled } from "@/posthog";
import PostHogSurveyForm, { type PostHogSurvey } from "./PostHogSurveyForm";
import { OwnerStep } from "./ownerSetup/OwnerStep";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";
import { newOrganizationLandingPath } from "./newOrganizationLandingPath";

const OWNER_SETUP_SURVEY_NAME = "Owner Setup Survey";

function isEmailValid(email: string) {
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  return emailRegex.test(email);
}

function isPasswordValid(password: string) {
  if (password.length < 8) return false;
  if (!/[0-9]/.test(password)) return false;
  if (!/[A-Z]/.test(password)) return false;
  return true;
}

function validateOwnerFields(values: {
  email: string;
  firstName: string;
  lastName: string;
  password: string;
  confirmPassword: string;
}): Record<string, string> {
  const errors: Record<string, string> = {};

  if (!values.email.trim()) {
    errors.email = "Email is required.";
  } else if (!isEmailValid(values.email.trim())) {
    errors.email = "Please enter a valid email address.";
  }

  if (!values.firstName.trim()) {
    errors.firstName = "First name is required.";
  }

  if (!values.lastName.trim()) {
    errors.lastName = "Last name is required.";
  }

  if (!values.password) {
    errors.password = "Password is required.";
  } else if (!isPasswordValid(values.password)) {
    errors.password = "Password must be 8+ characters with at least 1 number and 1 capital letter.";
  }

  if (!values.confirmPassword) {
    errors.confirmPassword = "Please confirm your password.";
  } else if (values.confirmPassword !== values.password) {
    errors.confirmPassword = "Passwords do not match.";
  }

  return errors;
}

async function submitOwnerSetup(args: {
  email: string;
  firstName: string;
  lastName: string;
  password: string;
  setError: (error: string | null) => void;
  setLoading: (loading: boolean) => void;
  setPendingOrganizationSlug: (slug: string | null) => void;
  setActiveSurvey: (survey: PostHogSurvey | null) => void;
  setStep: (step: "owner" | "survey") => void;
}) {
  args.setError(null);
  args.setLoading(true);

  try {
    const response = await fetch("/api/v1/setup-owner", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        email: args.email.trim(),
        first_name: args.firstName.trim(),
        last_name: args.lastName.trim(),
        password: args.password,
      }),
    });

    if (!response.ok) {
      try {
        const data = await response.json();
        args.setError(data.message || "Failed to set up owner account");
      } catch {
        if (response.status === 409) {
          args.setError("This instance is already initialized.");
        } else {
          args.setError(`Failed to set up owner account (${response.status})`);
        }
      }
      return;
    }

    const data: { organization_id: string; organization_slug: string } = await response.json();
    const orgSlug = data.organization_slug;

    if (!isPostHogEnabled) {
      window.location.href = newOrganizationLandingPath(orgSlug);
      return;
    }

    args.setPendingOrganizationSlug(orgSlug);
    posthog.getActiveMatchingSurveys((surveys) => {
      const usableSurveys = (surveys as PostHogSurvey[]).filter(
        (survey) => Array.isArray(survey.questions) && survey.questions.length > 0,
      );

      const selectedSurvey =
        usableSurveys.find((survey) => survey.name === OWNER_SETUP_SURVEY_NAME) ?? usableSurveys[0];

      if (!selectedSurvey) {
        window.location.href = newOrganizationLandingPath(orgSlug);
        return;
      }

      args.setActiveSurvey(selectedSurvey);
      args.setStep("survey");
    });
  } catch {
    args.setError("Network error occurred");
  } finally {
    args.setLoading(false);
  }
}

const OwnerSetup: React.FC = () => {
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [step, setStep] = useState<"owner" | "survey">("owner");
  const [pendingOrganizationSlug, setPendingOrganizationSlug] = useState<string | null>(null);
  const [activeSurvey, setActiveSurvey] = useState<PostHogSurvey | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useReportPageReady(true);

  const handleOwnerSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const errors = validateOwnerFields({ email, firstName, lastName, password, confirmPassword });
    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) {
      return;
    }
    void submitOwnerSetup({
      email,
      firstName,
      lastName,
      password,
      setError,
      setLoading,
      setPendingOrganizationSlug,
      setActiveSurvey,
      setStep,
    });
  };

  return (
    <div
      className={cn("min-h-screen flex items-center justify-center bg-slate-100 px-4 py-8", appDarkModeClasses.surface)}
    >
      <div
        className={cn(
          "max-w-md w-full rounded-lg bg-white p-8 shadow-sm",
          appDarkModeClasses.modalEdge,
          appDarkModeClasses.surfaceRaised,
        )}
      >
        {step === "owner" && (
          <OwnerStep
            email={email}
            firstName={firstName}
            lastName={lastName}
            password={password}
            confirmPassword={confirmPassword}
            loading={loading}
            error={error}
            fieldErrors={fieldErrors}
            onEmailChange={setEmail}
            onFirstNameChange={setFirstName}
            onLastNameChange={setLastName}
            onPasswordChange={setPassword}
            onConfirmPasswordChange={setConfirmPassword}
            onSubmit={handleOwnerSubmit}
          />
        )}

        {step === "survey" && activeSurvey && pendingOrganizationSlug && (
          <PostHogSurveyForm survey={activeSurvey} redirectTo={newOrganizationLandingPath(pendingOrganizationSlug)} />
        )}
      </div>
    </div>
  );
};

export default OwnerSetup;
