import { factoriesMaterializeFactoryAutomationDefaults } from "@/api-client";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { parseCanvasYamlForImport } from "@/pages/app/lib/workflow-spec-files";

import type { PlanningReviewStep } from "../pages/planningReviewMockup";
import { PLANNING_SETTINGS_COPY } from "../pages/planningSettingsCopy";
import { defaultRefineTaskStep } from "./refinementPrompt";

export async function loadDefaultRefinementPrompt(input: {
  organizationId: string;
  factoryId: string;
  automationId: string;
}): Promise<PlanningReviewStep | null> {
  try {
    const response = await factoriesMaterializeFactoryAutomationDefaults(
      withOrganizationHeader({
        organizationId: input.organizationId,
        path: { factoryId: input.factoryId, automationId: input.automationId },
        body: {},
      }),
    );
    const parsed = parseCanvasYamlForImport(response.data?.canvasYaml ?? "");
    if (!parsed.ok) {
      showErrorToast(PLANNING_SETTINGS_COPY.restorePromptError);
      return null;
    }

    const step = defaultRefineTaskStep(parsed.spec);
    if (!step?.prompt) {
      showErrorToast(PLANNING_SETTINGS_COPY.restorePromptError);
      return null;
    }
    return step;
  } catch (error) {
    showErrorToast(getApiErrorMessage(error, PLANNING_SETTINGS_COPY.restorePromptError));
    return null;
  }
}
