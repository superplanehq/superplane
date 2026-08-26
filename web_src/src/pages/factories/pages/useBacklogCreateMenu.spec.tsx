import { act, renderHook, waitFor } from "@testing-library/react";
import type { ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { BacklogIntakeItemsProvider } from "./BacklogIntakeItemsContext";
import { useBacklogCreateMenu } from "./useBacklogCreateMenu";

const useFactoryIntakes = vi.fn();
const createWorkOrder = vi.fn();
const importFactoryIntakeItem = vi.fn();
const useSearchFactoryIntakeItems = vi.fn();

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateWorkOrder: () => ({ mutateAsync: createWorkOrder, isPending: false }),
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => useFactoryIntakes(),
  useSearchFactoryIntakeItems: (...args: unknown[]) => useSearchFactoryIntakeItems(...args),
  useImportFactoryIntakeItem: () => ({ mutateAsync: importFactoryIntakeItem, isPending: false }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

const intake = {
  id: "intake-github-issues",
  name: "GitHub issues",
  source: "SOURCE_GITHUB_ISSUES" as const,
  healthy: true,
};

const liveItem = {
  id: "12",
  key: "#12",
  title: "Handle duplicate refunds",
  body: "Retrying a refund posts twice.",
  url: "https://github.com/acme/payments/issues/12",
};

function wrapper(catalogItems: { id: string; intakeId: string; key: string; title: string; body: string }[] = []) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return <BacklogIntakeItemsProvider catalog={{ items: catalogItems }}>{children}</BacklogIntakeItemsProvider>;
  };
}

describe("useBacklogCreateMenu", () => {
  beforeEach(() => {
    useFactoryIntakes.mockReturnValue({ data: [intake], isLoading: false });
    useSearchFactoryIntakeItems.mockReturnValue({ data: [liveItem], isLoading: false, isError: false });
    createWorkOrder.mockReset();
    importFactoryIntakeItem.mockReset();
  });

  it("searches the focused intake and imports through the intake RPC", async () => {
    const onImported = vi.fn();
    const { result } = renderHook(() => useBacklogCreateMenu("org-1", "factory-1", onImported), {
      wrapper: wrapper(),
    });

    act(() => {
      result.current.setFocusedIntake(intake.id);
    });

    await waitFor(() => {
      expect(result.current.items).toEqual([
        {
          id: "12",
          intakeId: intake.id,
          key: "#12",
          title: "Handle duplicate refunds",
          body: "Retrying a refund posts twice.",
        },
      ]);
    });
    expect(useSearchFactoryIntakeItems).toHaveBeenCalledWith("org-1", "factory-1", intake.id, "", true, 5);

    importFactoryIntakeItem.mockResolvedValue({
      id: "wo-imported-12",
      title: "Handle duplicate refunds",
      state: "STATE_DRAFT",
    });
    await act(async () => {
      await result.current.importItem(result.current.items[0]!);
    });

    expect(importFactoryIntakeItem).toHaveBeenCalledWith({ intakeId: intake.id, itemId: "12" });
    expect(createWorkOrder).not.toHaveBeenCalled();
    expect(onImported).toHaveBeenCalledWith("wo-imported-12", {
      id: "wo-imported-12",
      title: "Handle duplicate refunds",
      state: "STATE_DRAFT",
    });
  });

  it("creates from the Storybook catalog without calling import", async () => {
    const onImported = vi.fn();
    const catalogItem = {
      id: "gh-issue-12",
      intakeId: intake.id,
      key: "#12",
      title: "Handle duplicate refunds on retry",
      body: "A second refund request posts a second credit.",
    };
    const { result } = renderHook(() => useBacklogCreateMenu("org-1", "factory-1", onImported), {
      wrapper: wrapper([catalogItem]),
    });

    act(() => {
      result.current.setFocusedIntake(intake.id);
    });

    expect(useSearchFactoryIntakeItems).toHaveBeenCalledWith("org-1", "factory-1", intake.id, "", false, 5);
    await act(async () => {
      await result.current.importItem(result.current.items[0]!);
    });

    expect(createWorkOrder).toHaveBeenCalledWith({
      title: catalogItem.title,
      description: catalogItem.body,
    });
    expect(importFactoryIntakeItem).not.toHaveBeenCalled();
    expect(onImported).not.toHaveBeenCalled();
  });

  it("pages catalog matches and asks the live search for the next limit", async () => {
    const catalogItems = Array.from({ length: 8 }, (_, index) => ({
      id: `gh-${index}`,
      intakeId: intake.id,
      key: `#${index}`,
      title: `Issue ${index}`,
      body: "",
    }));
    const { result } = renderHook(() => useBacklogCreateMenu("org-1", "factory-1"), {
      wrapper: wrapper(catalogItems),
    });

    act(() => {
      result.current.setFocusedIntake(intake.id);
    });

    expect(result.current.items).toHaveLength(5);
    expect(result.current.hasMore).toBe(true);

    act(() => {
      result.current.loadMore();
    });

    expect(result.current.items).toHaveLength(8);
    expect(result.current.hasMore).toBe(false);
  });
});
