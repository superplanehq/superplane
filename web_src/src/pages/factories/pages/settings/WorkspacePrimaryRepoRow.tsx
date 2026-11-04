import { useState } from "react";
import { ChevronDown } from "lucide-react";

import { Button } from "@/components/ui/button";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";

import { IntegrationChoiceIcon } from "../onboarding/onboardingSteps";
import type { VcsHostId } from "../onboarding/onboardingFixtures";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { UpdateUnfinishedTasksDialog } from "./UpdateUnfinishedTasksDialog";
import { useFactoryRepository } from "./useFactoryRepository";
import { useRepositoryBranchesPrefetch, useRepositoryDefaultBranch } from "./useRepositoryDefaultBranch";
import { RepositoryBranchField } from "./WorkspaceRepositoryFields";

const PANEL_ID = "agent-cross-repo-panel-primary";

export function WorkspacePrimaryRepoRow(props: {
  organizationId: string;
  factoryId: string;
  integrationId: string;
  repository: string;
  savedBranch: string;
  catalogRepositories?: { fullName?: string | null; defaultBranch?: string | null }[];
  host: VcsHostId;
  canUpdate: boolean;
  expanded: boolean;
  onToggle: () => void;
}) {
  const updateRepository = useFactoryRepository(props.organizationId, props.factoryId);
  const [pendingBranch, setPendingBranch] = useState<string | null>(null);
  const hostDefaultBranch = useRepositoryDefaultBranch(props);
  const prefetchBranches = useRepositoryBranchesPrefetch(props);
  const branch = props.savedBranch || hostDefaultBranch;

  const save = async (retargetTasks: boolean) => {
    const defaultBranch = pendingBranch;
    setPendingBranch(null);
    if (!defaultBranch) return;
    try {
      await updateRepository.mutateAsync({ repository: props.repository, defaultBranch, retargetTasks });
      showSuccessToast(AGENT_RESOURCES_COPY.crossRepoUpdated);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.crossRepoUpdateFailed));
    }
  };

  return (
    <li className="min-w-0" data-testid="agent-cross-repo-primary">
      <Button
        type="button"
        variant="ghost"
        className="h-auto w-full min-w-0 justify-start gap-3 rounded-none px-3 py-2.5 text-left font-normal hover:bg-accent/40"
        aria-expanded={props.expanded}
        aria-controls={PANEL_ID}
        aria-label={AGENT_RESOURCES_COPY.editRepository(props.repository)}
        onClick={props.onToggle}
        {...prefetchBranches}
        data-testid="agent-cross-repo-toggle-primary"
      >
        <IntegrationChoiceIcon name={props.host} />
        <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{props.repository}</span>
        <span className="shrink-0 truncate text-[12px] text-muted-foreground">
          {branch ? `${branch} · ` : ""}
          {AGENT_RESOURCES_COPY.primaryRepository}
        </span>
        <ChevronDown
          className={cn("size-4 shrink-0 text-muted-foreground transition-transform", props.expanded && "rotate-180")}
          aria-hidden
        />
      </Button>
      {props.expanded ? (
        <div id={PANEL_ID} className="border-t border-border bg-muted/30 px-3 py-3">
          <div className="grid gap-4 sm:grid-cols-2">
            <RepositoryBranchField
              id="agent-cross-repo-primary-branch"
              organizationId={props.organizationId}
              integrationId={props.integrationId}
              repository={props.repository}
              branch={props.savedBranch}
              defaultBranch={hostDefaultBranch}
              disabled={!props.canUpdate || updateRepository.isPending}
              onChange={(next) => {
                if (next && next !== props.savedBranch) setPendingBranch(next);
              }}
            />
          </div>
          <p className="mt-2 text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.primaryRepositoryHint}</p>
        </div>
      ) : null}
      <UpdateUnfinishedTasksDialog
        open={pendingBranch !== null}
        previousRepository={props.repository}
        nextRepository={props.repository}
        previousBranch={props.savedBranch}
        nextBranch={pendingBranch ?? ""}
        onOpenChange={(open) => !open && setPendingBranch(null)}
        onConfirm={() => void save(true)}
        onKeepTasks={() => void save(false)}
      />
    </li>
  );
}
