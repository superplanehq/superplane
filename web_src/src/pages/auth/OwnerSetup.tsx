import React, { useCallback, useEffect, useState } from "react";
import { FleetManagerWizardScreen } from "@/components/fleet-manager-setup/FleetManagerWizard";
import { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";
import { FLEET_MANAGER_WIZARD_STEPS, type FleetManagerWizardStep } from "@/lib/fleetManagerWizard";
import { posthog, isPostHogEnabled } from "@/posthog";
import { FirstRunShell } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import type { FirstRunChrome } from "@/pages/factories/pages/onboarding/first-run/firstRunTypes";
import type { FirstRunSphereProps } from "@/pages/factories/pages/onboarding/first-run/FirstRunSpherePane";
import PostHogSurveyForm, { type PostHogSurvey } from "./PostHogSurveyForm";
import { FleetManagerStep } from "./ownerSetup/FleetManagerStep";
import { LicenseStep } from "./ownerSetup/LicenseStep";
import { OwnerStep } from "./ownerSetup/OwnerStep";
import { SmtpStep } from "./ownerSetup/SmtpStep";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { newOrganizationLandingPath } from "./newOrganizationLandingPath";

const OWNER_SETUP_SURVEY_NAME = "Owner Setup Survey";
const OWNER_SETUP_STEPS = 7;
const OWNER_SETUP_PREVIEW: FirstRunSphereProps = {
  level: 0,
  art: {
    mode: "globe",
    background: "#b09532",
    arrowColor: "#eeede9",
    count: 400,
  },
};

type OwnerSetupStep = "owner" | "license" | "smtp" | "fleet-intro" | `fleet-${FleetManagerWizardStep}` | "survey";

const RESUMABLE_STEPS: OwnerSetupStep[] = [
  "license",
  "smtp",
  "fleet-intro",
  "fleet-provider",
  "fleet-fleets",
  "fleet-install",
];

type SetupOwnerStatus = "disabled" | "required" | "in_progress" | "complete";

interface SetupOwnerStatusResponse {
  status: SetupOwnerStatus;
  step?: string;
}

type OwnerSetupEntry =
  | { kind: "redirect"; to: string }
  | { kind: "owner" }
  | { kind: "resume"; step: OwnerSetupStep; email: string; orgSlug: string };

function isResumableStep(step: string | undefined): step is OwnerSetupStep {
  return (RESUMABLE_STEPS as string[]).includes(step ?? "");
}

async function fetchSetupOwnerStatus(): Promise<SetupOwnerStatusResponse> {
  const response = await fetch("/api/v1/setup-owner/status", { credentials: "include" });
  if (!response.ok) {
    throw new Error(`Failed to load setup status (${response.status})`);
  }
  return (await response.json()) as SetupOwnerStatusResponse;
}

async function fetchSignedInAccount(): Promise<{ email: string } | null> {
  try {
    const response = await fetch("/account", { credentials: "include" });
    if (!response.ok) return null;
    return (await response.json()) as { email: string };
  } catch {
    return null;
  }
}

// The organization slug always comes from the signed-in account. The
// anonymous status endpoint never returns organization data.
async function fetchFirstOrganizationSlug(): Promise<string | null> {
  try {
    const response = await fetch("/organizations", { credentials: "include" });
    if (!response.ok) return null;
    const organizations = (await response.json()) as { slug: string }[];
    return organizations[0]?.slug ?? null;
  } catch {
    return null;
  }
}

async function resolveOwnerSetupEntry(): Promise<OwnerSetupEntry> {
  let status: SetupOwnerStatusResponse;
  try {
    status = await fetchSetupOwnerStatus();
  } catch {
    return { kind: "redirect", to: "/login" };
  }

  if (status.status === "disabled") {
    return { kind: "redirect", to: "/login" };
  }

  if (status.status === "required") {
    return { kind: "owner" };
  }

  const account = await fetchSignedInAccount();

  if (status.status === "complete") {
    return { kind: "redirect", to: account ? "/" : "/login" };
  }

  if (!account) {
    return { kind: "redirect", to: "/login?redirect=/setup" };
  }

  const orgSlug = await fetchFirstOrganizationSlug();
  if (!orgSlug) {
    return { kind: "redirect", to: "/" };
  }

  const step = isResumableStep(status.step) ? status.step : "license";
  return { kind: "resume", step, email: account.email, orgSlug };
}

// Forward and back save the new step so a refresh resumes the same page.
// Clearing the step closes the wizard; the survey and the organization
// landing are not resumable setup pages.
async function persistSetupProgress(step: string) {
  try {
    await fetch("/api/v1/setup-owner/progress", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ step }),
    });
  } catch {
    // Progress persistence is best-effort. The next load reconciles.
  }
}

function writeSetupStepToAddress(step: OwnerSetupStep) {
  if (!isResumableStep(step)) return;
  window.history.replaceState(window.history.state, "", `/setup?step=${step}`);
}

function ownerSetupStepNumber(step: OwnerSetupStep) {
  if (step === "owner") return 1;
  if (step === "license") return 2;
  if (step === "smtp") return 3;
  if (step === "fleet-intro") return 4;
  if (step === "fleet-provider") return 5;
  if (step === "fleet-fleets") return 6;
  if (step === "fleet-install") return 7;
  return OWNER_SETUP_STEPS;
}

function fleetManagerOrganizationId(slug: string | null) {
  return slug ?? "";
}

function fleetWizardStep(step: OwnerSetupStep): FleetManagerWizardStep | null {
  if (!step.startsWith("fleet-") || step === "fleet-intro") return null;
  return step.slice("fleet-".length) as FleetManagerWizardStep;
}

function fleetStepBefore(step: OwnerSetupStep): OwnerSetupStep {
  const active = fleetWizardStep(step);
  if (!active) return "fleet-intro";
  const index = FLEET_MANAGER_WIZARD_STEPS.indexOf(active);
  if (index <= 0) return "fleet-intro";
  return `fleet-${FLEET_MANAGER_WIZARD_STEPS[index - 1]}`;
}

function ownerSetupBackTarget(step: OwnerSetupStep): OwnerSetupStep | null {
  if (fleetWizardStep(step)) return fleetStepBefore(step);
  if (step === "fleet-intro") return "smtp";
  if (step === "smtp") return "license";
  return null;
}

function OwnerFleetSetupSteps({
  step,
  pendingOrganizationSlug,
  activeFleetStep,
  fleetStepIndex,
  fleetSetup,
  previousFleetStep,
  finishInstallation,
  setStep,
}: {
  step: OwnerSetupStep;
  pendingOrganizationSlug: string | null;
  activeFleetStep: FleetManagerWizardStep | null;
  fleetStepIndex: number;
  fleetSetup: ReturnType<typeof useFleetManagerSetup>;
  previousFleetStep: () => void;
  finishInstallation: () => void;
  setStep: (step: OwnerSetupStep) => void;
}) {
  if (!pendingOrganizationSlug) return null;
  if (step === "fleet-intro") {
    return <FleetManagerStep onContinue={() => setStep("fleet-provider")} onSkip={finishInstallation} />;
  }
  if (!activeFleetStep) return null;

  return (
    <FleetManagerWizardScreen
      step={activeFleetStep}
      setup={fleetSetup}
      onBack={previousFleetStep}
      onNext={() => {
        const next = FLEET_MANAGER_WIZARD_STEPS[fleetStepIndex + 1];
        if (next) setStep(`fleet-${next}`);
      }}
      onFinish={finishInstallation}
      onFinishLater={finishInstallation}
    />
  );
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
  onOwnerCreated: (step: OwnerSetupStep) => void;
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
    args.onOwnerCreated("license");
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
  if (!isPostHogEnabled()) {
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

interface OwnerSetupEntryActions {
  setEmail: (email: string) => void;
  setPendingOrganizationSlug: (slug: string | null) => void;
  setStep: (step: OwnerSetupStep) => void;
  setPhase: (phase: "loading" | "ready") => void;
}

function applyOwnerSetupEntry(entry: OwnerSetupEntry, actions: OwnerSetupEntryActions) {
  if (entry.kind === "redirect") {
    window.location.href = entry.to;
    return;
  }

  if (entry.kind === "resume") {
    actions.setEmail(entry.email);
    actions.setPendingOrganizationSlug(entry.orgSlug);
    actions.setStep(entry.step);
    writeSetupStepToAddress(entry.step);
  }

  actions.setPhase("ready");
}

function ownerSetupChrome(args: {
  step: OwnerSetupStep;
  email: string;
  goToStep: (step: OwnerSetupStep) => void;
}): FirstRunChrome | undefined {
  if (args.step === "survey") return undefined;

  const afterOwner = args.step !== "owner";
  const backTarget = ownerSetupBackTarget(args.step);

  return {
    stepIndex: ownerSetupStepNumber(args.step) - 1,
    stepCount: OWNER_SETUP_STEPS,
    email: afterOwner ? args.email : undefined,
    onLogOut: afterOwner ? () => (window.location.href = "/logout?redirect=/setup") : undefined,
    onBack: backTarget ? () => args.goToStep(backTarget) : undefined,
  };
}

function handleOwnerFormSubmit(
  e: React.FormEvent,
  values: { email: string; firstName: string; lastName: string; password: string; confirmPassword: string },
  actions: {
    setFieldErrors: (errors: Record<string, string>) => void;
    setError: (error: string | null) => void;
    setLoading: (loading: boolean) => void;
    setPendingOrganizationSlug: (slug: string | null) => void;
    onOwnerCreated: (step: OwnerSetupStep) => void;
  },
) {
  e.preventDefault();
  const errors = validateOwnerFields(values);
  actions.setFieldErrors(errors);
  if (Object.keys(errors).length > 0) {
    return;
  }
  void submitOwnerSetup({ ...values, ...actions });
}

const OwnerSetup: React.FC = () => {
  const [email, setEmail] = useState("");
  const [firstName, setFirstName] = useState("");
  const [lastName, setLastName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [phase, setPhase] = useState<"loading" | "ready">("loading");
  const [step, setStep] = useState<OwnerSetupStep>("owner");
  const [pendingOrganizationSlug, setPendingOrganizationSlug] = useState<string | null>(null);
  const [activeSurvey, setActiveSurvey] = useState<PostHogSurvey | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const activeFleetStep = fleetWizardStep(step);
  const fleetSetup = useFleetManagerSetup(
    activeFleetStep !== null,
    fleetManagerOrganizationId(pendingOrganizationSlug),
  );

  useReportPageReady(true);

  useEffect(() => {
    let cancelled = false;

    void resolveOwnerSetupEntry().then((entry) => {
      if (!cancelled) {
        applyOwnerSetupEntry(entry, { setEmail, setPendingOrganizationSlug, setStep, setPhase });
      }
    });

    return () => {
      cancelled = true;
    };
  }, []);

  const goToStep = useCallback((next: OwnerSetupStep) => {
    setStep(next);
    writeSetupStepToAddress(next);
    if (isResumableStep(next)) {
      void persistSetupProgress(next);
    }
  }, []);

  // The setup-owner request already saved this step in its transaction,
  // so advancing after account creation does not persist again.
  const onOwnerCreated = useCallback((next: OwnerSetupStep) => {
    setStep(next);
    writeSetupStepToAddress(next);
  }, []);

  const handleOwnerSubmit = (e: React.FormEvent) =>
    handleOwnerFormSubmit(
      e,
      { email, firstName, lastName, password, confirmPassword },
      { setFieldErrors, setError, setLoading, setPendingOrganizationSlug, onOwnerCreated },
    );

  const finishInstallation = useCallback(() => {
    if (!pendingOrganizationSlug) return;
    void persistSetupProgress("").then(() => {
      continueAfterOwnerSetup({ orgSlug: pendingOrganizationSlug, setActiveSurvey, setStep });
    });
  }, [pendingOrganizationSlug]);

  const fleetStepIndex = activeFleetStep ? FLEET_MANAGER_WIZARD_STEPS.indexOf(activeFleetStep) : -1;

  if (phase === "loading") {
    return null;
  }

  return (
    <FirstRunShell
      testId="owner-setup"
      contentSpacing="compact"
      visual="preview"
      sphere={OWNER_SETUP_PREVIEW}
      chrome={ownerSetupChrome({ step, email, goToStep })}
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

      {step === "license" && pendingOrganizationSlug && <LicenseStep onContinue={() => goToStep("smtp")} />}

      {step === "smtp" && pendingOrganizationSlug && (
        <SmtpStep onContinue={() => goToStep("fleet-intro")} onSkip={() => goToStep("fleet-intro")} />
      )}

      <OwnerFleetSetupSteps
        step={step}
        pendingOrganizationSlug={pendingOrganizationSlug}
        activeFleetStep={activeFleetStep}
        fleetStepIndex={fleetStepIndex}
        fleetSetup={fleetSetup}
        previousFleetStep={() => goToStep(fleetStepBefore(step))}
        finishInstallation={finishInstallation}
        setStep={goToStep}
      />

      {step === "survey" && activeSurvey && pendingOrganizationSlug && (
        <PostHogSurveyForm survey={activeSurvey} redirectTo={newOrganizationLandingPath(pendingOrganizationSlug)} />
      )}
    </FirstRunShell>
  );
};

export default OwnerSetup;
