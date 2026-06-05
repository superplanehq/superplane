import { useMemo } from "react";

import { buildCanvasFiles } from "./lib/canvas-files";

type UseFilesFromCanvasArgs = Parameters<typeof buildCanvasFiles>[0];

export function useFilesFromCanvas(args: UseFilesFromCanvasArgs) {
  const { canvasYamlPayload, panels, layout, canvasId, canvasName, consoleLoading, consoleError } = args;

  return useMemo(
    () =>
      buildCanvasFiles({
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
