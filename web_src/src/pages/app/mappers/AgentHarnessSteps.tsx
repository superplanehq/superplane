import { agentRunnerStepTitles } from "@/lib/agentRunnerSteps";
import { useMergeConfidenceCanvas } from "@/lib/mergeConfidenceCanvasContext";
import { FactoryNodeStepList } from "@/ui/factoryNodeChrome";

export function AgentHarnessSteps({ configuration }: { configuration: unknown }) {
  const steps = agentRunnerStepTitles(configuration, { expandMergeChecks: useMergeConfidenceCanvas() });
  if (steps.length === 0) {
    return null;
  }
  return <FactoryNodeStepList steps={steps} />;
}
