import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";
import { IntakeDefaultOwnerField } from "./IntakeDefaultOwnerField";

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [
      { metadata: { id: "alice" }, spec: { displayName: "Alice" } },
      { metadata: { id: "bob" }, spec: { displayName: "Bob" } },
    ],
    isLoading: false,
    isError: false,
  }),
}));
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ currentUserId: "alice" }) }));

vi.mock("@/hooks/useFactoryOwnerEligibility", () => ({
  useFactoryOwnerEligibility: () => ({
    data: {
      members: [
        { userId: "alice", eligible: true },
        { userId: "bob", eligible: false },
      ],
    },
    isPending: false,
    isError: false,
  }),
}));

describe("IntakeDefaultOwnerField", () => {
  it("requires a connected new owner and preserves a disconnected saved owner", async () => {
    const onChange = vi.fn();
    const { rerender } = render(
      <IntakeDefaultOwnerField organizationId="org" factoryId="factory" value="" required onChange={onChange} />,
    );
    fireEvent.click(screen.getByRole("combobox", { name: /Default task owner/ }));
    expect(screen.queryByRole("option", { name: "Bob" })).not.toBeInTheDocument();
    expect(screen.queryByRole("option", { name: "No owner" })).not.toBeInTheDocument();
    fireEvent.click(await screen.findByRole("option", { name: "Alice" }));
    expect(onChange).toHaveBeenCalledWith("alice");
    rerender(
      <IntakeDefaultOwnerField organizationId="org" factoryId="factory" value="bob" required onChange={onChange} />,
    );
    expect(screen.getByRole("combobox", { name: /Default task owner/ })).toHaveTextContent("Bob");
    expect(screen.getByText("This owner must reconnect Codex before tasks can run.")).toBeInTheDocument();
    expect(onChange).toHaveBeenCalledTimes(1);
  });
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
