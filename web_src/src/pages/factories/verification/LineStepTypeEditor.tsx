import { useState } from "react";

import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Switch } from "@/ui/switch";
import { LineStepEditorShell } from "@/pages/factories/FactoryLineStepFlow";

import type { LineStepType, VerificationCheck, VerificationSuiteOption, VerifyStepDraft } from "./types";
import { CheckKindLabel } from "./CheckOutcomeChip";

const STEP_TYPE_LABELS: Record<LineStepType, string> = {
  runApp: "Run app",
  verify: "Verify",
};

interface AppOption {
  id: string;
  name: string;
}

interface LineStepTypeEditorProps {
  index: number;
  value: VerifyStepDraft;
  suites: VerificationSuiteOption[];
  apps: AppOption[];
  onChange: (step: VerifyStepDraft) => void;
}

/**
 * Design variant of the line step editor with a step type choice. `Run app`
 * keeps the existing app and trigger fields; `Verify` shows the verification
 * suite, its rule set, and a blocking toggle per check.
 */
export function LineStepTypeEditor({ index, value, suites, apps, onChange }: LineStepTypeEditorProps) {
  const [step, setStep] = useState(value);
  const [appId, setAppId] = useState(apps[0]?.id ?? "");

  const update = (next: VerifyStepDraft) => {
    setStep(next);
    onChange(next);
  };

  const selectedSuite = suites.find((suite) => suite.id === step.suiteId);

  const selectSuite = (suiteId: string) => {
    const suite = suites.find((option) => option.id === suiteId);
    if (!suite) return;
    update({ ...step, suiteId, ruleSetName: suite.ruleSetName, checks: suite.checks });
  };

  const toggleCheckBlocking = (checkId: string, blocking: boolean) => {
    update({
      ...step,
      checks: step.checks.map((check) => (check.id === checkId ? { ...check, blocking } : check)),
    });
  };

  return (
    <LineStepEditorShell>
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <div className="w-full min-w-0 space-y-2">
          <Label htmlFor={`verify-step-name-${index}`}>Step name</Label>
          <Input
            id={`verify-step-name-${index}`}
            value={step.name}
            onChange={(event) => update({ ...step, name: event.target.value })}
            placeholder="verify"
          />
        </div>

        <div className="w-full min-w-0 space-y-2">
          <Label htmlFor={`verify-step-type-${index}`}>Step type</Label>
          <Select value={step.type} onValueChange={(type) => update({ ...step, type: type as LineStepType })}>
            <SelectTrigger id={`verify-step-type-${index}`} className="w-full">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {(Object.keys(STEP_TYPE_LABELS) as LineStepType[]).map((type) => (
                <SelectItem key={type} value={type}>
                  {STEP_TYPE_LABELS[type]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        {step.type === "runApp" ? (
          <div className="w-full min-w-0 space-y-2">
            <Label htmlFor={`verify-step-app-${index}`}>App</Label>
            <Select value={appId || undefined} onValueChange={setAppId}>
              <SelectTrigger id={`verify-step-app-${index}`} className="w-full">
                <SelectValue placeholder="Select app" />
              </SelectTrigger>
              <SelectContent>
                {apps.map((app) => (
                  <SelectItem key={app.id} value={app.id}>
                    {app.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        ) : (
          <div className="w-full min-w-0 space-y-2">
            <Label htmlFor={`verify-step-suite-${index}`}>Verification suite</Label>
            <Select value={step.suiteId || undefined} onValueChange={selectSuite}>
              <SelectTrigger id={`verify-step-suite-${index}`} className="w-full">
                <SelectValue placeholder="Select suite" />
              </SelectTrigger>
              <SelectContent>
                {suites.map((suite) => (
                  <SelectItem key={suite.id} value={suite.id}>
                    {suite.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            {selectedSuite ? (
              <p className="text-xs text-muted-foreground">Rule set: {selectedSuite.ruleSetName}</p>
            ) : null}
          </div>
        )}
      </div>

      {step.type === "verify" && selectedSuite ? (
        <div className="mt-4 rounded-md border border-border bg-background">
          <div className="border-b border-border px-3 py-2">
            <p className="workspace-section-label text-muted-foreground">Checks</p>
            <p className="text-[12px] text-muted-foreground">
              Blocking checks stop the line when they find a problem. Advisory checks only record findings.
            </p>
          </div>
          <ul className="divide-y divide-border">
            {step.checks.map((check) => (
              <ChecksRow key={check.id} check={check} onBlockingChange={toggleCheckBlocking} />
            ))}
          </ul>
        </div>
      ) : null}
    </LineStepEditorShell>
  );
}

function ChecksRow({
  check,
  onBlockingChange,
}: {
  check: VerificationCheck;
  onBlockingChange: (checkId: string, blocking: boolean) => void;
}) {
  return (
    <li className="flex flex-wrap items-center justify-between gap-2 px-3 py-2">
      <div className="flex min-w-0 flex-col gap-0.5">
        <span className="text-[13px] font-medium text-foreground">{check.name}</span>
        <CheckKindLabel kind={check.kind} />
      </div>
      <label className="flex items-center gap-2 text-[12px] text-muted-foreground">
        {check.blocking ? "Blocking" : "Advisory"}
        <Switch
          checked={check.blocking}
          onCheckedChange={(blocking) => onBlockingChange(check.id, blocking)}
          aria-label={`Blocking for ${check.name}`}
        />
      </label>
    </li>
  );
}
