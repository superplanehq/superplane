import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import type { SuperplaneUsersUser } from "@/api-client";
import { WorkOrderAssigneePicker } from "./WorkOrderAssigneePicker";

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
  buildUser("carol", "Carol Clark"),
  buildUser("dan", "Dan Davis"),
];

function renderPicker(overrides: Partial<Parameters<typeof WorkOrderAssigneePicker>[0]> = {}) {
  const onChange = vi.fn();
  render(<WorkOrderAssigneePicker organizationId="org-1" selectedIds={[]} onChange={onChange} {...overrides} />);
  return { onChange };
}

function renderedNames() {
  return screen
    .getAllByRole("option")
    .map((item) => item.textContent ?? "")
    .filter((name) => !name.includes("No owner"));
}

describe("WorkOrderAssigneePicker", () => {
  it("sorts users alphabetically when nobody is assigned", () => {
    renderPicker();

    expect(renderedNames()).toEqual([
      expect.stringContaining("Alice Anderson"),
      expect.stringContaining("Bob Brown"),
      expect.stringContaining("Carol Clark"),
      expect.stringContaining("Dan Davis"),
    ]);
  });

  it("pins currently-assigned users to the top regardless of alphabetical order", () => {
    renderPicker({ selectedIds: ["dan", "bob"], pinnedIds: ["dan", "bob"] });

    expect(renderedNames()).toEqual([
      expect.stringContaining("Bob Brown"),
      expect.stringContaining("Dan Davis"),
      expect.stringContaining("Alice Anderson"),
      expect.stringContaining("Carol Clark"),
    ]);
  });

  it("falls back to selectedIds for pinning when pinnedIds is not provided", () => {
    renderPicker({ selectedIds: ["carol"] });

    expect(renderedNames()[0]).toContain("Carol Clark");
  });

  it("keeps the pinned order stable while the live selection changes mid-session", () => {
    const { rerender } = render(
      <WorkOrderAssigneePicker
        organizationId="org-1"
        selectedIds={["dan", "bob"]}
        pinnedIds={["dan", "bob"]}
        onChange={vi.fn()}
      />,
    );

    expect(renderedNames()[0]).toContain("Bob Brown");

    rerender(
      <WorkOrderAssigneePicker
        organizationId="org-1"
        selectedIds={["dan"]}
        pinnedIds={["dan", "bob"]}
        onChange={vi.fn()}
      />,
    );

    expect(renderedNames()).toEqual([
      expect.stringContaining("Bob Brown"),
      expect.stringContaining("Dan Davis"),
      expect.stringContaining("Alice Anderson"),
      expect.stringContaining("Carol Clark"),
    ]);
  });

  it("filters the list from the search field", () => {
    renderPicker();

    fireEvent.change(screen.getByPlaceholderText("Search people"), { target: { value: "dan" } });

    expect(screen.getByRole("option", { name: "Dan Davis" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Alice Anderson" })).not.toBeInTheDocument();
  });

  it("clears the owner from the No owner row", () => {
    const { onChange } = renderPicker({ selectedIds: ["alice"] });

    fireEvent.click(screen.getByRole("option", { name: "No owner" }));

    expect(onChange).toHaveBeenCalledWith([]);
  });

  it("replaces the current owner when another person is selected", () => {
    const { onChange } = renderPicker({ selectedIds: ["alice"] });

    fireEvent.click(screen.getByRole("option", { name: "Bob Brown" }));

    expect(onChange).toHaveBeenCalledWith(["bob"]);
    expect(onChange).not.toHaveBeenCalledWith(["alice", "bob"]);
  });
});
