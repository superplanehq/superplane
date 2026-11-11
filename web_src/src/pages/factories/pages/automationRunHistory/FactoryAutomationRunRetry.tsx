import { useMutation, useQueryClient } from "@tanstack/react-query";
import { canvasesReemitTriggerEvent, type CanvasesCanvasRun } from "@/api-client";
import { Button } from "@/components/ui/button";
import { getApiErrorMessage } from "@/lib/errors";
import { showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function FactoryAutomationRunRetry({
  organizationId,
  canvasId,
  run,
  enabled,
  onRetried,
}: {
  organizationId: string;
  canvasId: string;
  run: CanvasesCanvasRun;
  enabled: boolean;
  onRetried: () => void;
}) {
  const queryClient = useQueryClient();
  const retry = useMutation({
    mutationFn: async () => {
      if (!run.rootEvent?.nodeId || !run.rootEvent?.id) throw new Error("The source event is missing.");
      await canvasesReemitTriggerEvent(
        withOrganizationHeader({
          organizationId,
          path: { canvasId, nodeId: run.rootEvent.nodeId, eventId: run.rootEvent.id },
        }),
      );
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["canvases"] });
      showSuccessToast("Run retry requested.");
      onRetried();
    },
  });
  return (
    <div className="mb-4 flex flex-wrap items-center justify-end gap-3">
      {retry.isError ? (
        <p role="alert" className="mr-auto text-sm text-destructive">
          {getApiErrorMessage(retry.error, "Could not retry this run. Try again.")}
        </p>
      ) : null}
      <Button
        variant="outline"
        size="sm"
        disabled={
          !enabled || retry.isPending || run.state !== "STATE_FINISHED" || !run.rootEvent?.id || !run.rootEvent.nodeId
        }
        onClick={() => retry.mutate()}
      >
        {retry.isPending ? "Retrying run" : "Retry run"}
      </Button>
    </div>
  );
}
