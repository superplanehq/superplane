import React, { useState } from "react";
import { posthog, isPostHogEnabled } from "@/posthog";
import PostHogSurveyForm, { type PostHogSurvey } from "./PostHogSurveyForm";
import { OwnerStep } from "./ownerSetup/OwnerStep";
import { PrivateNetworkStep } from "./ownerSetup/PrivateNetworkStep";
import { SmtpPromptStep } from "./ownerSetup/SmtpPromptStep";
import { SmtpConfigStep } from "./ownerSetup/SmtpConfigStep";
import { useReportPageReady } from "@/hooks/useReportPageReady";

const OWNER_SETUP_SURVEY_NAME = "Owner Setup Survey";

const OwnerSetup: React.FC = () => {
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [smtpHost, setSmtpHost] = useState("");
  const [smtpPort, setSmtpPort] = useState("");
  const [smtpUsername, setSmtpUsername] = useState("");
  const [smtpPassword, setSmtpPassword] = useState("");
  const [smtpFromName, setSmtpFromName] = useState("");
  const [smtpFromEmail, setSmtpFromEmail] = useState("");
  const [smtpUseTLS, setSmtpUseTLS] = useState(true);
  const [allowPrivateNetworkAccess, setAllowPrivateNetworkAccess] = useState(false);
  const [step, setStep] = useState<"owner" | "privateNetwork" | "smtpPrompt" | "smtpConfig" | "survey">("owner");
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

  const validateSMTPFields = () => {
    const errors: Record<string, string> = {};

    if (!smtpHost.trim()) {
      errors.smtpHost = "SMTP host is required.";
    }

    if (!smtpPort.trim()) {
      errors.smtpPort = "SMTP port is required.";
    } else if (!/^[0-9]+$/.test(smtpPort.trim())) {
      errors.smtpPort = "SMTP port must be a number.";
    }

    if (!smtpFromEmail.trim()) {
      errors.smtpFromEmail = "SMTP from email is required.";
    } else if (!isEmailValid(smtpFromEmail.trim())) {
      errors.smtpFromEmail = "Please enter a valid from email address.";
    }

    if (smtpUsername.trim() && !smtpPassword) {
      errors.smtpPassword = "SMTP password is required when username is provided.";
    }

    setFieldErrors(errors);
    return errors;
  };

  const submitSetup = async (enableSMTP: boolean) => {
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
          smtp_enabled: enableSMTP,
          smtp_host: enableSMTP ? smtpHost.trim() : "",
          smtp_port: enableSMTP && smtpPort ? Number(smtpPort) : 0,
          smtp_username: enableSMTP ? smtpUsername.trim() : "",
          smtp_password: enableSMTP ? smtpPassword : "",
          smtp_from_name: enableSMTP ? smtpFromName.trim() : "",
          smtp_from_email: enableSMTP ? smtpFromEmail.trim() : "",
          smtp_use_tls: enableSMTP ? smtpUseTLS : false,
          allow_private_network_access: allowPrivateNetworkAccess,
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

  const handleOwnerNext = (e: React.FormEvent) => {
    e.preventDefault();
    const errors = validateOwnerFields();
    if (Object.keys(errors).length > 0) {
      return;
    }
    setStep("privateNetwork");
  };

  const handlePrivateNetworkNext = () => {
    setStep("smtpPrompt");
  };

  const handlePrivateNetworkBack = () => {
    setError(null);
    setFieldErrors({});
    setStep("owner");
  };

  const handleSkipSMTP = () => {
    setFieldErrors({});
    submitSetup(false);
  };

  const handleEnableSMTP = () => {
    setError(null);
    setFieldErrors({});
    setStep("smtpConfig");
  };

  const handleSMTPPromptBack = () => {
    setError(null);
    setFieldErrors({});
    setStep("privateNetwork");
  };

  const handleSMTPConfigBack = () => {
    setError(null);
    setFieldErrors({});
    setStep("smtpPrompt");
  };

  const handleSubmitSMTP = (e: React.FormEvent) => {
    e.preventDefault();
    const errors = validateSMTPFields();
    if (Object.keys(errors).length > 0) {
      return;
    }
    submitSetup(true);
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-slate-100 px-4 py-8">
      <div className="max-w-md w-full bg-white dark:bg-gray-900 rounded-lg outline outline-gray-950/10 shadow-sm p-8">
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
            onNext={handleOwnerNext}
          />
        )}

        {step === "privateNetwork" && (
          <PrivateNetworkStep
            allowPrivateNetworkAccess={allowPrivateNetworkAccess}
            loading={loading}
            error={error}
            onAllowPrivateNetworkAccessChange={setAllowPrivateNetworkAccess}
            onBack={handlePrivateNetworkBack}
            onNext={handlePrivateNetworkNext}
          />
        )}

        {step === "smtpPrompt" && (
          <SmtpPromptStep
            loading={loading}
            error={error}
            onBack={handleSMTPPromptBack}
            onEnableSMTP={handleEnableSMTP}
            onSkipSMTP={handleSkipSMTP}
          />
        )}

        {step === "survey" && activeSurvey && pendingOrganizationId && (
          <PostHogSurveyForm survey={activeSurvey} redirectTo={`/${pendingOrganizationId}`} />
        )}

        {step === "smtpConfig" && (
          <SmtpConfigStep
            smtpHost={smtpHost}
            smtpPort={smtpPort}
            smtpUsername={smtpUsername}
            smtpPassword={smtpPassword}
            smtpFromName={smtpFromName}
            smtpFromEmail={smtpFromEmail}
            smtpUseTLS={smtpUseTLS}
            loading={loading}
            error={error}
            fieldErrors={fieldErrors}
            onSmtpHostChange={setSmtpHost}
            onSmtpPortChange={setSmtpPort}
            onSmtpUsernameChange={setSmtpUsername}
            onSmtpPasswordChange={setSmtpPassword}
            onSmtpFromNameChange={setSmtpFromName}
            onSmtpFromEmailChange={setSmtpFromEmail}
            onSmtpUseTLSChange={setSmtpUseTLS}
            onBack={handleSMTPConfigBack}
            onSkipSMTP={handleSkipSMTP}
            onSubmit={handleSubmitSMTP}
          />
        )}
      </div>
    </div>
  );
};

export default OwnerSetup;
