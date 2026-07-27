import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "vitest";

import { OrgWorkspaceHarness } from "@/pages/__fixtures__/OrgWorkspaceHarness";

import { activeWorkItemData } from "./__fixtures__/activeWorkItemData";
import { projectXWorkspaceData } from "./__fixtures__/workspaceData";
import { ActiveWorkItemPage } from "./ActiveWorkItemPage";
import { WorkspacePage } from "./index";

describe("ActiveWorkItemPage", () => {
  it("shows the chronological work record and plan checkpoint", async () => {
    render(
      <OrgWorkspaceHarness
        startAt="workItem"
        appElement={<WorkspacePage data={projectXWorkspaceData} />}
        workItemElement={<ActiveWorkItemPage data={activeWorkItemData} />}
      />,
    );

    expect(await screen.findByRole("heading", { name: "Add SSO session recovery" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Chronology" })).toBeInTheDocument();
    expect(screen.getByText("Plan v2 ready for approval")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Approve updated plan" })).toBeInTheDocument();
    expect(screen.getByRole("textbox", { name: "Direct this work" })).toBeInTheDocument();
  });

  it("adds steering to the timeline and approves the revised plan", async () => {
    const user = userEvent.setup();
    render(
      <OrgWorkspaceHarness startAt="workItem" workItemElement={<ActiveWorkItemPage data={activeWorkItemData} />} />,
    );

    await user.click(await screen.findByRole("button", { name: "Steer from Plan v2 ready for approval" }));
    await user.type(screen.getByRole("textbox", { name: "Direct this work" }), "Keep the recovery cookie HttpOnly.");
    await user.click(screen.getByRole("button", { name: "Send direction" }));

    expect(screen.getByText("Keep the recovery cookie HttpOnly.")).toBeInTheDocument();
    expect(screen.getByText("In response to Plan v2 ready for approval")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Approve updated plan" }));

    expect(screen.getByText("Plan v2 approved by you")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Approve updated plan" })).not.toBeInTheDocument();
  });

  it("pauses and resumes active work", async () => {
    const user = userEvent.setup();
    render(
      <OrgWorkspaceHarness startAt="workItem" workItemElement={<ActiveWorkItemPage data={activeWorkItemData} />} />,
    );

    await user.click(await screen.findByRole("button", { name: "Pause work" }));
    expect(screen.getByRole("button", { name: "Resume work" })).toBeInTheDocument();
    expect(screen.getByText("Paused by you")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Resume work" }));
    expect(screen.getByRole("button", { name: "Pause work" })).toBeInTheDocument();
    expect(screen.getByText("Work resumed")).toBeInTheDocument();
  });

  it("opens an active work item from the factory overview", async () => {
    const user = userEvent.setup();
    render(
      <OrgWorkspaceHarness
        startAt="app"
        appElement={<WorkspacePage data={projectXWorkspaceData} />}
        workItemElement={<ActiveWorkItemPage data={activeWorkItemData} />}
      />,
    );

    await user.click(await screen.findByRole("button", { name: /Add SSO session recovery/ }));

    expect(await screen.findByRole("heading", { name: "Add SSO session recovery" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Chronology" })).toBeInTheDocument();
  });
});
