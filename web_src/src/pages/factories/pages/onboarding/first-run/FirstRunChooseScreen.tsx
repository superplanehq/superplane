import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { ChevronDown } from "lucide-react";
import { useState } from "react";

import { RepositoryPicker } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome } from "./firstRunTypes";

export function FirstRunChooseScreen({
  repositories,
  selectedRepository,
  loading,
  chrome,
  onSelectRepository,
  onEditConnection,
  onContinue,
}: {
  repositories: string[];
  selectedRepository: string | null;
  /** True while the repository list loads or refreshes; hides stale entries. */
  loading?: boolean;
  chrome?: FirstRunChrome;
  onSelectRepository: (repository: string) => void;
  onEditConnection: () => void;
  onContinue: () => void;
}) {
  const copy = FIRST_RUN_COPY.choose;
  const [whyMissingOpen, setWhyMissingOpen] = useState(false);

  return (
    <FirstRunShell testId="first-run-choose" chrome={chrome}>
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
              onSelect={onSelectRepository}
            />
          )}
          <p className="mt-3 text-[13px] text-muted-foreground">
            {copy.missingRepository}{" "}
            <button
              type="button"
              onClick={onEditConnection}
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
          <Button
            type="button"
            className="w-full"
            disabled={!selectedRepository}
            onClick={onContinue}
            data-testid="first-run-continue-to-tickets"
          >
            {selectedRepository ? copy.continueReady : copy.continue}
          </Button>
          <p className="text-[12px] text-muted-foreground">{copy.moreLater}</p>
        </div>

        <details
          className="rounded-lg border border-border p-3 text-left open:pb-4"
          open={whyMissingOpen}
          onToggle={(event) => setWhyMissingOpen(event.currentTarget.open)}
          data-testid="first-run-choose-why-missing"
        >
          <summary className="flex cursor-pointer list-none items-center justify-between gap-2 text-[13px] text-muted-foreground [&::-webkit-details-marker]:hidden">
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

/** Mirrors the repository picker layout: a search field and a short list. */
function RepositoryListLoading() {
  return (
    <div className="space-y-3" data-testid="first-run-repositories-loading" aria-hidden>
      <div className="h-9 animate-pulse rounded-md bg-accent/40" />
      <div className="rounded-lg border border-border">
        <ul className="divide-y divide-border">
          {[0, 1, 2].map((row) => (
            <li key={row} className="flex items-center gap-3 px-3 py-2.5">
              <div className="size-4 shrink-0 animate-pulse rounded bg-accent/40" />
              <div className="h-3.5 w-40 animate-pulse rounded bg-accent/40" />
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
}
