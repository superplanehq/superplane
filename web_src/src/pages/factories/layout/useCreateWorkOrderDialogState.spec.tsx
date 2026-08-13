import { act, renderHook } from "@testing-library/react";
import { type ReactNode } from "react";
import { MemoryRouter, useLocation, useNavigate } from "react-router";
import { describe, expect, it } from "vitest";

import { createWorkOrderPath, linesPath, workOrderDetailPath, workOrdersPath } from "../lib/factoryPagePaths";
import { useCreateWorkOrderDialogState } from "./useCreateWorkOrderDialogState";

const ORGANIZATION_ID = "org-1";
const FACTORY_ID = "factory-1";

function wrapper(path: string) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <MemoryRouter initialEntries={[path]}>{children}</MemoryRouter>;
  };
}

function useDialogState(canCreate: boolean) {
  const location = useLocation();
  const navigate = useNavigate();
  const state = useCreateWorkOrderDialogState(ORGANIZATION_ID, FACTORY_ID, canCreate);
  return { ...state, pathname: location.pathname, navigate };
}

describe("useCreateWorkOrderDialogState", () => {
  it("does not open the create dialog from the deep link without create permission", () => {
    const { result } = renderHook(() => useDialogState(false), {
      wrapper: wrapper(createWorkOrderPath(ORGANIZATION_ID, FACTORY_ID)),
    });

    expect(result.current.createWorkOrderOpen).toBe(false);
  });

  it("opens the create dialog from the deep link when create is allowed", () => {
    const { result } = renderHook(() => useDialogState(true), {
      wrapper: wrapper(createWorkOrderPath(ORGANIZATION_ID, FACTORY_ID)),
    });

    expect(result.current.createWorkOrderOpen).toBe(true);
  });

  it("replaces the deep link with the new work order path after create", () => {
    const { result } = renderHook(() => useDialogState(true), {
      wrapper: wrapper(createWorkOrderPath(ORGANIZATION_ID, FACTORY_ID)),
    });

    act(() => {
      result.current.completeCreateWorkOrder("order-1");
    });

    expect(result.current.createWorkOrderOpen).toBe(false);
    expect(result.current.pathname).toBe(workOrderDetailPath(ORGANIZATION_ID, FACTORY_ID, "order-1"));
  });

  it("does not open from New Work Order when create is not allowed", () => {
    const { result } = renderHook(() => useDialogState(false), {
      wrapper: wrapper(workOrdersPath(ORGANIZATION_ID, FACTORY_ID)),
    });

    act(() => {
      result.current.openCreateWorkOrder();
    });

    expect(result.current.createWorkOrderOpen).toBe(false);
  });

  it("closes an opened dialog when the path changes", () => {
    const { result } = renderHook(() => useDialogState(true), {
      wrapper: wrapper(workOrdersPath(ORGANIZATION_ID, FACTORY_ID)),
    });

    act(() => {
      result.current.openCreateWorkOrder();
    });

    expect(result.current.createWorkOrderOpen).toBe(true);

    act(() => {
      result.current.navigate(linesPath(ORGANIZATION_ID, FACTORY_ID));
    });

    expect(result.current.createWorkOrderOpen).toBe(false);
  });
});
