import React, { useCallback, useEffect, useRef, useState } from "react";
import { FleetManagerWizardScreen } from "@/components/fleet-manager-setup/FleetManagerWizard";
import { Button } from "@/components/ui/button";
import { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";
import { FLEET_MANAGER_WIZARD_STEPS, type FleetManagerWizardStep } from "@/lib/fleetManagerWizard";
import { FirstRunShell } from "@/pages/factories/pages/onboarding/first-run/FirstRunShell";
import type { FirstRunChrome } from "@/pages/factories/pages/onboarding/first-run/firstRunTypes";
import type { FirstRunSphereProps } from "@/pages/factories/pages/onboarding/first-run/FirstRunSpherePane";
import { FleetManagerStep } from "./ownerSetup/FleetManagerStep";
import { LicenseStep } from "./ownerSetup/LicenseStep";
import { OWNER_SETUP_COPY } from "./ownerSetup/ownerSetupCopy";
import { OwnerStep } from "./ownerSetup/OwnerStep";
import { SmtpStep } from "./ownerSetup/SmtpStep";
import { useFinishInstallation } from "./ownerSetup/useFinishInstallation";
import { useOwnerAccountForm } from "./ownerSetup/useOwnerAccountForm";
import { useReportPageReady } from "@/hooks/useReportPageReady";
import { newOrganizationLandingPath } from "./newOrganizationLandingPath";

const OWNER_SETUP_STEPS = 8;
const OWNER_SETUP_PREVIEW: FirstRunSphereProps = {
  level: 0,
  art: {
    mode: "globe",
    background: "#b09532",
    arrowColor: "#eeede9",
    count: 400,
  },
};

type OwnerSetupStep = "owner" | "license" | "smtp" | "fleet-intro" | `fleet-${FleetManagerWizardStep}`;

const RESUMABLE_STEPS: OwnerSetupStep[] = [
  "license",
  "smtp",
  "fleet-intro",
  ...FLEET_MANAGER_WIZARD_STEPS.map((step) => `fleet-${step}` as OwnerSetupStep),
];

type SetupOwnerStatus = "disabled" | "required" | "in_progress" | "complete";

interface SetupOwnerStatusResponse {
  status: SetupOwnerStatus;
  step?: string;
  beacon_enabled?: boolean;
}

type OwnerSetupEntry =
  | { kind: "redirect"; to: string }
  | { kind: "owner"; beaconEnabled: boolean }
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

async function fetchSignedInAccount(): Promise<{ email: string; installation_admin: boolean } | null> {
  try {
    const response = await fetch("/account", { credentials: "include" });
    if (!response.ok) return null;
    const account = (await response.json()) as { email?: string; installation_admin?: boolean };
    if (!account.email) return null;
    return { email: account.email, installation_admin: account.installation_admin === true };
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
    return { kind: "owner", beaconEnabled: status.beacon_enabled === true };
  }

  const account = await fetchSignedInAccount();

  if (status.status === "complete") {
    return { kind: "redirect", to: account ? "/" : "/login" };
  }

  if (!account) {
    return { kind: "redirect", to: "/login?redirect=/setup" };
  }

  if (!account.installation_admin) {
    return { kind: "redirect", to: "/" };
  }

  const orgSlug = await fetchFirstOrganizationSlug();
  if (!orgSlug) {
    return { kind: "redirect", to: "/" };
  }

  const step = isResumableStep(status.step) ? status.step : "license";
  return { kind: "resume", step, email: account.email, orgSlug };
}

// Forward and back save the new step so a refresh resumes the same page.
// Clearing the step closes the wizard. The organization landing is not a
// resumable setup page.
async function persistSetupProgress(step: string): Promise<boolean> {
  try {
    const response = await fetch("/api/v1/setup-owner/progress", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      credentials: "include",
      body: JSON.stringify({ step }),
    });
    if (response.ok) return true;
    return step === "" && response.status === 409;
  } catch {
    return false;
  }
}

function useQueuedSetupProgress() {
  const queue = useRef(Promise.resolve());

  return useCallback((step: string) => {
    const save = queue.current.then(
      () => persistSetupProgress(step),
      () => persistSetupProgress(step),
    );
    queue.current = save.then(
      () => undefined,
      () => undefined,
    );
    return save;
  }, []);
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
  if (step === "fleet-values") return 7;
  if (step === "fleet-install") return 8;
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

interface OwnerSetupEntryActions {
  setEmail: (email: string) => void;
  setPendingOrganizationSlug: (slug: string | null) => void;
  setStep: (step: OwnerSetupStep) => void;
  setPhase: (phase: "loading" | "ready") => void;
  setBeaconEnabled: (enabled: boolean) => void;
}

function applyOwnerSetupEntry(entry: OwnerSetupEntry, actions: OwnerSetupEntryActions) {
  if (entry.kind === "redirect") {
    window.location.href = entry.to;
    return;
  }

  if (entry.kind === "owner") {
    actions.setBeaconEnabled(entry.beaconEnabled);
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

const OwnerSetup: React.FC = () => {
  const [phase, setPhase] = useState<"loading" | "ready">("loading");
  const [step, setStep] = useState<OwnerSetupStep>("owner");
  const [beaconEnabled, setBeaconEnabled] = useState(false);
  const [pendingOrganizationSlug, setPendingOrganizationSlug] = useState<string | null>(null);
  const activeFleetStep = fleetWizardStep(step);
  const fleetSetup = useFleetManagerSetup(
    activeFleetStep !== null,
    fleetManagerOrganizationId(pendingOrganizationSlug),
  );
  const saveSetupProgress = useQueuedSetupProgress();
  const { closeError, isClosing, finishInstallation } = useFinishInstallation({
    organizationSlug: pendingOrganizationSlug,
    saveSetupProgress,
    onClosed: (orgSlug) => {
      window.location.href = newOrganizationLandingPath(orgSlug);
    },
  });

  useReportPageReady(true);

  const onOwnerCreated = useCallback((next: OwnerSetupStep) => {
    setStep(next);
    writeSetupStepToAddress(next);
  }, []);

  const account = useOwnerAccountForm({
    setPendingOrganizationSlug,
    onOwnerCreated,
    shareInstallReport: beaconEnabled,
  });

  useEffect(() => {
    let cancelled = false;

    void resolveOwnerSetupEntry().then((entry) => {
      if (!cancelled) {
        applyOwnerSetupEntry(entry, {
          setEmail: account.setEmail,
          setPendingOrganizationSlug,
          setStep,
          setPhase,
          setBeaconEnabled,
        });
      }
    });

    return () => {
      cancelled = true;
    };
  }, [account.setEmail]);

  const goToStep = useCallback(
    (next: OwnerSetupStep) => {
      setStep(next);
      writeSetupStepToAddress(next);
      if (isResumableStep(next)) {
        void saveSetupProgress(next);
      }
    },
    [saveSetupProgress],
  );

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
      chrome={ownerSetupChrome({ step, email: account.email, goToStep })}
    >
      {closeError && (
        <div className="mb-6" role="alert" data-testid="owner-setup-close-error">
          <p className="text-sm text-destructive">{OWNER_SETUP_COPY.closeError}</p>
          <Button type="button" className="mt-3" onClick={finishInstallation} disabled={isClosing}>
            {OWNER_SETUP_COPY.closeRetry}
          </Button>
        </div>
      )}
      {step === "owner" && (
        <OwnerStep
          email={account.email}
          firstName={account.firstName}
          lastName={account.lastName}
          password={account.password}
          confirmPassword={account.confirmPassword}
          loading={account.loading}
          error={account.error}
          fieldErrors={account.fieldErrors}
          onEmailChange={account.setEmail}
          onFirstNameChange={account.setFirstName}
          onLastNameChange={account.setLastName}
          onPasswordChange={account.setPassword}
          onConfirmPasswordChange={account.setConfirmPassword}
          showInstallReport={beaconEnabled}
          shareOwnerIdentity={account.shareOwnerIdentity}
          onShareOwnerIdentityChange={account.setShareOwnerIdentity}
          onSubmit={account.handleOwnerSubmit}
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
    </FirstRunShell>
  );
};

export default OwnerSetup;
