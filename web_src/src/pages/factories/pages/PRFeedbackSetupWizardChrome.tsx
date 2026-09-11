import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsCopy";

export function PRFeedbackSetupWizardShell({
  testId,
  preview,
  children,
}: {
  testId: string;
  preview: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex h-full min-h-0" data-testid={testId}>
      <div className="flex min-h-0 flex-1 justify-center overflow-y-auto px-8 py-12 lg:px-12">
        <div className="my-auto w-full max-w-lg space-y-6">{children}</div>
      </div>
      {preview}
    </div>
  );
}

export function PRFeedbackSetupPreviewPane({
  label,
  caption,
  testId,
  captionTestId,
  children,
}: {
  label: string;
  caption: string;
  testId: string;
  captionTestId: string;
  children: ReactNode;
}) {
  return (
    <aside
      className={cn(
        "relative hidden overflow-hidden border-l border-border lg:flex lg:w-[44%] lg:flex-col",
        "bg-[radial-gradient(ellipse_at_50%_45%,#faf9fd_0%,#f1eff7_70%)]",
        "dark:bg-[radial-gradient(ellipse_at_50%_45%,#14100a_0%,#0d0c08_70%)]",
      )}
      aria-label={label}
      data-testid={testId}
    >
      <PreviewRings />
      <div className="relative z-10 flex flex-1 items-center justify-center px-8">{children}</div>
      <p
        className="absolute inset-x-0 bottom-4 z-10 px-6 text-center font-mono text-[11px] uppercase tracking-[0.14em] text-muted-foreground"
        data-testid={captionTestId}
      >
        {caption}
      </p>
    </aside>
  );
}

export function PRFeedbackSetupPreviewPrLabel() {
  return (
    <p className="mb-3 font-mono text-[11px] tracking-[0.04em] text-muted-foreground">
      {PR_FEEDBACK_SETTINGS_COPY.wizardPreviewPrLabel} {PR_FEEDBACK_SETTINGS_COPY.wizardPreviewPrNumber}
    </p>
  );
}

function PreviewRings() {
  return (
    <div className="absolute inset-0 flex items-center justify-center" aria-hidden>
      {[280, 380, 480].map((diameter) => (
        <span
          key={diameter}
          className="absolute rounded-full border border-[#5b33ad]/12 dark:border-[#f6a821]/10"
          style={{ width: diameter, height: diameter, top: "46%", left: "50%", transform: "translate(-50%, -50%)" }}
        />
      ))}
    </div>
  );
}
