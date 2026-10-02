import { factoriesMaterializeFactoryAutomationDefaults } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { parseCanvasYamlForImport } from "@/pages/app/lib/workflow-spec-files";

import type { PlanningReviewStep } from "../pages/planningReviewMockup";
import { PLANNING_SETTINGS_COPY } from "../pages/planningSettingsCopy";
import { defaultRefineTaskStep } from "./refinementPrompt";

export async function loadDefaultRefinementPrompt(input: {
  organizationId: string;
  factoryId: string;
  automationId: string;
}): Promise<PlanningReviewStep> {
  const response = await factoriesMaterializeFactoryAutomationDefaults(
    withOrganizationHeader({
      organizationId: input.organizationId,
      path: { factoryId: input.factoryId, automationId: input.automationId },
      body: {},
    }),
  );
  const parsed = parseCanvasYamlForImport(response.data?.canvasYaml ?? "");
  if (!parsed.ok) {
    throw new Error(PLANNING_SETTINGS_COPY.restorePromptError);
  }

  const step = defaultRefineTaskStep(parsed.spec);
  if (!step?.prompt) {
    throw new Error(PLANNING_SETTINGS_COPY.restorePromptError);
  }
  return step;
}
