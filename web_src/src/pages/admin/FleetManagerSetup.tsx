import { useState } from "react";

import { FleetManagerWizardScreen } from "@/components/fleet-manager-setup/FleetManagerWizard";
import { Button } from "@/components/ui/button";
import { useFleetManagerSetup } from "@/hooks/useFleetManagerSetup";
import { FLEET_MANAGER_WIZARD_STEPS, type FleetManagerWizardStep } from "@/lib/fleetManagerWizard";

export default function FleetManagerSetup() {
  const [step, setStep] = useState<FleetManagerWizardStep>("provider");
  const [complete, setComplete] = useState(false);
  const setup = useFleetManagerSetup();
  const stepIndex = FLEET_MANAGER_WIZARD_STEPS.indexOf(step);

  if (complete) {
    return (
      <div className="max-w-2xl space-y-4">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Fleet Manager setup saved</h1>
        <p className="text-sm text-gray-500 dark:text-gray-400">
          Return to this page to update fleets or recover an incomplete installation.
        </p>
        <Button type="button" variant="outline" onClick={() => setComplete(false)}>
          Configure Fleet Manager
        </Button>
      </div>
    );
  }

  return (
    <div className="max-w-3xl space-y-5">
      <header className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Set up Fleet Manager</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Generate standalone chart values for your runner infrastructure.
          </p>
        </div>
        <p className="text-xs text-gray-500">
          Step {stepIndex + 1} of {FLEET_MANAGER_WIZARD_STEPS.length}
        </p>
      </header>
      <section className="rounded-lg border border-slate-200 bg-white p-6 dark:border-gray-700 dark:bg-gray-900">
        <FleetManagerWizardScreen
          step={step}
          setup={setup}
          showBackButton={stepIndex > 0}
          onBack={() => setStep(FLEET_MANAGER_WIZARD_STEPS[stepIndex - 1] ?? "provider")}
          onNext={() => setStep(FLEET_MANAGER_WIZARD_STEPS[stepIndex + 1] ?? "install")}
          onFinish={() => setComplete(true)}
          onFinishLater={() => setComplete(true)}
        />
      </section>
    </div>
  );
}
