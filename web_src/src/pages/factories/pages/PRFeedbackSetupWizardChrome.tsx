import { BrandArrowArt } from "@/components/branding/BrandArrowArt";
import "@/components/branding/brandSurface.css";
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
    <div className="brand-surface brand-configure flex h-full min-h-0" data-testid={testId}>
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
      className="relative hidden overflow-hidden border-l border-border lg:flex lg:w-[44%] lg:flex-col"
      aria-label={label}
      data-testid={testId}
    >
      <BrandArrowArt variant="teal" minWidth={1024} />
      <div className="configure-preview-content relative z-10 flex flex-1 items-center justify-center px-8 py-24">
        {children}
      </div>
      <p
        className="configure-preview-caption absolute inset-x-0 bottom-6 z-10 px-6 text-center font-mono text-[11px] uppercase tracking-[0.1em]"
        data-testid={captionTestId}
      >
        {caption}
      </p>
    </aside>
  );
}

export function PRFeedbackSetupPreviewPrLabel() {
  return (
    <p className="mb-3 inline-block rounded-md bg-card px-3 py-1.5 font-mono text-[11px] tracking-[0.04em] text-muted-foreground">
      {PR_FEEDBACK_SETTINGS_COPY.wizardPreviewPrLabel} {PR_FEEDBACK_SETTINGS_COPY.wizardPreviewPrNumber}
    </p>
  );
}
