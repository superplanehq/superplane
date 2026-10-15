import { Terminal } from "lucide-react";
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
      titleClassName="text-sm font-semibold"
      description={AGENT_RESOURCES_COPY.setupStepsHelper}
      icon={<Terminal className="size-4" strokeWidth={1.75} aria-hidden />}
      iconClassName="border-border bg-muted/60 text-foreground"
      className="shadow-sm"
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
        <SetupStepsPreview commands={commands} />
      )}
    </FactorySettingsCard>
  );
}

function SetupStepsPreview({ commands }: { commands: string[] }) {
  const visible = commands.slice(0, PREVIEW_COMMAND_LIMIT);
  const truncated = commands.length > PREVIEW_COMMAND_LIMIT;
  return (
    <div
      className="rounded-lg border border-border bg-muted/40 px-3 py-2 font-mono text-[12px] leading-6"
      data-testid="agent-setup-steps-preview"
    >
      {visible.map((command, index) => (
        <div key={`${index}-${command}`} className="truncate text-muted-foreground">
          <span aria-hidden className="mr-2 select-none">
            $
          </span>
          {command}
        </div>
      ))}
      {truncated ? (
        <div aria-hidden className="text-muted-foreground/60 select-none">
          …
        </div>
      ) : null}
    </div>
  );
}
