import { fireEvent, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { SuperplaneUsersUser } from "@/api-client";
import { WorkOrderAssigneePicker } from "./WorkOrderAssigneePicker";

const { permissionsState } = vi.hoisted(() => ({
  permissionsState: { currentUserId: undefined as string | undefined },
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: vi.fn(() => ({ data: mockUsers, isLoading: false })),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => permissionsState,
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

const alphabeticalNames = [
  expect.stringContaining("Alice Anderson"),
  expect.stringContaining("Bob Brown"),
  expect.stringContaining("Carol Clark"),
  expect.stringContaining("Dan Davis"),
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
  beforeEach(() => {
    permissionsState.currentUserId = undefined;
  });

  it("keeps No owner first and sorts people alphabetically when the signed-in user is missing", () => {
    permissionsState.currentUserId = "missing-user";
    renderPicker({ selectedIds: ["dan"] });

    expect(screen.getAllByRole("option")[0]).toHaveTextContent("No owner");
    expect(renderedNames()).toEqual(alphabeticalNames);
  });

  it("puts the signed-in user first and keeps the other people alphabetical", () => {
    permissionsState.currentUserId = "carol";
    renderPicker({ selectedIds: ["dan"] });

    expect(renderedNames()).toEqual([
      expect.stringContaining("Carol Clark"),
      expect.stringContaining("Alice Anderson"),
      expect.stringContaining("Bob Brown"),
      expect.stringContaining("Dan Davis"),
    ]);
  });

  it("does not keep the signed-in user visible when the search does not match", () => {
    permissionsState.currentUserId = "carol";
    renderPicker();

    fireEvent.change(screen.getByPlaceholderText("Search people"), { target: { value: "dan" } });

    expect(screen.getByRole("option", { name: "Dan Davis" })).toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "Carol Clark" })).not.toBeInTheDocument();
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

it("keeps an ineligible selected owner visible and permits clearing", () => {
  const { onChange } = renderPicker({
    selectedIds: ["removed"],
    eligibility: { subscription: true, ownershipChangeAllowed: true, members: [{ userId: "bob", eligible: true }] },
  });
  expect(screen.getByRole("option", { name: /Unavailable member/ })).toHaveAttribute("aria-disabled", "true");
  fireEvent.click(screen.getByRole("option", { name: "No owner" }));
  expect(onChange).toHaveBeenCalledWith([]);
});

it.each(["hover", "focus"] as const)(
  "keeps disconnected owners disabled and explains why on %s",
  async (interaction) => {
    const user = userEvent.setup();
    const reason = "Connect a ChatGPT subscription in this workspace.";
    permissionsState.currentUserId = "alice";
    const { onChange } = renderPicker({
      eligibility: {
        subscription: true,
        ownershipChangeAllowed: true,
        members: [
          { userId: "alice", eligible: false, reason },
          { userId: "bob", eligible: true },
          { userId: "carol", eligible: false, reason },
        ],
      },
    });

    const currentUser = screen.getByRole("option", { name: "Alice Anderson" });
    const otherMember = screen.getByRole("option", { name: "Carol Clark" });
    expect(currentUser).toHaveAttribute("aria-disabled", "true");
    expect(otherMember).toHaveAttribute("aria-disabled", "true");
    expect(screen.queryByText(reason)).not.toBeInTheDocument();
    fireEvent.click(currentUser);
    fireEvent.click(otherMember);
    expect(onChange).not.toHaveBeenCalled();

    if (interaction === "hover") {
      await user.hover(currentUser);
    } else {
      await user.tab();
      expect(currentUser).toHaveFocus();
    }

    expect(await screen.findByRole("tooltip")).toHaveTextContent(reason);
    expect(currentUser).toHaveAccessibleDescription(reason);
    if (interaction === "focus") {
      await user.keyboard("{Enter}");
      expect(onChange).not.toHaveBeenCalled();
    }
    fireEvent.click(screen.getByRole("option", { name: "Bob Brown" }));
    expect(onChange).toHaveBeenCalledWith(["bob"]);
  },
);

it("blocks both clearing and assignment while the task stops", () => {
  permissionsState.currentUserId = "alice";
  const { onChange } = renderPicker({
    selectedIds: ["alice"],
    eligibility: {
      subscription: true,
      ownershipChangeAllowed: false,
      restrictionReason: "Wait for this task to stop.",
    },
  });
  fireEvent.click(screen.getByRole("option", { name: "No owner" }));
  fireEvent.click(screen.getByRole("option", { name: /Alice Anderson/ }));
  expect(onChange).not.toHaveBeenCalled();
  expect(screen.getByText("Wait for this task to stop.")).toBeInTheDocument();
});
