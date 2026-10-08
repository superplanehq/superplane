import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

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

const mockUsers: SuperplaneUsersUser[] = [buildUser("alice", "Alice Anderson"), buildUser("bob", "Bob Brown")];

describe("WorkOrderAssigneesPopover", () => {
  it("saves the person you click and closes the list", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);

    render(
      <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={["alice"]} onSave={onSave}>
        <button>Owner</button>
      </WorkOrderAssigneesPopover>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Owner" }));
    fireEvent.click(screen.getByRole("option", { name: "Bob Brown" }));

    await waitFor(() => expect(onSave).toHaveBeenCalledWith(["bob"]));
    expect(screen.queryByPlaceholderText("Search people")).not.toBeInTheDocument();
  });

  it("clears the owner when you choose No owner", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);

    render(
      <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={["alice"]} onSave={onSave}>
        <button>Owner</button>
      </WorkOrderAssigneesPopover>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Owner" }));
    fireEvent.click(screen.getByRole("option", { name: "No owner" }));

    await waitFor(() => expect(onSave).toHaveBeenCalledWith([]));
  });

  it("does not call onSave when you click the current owner", async () => {
    const onSave = vi.fn().mockResolvedValue(undefined);

    render(
      <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={["alice"]} onSave={onSave}>
        <button>Owner</button>
      </WorkOrderAssigneesPopover>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Owner" }));
    fireEvent.click(screen.getByRole("option", { name: "Alice Anderson" }));

    await waitFor(() => expect(screen.queryByPlaceholderText("Search people")).not.toBeInTheDocument());
    expect(onSave).not.toHaveBeenCalled();
  });

  it("keeps the current owner at the top of the list", () => {
    render(
      <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={["bob"]} onSave={vi.fn()}>
        <button>Owner</button>
      </WorkOrderAssigneesPopover>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Owner" }));

    const items = screen.getAllByRole("option").map((item) => item.textContent ?? "");
    expect(items[0]).toContain("No owner");
    expect(items[1]).toContain("Bob Brown");
    expect(items[2]).toContain("Alice Anderson");
  });

  it("stays open when saving fails", async () => {
    const onSave = vi.fn().mockRejectedValue(new Error("nope"));

    render(
      <WorkOrderAssigneesPopover organizationId="org-1" selectedIds={["alice"]} onSave={onSave}>
        <button>Owner</button>
      </WorkOrderAssigneesPopover>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Owner" }));
    fireEvent.click(screen.getByRole("option", { name: "Bob Brown" }));

    await waitFor(() => expect(onSave).toHaveBeenCalledWith(["bob"]));
    expect(screen.getByPlaceholderText("Search people")).toBeInTheDocument();
  });
});
