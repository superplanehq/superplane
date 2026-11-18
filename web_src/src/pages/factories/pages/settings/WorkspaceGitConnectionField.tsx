import { Link } from "react-router";

import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { usePermissions } from "@/contexts/usePermissions";
import { useConnectedIntegrations } from "@/hooks/useIntegrations";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";

import { IntegrationChoiceIcon } from "../onboarding/onboardingSteps";
import { factorySettingsSectionPath } from "../../lib/factoryPagePaths";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { useFactoryRepository } from "./useFactoryRepository";
import type { useWorkspaceRepositoryChoices } from "./useWorkspaceRepositoryChoices";

type ConnectionWorkspace = { onboarding?: { appRepository?: string; defaultBranch?: string } | null };

export function WorkspaceGitConnectionField(props: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: ConnectionWorkspace | null | undefined;
  choices: ReturnType<typeof useWorkspaceRepositoryChoices>;
  canUpdate: boolean;
}) {
  const { canAct } = usePermissions();
  const repository = props.factory?.onboarding?.appRepository?.trim() ?? "";
  if (!repository || !canAct("integrations", "read")) return null;
  return <GitConnectionPicker {...props} repository={repository} />;
}

function GitConnectionPicker(props: Parameters<typeof WorkspaceGitConnectionField>[0] & { repository: string }) {
  const { host, integrationId } = props.choices;
  const integrations = useConnectedIntegrations(props.organizationId);
  const updateRepository = useFactoryRepository(props.organizationId, props.factoryId);
  const options = (integrations.data ?? []).filter(
    (integration) =>
      integration.metadata?.integrationName === host &&
      (integration.status?.state === "ready" || integration.metadata?.id === integrationId) &&
      Boolean(integration.metadata?.id),
  );
  const selected = options.some((integration) => integration.metadata?.id === integrationId) ? integrationId : "";

  const choose = async (integrationId: string) => {
    try {
      await updateRepository.mutateAsync({
        repository: props.repository,
        defaultBranch: props.factory?.onboarding?.defaultBranch?.trim() ?? "",
        integrationId,
      });
      showSuccessToast(AGENT_RESOURCES_COPY.gitConnectionUpdated);
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, AGENT_RESOURCES_COPY.gitConnectionUpdateFailed));
    }
  };

  return (
    <div
      className={cn(
        "mb-3 flex flex-wrap items-center gap-3 rounded-lg border px-3 py-2.5",
        selected ? "border-border" : "border-amber-300 bg-amber-50 dark:border-amber-800 dark:bg-amber-950/40",
      )}
      data-testid="agent-git-connection"
    >
      <IntegrationChoiceIcon name={host} />
      <div className="min-w-0 flex-1">
        <p className="text-[13px] font-medium">{AGENT_RESOURCES_COPY.gitConnectionLabel}</p>
        <p className="text-[12px] text-muted-foreground">
          {selected ? AGENT_RESOURCES_COPY.gitConnectionHelper : AGENT_RESOURCES_COPY.gitConnectionMissing}{" "}
          <Link
            to={factorySettingsSectionPath(props.organizationId, props.factoryKey, "organization", "integrations")}
            className="underline underline-offset-2"
          >
            {AGENT_RESOURCES_COPY.gitConnectionManage}
          </Link>
        </p>
      </div>
      <Select
        value={selected}
        onValueChange={(value) => void choose(value)}
        disabled={!props.canUpdate || updateRepository.isPending || options.length === 0}
      >
        <SelectTrigger
          className="w-full sm:w-64"
          aria-label={AGENT_RESOURCES_COPY.gitConnectionLabel}
          data-testid="agent-git-connection-select"
        >
          <SelectValue
            placeholder={
              options.length === 0 ? AGENT_RESOURCES_COPY.gitConnectionEmpty : AGENT_RESOURCES_COPY.gitConnectionChoose
            }
          />
        </SelectTrigger>
        <SelectContent position="popper">
          {options.map((integration) => (
            <SelectItem key={integration.metadata?.id} value={integration.metadata?.id ?? ""}>
              {integration.metadata?.name || integration.metadata?.id}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
