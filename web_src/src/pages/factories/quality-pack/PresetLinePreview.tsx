import { AppWindow, ShieldCheck } from "lucide-react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";
import { LineStepArrow } from "@/pages/factories/FactoryLineStepFlow";
import type { PresetLineStep } from "@/pages/factories/verification/types";

interface PresetLinePreviewProps {
  lineName: string;
  steps: PresetLineStep[];
  onInstall: () => void;
}

/**
 * Install-time preview of a preset line. The verify step is expanded to show
 * the checks it runs, so the gate is visible before install.
 */
export function PresetLinePreview({ lineName, steps, onInstall }: PresetLinePreviewProps) {
  return (
    <section className={cn(factoryCardClassName, "flex w-full max-w-xl flex-col")} aria-label="Line preview">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-3">
        <div className="flex flex-col gap-0.5">
          <h3 className="workspace-section-title text-foreground">Line preview: {lineName}</h3>
          <p className="text-[12px] text-muted-foreground">
            Work orders dispatched to this line pass through these steps in order.
          </p>
        </div>
        <div className="flex flex-col items-end gap-1">
          <Button size="sm" onClick={onInstall}>
            Install line
          </Button>
          <p className="text-[11px] text-muted-foreground">This will not start a run yet.</p>
        </div>
      </header>

      <div className="flex flex-col items-stretch px-4 py-4">
        {steps.map((step, index) => (
          <div key={step.name} className="flex flex-col items-stretch">
            {index > 0 ? <LineStepArrow className="self-center" /> : null}
            <PresetStepCard step={step} />
          </div>
        ))}
      </div>
    </section>
  );
}

function PresetStepCard({ step }: { step: PresetLineStep }) {
  const isVerify = step.type === "verify";
  const StepIcon = isVerify ? ShieldCheck : AppWindow;
  return (
    <div
      className={cn(
        "rounded-xl border p-3",
        isVerify
          ? "border-blue-200 bg-blue-50/60 dark:border-blue-900 dark:bg-blue-950/30"
          : "border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-900",
      )}
    >
      <div className="flex items-center gap-2">
        <StepIcon className="size-4 shrink-0 text-muted-foreground" aria-hidden />
        <span className="text-[13px] font-medium text-foreground">{step.name}</span>
        <span className="rounded border border-border px-1.5 py-0.5 text-[11px] font-medium uppercase tracking-wide text-muted-foreground">
          {isVerify ? "Verify" : "Run app"}
        </span>
      </div>
      <p className="mt-1 text-[13px] text-muted-foreground">{step.summary}</p>
      {isVerify && step.checkNames && step.checkNames.length > 0 ? (
        <ul className="mt-2 grid gap-1 sm:grid-cols-2">
          {step.checkNames.map((checkName) => (
            <li
              key={checkName}
              className="rounded border border-border bg-background px-2 py-1 text-[12px] text-foreground"
            >
              {checkName}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
