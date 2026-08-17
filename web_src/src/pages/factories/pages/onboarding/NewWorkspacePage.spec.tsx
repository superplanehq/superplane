import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { NewWorkspacePage } from "./NewWorkspacePage";

const createFactory = vi.fn().mockResolvedValue({ id: "factory-1", key: "PAY", name: "Payments" });

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateFactory: () => ({ mutateAsync: createFactory, isPending: false }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationInviteLink: () => ({ data: undefined, isLoading: false }),
}));

function CurrentPath() {
  return <span>{useLocation().pathname}</span>;
}

function renderPage() {
  render(
    <MemoryRouter initialEntries={["/org-1/workspaces/new"]}>
      <Routes>
        <Route path="/:organizationId/workspaces/new" element={<NewWorkspacePage />} />
        <Route path="/:organizationId/workspaces/:factoryKey/onboarding" element={<CurrentPath />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("NewWorkspacePage", () => {
  it("creates the workspace from the name step and opens its setup wizard", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("Workspace name"), "Payments");
    await user.click(screen.getByRole("button", { name: "Continue to version control" }));

    expect(createFactory).toHaveBeenCalledWith({ name: "Payments", description: "", key: "" });
    expect(await screen.findByText("/org-1/workspaces/PAY/onboarding")).toBeInTheDocument();
  });

  it("keeps version control locked until the workspace exists", async () => {
    const user = userEvent.setup();
    renderPage();

    await user.type(screen.getByLabelText("Workspace name"), "Payments");

    expect(screen.getByRole("button", { name: /Version control/ })).toBeDisabled();
  });
});
