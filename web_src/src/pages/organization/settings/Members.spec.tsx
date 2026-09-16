import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

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

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({
    canAct: () => true,
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationUsers: () => ({
    data: [
      {
        metadata: { id: "user-1", email: LONG_EMAIL },
        spec: { displayName: "Ada Lovelace" },
        status: {
          isOwner: true,
          roles: [{ roleName: "admin", roleDisplayName: "Admin" }],
        },
      },
    ],
    isLoading: false,
    error: null,
  }),
  useOrganizationRoles: () => ({
    data: [{ metadata: { name: "admin" }, spec: { displayName: "Admin" } }],
    isLoading: false,
    error: null,
  }),
  useOrganizationInviteLink: () => ({
    data: { token: "invite-token", enabled: false },
    isLoading: false,
    error: null,
  }),
  useAssignRole: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useSetUserOwner: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useRemoveOrganizationSubject: () => ({ mutateAsync: vi.fn(), isPending: false }),
  useUpdateOrganizationInviteLink: () => ({ mutateAsync: vi.fn(), isPending: false }),
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
