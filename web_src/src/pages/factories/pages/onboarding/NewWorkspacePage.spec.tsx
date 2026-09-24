import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { NewWorkspacePage } from "./NewWorkspacePage";

const createFactory = vi.fn();
const listedFactories: Array<{ name: string }> = [];
const githubAppAvailability = { resolved: true, available: true, failed: false, retry: vi.fn() };

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateFactory: () => ({ mutateAsync: createFactory, isPending: false }),
  useFactories: () => ({ data: listedFactories, isLoading: false }),
}));

vi.mock("./useGithubAppAvailability", () => ({
  useGithubAppAvailability: () => githubAppAvailability,
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
    githubAppAvailability.resolved = true;
    githubAppAvailability.available = true;
    githubAppAvailability.failed = false;
    githubAppAvailability.retry.mockReset();
  });

  it("creates the workspace with a placeholder name and opens the setup wizard", async () => {
    renderPage();

    expect(await screen.findByText("/org-1/workspaces/new/setup")).toBeInTheDocument();
    expect(createFactory).toHaveBeenCalledWith({ name: "New workspace", description: "", key: "" });
  });

  it("avoids a placeholder name that the organization already uses", async () => {
    listedFactories.push({ name: "New workspace" });
    renderPage();

    expect(await screen.findByText("/org-1/workspaces/new/setup")).toBeInTheDocument();
    expect(createFactory).toHaveBeenCalledWith({ name: "New workspace 2", description: "", key: "" });
  });

  it("keeps the user on the page when creation fails", async () => {
    createFactory.mockRejectedValue(new Error("Workspace limit reached"));
    renderPage();

    expect(await screen.findByText("Workspace limit reached")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Try again" })).toBeInTheDocument();
  });

  it("blocks setup and creates nothing when the GitHub App is not configured", async () => {
    githubAppAvailability.available = false;
    renderPage();

    expect(await screen.findByTestId("github-app-required")).toBeInTheDocument();
    expect(createFactory).not.toHaveBeenCalled();
  });

  it("waits for the integration catalog before creating the workspace", () => {
    githubAppAvailability.resolved = false;
    renderPage();

    expect(createFactory).not.toHaveBeenCalled();
    expect(screen.queryByTestId("github-app-required")).not.toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Checking GitHub setup…");
  });

  it("shows progress while the workspace is created", () => {
    createFactory.mockReturnValue(new Promise(() => undefined));

    renderPage();

    expect(screen.getByRole("status")).toHaveTextContent("Creating workspace…");
  });

  it("does not create a workspace when the catalog request fails", () => {
    githubAppAvailability.failed = true;
    githubAppAvailability.available = false;
    renderPage();

    expect(screen.getByText("SuperPlane could not check the GitHub App.")).toBeInTheDocument();
    expect(createFactory).not.toHaveBeenCalled();
  });
});
