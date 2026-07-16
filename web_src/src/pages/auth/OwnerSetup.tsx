import React, { useState } from "react";
import { posthog, isPostHogEnabled } from "@/posthog";
import PostHogSurveyForm, { type PostHogSurvey } from "./PostHogSurveyForm";
import { OwnerStep } from "./ownerSetup/OwnerStep";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { cn } from "@/lib/utils";

const OWNER_SETUP_SURVEY_NAME = "Owner Setup Survey";

const OwnerSetup: React.FC = () => {
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [step, setStep] = useState<"owner" | "survey">("owner");
  const [pendingOrganizationId, setPendingOrganizationId] = useState<string | null>(null);
  const [activeSurvey, setActiveSurvey] = useState<PostHogSurvey | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});

  useReportPageReady(true);

  const isEmailValid = (email: string) => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
    return emailRegex.test(email);
  };

  const isPasswordValid = (password: string) => {
    if (password.length < 8) return false;
    if (!/[0-9]/.test(password)) return false;
    if (!/[A-Z]/.test(password)) return false;
    return true;
  };

  const validateOwnerFields = () => {
    const errors: Record<string, string> = {};

    if (!email.trim()) {
      errors.email = "Email is required.";
    } else if (!isEmailValid(email.trim())) {
      errors.email = "Please enter a valid email address.";
    }

    if (!firstName.trim()) {
      errors.firstName = "First name is required.";
    }

    if (!lastName.trim()) {
      errors.lastName = "Last name is required.";
    }

    if (!password) {
      errors.password = "Password is required.";
    } else if (!isPasswordValid(password)) {
      errors.password = "Password must be 8+ characters with at least 1 number and 1 capital letter.";
    }

    if (!confirmPassword) {
      errors.confirmPassword = "Please confirm your password.";
    } else if (confirmPassword !== password) {
      errors.confirmPassword = "Passwords do not match.";
    }

    setFieldErrors(errors);
    return errors;
  };

  const submitSetup = async () => {
    setError(null);

    setLoading(true);

    try {
      const response = await fetch("/api/v1/setup-owner", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        body: JSON.stringify({
          email: email.trim(),
          first_name: firstName.trim(),
          last_name: lastName.trim(),
          password,
        }),
      });

      if (!response.ok) {
        try {
          const data = await response.json();
          setError(data.message || "Failed to set up owner account");
        } catch {
          if (response.status === 409) {
            setError("This instance is already initialized.");
          } else {
            setError(`Failed to set up owner account (${response.status})`);
          }
        }
        return;
      }

      const data: { organization_id: string } = await response.json();
      const orgId = data.organization_id;

      if (!isPostHogEnabled) {
        window.location.href = `/${orgId}`;
        return;
      }

      setPendingOrganizationId(orgId);
      posthog.getActiveMatchingSurveys((surveys) => {
        const usableSurveys = (surveys as PostHogSurvey[]).filter(
          (survey) => Array.isArray(survey.questions) && survey.questions.length > 0,
        );

        const selectedSurvey =
          usableSurveys.find((survey) => survey.name === OWNER_SETUP_SURVEY_NAME) ?? usableSurveys[0];

        if (!selectedSurvey) {
          window.location.href = `/${orgId}`;
          return;
        }

        setActiveSurvey(selectedSurvey);
        setStep("survey");
      });
    } catch {
      setError("Network error occurred");
    } finally {
      setLoading(false);
    }
  };

  const handleOwnerSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const errors = validateOwnerFields();
    if (Object.keys(errors).length > 0) {
      return;
    }
    submitSetup();
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

        {step === "survey" && activeSurvey && pendingOrganizationId && (
          <PostHogSurveyForm survey={activeSurvey} redirectTo={`/${pendingOrganizationId}`} />
        )}
      </div>
    </div>
  );
};

export default OwnerSetup;
