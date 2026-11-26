import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
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

let eligibleMembers = [
  { userId: "alice", eligible: true },
  { userId: "bob", eligible: false },
];
let eligibilityPending = false;
let eligibilityError = false;
const refetchEligibility = vi.fn(async () => ({}));
vi.mock("@/hooks/useFactoryOwnerEligibility", () => ({
  useFactoryOwnerEligibility: () => ({
    data: { members: eligibleMembers },
    isPending: eligibilityPending,
    isError: eligibilityError,
    refetch: refetchEligibility,
  }),
}));
vi.mock("../codex/CodexConnectionSetupDialog", () => ({
  CodexConnectionSetupDialog: ({ open, onConnected }: { open: boolean; onConnected?: () => Promise<void> }) =>
    open ? (
      <button type="button" onClick={() => void onConnected?.()}>
        Finish Codex sign-in
      </button>
    ) : null,
}));

describe("IntakeDefaultOwnerField", () => {
  beforeEach(() => {
    eligibleMembers = [
      { userId: "alice", eligible: true },
      { userId: "bob", eligible: false },
    ];
    eligibilityPending = false;
    eligibilityError = false;
    refetchEligibility.mockClear();
  });

  it.each(["pending", "error"])("suppresses connection prompts while eligibility is %s", (state) => {
    eligibleMembers = [
      { userId: "alice", eligible: false },
      { userId: "bob", eligible: false },
    ];
    eligibilityPending = state === "pending";
    eligibilityError = state === "error";
    const onChange = vi.fn();
    render(
      <IntakeDefaultOwnerField organizationId="org" factoryId="factory" value="bob" required onChange={onChange} />,
    );

    expect(screen.getByRole("combobox", { name: /Default task owner/ })).toBeDisabled();
    expect(screen.queryByText("No members have ChatGPT connected.")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Connect Codex" })).not.toBeInTheDocument();
    expect(screen.getByText("This owner must reconnect Codex before tasks can run.")).toBeInTheDocument();
    if (state === "error") {
      expect(screen.getByRole("alert")).toBeInTheDocument();
      fireEvent.click(screen.getByRole("button", { name: "Try again" }));
      expect(refetchEligibility).toHaveBeenCalledTimes(1);
    }
    expect(onChange).not.toHaveBeenCalled();
  });

  it("offers Codex connection when only another member is eligible", () => {
    eligibleMembers = [
      { userId: "alice", eligible: false },
      { userId: "bob", eligible: true },
    ];
    render(<IntakeDefaultOwnerField organizationId="org" factoryId="factory" value="" required onChange={vi.fn()} />);
    expect(screen.getByRole("combobox", { name: /Default task owner/ })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Connect Codex" })).toBeInTheDocument();
  });

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
  it.each([{ members: [] }, { members: [{ userId: "bob", eligible: false }] }])(
    "disables the empty owner picker and connects the signed-in member (%j)",
    async ({ members }) => {
      eligibleMembers = members;
      const onChange = vi.fn();
      const { rerender } = render(
        <IntakeDefaultOwnerField organizationId="org" factoryId="factory" value="" required onChange={onChange} />,
      );
      expect(screen.getByRole("combobox", { name: /Default task owner/ })).toBeDisabled();
      expect(screen.getByText("No members have ChatGPT connected.")).toBeInTheDocument();
      fireEvent.click(screen.getByRole("button", { name: "Connect Codex" }));
      fireEvent.click(await screen.findByRole("button", { name: "Finish Codex sign-in" }));
      await waitFor(() => expect(onChange).toHaveBeenCalledWith("alice"));
      await waitFor(() =>
        expect(screen.queryByRole("button", { name: "Finish Codex sign-in" })).not.toBeInTheDocument(),
      );
      eligibleMembers = [{ userId: "alice", eligible: true }];
      rerender(
        <IntakeDefaultOwnerField organizationId="org" factoryId="factory" value="alice" required onChange={onChange} />,
      );
      expect(screen.getByRole("combobox", { name: /Default task owner/ })).toBeEnabled();
      expect(screen.getByRole("combobox", { name: /Default task owner/ })).toHaveTextContent("Alice");
      expect(screen.queryByText("No members have ChatGPT connected.")).not.toBeInTheDocument();
    },
  );
});
