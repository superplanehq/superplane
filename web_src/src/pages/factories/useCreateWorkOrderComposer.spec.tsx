import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

const { createMutate, dispatchMutate } = vi.hoisted(() => ({
  createMutate: vi.fn(),
  dispatchMutate: vi.fn(),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateWorkOrder: () => ({ mutateAsync: createMutate, isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: dispatchMutate, isPending: false }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

import { useCreateWorkOrderComposer } from "./useCreateWorkOrderComposer";

describe("useCreateWorkOrderComposer", () => {
  const onClose = vi.fn();
  const onCreated = vi.fn();

  beforeEach(() => {
    createMutate.mockReset();
    dispatchMutate.mockReset();
    onClose.mockReset();
    onCreated.mockReset();
  });

  it("marks Send to line as loading while the work order is created", async () => {
    let resolveCreate: (order: { id: string }) => void = () => {};
    createMutate.mockImplementation(
      () =>
        new Promise<{ id: string }>((resolve) => {
          resolveCreate = resolve;
        }),
    );
    dispatchMutate.mockResolvedValue({});

    const { result } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    act(() => {
      result.current.updateTitle("Ship the refunds line");
      result.current.setSelectedLineName("plan-and-implement");
    });

    act(() => {
      void result.current.handleSendToLine();
    });

    expect(result.current.isSendingToLine).toBe(true);
    expect(result.current.isSavingDraft).toBe(false);

    await act(async () => {
      resolveCreate({ id: "order-1" });
    });
  });

  it("opens the new work order without closing to the list first", async () => {
    createMutate.mockResolvedValue({ id: "order-1" });

    const { result } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    act(() => {
      result.current.updateTitle("Ship the refunds line");
    });

    await act(async () => {
      await result.current.handleSaveDraft();
    });

    expect(onCreated).toHaveBeenCalledWith("order-1");
    expect(onClose).not.toHaveBeenCalled();
  });

  it("keeps the first 256 characters of a long pasted title", () => {
    const { result } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    act(() => {
      result.current.updateTitle("a".repeat(300));
    });

    expect(result.current.title).toHaveLength(256);
  });

  it("keeps the first 5000 characters of a long pasted description", () => {
    const { result } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    act(() => {
      result.current.updateDescription("a".repeat(5200));
    });

    expect(result.current.description).toHaveLength(5000);
  });
});
