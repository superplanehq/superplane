import type { OrganizationsIntegrationResourceRef } from "@/api-client";
import { cn } from "@/lib/utils";
import { Loader2, XCircle } from "lucide-react";

import { catalogStatusCheckNames, isChecksHandlerCIIntegration, type ChecksToolsAccess } from "./checksPRFeedbackSetup";
import {
  checksPreviewAttemptsLabel,
  checksPreviewReadingLabel,
  checksPreviewRows,
  checksPreviewToolOutcome,
  checksSetupPreviewCaption,
  type ChecksPreviewRow,
  type ChecksPreviewToolOutcome,
} from "./checksPRFeedbackPreview";
import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsCopy";
import { PRFeedbackSetupPreviewPane, PRFeedbackSetupPreviewPrLabel } from "./PRFeedbackSetupWizardChrome";

export function ChecksPRFeedbackSetupPreview({
  step,
  catalog,
  selectedNames,
  catalogEmpty,
  maximumAttempts,
  toolsAccess,
  suggestedNames,
  runnerIntegrationIds,
  connected,
}: {
  step: "checks" | "tools";
  catalog: OrganizationsIntegrationResourceRef[];
  selectedNames: string[];
  catalogEmpty: boolean;
  maximumAttempts: number;
  toolsAccess: ChecksToolsAccess;
  suggestedNames: string[];
  runnerIntegrationIds: string[];
  connected: Array<{ status?: { state?: string }; metadata?: { id?: string; integrationName?: string } }>;
}) {
  const rows = checksPreviewRows(catalogStatusCheckNames(catalog), selectedNames);
  const toolOutcome = checksPreviewToolOutcome({
    toolsAccess,
    suggestedNames,
    selectedIds: runnerIntegrationIds,
    connected: readyCIIntegrations(connected),
  });
  const caption = checksSetupPreviewCaption({
    step,
    catalogEmpty,
    selectedCount: selectedNames.filter((name) => name.trim()).length,
    toolOutcome,
  });
  const attemptsLabel = step === "checks" && !catalogEmpty ? checksPreviewAttemptsLabel(maximumAttempts) : "";

  return (
    <PRFeedbackSetupPreviewPane
      label={PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksLabel}
      caption={caption}
      testId="checks-setup-preview"
      captionTestId="checks-setup-preview-caption"
    >
      <div className="w-full max-w-sm space-y-3">
        <ChecksPreviewCard rows={rows} catalogEmpty={catalogEmpty} attemptsLabel={attemptsLabel} />
        {step === "tools" ? <ChecksPreviewToolOutcomeView outcome={toolOutcome} /> : null}
      </div>
    </PRFeedbackSetupPreviewPane>
  );
}

function readyCIIntegrations(
  connected: Array<{ status?: { state?: string }; metadata?: { id?: string; integrationName?: string } }>,
) {
  return connected.filter(
    (integration) =>
      integration.status?.state === "ready" &&
      integration.metadata?.id &&
      isChecksHandlerCIIntegration(integration.metadata.integrationName),
  );
}

function ChecksPreviewCard({
  rows,
  catalogEmpty,
  attemptsLabel,
}: {
  rows: ChecksPreviewRow[];
  catalogEmpty: boolean;
  attemptsLabel: string;
}) {
  return (
    <div className="w-full max-w-sm">
      <PRFeedbackSetupPreviewPrLabel />
      <article
        className="rounded-lg border border-border bg-card p-4 shadow-sm"
        data-testid="checks-setup-preview-card"
      >
        <h2 className="mb-3 text-[13px] font-medium text-foreground">
          {PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksHeading}
        </h2>
        {catalogEmpty || rows.length === 0 ? (
          <p className="text-[13px] text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksNoneYet}</p>
        ) : (
          <ul className="space-y-2">
            {rows.map((row, index) => (
              <ChecksPreviewRowView key={row.name} row={row} index={index} />
            ))}
          </ul>
        )}
        {attemptsLabel ? (
          <p className="mt-3 text-[12px] text-muted-foreground" data-testid="checks-setup-preview-attempts">
            {attemptsLabel}
          </p>
        ) : null}
      </article>
    </div>
  );
}

function ChecksPreviewRowView({ row, index }: { row: ChecksPreviewRow; index: number }) {
  const outcome = row.selected
    ? PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksFixing
    : PR_FEEDBACK_SETTINGS_COPY.wizardPreviewChecksIgnored;

  return (
    <li
      className={cn(
        "animate-in fade-in flex items-center gap-2 text-[13px] duration-300",
        row.selected ? "opacity-100" : "opacity-50",
      )}
      style={{ animationDelay: `${index * 60}ms` }}
      data-testid={`checks-setup-preview-check-${row.name}`}
      data-selected={row.selected ? "true" : "false"}
      data-outcome={outcome}
    >
      <XCircle
        className={cn("size-3.5 shrink-0", row.selected ? "text-red-500 dark:text-red-400" : "text-muted-foreground")}
        aria-hidden
      />
      <span className="min-w-0 flex-1 truncate font-medium">{row.name}</span>
      <span
        className={cn(
          "shrink-0 text-[12px] font-medium",
          row.selected ? "text-[#4c2a94] dark:text-[#f6a821]" : "text-muted-foreground",
        )}
      >
        {outcome}
      </span>
    </li>
  );
}

function ChecksPreviewToolOutcomeView({ outcome }: { outcome: ChecksPreviewToolOutcome }) {
  if (outcome.kind === "granted") {
    const label = checksPreviewReadingLabel(outcome.labels);
    if (!label) {
      return null;
    }
    return (
      <p
        className="animate-in fade-in slide-in-from-bottom-1 inline-flex w-full items-center justify-center gap-2 text-[12px] font-medium text-[#4c2a94] duration-300 dark:text-[#f6a821]"
        data-testid="checks-setup-preview-tool-outcome"
        role="status"
      >
        <Loader2 className="size-3.5 motion-safe:animate-spin" aria-hidden />
        {label}
      </p>
    );
  }

  if (outcome.kind === "no-access") {
    return (
      <p
        className="animate-in fade-in slide-in-from-bottom-1 text-center text-[12px] font-medium text-muted-foreground duration-300"
        data-testid="checks-setup-preview-tool-outcome"
      >
        {PR_FEEDBACK_SETTINGS_COPY.wizardPreviewToolsNoAccess}
      </p>
    );
  }

  return null;
}
