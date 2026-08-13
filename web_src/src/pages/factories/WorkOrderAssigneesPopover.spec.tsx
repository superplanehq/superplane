import { fireEvent, render, screen, within } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { SuperplaneUsersUser } from "@/api-client";
import { WorkOrderAssigneesPopover } from "./WorkOrderAssigneesPopover";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: vi.fn(() => ({ data: mockUsers, isLoading: false })),
}));

function buildUser(id: string, displayName: string): SuperplaneUsersUser {
  return {
    metadata: { id, email: `${id}@example.com` },
    spec: { displayName },
  } as SuperplaneUsersUser;
}

const mockUsers: SuperplaneUsersUser[] = [
  buildUser("alice", "Alice Anderson"),
  buildUser("bob", "Bob Brown"),
];

function checkboxFor(name: string) {
  const item = screen.getAllByRole("listitem").find((el) => el.textContent?.includes(name));
  if (!item) {
    throw new Error(`Could not find list item for ${name}`);
  }
  return within(item).getByRole("checkbox");
}

describe("WorkOrderAssigneesPopover", () => {
  it("keeps unchecked assignees unchecked across an unrelated parent re-render, and saves the reduced list", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);

    function Host({ selectedIds }: { selectedIds: string[] }) {
      return (
        <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={selectedIds} onSave={onSave}>
          <button>Assignees</button>
        </WorkOrderAssigneesPopover>
      );
    }

    const { rerender } = render(<Host selectedIds={["alice", "bob"]} />);

    fireEvent.click(screen.getByRole("button", { name: "Assignees" }));

    const bobCheckbox = checkboxFor("Bob Brown");
    expect(bobCheckbox).toBeChecked();
    fireEvent.click(bobCheckbox);
    expect(checkboxFor("Bob Brown")).not.toBeChecked();

    // Simulate an unrelated parent re-render (e.g. a websocket-triggered
    // refetch) that produces a new-but-content-equal `selectedIds` array
    // while the popover is still open.
    rerender(<Host selectedIds={[...["alice", "bob"]]} />);

    expect(checkboxFor("Bob Brown")).not.toBeChecked();
    expect(checkboxFor("Alice Anderson")).toBeChecked();

    fireEvent.click(screen.getByTestId("work-order-save-assignees"));

    await vi.waitFor(() => expect(onSave).toHaveBeenCalledWith(["alice"]));
  });

  it("supports unassigning everyone", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);

    render(
      <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={["alice", "bob"]} onSave={onSave}>
        <button>Assignees</button>
      </WorkOrderAssigneesPopover>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Assignees" }));

    fireEvent.click(checkboxFor("Alice Anderson"));
    fireEvent.click(checkboxFor("Bob Brown"));

    fireEvent.click(screen.getByTestId("work-order-save-assignees"));

    await vi.waitFor(() => expect(onSave).toHaveBeenCalledWith([]));
  });

  it("does not call onSave when closing without any actual change", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);

    render(
      <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={["alice"]} onSave={onSave}>
        <button>Assignees</button>
      </WorkOrderAssigneesPopover>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Assignees" }));

    // Toggle off and back on again — net no-op.
    const aliceCheckbox = checkboxFor("Alice Anderson");
    fireEvent.click(aliceCheckbox);
    fireEvent.click(checkboxFor("Alice Anderson"));

    fireEvent.click(screen.getByTestId("work-order-save-assignees"));

    await vi.waitFor(() => expect(screen.queryByTestId("work-order-save-assignees")).not.toBeInTheDocument());
    expect(onSave).not.toHaveBeenCalled();
  });

  it("pins assigned users to the top of the list", () => {
    render(
      <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={["bob"]} onSave={vi.fn()}>
        <button>Assignees</button>
      </WorkOrderAssigneesPopover>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Assignees" }));

    const items = screen.getAllByRole("listitem");
    expect(items[0].textContent).toContain("Bob Brown");
    expect(items[1].textContent).toContain("Alice Anderson");
  });
});
