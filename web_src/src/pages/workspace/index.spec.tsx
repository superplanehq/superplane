import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { OrgWorkspaceHarness } from "@/pages/__fixtures__/OrgWorkspaceHarness";

import { projectXWorkspaceData } from "./__fixtures__/workspaceData";
import { WorkspacePage } from "./index";

describe("WorkspacePage", () => {
  it("shows the Factory overview and work order queues", async () => {
    const user = userEvent.setup();
    render(<OrgWorkspaceHarness startAt="app" appElement={<WorkspacePage data={projectXWorkspaceData} />} />);

    expect(await screen.findByRole("heading", { name: "Project X Factory" })).toBeInTheDocument();
    expect(screen.getByText("Software Factory")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Needs attention" })).toBeInTheDocument();
    expect(screen.getByText("Choose session recovery fallback")).toBeInTheDocument();
    expect(screen.getByText("Work orders completed")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Factory automations" })).not.toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Work Orders" }));

    expect(screen.getByRole("heading", { name: "Running" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Recently done" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Unsuccessful" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Repository velocity" })).not.toBeInTheDocument();
  });

  it("shows the Factory automations and repository velocity", async () => {
    const user = userEvent.setup();
    render(<OrgWorkspaceHarness startAt="app" appElement={<WorkspacePage data={projectXWorkspaceData} />} />);

    await user.click(screen.getByRole("tab", { name: "Automations" }));

    expect(screen.getByRole("heading", { name: "Factory automations" })).toBeInTheDocument();
    expect(screen.getByText("Issue intake")).toBeInTheDocument();
    expect(screen.getByText("Plan and build")).toBeInTheDocument();

    await user.click(screen.getByRole("tab", { name: "Velocity" }));

    expect(screen.getByRole("heading", { name: "Repository velocity" })).toBeInTheDocument();
    expect(screen.getByText("Last 14 days")).toBeInTheDocument();
    expect(screen.getByText("Team total")).toBeInTheDocument();
    expect(screen.getByText("Human-authored")).toBeInTheDocument();
    expect(screen.getByText("Factory-authored")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Delivery indicators" })).toBeInTheDocument();
    expect(screen.getAllByText("Merged pull requests")).toHaveLength(2);
    expect(screen.getByRole("heading", { name: "Tracked execution cost" })).toBeInTheDocument();
    expect(screen.getByText("Third-party service charges are excluded.", { exact: false })).toBeInTheDocument();
    expect(screen.queryByText("@maya-chen")).not.toBeInTheDocument();
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
