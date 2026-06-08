import { useMemo } from "react";

import { buildAppFiles } from "./lib/app-files";

type UseAppFilesArgs = Parameters<typeof buildAppFiles>[0];

export function useAppFiles(args: UseAppFilesArgs) {
  const { canvas, canvasNodes, panels, layout, canvasId, canvasName, consoleLoading, consoleError } = args;

  return useMemo(
    () =>
      buildAppFiles({
        canvas,
        canvasNodes,
        panels,
        layout,
        canvasId,
        canvasName,
        consoleLoading,
        consoleError,
      }),
    [canvas, canvasName, canvasId, canvasNodes, consoleError, consoleLoading, layout, panels],
  );
}
