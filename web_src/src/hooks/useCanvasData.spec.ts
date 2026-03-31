import { QueryClient } from "@tanstack/react-query";
import { describe, expect, it } from "vitest";

import { canvasKeys } from "@/hooks/useCanvasData";

describe("canvasKeys.nodeExecution", () => {
  it("omits a trailing undefined when limit is not provided", () => {
    expect(canvasKeys.nodeExecution("canvas-1", "node-1")).toEqual([
      "canvases",
      "nodeExecutions",
      "canvas-1",
      "node-1",
    ]);
  });

  it("matches limited cached queries when invalidating by prefix", async () => {
    const queryClient = new QueryClient();
    const cachedKey = canvasKeys.nodeExecution("canvas-1", "node-1", undefined, 10);

    queryClient.setQueryData(cachedKey, { executions: [] });

    await queryClient.invalidateQueries({
      queryKey: canvasKeys.nodeExecution("canvas-1", "node-1"),
    });

    expect(queryClient.getQueryState(cachedKey)?.isInvalidated).toBe(true);
  });
});
