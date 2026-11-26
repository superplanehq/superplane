import { useRef, useState } from "react";
import { useNavigate } from "react-router";
import { useRunFactoryCustomAutomation } from "./useFactoryCustomAutomationData";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { factoryCustomAutomationRunPath } from "@/pages/factories/lib/customAutomationPaths";
import type { ColumnAutomation } from "@/pages/factories/lib/columnAutomations";

export function useBacklogScheduleRun(organizationId: string, factoryId: string, factoryKey: string, lineId?: string) {
  const navigate = useNavigate();
  const runAutomation = useRunFactoryCustomAutomation(organizationId, factoryId);
  const [runningAutomationId, setRunningAutomationId] = useState<string>();
  const running = useRef(false);
  const onRun = async (automation: ColumnAutomation) => {
    if (!automation.canvasId || !lineId || running.current) return;
    running.current = true;
    setRunningAutomationId(automation.canvasId);
    try {
      const started = await runAutomation.mutateAsync({ automationId: automation.canvasId, workOrderId: "" });
      if (started.runId)
        navigate(
          factoryCustomAutomationRunPath(organizationId, factoryKey, automation.canvasId, started.runId),
        );
    } catch (error) {
      showErrorToast(getApiErrorMessage(error, "SuperPlane could not start the automation."));
    } finally {
      running.current = false;
      setRunningAutomationId(undefined);
    }
  };
  return { onRun, runPending: Boolean(runningAutomationId), runningAutomationId };
}
