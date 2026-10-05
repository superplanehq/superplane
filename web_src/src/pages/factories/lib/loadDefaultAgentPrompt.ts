import { factoriesMaterializeFactoryAutomationDefaults } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { parseCanvasYamlForImport } from "@/pages/app/lib/workflow-spec-files";

import type { PlanningReviewStep } from "../pages/planningReviewMockup";
import { defaultAgentPromptSteps, RESTORE_DEFAULT_PROMPT_COPY } from "./defaultAgentPrompt";

export async function loadDefaultAgentPrompt(input: {
  organizationId: string;
  factoryId: string;
  automationId: string;
  agentNodeId: string;
}): Promise<PlanningReviewStep[]> {
  const response = await factoriesMaterializeFactoryAutomationDefaults(
    withOrganizationHeader({
      organizationId: input.organizationId,
      path: { factoryId: input.factoryId, automationId: input.automationId },
      body: {},
    }),
  );
  const parsed = parseCanvasYamlForImport(response.data?.canvasYaml ?? "");
  if (!parsed.ok) {
    throw new Error(RESTORE_DEFAULT_PROMPT_COPY.error);
  }

  const steps = defaultAgentPromptSteps(parsed.spec, input.agentNodeId);
  if (!steps?.some((step) => step.prompt)) {
    throw new Error(RESTORE_DEFAULT_PROMPT_COPY.error);
  }
  return steps;
}
