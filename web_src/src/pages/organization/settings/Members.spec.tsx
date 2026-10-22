import { fireEvent, render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";
import { Members } from "./Members";

const LONG_EMAIL = "very.long.email.address.that.should.not.truncate@example.com";

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/hooks/useReportPageReady", () => ({
  useReportPageReady: vi.fn(),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: { id: "user-viewer" } }),
}));

let seatLimit: number | undefined;
let loading = false;
let refreshing = false;
let memberError: Error | null = null;
let ancillaryError: Error | null = null;
let manage = true;
let inviteEnabled = true;
const updateInvite = vi.fn(async () => {});
beforeEach(() => {
  seatLimit = undefined;
  loading = false;
  refreshing = false;
  memberError = null;
  ancillaryError = null;
  manage = true;
  inviteEnabled = true;
  updateInvite.mockClear();
});

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: (_resource: string, action: string) => action === "read" || manage,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationMembers: () => ({
    seatLimit,
    isSuccess: !loading && !memberError,
    users: [
      {
        metadata: { id: "user-1", email: LONG_EMAIL },
        spec: { displayName: "Ada Lovelace" },
        status: {
          isOwner: true,
          roles: [{ roleName: "admin", roleDisplayName: "Admin" }],
        },
      },
    ],
    isLoading: loading,
    isFetching: refreshing,
    error: memberError,
  }),
  useOrganizationRoles: () => ({
    data: [{ metadata: { name: "admin" }, spec: { displayName: "Admin" } }],
    isLoading: false,
    error: ancillaryError,
  }),
  useOrganizationInviteLink: () => ({
    data: { token: "invite-token", enabled: inviteEnabled },
    isLoading: false,
    error: ancillaryError,
  }),
  useAssignRole: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useSetUserOwner: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useRemoveOrganizationSubject: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateOrganizationInviteLink: () => ({ mutateAsync: updateInvite, isPending: false }),
  useResetOrganizationInviteLink: () => ({ mutateAsync: vi.fn(), isPending: false }),
}));

vi.mock("@/lib/toast", () => ({
  showSuccessToast: vi.fn(),
  showErrorToast: vi.fn(),
}));

function renderMembers() {
  return render(
    <TooltipProvider>
      <Members organizationId="org-1" />
    </TooltipProvider>,
  );
}

describe("Members", () => {
  it("scrolls the members table horizontally instead of clipping emails", () => {
    renderMembers();

    const email = screen.getByText(LONG_EMAIL);
    expect(email).toBeInTheDocument();
    expect(email.className).not.toContain("truncate");
    expect(email.className).not.toContain("max-w-[26rem]");

    const table = screen.getByRole("table");
    const scrollContainer = table.parentElement?.parentElement;
    expect(scrollContainer?.className).toContain("overflow-x-auto");
    expect(scrollContainer?.className).not.toContain("overflow-x-hidden");
    expect(scrollContainer?.className).toContain("whitespace-nowrap");
  });
});

for (const state of [
  { name: "finite", limit: 3, summary: "1 of 3 seats used", blocked: false },
  { name: "unlimited", limit: undefined, summary: "1 members · Unlimited seats", blocked: false },
  { name: "full", limit: 1, summary: "Seat limit reached.", blocked: true },
  { name: "exceeded", limit: 0, summary: "Seat limit exceeded.", blocked: true },
]) {
  it(`shows ${state.name} seats and controls sharing`, () => {
    seatLimit = state.limit;
    renderMembers();
    expect(screen.getByText(state.summary)).toBeInTheDocument();
    expect(screen.getByRole<HTMLButtonElement>("button", { name: "Copy link" }).disabled).toBe(state.blocked);
    expect(screen.getByRole<HTMLButtonElement>("button", { name: "generate a new link" }).disabled).toBe(state.blocked);
    expect(screen.getByRole<HTMLButtonElement>("switch", { name: "Toggle invite link" }).disabled).toBe(false);
  });
}
it("keeps an enabled link saved while capacity fails, and restores sharing after refresh", () => {
  loading = true;
  const { rerender } = renderMembers();
  expect(screen.getByText("Loading seat usage...")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Copy link" })).toBeDisabled();
  loading = false;
  memberError = new Error("offline");
  rerender(
    <TooltipProvider>
      <Members organizationId="org-1" />
    </TooltipProvider>,
  );
  expect(screen.getByText(/Failed to load seat usage/)).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Copy link" })).toBeDisabled();
  memberError = null;
  seatLimit = 1;
  rerender(
    <TooltipProvider>
      <Members organizationId="org-1" />
    </TooltipProvider>,
  );
  expect(screen.getByRole("button", { name: "Copy link" })).toBeDisabled();
  seatLimit = 2;
  rerender(
    <TooltipProvider>
      <Members organizationId="org-1" />
    </TooltipProvider>,
  );
  expect(screen.getByRole("button", { name: "Copy link" })).toBeEnabled();
  expect(updateInvite).not.toHaveBeenCalled();
});
it("allows disabling while full and blocks enabling while capacity is unknown", () => {
  seatLimit = 1;
  const { rerender } = renderMembers();
  fireEvent.click(screen.getByRole("switch", { name: "Toggle invite link" }));
  expect(updateInvite).toHaveBeenCalledWith(false);
  inviteEnabled = false;
  loading = true;
  rerender(
    <TooltipProvider>
      <Members organizationId="org-1" />
    </TooltipProvider>,
  );
  expect(screen.getByRole("switch", { name: "Toggle invite link" })).toBeDisabled();
});
it("shows seats for read-only members despite role or invite request failures", () => {
  manage = false;
  ancillaryError = new Error("Forbidden");
  seatLimit = 4;
  renderMembers();
  expect(screen.getByText("1 of 4 seats used")).toBeInTheDocument();
});

it("blocks sharing while a successful cached capacity response refreshes", () => {
  refreshing = true;
  const { rerender } = renderMembers();
  expect(screen.getByText("Loading seat usage...")).toBeInTheDocument();
  expect(screen.getByRole("button", { name: "Copy link" })).toBeDisabled();
  refreshing = false;
  rerender(
    <TooltipProvider>
      <Members organizationId="org-1" />
    </TooltipProvider>,
  );
  expect(screen.getByRole("button", { name: "Copy link" })).toBeEnabled();
});
