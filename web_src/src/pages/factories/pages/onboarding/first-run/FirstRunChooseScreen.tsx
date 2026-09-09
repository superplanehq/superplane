import { LoadingButton } from "@/components/ui/loading-button";
import { cn } from "@/lib/utils";
import { ChevronDown, Loader2 } from "lucide-react";
import { useState } from "react";

import { RepositoryPicker } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";

export function FirstRunChooseScreen({
  repositories,
  selectedRepository,
  loading,
  saving = false,
  chrome,
  onSelectRepository,
  onEditConnection,
  onContinue,
}: {
  repositories: string[];
  selectedRepository: string | null;
  /** True while the repository list loads or refreshes; hides stale entries. */
  loading?: boolean;
  saving?: boolean;
  chrome?: FirstRunChrome;
  onSelectRepository: (repository: string) => void;
  onEditConnection: () => void;
  onContinue: () => void;
}) {
  const copy = FIRST_RUN_COPY.choose;
  const [whyMissingOpen, setWhyMissingOpen] = useState(false);
  const busy = Boolean(loading || saving);

  return (
    <FirstRunShell testId="first-run-choose" chrome={chrome} busy={busy}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.repositoryHelper}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        <FirstRunPanel>
          {loading ? (
            <RepositoryListLoading />
          ) : (
            <RepositoryPicker
              host="github"
              repos={repositories}
              selectedRepo={selectedRepository}
              disabled={busy}
              onSelect={onSelectRepository}
            />
          )}
          <p className="mt-3 text-[13px] text-muted-foreground">
            {copy.missingRepository}{" "}
            <button
              type="button"
              onClick={onEditConnection}
              disabled={busy}
              className="font-medium text-foreground underline underline-offset-2 hover:no-underline"
            >
              {copy.editConnection}
            </button>
          </p>
          <p className="mt-1 text-[12px] text-muted-foreground" data-testid="first-run-choose-access-hint">
            {copy.accessHint}
          </p>
        </FirstRunPanel>

        <div className="space-y-2">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!selectedRepository || busy}
            loading={saving}
            loadingText={copy.saving}
            onClick={onContinue}
            data-testid="first-run-continue-to-tickets"
          >
            {selectedRepository ? copy.continueReady : copy.continue}
          </LoadingButton>
          <p className="text-[12px] text-muted-foreground">{copy.moreLater}</p>
        </div>

        <details
          className="rounded-lg border border-border p-3 text-left open:pb-4"
          open={whyMissingOpen}
          onToggle={(event) => setWhyMissingOpen(event.currentTarget.open)}
          aria-disabled={busy || undefined}
          data-testid="first-run-choose-why-missing"
        >
          <summary
            className={cn(
              "flex cursor-pointer list-none items-center justify-between gap-2 text-[13px] text-muted-foreground [&::-webkit-details-marker]:hidden",
              busy && "pointer-events-none opacity-50",
            )}
          >
            {copy.missingTitle}
            <ChevronDown
              className={cn("size-3.5 shrink-0 transition-transform", whyMissingOpen && "rotate-180")}
              aria-hidden
            />
          </summary>
          <ul className="mt-2 list-disc space-y-1 pl-4 text-[12px] text-muted-foreground">
            {copy.missingReasons.map((reason) => (
              <li key={reason}>{reason}</li>
            ))}
          </ul>
        </details>
      </div>
    </FirstRunShell>
  );
}

function RepositoryListLoading() {
  return (
    <div
      className="flex min-h-32 items-center justify-center gap-2 text-[13px] text-muted-foreground"
      data-testid="first-run-repositories-loading"
      role="status"
    >
      <Loader2 className="size-4 animate-spin" data-testid="first-run-repositories-spinner" aria-hidden />
      <span>{FIRST_RUN_COPY.choose.loading}</span>
    </div>
  );
}
