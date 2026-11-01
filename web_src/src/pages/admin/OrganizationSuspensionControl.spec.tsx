import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { OrganizationSuspensionControl } from "./OrganizationSuspensionControl";
import { organizationKeys } from "@/hooks/useOrganizationData";

const state = { suspended: false, cancellationComplete: false, historyError: false, mutationError: false };
const suspend = vi.fn(async (_options: { body: { reason: string } }) => {
  if (state.mutationError) throw new Error("Unavailable");
  state.suspended = true;
  return { data: {} };
});
const reactivate = vi.fn(async () => {
  if (state.mutationError) throw new Error("Cancellation is pending");
  state.suspended = false;
  return { data: {} };
});
vi.mock("@/api-client/sdk.gen", () => ({
  adminOrganizationsListSuspensionHistory: async () => {
    if (state.historyError) throw new Error("Unavailable");
    return {
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
    };
  },
  adminOrganizationsSuspendOrganization: suspend,
  adminOrganizationsReactivateOrganization: reactivate,
}));

function renderControl() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const onUpdate = vi.fn();
  render(
    <QueryClientProvider client={client}>
      <OrganizationSuspensionControl organization={{ id: "org-1", name: "Example" }} onUpdate={onUpdate} />
    </QueryClientProvider>,
  );
  return { client, onUpdate };
}

describe("Organization suspension control", () => {
  beforeEach(() => {
    state.suspended = false;
    state.cancellationComplete = false;
    state.historyError = false;
    state.mutationError = false;
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

  it("reactivates only after confirmation and refreshes member and admin data", async () => {
    state.suspended = true;
    state.cancellationComplete = true;
    const { client, onUpdate } = renderControl();
    client.setQueryData(organizationKeys.details("org-1"), { suspended: true });
    fireEvent.click(await screen.findByRole("button", { name: "Reactivate organization" }));
    expect(screen.getByRole("dialog")).toHaveTextContent("Discarded work will not restart.");
    expect(reactivate).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Keep current status" }));
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
    expect(reactivate).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Reactivate organization" }));
    fireEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Reactivate organization" }));
    await waitFor(() => expect(screen.getByText("Status: Active")).toBeInTheDocument());
    expect(reactivate).toHaveBeenCalledTimes(1);
    expect(onUpdate).toHaveBeenCalledTimes(1);
    expect(client.getQueryState(organizationKeys.details("org-1"))?.isInvalidated).toBe(true);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("rejects whitespace and overlong reasons but accepts 2000 Unicode characters", async () => {
    renderControl();
    fireEvent.click(await screen.findByRole("button", { name: "Suspend organization" }));
    const confirmation = within(screen.getByRole("dialog")).getByRole("button", { name: "Suspend organization" });
    const reason = screen.getByLabelText("Suspension reason");
    for (const value of [" \n\t", "界".repeat(2001)]) {
      fireEvent.change(reason, { target: { value } });
      expect(confirmation).toBeDisabled();
      fireEvent.click(confirmation);
      expect(suspend).not.toHaveBeenCalled();
    }
    fireEvent.change(reason, { target: { value: `  ${"界".repeat(2000)}  ` } });
    expect(confirmation).toBeEnabled();
    fireEvent.click(confirmation);
    fireEvent.click(confirmation);
    await waitFor(() => expect(suspend).toHaveBeenCalledTimes(1));
    expect(suspend.mock.calls[0][0].body.reason).toBe("界".repeat(2000));
  });

  it("preserves the reason after a failed suspension and permits retry", async () => {
    state.mutationError = true;
    const { onUpdate } = renderControl();
    fireEvent.click(await screen.findByRole("button", { name: "Suspend organization" }));
    fireEvent.change(screen.getByLabelText("Suspension reason"), { target: { value: "Abuse" } });
    const confirmation = within(screen.getByRole("dialog")).getByRole("button", { name: "Suspend organization" });
    fireEvent.click(confirmation);
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not change suspension status.");
    expect(screen.getByLabelText("Suspension reason")).toHaveValue("Abuse");
    expect(onUpdate).not.toHaveBeenCalled();
    state.mutationError = false;
    fireEvent.click(confirmation);
    await waitFor(() => expect(screen.getByText("Status: Suspended")).toBeInTheDocument());
    expect(suspend).toHaveBeenCalledTimes(2);
    expect(onUpdate).toHaveBeenCalledTimes(1);
  });

  it("refreshes pending cancellation after a conflicting reactivation", async () => {
    state.suspended = true;
    state.cancellationComplete = true;
    state.mutationError = true;
    const { onUpdate } = renderControl();
    fireEvent.click(await screen.findByRole("button", { name: "Reactivate organization" }));
    state.cancellationComplete = false;
    fireEvent.click(within(screen.getByRole("dialog")).getByRole("button", { name: "Reactivate organization" }));
    expect(await screen.findByRole("alert")).toHaveTextContent("wait for existing work to stop");
    await waitFor(() =>
      expect(
        within(screen.getByRole("dialog")).getByRole("button", { name: "Reactivate organization" }),
      ).toBeDisabled(),
    );
    expect(onUpdate).not.toHaveBeenCalled();
    expect(screen.getByText("Status: Suspended")).toBeInTheDocument();
  });

  it("blocks mutations when history fails and recovers through Try again", async () => {
    state.historyError = true;
    renderControl();
    expect(await screen.findByRole("alert")).toHaveTextContent("Could not load suspension history.");
    expect(screen.getByRole("button", { name: "Suspend organization" })).toBeDisabled();
    state.historyError = false;
    fireEvent.click(screen.getByRole("button", { name: "Try again" }));
    await waitFor(() => expect(screen.getByRole("button", { name: "Suspend organization" })).toBeEnabled());
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
    expect(suspend).not.toHaveBeenCalled();
  });
});
