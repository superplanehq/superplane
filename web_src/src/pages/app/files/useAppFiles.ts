import { useMemo } from "react";

import { buildAppFiles } from "./lib/app-files";

type UseAppFilesArgs = Parameters<typeof buildAppFiles>[0];

export function useAppFiles(args: UseAppFilesArgs) {
  const { canvasYamlPayload, panels, layout, canvasId, canvasName, consoleLoading, consoleError } = args;

  return useMemo(
    () =>
      buildAppFiles({
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
