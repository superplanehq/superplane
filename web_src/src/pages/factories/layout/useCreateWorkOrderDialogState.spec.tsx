import { act, renderHook } from "@testing-library/react";
import { type ReactNode } from "react";
import { MemoryRouter, useLocation, useNavigate } from "react-router";
import { describe, expect, it } from "bun:test";

import {
  createWorkOrderPath,
  factoryLineDetailPath,
  linesPath,
  workOrderDetailPath,
  workOrdersPath,
} from "../lib/factoryPagePaths";
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
  return {
    ...state,
    pathname: location.pathname,
    href: `${location.pathname}${location.search}`,
    locationState: location.state,
    navigate,
  };
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

  it("opens the created task the same way as a GitHub import", () => {
    const { result } = renderHook(() => useDialogState(true), {
      wrapper: wrapper(createWorkOrderPath(ORGANIZATION_ID, FACTORY_KEY)),
    });
    const order = { id: "order-1", number: "101", title: "Ship the refunds line" };

    act(() => {
      result.current.completeCreateWorkOrder("101", order);
    });

    expect(result.current.createWorkOrderOpen).toBe(false);
    expect(result.current.href).toBe(
      workOrderDetailPath(ORGANIZATION_ID, FACTORY_KEY, "101", "line-plan-and-implement"),
    );
    expect(result.current.locationState).toEqual({ peekOrder: order });
  });

  it("keeps the current line when create finishes from a board", () => {
    const { result } = renderHook(() => useDialogState(true, "line-plan-and-implement"), {
      wrapper: wrapper(factoryLineDetailPath(ORGANIZATION_ID, FACTORY_KEY, "line-hotfix")),
    });

    act(() => {
      result.current.completeCreateWorkOrder("102");
    });

    expect(result.current.href).toBe(workOrderDetailPath(ORGANIZATION_ID, FACTORY_KEY, "102", "line-hotfix"));
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
