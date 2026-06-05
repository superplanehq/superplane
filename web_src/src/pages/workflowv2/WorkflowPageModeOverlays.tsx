import type { ComponentProps } from "react";

import { WorkflowDashboardOverlay } from "./dashboard/WorkflowDashboardOverlay";
import { WorkflowFilesOverlayLayer } from "./WorkflowFilesOverlayLayer";
import { WorkflowMemoryOverlayLayer } from "./WorkflowMemoryOverlayLayer";
import type { getWorkflowViewFlagsFromSearchParams } from "./viewState";

type UrlViewFlags = ReturnType<typeof getWorkflowViewFlagsFromSearchParams>;

type WorkflowPageModeOverlaysProps = {
  urlViewFlags: UrlViewFlags;
  dashboard: Omit<ComponentProps<typeof WorkflowDashboardOverlay>, "isDashboardMode">;
  memory: Omit<ComponentProps<typeof WorkflowMemoryOverlayLayer>, "isMemoryMode">;
  files: Omit<ComponentProps<typeof WorkflowFilesOverlayLayer>, "isFilesMode">;
};

export function WorkflowPageModeOverlays({ urlViewFlags, dashboard, memory, files }: WorkflowPageModeOverlaysProps) {
  return (
    <>
      <WorkflowDashboardOverlay isDashboardMode={urlViewFlags.isDashboardMode} {...dashboard} />
      <WorkflowMemoryOverlayLayer isMemoryMode={urlViewFlags.isMemoryMode} {...memory} />
      <WorkflowFilesOverlayLayer isFilesMode={urlViewFlags.isFilesMode} {...files} />
    </>
  );
}
