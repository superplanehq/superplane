import { renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import {
  resetDiscussionHandlerProvisionAttempts,
  useAutoProvisionDiscussionHandler,
} from "./useAutoProvisionDiscussionHandler";

describe("useAutoProvisionDiscussionHandler", () => {
  beforeEach(() => {
    resetDiscussionHandlerProvisionAttempts();
  });

  it("provisions the next workspace after the previous workspace fails", async () => {
    const listHandlers = vi.fn().mockResolvedValue([]);
    const createA = vi.fn().mockRejectedValue(new Error("failed"));
    const createB = vi.fn().mockResolvedValue({ id: "handler-b" });

    const { result, rerender } = renderHook(
      ({ factoryId, createHandler }: { factoryId: string; createHandler: typeof createA }) =>
        useAutoProvisionDiscussionHandler({
          factoryId,
          enabled: true,
          listHandlers,
          createHandler,
        }),
      { initialProps: { factoryId: "factory-a", createHandler: createA } },
    );

    await waitFor(() => {
      expect(result.current.failed).toBe(true);
    });
    expect(createA).toHaveBeenCalledTimes(1);

    rerender({ factoryId: "factory-b", createHandler: createB });

    await waitFor(() => {
      expect(createB).toHaveBeenCalledTimes(1);
    });
    expect(result.current.failed).toBe(false);
  });
});
