import { useMemo } from "react";

import { buildWorkflowFiles } from "./lib/canvas-files";

type UseWorkflowFilesFromCanvasArgs = Parameters<typeof buildWorkflowFiles>[0];

export function useWorkflowFilesFromCanvas(args: UseWorkflowFilesFromCanvasArgs) {
  const { canvasYamlPayload, panels, layout, canvasId, canvasName, consoleLoading, consoleError } = args;

  return useMemo(
    () =>
      buildWorkflowFiles({
        canvasYamlPayload,
        panels,
        layout,
        canvasId,
        canvasName,
        consoleLoading,
        consoleError,
      }),
    [canvasName, canvasId, canvasYamlPayload, consoleError, consoleLoading, layout, panels],
  );
}
