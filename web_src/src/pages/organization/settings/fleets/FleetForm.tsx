import { useState, type FormEvent } from "react";
import { Button } from "@/components/ui/button";
import { useFleetConfiguration } from "@/hooks/useFleetConfiguration";
import { FleetConfigurationFields } from "./FleetConfigurationFields";
import { FleetWizardProgress } from "./FleetWizardProgress";

const steps = ["Fleet", "Runner version", "Platform", "Resources"] as const;
const sections = ["identity", "version", "platform", "resources"] as const;
const descriptions = [
  "Choose a stable ID for this fleet. You cannot change it after creation.",
  "Choose the runner release that machines in this fleet must use.",
  "Set the operating system and CPU architecture for this fleet.",
  "Set the resources and capabilities that runners must provide.",
];

export function FleetForm({
  organizationId,
  onSaved,
  onCancel,
  refresh,
}: {
  organizationId: string;
  onSaved: (fleetId: string) => void;
  onCancel: () => void;
  refresh: () => void;
}) {
  const [step, setStep] = useState(0);
  const { draft, setDraft, error, saving, save } = useFleetConfiguration({ organizationId, onSaved, refresh });
  const submit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (step < steps.length - 1) {
      setStep(step + 1);
      return;
    }
    void save();
  };
  return (
    <form onSubmit={submit} className="space-y-5 rounded-lg border border-border bg-card p-5 text-[13px]">
      <h2 className="text-base font-medium">Create fleet</h2>
      <FleetWizardProgress steps={steps} current={step} />
      <div>
        <h3 className="font-medium">{steps[step]}</h3>
        <p className="mt-1 text-muted-foreground">{descriptions[step]}</p>
      </div>
      <FleetConfigurationFields section={sections[step]} draft={draft} onChange={setDraft} disabled={saving} />
      {error && (
        <p role="alert" className="text-destructive">
          {error}
        </p>
      )}
      <div className="flex items-center justify-between gap-2 border-t border-border pt-4">
        <Button type="button" size="sm" variant="ghost" onClick={onCancel} disabled={saving}>
          Cancel
        </Button>
        <div className="flex gap-2">
          {step > 0 && (
            <Button type="button" size="sm" variant="outline" onClick={() => setStep(step - 1)} disabled={saving}>
              Back
            </Button>
          )}
          <Button type="submit" size="sm" disabled={saving}>
            {saving ? "Creating…" : step === steps.length - 1 ? "Create fleet" : "Next"}
          </Button>
        </div>
      </div>
    </form>
  );
}
