import type { ComponentProps } from "react";

import { WorkflowConsoleOverlay } from "./console/WorkflowConsoleOverlay";
import { WorkflowFilesOverlayLayer } from "./WorkflowFilesOverlayLayer";
import { WorkflowMemoryOverlayLayer } from "./WorkflowMemoryOverlayLayer";
import type { getWorkflowViewFlagsFromSearchParams } from "./viewState";

type UrlViewFlags = ReturnType<typeof getWorkflowViewFlagsFromSearchParams>;

type WorkflowPageModeOverlaysProps = {
  urlViewFlags: UrlViewFlags;
  console: Omit<ComponentProps<typeof WorkflowConsoleOverlay>, "isConsoleMode">;
  memory: Omit<ComponentProps<typeof WorkflowMemoryOverlayLayer>, "isMemoryMode">;
  files: Omit<ComponentProps<typeof WorkflowFilesOverlayLayer>, "isFilesMode">;
};

export function WorkflowPageModeOverlays({ urlViewFlags, console, memory, files }: WorkflowPageModeOverlaysProps) {
  return (
    <>
      <WorkflowConsoleOverlay isConsoleMode={urlViewFlags.isConsoleMode} {...console} />
      <WorkflowMemoryOverlayLayer isMemoryMode={urlViewFlags.isMemoryMode} {...memory} />
      <WorkflowFilesOverlayLayer isFilesMode={urlViewFlags.isFilesMode} {...files} />
    </>
  );
}
