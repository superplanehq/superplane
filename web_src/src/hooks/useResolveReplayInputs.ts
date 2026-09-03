import { useQuery } from "@tanstack/react-query";
import { canvasesResolveReplayInputs, type CanvasesResolvedReplayInput } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function useResolveReplayInputs({
  canvasId,
  nodeId,
  sourceExecutionId,
}: {
  canvasId: string | null;
  nodeId: string;
  sourceExecutionId?: string;
}) {
  return useQuery<CanvasesResolvedReplayInput[]>({
    queryKey: ["replay-inputs", canvasId, nodeId, sourceExecutionId],
    queryFn: async () => {
      if (!canvasId) {
        throw new Error("Canvas is required");
      }

      const response = await canvasesResolveReplayInputs(
        withOrganizationHeader({
          path: { canvasId, nodeId },
          query: { sourceExecutionId },
        }),
      );
      return response.data?.inputs ?? [];
    },
    enabled: Boolean(canvasId) && Boolean(sourceExecutionId),
  });
}
