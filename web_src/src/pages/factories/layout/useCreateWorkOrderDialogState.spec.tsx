import { act, renderHook } from "@testing-library/react";
import { type ReactNode } from "react";
import { MemoryRouter, useLocation, useNavigate } from "react-router";
import { describe, expect, it } from "vitest";

import { createWorkOrderPath, factoryHomePath, linesPath, workOrdersPath } from "../lib/factoryPagePaths";
import { useCreateWorkOrderDialogState } from "./useCreateWorkOrderDialogState";

const ORGANIZATION_ID = "org-1";
const FACTORY_KEY = "RF";

function wrapper(path: string) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <MemoryRouter initialEntries={[path]}>{children}</MemoryRouter>;
  };
}

function useDialogState(canCreate: boolean, firstLineId = "line-plan-and-implement") {
  const location = useLocation();
  const navigate = useNavigate();
  const state = useCreateWorkOrderDialogState(ORGANIZATION_ID, FACTORY_KEY, canCreate, firstLineId);
  return { ...state, pathname: location.pathname, navigate };
}

describe("useCreateWorkOrderDialogState", () => {
  it("does not open the create dialog from the deep link without create permission", () => {
    const { result } = renderHook(() => useDialogState(false), {
      wrapper: wrapper(createWorkOrderPath(ORGANIZATION_ID, FACTORY_KEY)),
    });

    expect(result.current.createWorkOrderOpen).toBe(false);
  });

  it("opens the create dialog from the deep link when create is allowed", () => {
    const { result } = renderHook(() => useDialogState(true), {
      wrapper: wrapper(createWorkOrderPath(ORGANIZATION_ID, FACTORY_KEY)),
    });

    expect(result.current.createWorkOrderOpen).toBe(true);
  });

  it("replaces the deep link with the line board after create", () => {
    const { result } = renderHook(() => useDialogState(true), {
      wrapper: wrapper(createWorkOrderPath(ORGANIZATION_ID, FACTORY_KEY)),
    });

    act(() => {
      result.current.completeCreateWorkOrder("101");
    });

    expect(result.current.createWorkOrderOpen).toBe(false);
    expect(result.current.pathname).toBe(factoryHomePath(ORGANIZATION_ID, FACTORY_KEY, "line-plan-and-implement"));
  });

  it("does not open from New Task when create is not allowed", () => {
    const { result } = renderHook(() => useDialogState(false), {
      wrapper: wrapper(workOrdersPath(ORGANIZATION_ID, FACTORY_KEY)),
    });

    act(() => {
      result.current.openCreateWorkOrder();
    });

    expect(result.current.createWorkOrderOpen).toBe(false);
  });

  it("closes an opened dialog when the path changes", () => {
    const { result } = renderHook(() => useDialogState(true), {
      wrapper: wrapper(workOrdersPath(ORGANIZATION_ID, FACTORY_KEY)),
    });

    act(() => {
      result.current.openCreateWorkOrder();
    });

    expect(result.current.createWorkOrderOpen).toBe(true);

    act(() => {
      result.current.navigate(linesPath(ORGANIZATION_ID, FACTORY_KEY));
    });

    expect(result.current.createWorkOrderOpen).toBe(false);
  });
});
