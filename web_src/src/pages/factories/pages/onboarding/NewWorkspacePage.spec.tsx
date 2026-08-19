import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { NewWorkspacePage } from "./NewWorkspacePage";

const createFactory = vi.fn();
const listedFactories: Array<{ name: string }> = [];

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateFactory: () => ({ mutateAsync: createFactory, isPending: false }),
  useFactories: () => ({ data: listedFactories, isLoading: false }),
}));

function CurrentPath() {
  return <span>{useLocation().pathname}</span>;
}

function renderPage() {
  render(
    <MemoryRouter initialEntries={["/org-1/workspaces/new"]}>
      <Routes>
        <Route path="/:organizationId/workspaces/new" element={<NewWorkspacePage />} />
        <Route path="/:organizationId/workspaces/:factoryKey/setup" element={<CurrentPath />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("NewWorkspacePage", () => {
  beforeEach(() => {
    createFactory.mockReset();
    createFactory.mockResolvedValue({ id: "factory-1", key: "NEW", name: "New workspace" });
    listedFactories.length = 0;
  });

  it("creates the workspace with a placeholder name and opens the setup wizard", async () => {
    renderPage();

    expect(await screen.findByText("/org-1/workspaces/NEW/setup")).toBeInTheDocument();
    expect(createFactory).toHaveBeenCalledWith({ name: "New workspace", description: "", key: "" });
  });

  it("avoids a placeholder name that the organization already uses", async () => {
    listedFactories.push({ name: "New workspace" });
    renderPage();

    expect(await screen.findByText("/org-1/workspaces/NEW/setup")).toBeInTheDocument();
    expect(createFactory).toHaveBeenCalledWith({ name: "New workspace 2", description: "", key: "" });
  });

  it("keeps the user on the page when creation fails", async () => {
    createFactory.mockRejectedValue(new Error("Workspace limit reached"));
    renderPage();

    expect(await screen.findByText("Workspace limit reached")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });
});
