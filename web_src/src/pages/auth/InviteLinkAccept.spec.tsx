import { render, screen } from "@testing-library/react";
import { MemoryRouter, Route, Routes } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { AccountContext, type AccountContextType } from "@/contexts/accountContextState";
import InviteLinkAccept from "./InviteLinkAccept";

vi.mock("@/hooks/useReportPageReady", () => ({
  useReportPageReady: () => undefined,
}));

const account: AccountContextType = {
  account: {
    id: "account-1",
    name: "Invited member",
    email: "invited@example.com",
    avatar_url: "",
    installation_admin: false,
    has_password: false,
  },
  loading: false,
  setupRequired: false,
  refreshAccount: async () => undefined,
};

function renderInvitation() {
  return render(
    <AccountContext.Provider value={account}>
      <MemoryRouter initialEntries={["/invitations/email-token"]}>
        <Routes>
          <Route path="/invitations/:token" element={<InviteLinkAccept />} />
          <Route path="/engineering" element={<p>Organization opened</p>} />
        </Routes>
      </MemoryRouter>
    </AccountContext.Provider>,
  );
}

describe("Email invitation acceptance", () => {
  beforeEach(() => vi.restoreAllMocks());

  it("accepts the invitation and opens its organization", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      if (String(input) !== "/api/v1/invitations/email-token/accept") {
        throw new Error(`unexpected fetch ${String(input)}`);
      }
      return Response.json({ organization_id: "org-1", organization_slug: "engineering" });
    });
    vi.stubGlobal("fetch", fetchMock);

    renderInvitation();

    expect(await screen.findByText("Organization opened")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/invitations/email-token/accept",
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
  });

  it("shows the rejection without opening the organization", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => Response.json({ message: "Sign in with the invited email address." }, { status: 403 })),
    );

    renderInvitation();

    expect(await screen.findByText("Sign in with the invited email address.")).toBeInTheDocument();
    expect(screen.queryByText("Organization opened")).not.toBeInTheDocument();
  });
});
