import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { createMutate, permissionsState } = vi.hoisted(() => ({
  createMutate: vi.fn(),
  permissionsState: { currentUserId: undefined as string | undefined },
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateWorkOrder: () => ({ mutateAsync: createMutate, isPending: false }),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => permissionsState,
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
    onClose.mockReset();
    onCreated.mockReset();
    permissionsState.currentUserId = undefined;
  });

  it("marks Create as loading while the task is created", async () => {
    let resolveCreate: (order: { id: string }) => void = () => {};
    createMutate.mockImplementation(
      () =>
        new Promise<{ id: string }>((resolve) => {
          resolveCreate = resolve;
        }),
    );

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

    act(() => {
      void result.current.handleCreate();
    });

    expect(result.current.isCreating).toBe(true);

    await act(async () => {
      resolveCreate({ id: "order-1" });
    });
  });

  it("seeds assigneeIds with the current user once me resolves", () => {
    permissionsState.currentUserId = "user-me";

    const { result } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    expect(result.current.assigneeIds).toEqual(["user-me"]);
  });

  it("keeps assigneeIds empty when there is no current user", () => {
    const { result } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    expect(result.current.assigneeIds).toEqual([]);
  });

  it("does not override a manual assignment made before me resolves", () => {
    const { result, rerender } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    act(() => {
      result.current.setAssigneeIds(["user-manual"]);
    });

    permissionsState.currentUserId = "user-me";
    rerender();

    expect(result.current.assigneeIds).toEqual(["user-manual"]);
  });

  it("does not clobber a manual change made after me resolves", () => {
    permissionsState.currentUserId = "user-me";

    const { result } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    expect(result.current.assigneeIds).toEqual(["user-me"]);

    act(() => {
      result.current.setAssigneeIds([]);
    });

    expect(result.current.assigneeIds).toEqual([]);
  });

  it("opens the new task without closing to the list first", async () => {
    createMutate.mockResolvedValue({ id: "order-1", number: "101" });

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
      await result.current.handleCreate();
    });

    expect(onCreated).toHaveBeenCalledWith("101", { id: "order-1", number: "101" });
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

  it("creates from a request draft and fills the title from the first body line", async () => {
    createMutate.mockResolvedValue({ id: "order-1", number: "101" });

    const { result } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    await act(async () => {
      await result.current.handleCreate({
        title: "",
        description: "Refunds fail on retry.",
      });
    });

    expect(createMutate).toHaveBeenCalledWith({
      title: "Refunds fail on retry.",
      description: "Refunds fail on retry.",
      assigneeIds: [],
    });
    expect(onCreated).toHaveBeenCalledWith("101", { id: "order-1", number: "101" });
  });

  it("uses New task when a request draft has no title text", async () => {
    createMutate.mockResolvedValue({ id: "order-1", number: "102" });

    const { result } = renderHook(() =>
      useCreateWorkOrderComposer({
        organizationId: "org-1",
        factoryId: "factory-1",
        onClose,
        onCreated,
      }),
    );

    await act(async () => {
      await result.current.handleCreate({
        title: "",
        description: "![receipt.png](sp-file://file-2)",
      });
    });

    expect(createMutate).toHaveBeenCalledWith({
      title: "New task",
      description: "![receipt.png](sp-file://file-2)",
      assigneeIds: [],
    });
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
