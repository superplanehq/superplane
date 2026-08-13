import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

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
  render(
    <WorkOrderAssigneePicker
      organizationId="org-1"
      selectedIds={[]}
      onChange={onChange}
      {...overrides}
    />,
  );
  return { onChange };
}

function renderedNames() {
  return screen.getAllByRole("listitem").map((item) => item.textContent?.trim());
}

describe("WorkOrderAssigneePicker", () => {
  it("sorts users alphabetically when nobody is assigned", () => {
    renderPicker();

    const names = renderedNames();
    expect(names).toEqual([
      expect.stringContaining("Alice Anderson"),
      expect.stringContaining("Bob Brown"),
      expect.stringContaining("Carol Clark"),
      expect.stringContaining("Dan Davis"),
    ]);
  });

  it("pins currently-assigned users to the top regardless of alphabetical order", () => {
    renderPicker({ selectedIds: ["dan", "bob"], pinnedIds: ["dan", "bob"] });

    const names = renderedNames();
    // Pinned users (bob, dan) come first, alphabetically among themselves,
    // followed by the rest alphabetically.
    expect(names).toEqual([
      expect.stringContaining("Bob Brown"),
      expect.stringContaining("Dan Davis"),
      expect.stringContaining("Alice Anderson"),
      expect.stringContaining("Carol Clark"),
    ]);
  });

  it("falls back to selectedIds for pinning when pinnedIds is not provided", () => {
    renderPicker({ selectedIds: ["carol"] });

    const names = renderedNames();
    expect(names[0]).toContain("Carol Clark");
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

    // Simulate the user unchecking "bob" mid-session: selectedIds changes,
    // but pinnedIds (captured when the popover opened) stays the same, so
    // the list order must not change.
    rerender(
      <WorkOrderAssigneePicker
        organizationId="org-1"
        selectedIds={["dan"]}
        pinnedIds={["dan", "bob"]}
        onChange={vi.fn()}
      />,
    );

    const names = renderedNames();
    expect(names).toEqual([
      expect.stringContaining("Bob Brown"),
      expect.stringContaining("Dan Davis"),
      expect.stringContaining("Alice Anderson"),
      expect.stringContaining("Carol Clark"),
    ]);
  });
});
