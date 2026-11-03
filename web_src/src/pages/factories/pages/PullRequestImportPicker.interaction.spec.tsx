import type { FactoriesFactoryPullRequestImportItem } from "@/api-client";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { PullRequestImportPicker } from "./PullRequestImportPicker";
import { CodexActionCancelled } from "../codex/codexActionContext";

let archivedTask = false;
let enabled = true;
let canCreate = true;
let canUpdate = true;
const prepareOpen = vi.fn(async () => true);
const importPR = vi.fn(async (_input: { lineId: string; pullRequestUrl: string }) => ({
  id: "new-task",
  number: "47",
}));
const refetch = vi.fn();
const searchPRs = vi.fn(
  (_organizationId: string, _factoryId: string, _query: string, _limit: number, _open: boolean) => ({
    data: [
      { url: "https://github.com/acme/app/pull/42", title: "New PR", number: "42" },
      {
        url: "https://github.com/acme/app/pull/43",
        title: "Existing PR",
        number: "43",
        existingTask: {
          id: "existing-task",
          number: "46",
          ...(archivedTask ? { state: "STATE_CLOSED", result: "RESULT_ARCHIVED" } : {}),
        },
      },
    ] as FactoriesFactoryPullRequestImportItem[],
    isLoading: false,
    isFetching: false,
    isError: false,
    refetch,
  }),
);
vi.mock("@/hooks/useExperimentalFeature", () => ({ useExperimentalFeature: () => ({ has: () => enabled }) }));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (_resource: string, action: string) => (action === "create" ? canCreate : canUpdate),
  }),
}));
vi.mock("./useBacklogCreateConnection", () => ({ useBacklogCreateConnection: () => prepareOpen }));
vi.mock("@/hooks/useFactoryPullRequestImports", () => ({
  useImportFactoryPullRequest: () => ({ mutateAsync: importPR, isPending: false }),
  useSearchFactoryPullRequestImports: (...args: Parameters<typeof searchPRs>) => searchPRs(...args),
}));

beforeEach(() => {
  archivedTask = false;
  enabled = canCreate = canUpdate = true;
  prepareOpen.mockReset();
  prepareOpen.mockResolvedValue(true);
  importPR.mockReset();
  importPR.mockResolvedValue({ id: "new-task", number: "47" });
  searchPRs.mockClear();
});

function picker(mobile = false, onOpenTask = vi.fn()) {
  return (
    <PullRequestImportPicker
      organizationId="org"
      factoryId="workspace"
      lineId="selected-line"
      mobile={mobile}
      onOpenTask={onOpenTask}
    />
  );
}

describe("Pull request picker integration", () => {
  it("hides the importer when the flag is off or a required permission is missing", () => {
    enabled = false;
    const view = render(picker());
    expect(screen.queryByRole("button", { name: "Import pull request" })).toBeNull();
    enabled = true;
    canCreate = false;
    view.rerender(picker());
    expect(screen.queryByRole("button", { name: "Import pull request" })).toBeNull();
    canCreate = true;
    canUpdate = false;
    view.rerender(picker());
    expect(screen.queryByRole("button", { name: "Import pull request" })).toBeNull();
  });

  for (const mobile of [false, true]) {
    it(`imports into the selected line and opens the task on ${mobile ? "mobile" : "desktop"}`, async () => {
      const onOpenTask = vi.fn();
      render(picker(mobile, onOpenTask));
      fireEvent.click(screen.getByRole("button", { name: "Import pull request" }));
      const row = await screen.findByRole("button", { name: /New PR/ });
      fireEvent.click(row);
      fireEvent.click(row);
      await waitFor(() => expect(onOpenTask).toHaveBeenCalledWith("new-task", { id: "new-task", number: "47" }));
      expect(importPR).toHaveBeenCalledTimes(1);
      expect(importPR).toHaveBeenCalledWith({
        lineId: "selected-line",
        pullRequestUrl: "https://github.com/acme/app/pull/42",
      });
    });
  }

  it("resolves an existing task through the API before opening it", async () => {
    importPR.mockResolvedValueOnce({ id: "existing-task", number: "46" });
    const onOpenTask = vi.fn();
    render(picker(false, onOpenTask));
    fireEvent.click(screen.getByRole("button", { name: "Import pull request" }));
    fireEvent.click(await screen.findByRole("button", { name: /Already imported/ }));
    await waitFor(() =>
      expect(importPR).toHaveBeenCalledWith({
        lineId: "selected-line",
        pullRequestUrl: "https://github.com/acme/app/pull/43",
      }),
    );
    expect(onOpenTask).toHaveBeenCalledWith("existing-task", { id: "existing-task", number: "46" });
  });

  for (const cachedArchiveState of [true, false]) {
    it(`imports again when the cached archive state is ${cachedArchiveState}`, async () => {
      archivedTask = cachedArchiveState;
      const onOpenTask = vi.fn();
      render(picker(false, onOpenTask));
      fireEvent.click(screen.getByRole("button", { name: "Import pull request" }));
      fireEvent.click(await screen.findByRole("button", { name: /Existing PR/ }));
      await waitFor(() =>
        expect(importPR).toHaveBeenCalledWith({
          lineId: "selected-line",
          pullRequestUrl: "https://github.com/acme/app/pull/43",
        }),
      );
      expect(onOpenTask).toHaveBeenCalledWith("new-task", { id: "new-task", number: "47" });
    });
  }

  it("keeps connection cancellation retryable", async () => {
    prepareOpen.mockResolvedValueOnce(false);
    render(picker());
    fireEvent.click(screen.getByRole("button", { name: "Import pull request" }));
    await waitFor(() => expect(prepareOpen).toHaveBeenCalledTimes(1));
    expect(screen.queryByRole("textbox")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Import pull request" }));
    await screen.findByRole("textbox", { name: "Search pull requests" });
  });

  it("allows retry after connection cancellation during import", async () => {
    importPR.mockRejectedValueOnce(new CodexActionCancelled());
    render(picker());
    fireEvent.click(screen.getByRole("button", { name: "Import pull request" }));
    fireEvent.click(await screen.findByRole("button", { name: /New PR/ }));
    await waitFor(() => expect(importPR).toHaveBeenCalledTimes(1));
    expect(screen.queryByRole("alert")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /New PR/ }));
    await waitFor(() => expect(importPR).toHaveBeenCalledTimes(2));
  });

  it("delays provider searches until typing stops", async () => {
    render(picker());
    fireEvent.click(screen.getByRole("button", { name: "Import pull request" }));
    const input = await screen.findByRole("textbox", { name: "Search pull requests" });
    fireEvent.change(input, { target: { value: "retry" } });
    expect(searchPRs).not.toHaveBeenCalledWith("org", "workspace", "retry", 5, true);
    await waitFor(() => expect(searchPRs).toHaveBeenCalledWith("org", "workspace", "retry", 5, true));
  });
});
