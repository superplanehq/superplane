import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "bun:test";
import { ScheduledIntakeOwnerSettings } from "./ScheduledIntakeOwnerSettings";

vi.mock("../layout/factoriesLayoutContext", () => ({
  useOptionalFactoriesLayout: () => ({ factory: { onboarding: { codexSubscription: false } } }),
}));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, currentUserId: "alice" }),
}));
vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [{ metadata: { id: "alice" }, spec: { displayName: "Alice" } }],
    isLoading: false,
  }),
}));

describe("Scheduled intake owner settings", () => {
  it("shows and clears the saved optional owner", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn(async () => {});
    render(
      <ScheduledIntakeOwnerSettings
        organizationId="org"
        factoryId="factory"
        node={{
          id: "create-task",
          name: "Create task",
          configuration: { defaultOwnerId: "alice", title: "Keep title" },
        }}
        onSave={onSave}
      />,
    );
    expect(screen.getByRole("button", { name: "Default task owner" })).toHaveTextContent("Alice");
    await user.click(screen.getByRole("button", { name: "Default task owner" }));
    await user.click(await screen.findByRole("option", { name: "No owner" }));
    await user.click(screen.getByRole("button", { name: "Save default owner" }));
    await waitFor(() =>
      expect(onSave).toHaveBeenCalledWith(
        expect.objectContaining({ nodeId: "create-task", configuration: { defaultOwnerId: "", title: "Keep title" } }),
      ),
    );
  });
});
