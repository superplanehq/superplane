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
  chrome,
  onSelectRepository,
  onEditConnection,
  onContinue,
}: {
  repositories: string[];
  selectedRepository: string | null;
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
          <RepositoryPicker
            host="github"
            repos={repositories}
            selectedRepo={selectedRepository}
            onSelect={onSelectRepository}
          />
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
            {copy.continue}
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
