import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { OrganizationSuspensionControl } from "./OrganizationSuspensionControl";

const state = { suspended: false, cancellationComplete: false };
const suspend = vi.fn(async (_options: { body: { reason: string } }) => {
  state.suspended = true;
  return { data: {} };
});
const reactivate = vi.fn(async () => {
  state.suspended = false;
  return { data: {} };
});
vi.mock("@/api-client/sdk.gen", () => ({
  adminOrganizationsListSuspensionHistory: async () => ({
    data: {
      suspended: state.suspended,
      events: state.suspended
        ? [
            {
              id: "event-1",
              action: "suspend",
              reason: "Abusive automation",
              accountId: "admin-1",
              createdAt: new Date().toISOString(),
              cancellationCompletedAt: state.cancellationComplete ? new Date().toISOString() : undefined,
            },
          ]
        : [],
    },
  }),
  adminOrganizationsSuspendOrganization: suspend,
  adminOrganizationsReactivateOrganization: reactivate,
}));

function renderControl() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <OrganizationSuspensionControl organization={{ id: "org-1", name: "Example" }} onUpdate={() => {}} />
    </QueryClientProvider>,
  );
}

describe("Organization suspension control", () => {
  beforeEach(() => {
    state.suspended = false;
    state.cancellationComplete = false;
    suspend.mockClear();
    reactivate.mockClear();
  });
  it("requires a valid reason and an explicit confirmation", async () => {
    renderControl();
    await waitFor(() => expect(screen.getByRole("button", { name: "Suspend organization" })).toBeEnabled());
    fireEvent.click(screen.getByRole("button", { name: "Suspend organization" }));
    const confirmation = screen.getByRole("button", { name: "Suspend organization" });
    expect(confirmation).toBeDisabled();
    fireEvent.change(screen.getByLabelText("Suspension reason"), { target: { value: "  Unattended automation  " } });
    expect(suspend).not.toHaveBeenCalled();
    fireEvent.click(confirmation);
    await waitFor(() => expect(suspend).toHaveBeenCalledTimes(1));
    expect(suspend.mock.calls[0][0].body.reason).toBe("Unattended automation");
    await waitFor(() => expect(screen.getByRole("button", { name: "Reactivate organization" })).toBeDisabled());
  });
  it("shows private history and keeps reactivation disabled until cancellation completes", async () => {
    state.suspended = true;
    renderControl();
    expect(await screen.findByText("Abusive automation")).toBeInTheDocument();
    expect(screen.getByText("Actor: admin-1")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reactivate organization" })).toBeDisabled();
    expect(reactivate).not.toHaveBeenCalled();
  });
});
