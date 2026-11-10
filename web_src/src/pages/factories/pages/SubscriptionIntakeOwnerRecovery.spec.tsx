import { useState } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { SubscriptionIntakeOwnerRecovery } from "./SubscriptionIntakeOwnerRecovery";

const mocks = vi.hoisted(() => ({
  subscription: true,
  ownerId: "",
  members: [] as Array<{ userId: string; eligible: boolean }>,
  save: vi.fn(),
  refresh: vi.fn(),
}));
vi.mock("../layout/factoriesLayoutContext", () => ({
  useOptionalFactoriesLayout: () => ({
    organizationId: "org",
    factoryId: "factory",
    factory: { onboarding: { codexSubscription: mocks.subscription } },
  }),
}));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ currentUserId: "alice", canAct: () => true }),
}));
vi.mock("@/hooks/useFactoryOwnerEligibility", () => ({
  useFactoryOwnerEligibility: () => ({
    data: { members: mocks.members },
    isPending: false,
    isError: false,
    refetch: mocks.refresh,
  }),
}));
vi.mock("@/hooks/useFactoryData", () => ({
  useFactoryAutomations: () => ({ data: [], isPending: false, isError: false }),
}));
vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useFactoryIntakes: () => ({
    data: [{ id: "intake", name: "Jira issues", settings: { defaultOwnerId: mocks.ownerId } }],
    isPending: false,
    isError: false,
  }),
  useUpdateFactoryIntake: () => {
    const [error, setError] = useState<Error>();
    return {
      isPending: false,
      error,
      mutateAsync: async (input: unknown) => {
        try {
          await mocks.save(input);
          setError(undefined);
        } catch (cause) {
          setError(cause as Error);
          throw cause;
        }
      },
    };
  },
}));
vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [{ metadata: { id: "alice" }, spec: { displayName: "Alice" } }],
    isLoading: false,
  }),
}));
vi.mock("./useColumnCanvasAgentEditor", () => ({ useColumnCanvasAgentEditor: () => ({}) }));
vi.mock("../codex/CodexConnectionSetupDialog", () => ({
  CodexConnectionSetupDialog: ({ open, onConnected }: { open: boolean; onConnected: () => Promise<void> }) =>
    open ? (
      <button
        onClick={() => {
          mocks.members = [{ userId: "alice", eligible: true }];
          void onConnected();
        }}
      >
        Complete Codex sign-in
      </button>
    ) : null,
}));

beforeEach(() => {
  mocks.subscription = true;
  mocks.ownerId = "";
  mocks.members = [{ userId: "alice", eligible: true }];
  mocks.save.mockReset();
  mocks.save.mockResolvedValue(undefined);
  mocks.refresh.mockReset();
  mocks.refresh.mockResolvedValue(undefined);
});
function showRecovery() {
  return render(
    <QueryClientProvider client={new QueryClient()}>
      <SubscriptionIntakeOwnerRecovery />
    </QueryClientProvider>,
  );
}

describe("Subscription intake owner recovery", () => {
  it("hides owner recovery outside subscription mode and keeps a disconnected active default", () => {
    mocks.subscription = false;
    const { rerender } = showRecovery();
    expect(screen.queryByRole("button", { name: "Set default owner" })).not.toBeInTheDocument();
    mocks.subscription = true;
    mocks.ownerId = "alice";
    mocks.members = [{ userId: "alice", eligible: false }];
    rerender(
      <QueryClientProvider client={new QueryClient()}>
        <SubscriptionIntakeOwnerRecovery />
      </QueryClientProvider>,
    );
    expect(screen.queryByRole("button", { name: "Connect Codex" })).not.toBeInTheDocument();
  });
  it("connects in a modal and returns to owner setup with the caller selected", async () => {
    mocks.members = [];
    showRecovery();
    fireEvent.click(screen.getByRole("button", { name: "Connect Codex" }));
    fireEvent.click(screen.getByRole("button", { name: "Complete Codex sign-in" }));
    await waitFor(() =>
      expect(screen.getByRole("combobox", { name: /Default task owner/ })).toHaveTextContent("Alice"),
    );
    fireEvent.click(screen.getByRole("button", { name: "Save default owner" }));
    await waitFor(() =>
      expect(mocks.save).toHaveBeenCalledWith({ intakeId: "intake", settings: { defaultOwnerId: "alice" } }),
    );
    expect(mocks.refresh).toHaveBeenCalledTimes(1);
  });
  it("keeps a failed save visible and retries with the selected owner", async () => {
    mocks.save.mockRejectedValueOnce(new Error("Save failed"));
    showRecovery();
    fireEvent.click(screen.getByRole("button", { name: "Set default owner" }));
    fireEvent.click(screen.getByRole("combobox", { name: /Default task owner/ }));
    fireEvent.click(await screen.findByRole("option", { name: "Alice" }));
    fireEvent.click(screen.getByRole("button", { name: "Save default owner" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("Save failed");
    expect(screen.getByRole("combobox", { name: /Default task owner/ })).toHaveTextContent("Alice");
    fireEvent.click(screen.getByRole("button", { name: "Save default owner" }));
    await waitFor(() => expect(mocks.save).toHaveBeenCalledTimes(2));
  });
});
