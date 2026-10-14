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
        <SetupStepsTerminal commands={commands} />
      )}
    </FactorySettingsCard>
  );
}

function SetupStepsTerminal({ commands }: { commands: string[] }) {
  const visible = commands.slice(0, PREVIEW_COMMAND_LIMIT);
  const truncated = commands.length > PREVIEW_COMMAND_LIMIT;
  return (
    <div className="overflow-hidden rounded-lg bg-zinc-950 text-zinc-200" data-testid="agent-setup-steps-preview">
      <div className="flex items-center gap-1.5 border-b border-white/10 px-3 py-2">
        <span className="size-2 rounded-full bg-zinc-700" aria-hidden />
        <span className="size-2 rounded-full bg-zinc-700" aria-hidden />
        <span className="size-2 rounded-full bg-zinc-700" aria-hidden />
        <span className="ml-1 font-mono text-[11px] tracking-wider text-zinc-500 uppercase">
          {AGENT_RESOURCES_COPY.setupStepsTitle}
        </span>
        <span className="ml-auto rounded-full bg-white/10 px-2 py-0.5 text-[11px] whitespace-nowrap text-zinc-300">
          {AGENT_RESOURCES_COPY.setupStepsSummary(commands.length)}
        </span>
      </div>
      <div className="overflow-x-auto p-3 font-mono text-[12px] leading-6">
        {visible.map((command, index) => (
          <div key={`${index}-${command}`} className="flex gap-2">
            <span aria-hidden className="text-emerald-400/70 select-none">
              $
            </span>
            <span className="min-w-0 flex-1 truncate text-zinc-100">{command}</span>
          </div>
        ))}
        {truncated ? (
          <div aria-hidden className="text-zinc-500 select-none">
            …
          </div>
        ) : null}
      </div>
    </div>
  );
}
