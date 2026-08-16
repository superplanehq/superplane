import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { WorkOrderAssigneePicker } from "./WorkOrderAssigneePicker";

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
      {
        metadata: { id: "user-casey", email: "casey@superplane.dev" },
        spec: { displayName: "Casey Coder" },
      },
    ],
    isLoading: false,
  }),
}));

function memberOrder() {
  return screen.getAllByRole("listitem").map((row) => {
    const spans = row.querySelectorAll("span");
    return spans[spans.length - 1]?.textContent ?? "";
  });
}

describe("WorkOrderAssigneePicker", () => {
  it("lists members alphabetically when nobody is assigned", () => {
    render(<WorkOrderAssigneePicker organizationId="org-1" selectedIds={[]} onChange={vi.fn()} />);

    expect(memberOrder()).toEqual(["Alex Reviewer", "Blake Builder", "Casey Coder"]);
  });

  it("lists the assigned members first", () => {
    render(<WorkOrderAssigneePicker organizationId="org-1" selectedIds={["user-casey"]} onChange={vi.fn()} />);

    expect(memberOrder()).toEqual(["Casey Coder", "Alex Reviewer", "Blake Builder"]);
  });

  it("keeps the order stable while the selection changes", async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();

    render(<WorkOrderAssigneePicker organizationId="org-1" selectedIds={["user-casey"]} onChange={onChange} />);

    await user.click(screen.getAllByRole("checkbox")[1]);

    expect(onChange).toHaveBeenCalledWith(["user-casey", "user-alex"]);
    expect(memberOrder()).toEqual(["Casey Coder", "Alex Reviewer", "Blake Builder"]);
  });
});
