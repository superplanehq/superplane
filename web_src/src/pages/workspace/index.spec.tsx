import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { OrgWorkspaceHarness } from "@/pages/__fixtures__/OrgWorkspaceHarness";

import { projectXWorkspaceData } from "./__fixtures__/workspaceData";
import { WorkspacePage } from "./index";

describe("WorkspacePage", () => {
  it("shows the Project X software factory overview", async () => {
    render(<OrgWorkspaceHarness startAt="app" appElement={<WorkspacePage data={projectXWorkspaceData} />} />);

    expect(await screen.findByRole("heading", { name: "Project X" })).toBeInTheDocument();
    expect(screen.getByText("Software Factory")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Factory flow" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Active work" })).toBeInTheDocument();
    expect(screen.getByText("Add SSO session recovery")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Recent delivery" })).toBeInTheDocument();
  });

  it("starts a new factory work item", async () => {
    const user = userEvent.setup();
    const onCreateWork = vi.fn();
    render(
      <OrgWorkspaceHarness
        startAt="app"
        appElement={<WorkspacePage data={projectXWorkspaceData} onCreateWork={onCreateWork} />}
      />,
    );

    await user.click(await screen.findByRole("button", { name: "New work" }));
    await user.type(screen.getByLabelText("Work item"), "Upgrade the billing API");
    await user.type(screen.getByLabelText("Goal"), "Move Project X to the new billing endpoint.");
    await user.click(screen.getByRole("button", { name: "Start factory run" }));

    expect(onCreateWork).toHaveBeenCalledWith({
      title: "Upgrade the billing API",
      goal: "Move Project X to the new billing endpoint.",
    });
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });
});
