import React, { useCallback, useState } from "react";
import { posthog, isPostHogEnabled } from "@/posthog";
import { FirstRunShell } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import PostHogSurveyForm, { type PostHogSurvey } from "./PostHogSurveyForm";
import { FleetManagerStep } from "./ownerSetup/FleetManagerStep";
import { LicenseStep } from "./ownerSetup/LicenseStep";
import { OWNER_SETUP_COPY } from "./ownerSetup/ownerSetupCopy";
import { OwnerSetupPane } from "./ownerSetup/OwnerSetupPane";
import { OwnerStep } from "./ownerSetup/OwnerStep";
import { SmtpStep } from "./ownerSetup/SmtpStep";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { newOrganizationLandingPath } from "./newOrganizationLandingPath";

const OWNER_SETUP_SURVEY_NAME = "Owner Setup Survey";
const OWNER_SETUP_STEPS = 4;

type OwnerSetupStep = "owner" | "license" | "smtp" | "fleet" | "survey";

function ownerSetupStepNumber(step: OwnerSetupStep) {
  if (step === "owner") return 1;
  if (step === "license") return 2;
  if (step === "smtp") return 3;
  if (step === "fleet") return 4;
  return OWNER_SETUP_STEPS;
}

function ownerSetupCaption(step: OwnerSetupStep) {
  if (step === "license") return OWNER_SETUP_COPY.pane.license;
  if (step === "smtp") return OWNER_SETUP_COPY.pane.smtp;
  if (step === "fleet") return OWNER_SETUP_COPY.pane.fleet;
  return OWNER_SETUP_COPY.pane.owner;
}

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
  setStep: (step: OwnerSetupStep) => void;
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
    args.setPendingOrganizationSlug(data.organization_slug);
    args.setStep("license");
  } catch {
    args.setError("Network error occurred");
  } finally {
    args.setLoading(false);
  }
}

function continueAfterOwnerSetup(args: {
  orgSlug: string;
  setActiveSurvey: (survey: PostHogSurvey | null) => void;
  setStep: (step: OwnerSetupStep) => void;
}) {
  if (!isPostHogEnabled) {
    window.location.href = newOrganizationLandingPath(args.orgSlug);
    return;
  }

  posthog.getActiveMatchingSurveys((surveys) => {
    const usableSurveys = (surveys as PostHogSurvey[]).filter(
      (survey) => Array.isArray(survey.questions) && survey.questions.length > 0,
    );

    const selectedSurvey = usableSurveys.find((survey) => survey.name === OWNER_SETUP_SURVEY_NAME) ?? usableSurveys[0];

    if (!selectedSurvey) {
      window.location.href = newOrganizationLandingPath(args.orgSlug);
      return;
    }

    args.setActiveSurvey(selectedSurvey);
    args.setStep("survey");
  });
}

const OwnerSetup: React.FC = () => {
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [step, setStep] = useState<OwnerSetupStep>("owner");
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
      setStep,
    });
  };

  const finishInstallation = useCallback(() => {
    if (pendingOrganizationSlug) {
      continueAfterOwnerSetup({ orgSlug: pendingOrganizationSlug, setActiveSurvey, setStep });
    }
  }, [pendingOrganizationSlug]);

  const afterOwner = step !== "owner" && step !== "survey";

  return (
    <FirstRunShell
      testId="owner-setup"
      contentSpacing="compact"
      chrome={
        step === "survey"
          ? undefined
          : {
              stepIndex: ownerSetupStepNumber(step) - 1,
              stepCount: OWNER_SETUP_STEPS,
              email: afterOwner ? email : undefined,
              onLogOut: afterOwner ? () => (window.location.href = "/logout") : undefined,
            }
      }
      aside={
        <OwnerSetupPane
          caption={ownerSetupCaption(step)}
          step={ownerSetupStepNumber(step)}
          stepCount={OWNER_SETUP_STEPS}
        />
      }
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

      {step === "license" && pendingOrganizationSlug && <LicenseStep onContinue={() => setStep("smtp")} />}

      {step === "smtp" && pendingOrganizationSlug && (
        <SmtpStep onContinue={() => setStep("fleet")} onSkip={() => setStep("fleet")} />
      )}

      {step === "fleet" && pendingOrganizationSlug && (
        <FleetManagerStep onContinue={finishInstallation} onSkip={finishInstallation} />
      )}

      {step === "survey" && activeSurvey && pendingOrganizationSlug && (
        <PostHogSurveyForm survey={activeSurvey} redirectTo={newOrganizationLandingPath(pendingOrganizationSlug)} />
      )}
    </FirstRunShell>
  );
};

export default OwnerSetup;
