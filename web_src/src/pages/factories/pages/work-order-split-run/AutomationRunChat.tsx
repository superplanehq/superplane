import type { FilesFile } from "@/api-client";
import { useWorkOrderFileUpload } from "@/hooks/useWorkOrderFileUpload";
import type { SplitRunSource } from "./splitRunSource";
import { useCanvas } from "@/hooks/useCanvasData";
import { CUSTOM_AUTOMATION_TEMPLATE_ID } from "../../lib/customAutomation";
import { WorkOrderIntentRequest } from "./WorkOrderIntentRequest";
import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";
import type { SplitRunPhase } from "./splitRunMocks";

/** Each run owns its conversation, including its continuation executions. */
export function AutomationRunChat({
  organizationId,
  factoryId,
  workOrderId,
  phase,
  canUpdate,
  description = "",
  files,
  source,
}: {
  organizationId: string;
  factoryId: string;
  workOrderId: string;
  phase: SplitRunPhase;
  canUpdate: boolean;
  description?: string;
  files?: FilesFile[];
  source?: SplitRunSource;
}) {
  const canvas = useCanvas(organizationId, phase.appId ?? "", { enabled: Boolean(phase.appId) });
  const custom = canvas.data?.spec?.nodes?.some((node) => {
    const template = node.metadata?.factoryTemplate;
    return (
      template && typeof template === "object" && "id" in template && template.id === CUSTOM_AUTOMATION_TEMPLATE_ID
    );
  });
  const uploads = useWorkOrderFileUpload({ organizationId, factoryId, orderId: workOrderId });
  const chat = useAnalysisPlanningSession({
    organizationId,
    factoryId,
    workOrderId,
    runId: phase.runId,
    enabled: Boolean(custom && phase.runId),
    canUpdate,
    isUploading: uploads.isUploading,
    uploadFiles: uploads.uploadFiles,
    refinementCanStart: phase.status === "running" || phase.status === "pending",
  });
  if (custom && chat.queryError)
    return (
      <p role="alert" className="text-sm text-destructive">
        The conversation did not load. Refresh the page to try again.
      </p>
    );
  if (!chat.session) return null;
  return (
    <WorkOrderIntentRequest
      title={phase.componentName}
      description={description}
      files={files}
      source={source}
      analysis={{ ...chat, mode: "automation", planningSessionExists: true }}
    />
  );
}
