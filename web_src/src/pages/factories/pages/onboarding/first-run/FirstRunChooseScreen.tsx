import { Button } from "@/components/ui/button";

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
      </div>
    </FirstRunShell>
  );
}
