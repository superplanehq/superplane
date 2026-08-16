import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { WorkOrderAssigneesPopover } from "./WorkOrderAssigneesPopover";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [
      {
        metadata: { id: "user-alex", email: "alex@superplane.dev" },
        spec: { displayName: "Alex Reviewer" },
      },
      {
        metadata: { id: "user-blake", email: "blake@superplane.dev" },
        spec: { displayName: "Blake Builder" },
      },
    ],
    isLoading: false,
  }),
}));

function popover(selectedIds: string[], onSave: (ids: string[]) => Promise<void>) {
  return (
    <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={selectedIds} onSave={onSave}>
      <button type="button">Owner</button>
    </WorkOrderAssigneesPopover>
  );
}

function memberOrder() {
  return screen.getAllByRole("listitem").map((row) => {
    const spans = row.querySelectorAll("span");
    return spans[spans.length - 1]?.textContent ?? "";
  });
}

describe("WorkOrderAssigneesPopover", () => {
  it("saves the deselection after the order refetches while the popover is open", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);

    const { rerender } = render(popover(["user-alex"], onSave));

    await user.click(screen.getByRole("button", { name: "Owner" }));
    await user.click(screen.getByRole("checkbox", { name: /Alex Reviewer/ }));

    rerender(popover(["user-alex"], onSave));

    await user.click(screen.getByTestId("work-order-save-assignees"));

    expect(onSave).toHaveBeenCalledWith([]);
  });

  it("seeds the draft from the current assignees each time it opens", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);

    const { rerender } = render(popover([], onSave));

    await user.click(screen.getByRole("button", { name: "Owner" }));
    await user.click(screen.getByRole("checkbox", { name: /Blake Builder/ }));
    await user.click(screen.getByTestId("work-order-save-assignees"));

    expect(onSave).toHaveBeenCalledWith(["user-blake"]);

    rerender(popover(["user-blake"], onSave));
    await user.click(screen.getByRole("button", { name: "Owner" }));

    expect(screen.getByRole("checkbox", { name: /Blake Builder/ })).toBeChecked();
  });

  it("regroups the assigned member to the top when reopened", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);

    const { rerender } = render(popover(["user-blake"], onSave));

    await user.click(screen.getByRole("button", { name: "Owner" }));
    expect(memberOrder()).toEqual(["Blake Builder", "Alex Reviewer"]);
    await user.keyboard("{Escape}");

    rerender(popover(["user-alex"], onSave));
    await user.click(screen.getByRole("button", { name: "Owner" }));

    expect(memberOrder()).toEqual(["Alex Reviewer", "Blake Builder"]);
  });

  it("discards an unsaved draft when the popover closes", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn().mockResolvedValue(undefined);

    render(popover(["user-alex"], onSave));

    await user.click(screen.getByRole("button", { name: "Owner" }));
    await user.click(screen.getByRole("checkbox", { name: /Alex Reviewer/ }));
    await user.keyboard("{Escape}");

    await user.click(screen.getByRole("button", { name: "Owner" }));

    expect(screen.getByRole("checkbox", { name: /Alex Reviewer/ })).toBeChecked();
  });
});
