import { useState } from "react";
import { CheckCircle2, ChevronLeft, Circle } from "lucide-react";
import type { SuperplaneOrganizationsFleet } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useRunnerRegistration } from "@/hooks/useRunnerRegistration";
import { FleetWizardProgress } from "./FleetWizardProgress";
import { RunnerInstructions } from "./RunnerInstructions";

const steps = ["Lifecycle", "Setup method"] as const;

export function ConnectRunner({
  organizationId,
  fleet,
  onClose,
  refresh,
}: {
  organizationId: string;
  fleet: SuperplaneOrganizationsFleet;
  onClose: () => void;
  refresh: () => void;
}) {
  const [step, setStep] = useState(0);
  const { mode, registration, status, error, create, chooseMode, retryStatus } = useRunnerRegistration({
    organizationId,
    fleetId: fleet.id!,
    refresh,
  });
  return (
    <div className="space-y-5 text-[13px]">
      <Button variant="ghost" size="sm" onClick={onClose} className="px-0 text-muted-foreground">
        <ChevronLeft className="size-3.5" aria-hidden="true" />
        Back to fleet
      </Button>
      <div className="space-y-5 rounded-lg border border-border bg-card p-5">
        <div>
          <h2 className="text-base font-medium">Connect runner</h2>
          <p className="mt-1 text-muted-foreground">Connect a machine to {fleet.id}.</p>
        </div>
        <FleetWizardProgress steps={steps} current={step} />
        {step === 0 ? (
          <RunnerLifecycleStep mode={mode} onModeChange={chooseMode} onNext={() => setStep(1)} />
        ) : (
          <>
            <div>
              <h3 className="font-medium">Choose how to run it</h3>
              <p className="mt-1 text-muted-foreground">
                Prepare your machine, then generate a short-lived registration command.
              </p>
            </div>
            <RunnerInstructions fleet={fleet} registration={registration}>
              {(ready) => (
                <>
                  {status === "ready" && (
                    <Button size="sm" disabled={!ready} onClick={() => void create()}>
                      {error ? "Retry registration" : "Generate registration"}
                    </Button>
                  )}
                  {status === "creating" && <p role="status">Creating registration…</p>}
                </>
              )}
            </RunnerInstructions>
            <RegistrationStatus status={status} create={create} />
            {error && (
              <p role="alert" className="text-destructive">
                {error}
              </p>
            )}
            {error && status === "waiting" && (
              <Button size="sm" variant="outline" onClick={retryStatus}>
                Retry status check
              </Button>
            )}
            <div className="flex justify-between border-t border-border pt-4">
              {status !== "connected" && (
                <Button
                  size="sm"
                  variant="outline"
                  disabled={status === "creating"}
                  onClick={() => {
                    chooseMode(mode);
                    setStep(0);
                  }}
                >
                  Back
                </Button>
              )}
              <Button size="sm" variant={status === "connected" ? "default" : "ghost"} onClick={onClose}>
                {status === "connected" ? "Done" : "Cancel"}
              </Button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function RegistrationStatus({ status, create }: { status: string; create: (replacement: boolean) => Promise<void> }) {
  if (status === "connected") return <p role="status">Runner connected. Registration confirmed by the server.</p>;
  if (status !== "expired" && status !== "terminated") return null;
  return (
    <div className="space-y-3">
      <p role="status">
        {status === "expired"
          ? "Registration expired. Create a new registration and use its command."
          : "Runner terminated. Create a new registration."}
      </p>
      <Button size="sm" onClick={() => void create(true)}>
        Create new registration
      </Button>
    </div>
  );
}

function RunnerLifecycleStep({
  mode,
  onModeChange,
  onNext,
}: {
  mode: string;
  onModeChange: (value: string) => void;
  onNext: () => void;
}) {
  return (
    <>
      <div>
        <h3 className="font-medium">Choose the runner lifecycle</h3>
        <p className="mt-1 text-muted-foreground">
          Choose whether the runner exits after one task or stays available for more work.
        </p>
      </div>
      <div role="group" aria-label="Runner lifecycle" className="grid gap-3 sm:grid-cols-2">
        {(
          [
            ["ephemeral", "Ephemeral", "Run one task, then exit."],
            ["reusable", "Reusable", "Keep running and accept more tasks."],
          ] as const
        ).map(([value, label, description]) => (
          <Button
            key={value}
            type="button"
            variant="outline"
            aria-label={label}
            aria-pressed={mode === value}
            className={`h-auto flex-col items-start gap-1 whitespace-normal rounded-lg p-4 text-left ${mode === value ? "border-foreground/40 bg-muted dark:bg-muted ring-1 ring-foreground/20" : ""}`}
            onClick={() => onModeChange(value)}
          >
            <span className="flex w-full items-center justify-between gap-4">
              <span className="text-[13px] font-semibold">{label}</span>
              {mode === value ? (
                <CheckCircle2 className="size-4 text-foreground" aria-hidden="true" />
              ) : (
                <Circle className="size-4 text-muted-foreground" aria-hidden="true" />
              )}
            </span>
            <span className="text-[12px] font-normal text-muted-foreground">{description}</span>
            {mode === value && <span className="mt-2 text-[11px] font-medium">Selected</span>}
          </Button>
        ))}
      </div>
      <div className="flex justify-end border-t border-border pt-4">
        <Button size="sm" onClick={onNext}>
          Next
        </Button>
      </div>
    </>
  );
}
