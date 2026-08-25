import { canvasesCommitCanvasStaging, canvasesPutCanvasStaging, type CanvasesCanvas } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { encodeRepositoryFileContent } from "@/pages/app/files/lib/repository-files";
import { materializeCanvasSpec } from "@/pages/app/lib/workflow-spec-files";
import { CANVAS_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";

import { applyIntakeSettingsToCanvas, type IntakeCanvasSettingsContext } from "./intakeAutomationSettings";
import type { IntakeSourceSettings } from "./intakeSourceSettingsModel";

interface SaveIntakeAutomationSettingsInput {
  canvasId: string;
  context: IntakeCanvasSettingsContext;
  canvas: CanvasesCanvas;
  settings: IntakeSourceSettings;
  stageCanvas?: (canvasId: string, canvasYaml: string) => Promise<unknown>;
  commitCanvas?: (canvasId: string) => Promise<unknown>;
  updateCanvas: (input: { name: string }) => Promise<unknown>;
}

export async function saveIntakeAutomationSettings({
  canvasId,
  context,
  canvas,
  settings,
  stageCanvas = stageIntakeCanvas,
  commitCanvas = commitIntakeCanvas,
  updateCanvas,
}: SaveIntakeAutomationSettingsInput): Promise<void> {
  const updatedCanvas = applyIntakeSettingsToCanvas(context, canvas, settings);
  await stageCanvas(canvasId, materializeCanvasSpec(updatedCanvas));
  await commitCanvas(canvasId);
  await updateCanvas({ name: settings.name });
}

async function stageIntakeCanvas(canvasId: string, canvasYaml: string): Promise<unknown> {
  return canvasesPutCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
      body: {
        operations: [
          {
            path: CANVAS_YAML_PATH,
            content: encodeRepositoryFileContent(canvasYaml),
          },
        ],
      },
    }),
  );
}

async function commitIntakeCanvas(canvasId: string): Promise<unknown> {
  return canvasesCommitCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
      body: { commitMessage: "Update intake settings" },
    }),
  );
}
