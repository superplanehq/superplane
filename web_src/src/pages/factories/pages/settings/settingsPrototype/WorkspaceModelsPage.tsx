import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

import { factoryCardClassName } from "../../factoryPageLayoutStyles";
import { AGENT_OPTIONS, type AgentHarnessId, integrationLabel } from "../../onboarding/onboardingFixtures";
import { useFactorySettingsLayout } from "../factorySettingsLayoutContext";
import { useWorkspaceConnections } from "./useWorkspaceConnections";
import { WorkspaceSettingsSection } from "./WorkspaceSettingsSection";
import { agentSummary } from "./workspaceConnections";

export function WorkspaceModelsPage() {
  const { factoryId } = useFactorySettingsLayout();
  const { connections, updateConnections } = useWorkspaceConnections(factoryId);

  return (
    <WorkspaceSettingsSection
      title="Models"
      description="Select the coding agent for this workspace. The agent changes the app repository and opens pull requests."
    >
      <div className="max-w-2xl space-y-6">
        <section className={cn("p-6", factoryCardClassName)} data-testid="workspace-settings-models">
          <h2 className="text-[13px] font-medium tracking-[-0.01em] text-foreground">Coding agent</h2>
          <p className="mt-1 text-[12px] text-muted-foreground">
            SuperPlane sends work orders to this agent. The agent writes code in the app repository and opens a pull
            request.
          </p>
          <p className="mt-3 text-[13px] font-medium">{agentSummary(connections)}</p>

          <div className="mt-4 grid gap-2">
            {AGENT_OPTIONS.map((option) => {
              const selected = connections.agent === option.id;
              return (
                <button
                  key={option.id}
                  type="button"
                  disabled={option.soon}
                  onClick={() => updateConnections({ ...connections, agent: option.id as AgentHarnessId })}
                  className={cn(
                    "relative flex items-start gap-3 rounded-lg border px-4 py-3 text-left transition-colors",
                    option.soon && "cursor-not-allowed border-border/70 bg-muted/20 opacity-70",
                    !option.soon && selected && "border-foreground bg-accent/40",
                    !option.soon && !selected && "border-border bg-background hover:bg-accent/30",
                  )}
                >
                  <IntegrationIcon integrationName={option.integrationId} className="mt-0.5 size-5" size={20} />
                  <span className="min-w-0">
                    <span className="block text-[13px] font-medium">{option.label}</span>
                    <span className="mt-0.5 block text-[12px] text-muted-foreground">{option.detail}</span>
                    <span className="mt-1 block text-[11px] text-muted-foreground">
                      {option.soon ? "Coming soon" : `Uses ${integrationLabel(option.integrationId)}`}
                    </span>
                  </span>
                </button>
              );
            })}
          </div>
        </section>
      </div>
    </WorkspaceSettingsSection>
  );
}
