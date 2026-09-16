import { cn } from "@/lib/utils";
import { Bug } from "lucide-react";

import { PRFeedbackSetupPreviewPane } from "./PRFeedbackSetupWizardChrome";
import { sentryIntakePreviewCaption, sentryIntakePreviewRows } from "./sentryIntakePreview";
import { SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";

export function SentryIntakeSetupPreview({ issueNames, hasProject }: { issueNames: string[]; hasProject: boolean }) {
  const rows = sentryIntakePreviewRows(issueNames);
  const caption = sentryIntakePreviewCaption(hasProject);

  return (
    <PRFeedbackSetupPreviewPane
      label={SENTRY_INTAKE_SETUP_COPY.wizardPreviewLabel}
      caption={caption}
      testId="sentry-setup-preview"
      captionTestId="sentry-setup-preview-caption"
    >
      <article
        className="w-full max-w-sm rounded-lg border border-border bg-card p-4 shadow-sm"
        data-testid="sentry-setup-preview-card"
      >
        <h2 className="mb-3 text-[13px] font-medium text-foreground">
          {SENTRY_INTAKE_SETUP_COPY.wizardPreviewHeading}
        </h2>
        <ul className="space-y-2">
          {rows.map((row, index) => (
            <li
              key={row.id}
              className={cn(
                "animate-in fade-in flex items-center gap-2 text-[13px] duration-300",
                hasProject ? "opacity-100" : "opacity-60",
              )}
              style={{ animationDelay: `${index * 40}ms` }}
              data-testid={`sentry-setup-preview-issue-${row.id}`}
            >
              <Bug className="size-3.5 shrink-0 text-red-500 dark:text-red-400" aria-hidden />
              <span className="min-w-0 flex-1 truncate font-medium">{row.title}</span>
              <span className="shrink-0 text-[12px] font-medium text-[#4c2a94] dark:text-[#f6a821]">
                {SENTRY_INTAKE_SETUP_COPY.wizardPreviewImporting}
              </span>
            </li>
          ))}
        </ul>
        <p className="mt-3 text-[12px] text-muted-foreground" data-testid="sentry-setup-preview-listen">
          {SENTRY_INTAKE_SETUP_COPY.wizardPreviewListening}
        </p>
      </article>
    </PRFeedbackSetupPreviewPane>
  );
}
