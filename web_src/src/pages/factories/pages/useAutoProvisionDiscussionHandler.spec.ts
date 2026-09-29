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

  it("creates the handler in the workspace that started the attempt", async () => {
    let resolveListA: (handlers: []) => void = () => {};
    const listA = vi.fn(
      () =>
        new Promise<[]>((resolve) => {
          resolveListA = resolve;
        }),
    );
    const listB = vi.fn().mockResolvedValue([]);
    const createA = vi.fn().mockResolvedValue({ id: "handler-a" });
    const createB = vi.fn().mockResolvedValue({ id: "handler-b" });

    const { rerender } = renderHook(
      ({
        factoryId,
        listHandlers,
        createHandler,
      }: {
        factoryId: string;
        listHandlers: typeof listA;
        createHandler: typeof createA;
      }) =>
        useAutoProvisionDiscussionHandler({
          factoryId,
          enabled: true,
          listHandlers,
          createHandler,
        }),
      { initialProps: { factoryId: "factory-a", listHandlers: listA, createHandler: createA } },
    );

    await waitFor(() => {
      expect(listA).toHaveBeenCalledTimes(1);
    });

    rerender({ factoryId: "factory-b", listHandlers: listB, createHandler: createB });
    resolveListA([]);

    await waitFor(() => {
      expect(createA).toHaveBeenCalledTimes(1);
      expect(createB).toHaveBeenCalledTimes(1);
    });
    expect(listB).toHaveBeenCalledTimes(1);
  });
});
