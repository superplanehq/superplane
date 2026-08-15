import { useMutation } from "@tanstack/react-query";
import { canvasesReplayNode, type CanvasesReplayNodeBody } from "@/api-client";
import { showSuccessToast } from "@/lib/toast";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export const REPLAY_NODE_ERROR_FALLBACK = "Failed to replay node";

export function useReplayNode({
  canvasId,
  nodeId,
  onReplayed,
}: {
  canvasId: string | null;
  nodeId: string;
  onReplayed?: (runId?: string) => void;
}) {
  return useMutation({
    mutationFn: async (body: CanvasesReplayNodeBody) => {
      if (!canvasId) {
        throw new Error("Canvas is required");
      }

      const response = await canvasesReplayNode(
        withOrganizationHeader({
          path: { canvasId, nodeId },
          body,
        }),
      );

      return response.data;
    },
    onSuccess: (data) => {
      showSuccessToast("Replay queued");
      onReplayed?.(data?.runId);
    },
    // No error toast: the modal stays open on failure and renders the same message
    // inline, so a toast would report it twice.
    onError: (error) => {
      console.error("Failed to replay node", error);
    },
  });
}
