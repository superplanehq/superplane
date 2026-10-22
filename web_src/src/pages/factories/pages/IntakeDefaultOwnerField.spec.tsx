import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { IntakeDefaultOwnerField } from "./IntakeDefaultOwnerField";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [{ metadata: { id: "alice" }, spec: { displayName: "Alice" } }],
    isLoading: false,
    isError: false,
  }),
}));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ currentUserId: "alice" }) }));

describe("IntakeDefaultOwnerField", () => {
  it("selects one member and clears an unavailable member", async () => {
    const onChange = vi.fn();
    const { rerender } = render(<IntakeDefaultOwnerField organizationId="org" value="" onChange={onChange} />);
    fireEvent.click(screen.getByRole("button", { name: "Default task owner" }));
    fireEvent.click(await screen.findByRole("option", { name: "Alice" }));
    expect(onChange).toHaveBeenCalledWith("alice");
    await waitFor(() => expect(screen.queryByRole("option", { name: "Alice" })).not.toBeInTheDocument());

    rerender(<IntakeDefaultOwnerField organizationId="org" value="removed" onChange={onChange} />);
    expect(screen.getByRole("button", { name: "Default task owner" })).toHaveTextContent("Unavailable member");
    fireEvent.click(screen.getByRole("button", { name: "Default task owner" }));
    fireEvent.click(await screen.findByRole("option", { name: "No owner" }));
    expect(onChange).toHaveBeenLastCalledWith("");
  });
});
