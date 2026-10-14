import { useNavigate } from "react-router";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { useFactoryAgentResources } from "@/hooks/useFactoryAgentResources";

import { factoryAgentPath } from "../../lib/factoryPagePaths";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";

const PREVIEW_COMMAND_LIMIT = 5;

export function WorkspaceSetupStepsSection({
  organizationId,
  factoryId,
  factoryKey,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  canUpdate: boolean;
}) {
  const navigate = useNavigate();
  const setup = useFactoryAgentResources(organizationId, factoryId, "KIND_SETUP_COMMAND");
  const stored = (setup.data ?? [])[0];
  const commands = stored?.commands ?? [];
  const editPath = `${factoryAgentPath(organizationId, factoryKey)}/setup-steps`;

  return (
    <FactorySettingsCard
      title={AGENT_RESOURCES_COPY.setupStepsTitle}
      description={AGENT_RESOURCES_COPY.setupStepsHelper}
      action={
        <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
          <Button
            type="button"
            size="sm"
            onClick={() => navigate(editPath)}
            disabled={!canUpdate}
            data-testid="agent-setup-steps-edit"
          >
            {AGENT_RESOURCES_COPY.editSetupSteps}
          </Button>
        </PermissionTooltip>
      }
      data-testid="agent-setup-steps"
    >
      {setup.isLoading ? (
        <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.setupStepsLoading}</p>
      ) : setup.isError ? (
        <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.setupStepsLoadError}</p>
      ) : commands.length === 0 ? null : (
        <div className="flex flex-col gap-2">
          <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.setupStepsSummary(commands.length)}</p>
          <pre className="overflow-x-auto rounded-lg border border-border bg-muted/40 p-3 font-mono text-[12px] leading-5">
            {commands.slice(0, PREVIEW_COMMAND_LIMIT).join("\n")}
            {commands.length > PREVIEW_COMMAND_LIMIT ? "\n…" : ""}
          </pre>
        </div>
      )}
    </FactorySettingsCard>
  );
}
