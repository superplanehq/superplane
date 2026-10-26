import { StrictMode } from "react";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { Link, MemoryRouter, Route, Routes } from "react-router";
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

function renderInvitation(path = "/invitations/email-token", options: { strictMode?: boolean } = {}) {
  const page = (
    <AccountContext.Provider value={account}>
      <MemoryRouter initialEntries={[path]}>
        <Link to="/invitations/email-token">Open original invitation</Link>
        <Link to="/invitations/next-token">Open next invitation</Link>
        <Routes>
          <Route path="/invitations/:token" element={<InviteLinkAccept />} />
          <Route path="/invite/:token" element={<InviteLinkAccept />} />
          <Route path="/engineering" element={<p>Organization opened</p>} />
        </Routes>
      </MemoryRouter>
    </AccountContext.Provider>
  );
  return render(options.strictMode ? <StrictMode>{page}</StrictMode> : page);
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

  it("accepts a different invitation after the current invitation fails", async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL) =>
      String(input).includes("next-token")
        ? Response.json({ organization_id: "org-1", organization_slug: "engineering" })
        : Response.json({ message: "This invitation was canceled." }, { status: 400 }),
    );
    vi.stubGlobal("fetch", fetchMock);

    renderInvitation();
    expect(await screen.findByText("This invitation was canceled.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("link", { name: "Open next invitation" }));

    expect(await screen.findByText("Organization opened")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  for (const previousOutcome of ["accepted", "rejected"]) {
    it(`ignores a late ${previousOutcome} response from a previous invitation`, async () => {
      let completePreviousRequest!: (response: Response) => void;
      const previousResponse = new Promise<Response>((resolve) => {
        completePreviousRequest = resolve;
      });
      const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
        if (String(input).includes("email-token")) {
          return previousResponse;
        }
        return Response.json({ message: "The current invitation expired." }, { status: 400 });
      });
      vi.stubGlobal("fetch", fetchMock);

      renderInvitation();
      await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
      fireEvent.click(screen.getByRole("link", { name: "Open next invitation" }));
      expect(await screen.findByText("The current invitation expired.")).toBeInTheDocument();

      await act(async () => {
        completePreviousRequest(
          previousOutcome === "accepted"
            ? Response.json({ organization_id: "org-1", organization_slug: "engineering" })
            : Response.json({ message: "The previous invitation was canceled." }, { status: 400 }),
        );
      });

      expect(screen.getByText("The current invitation expired.")).toBeInTheDocument();
      expect(screen.queryByText("Organization opened")).not.toBeInTheDocument();
      expect(screen.queryByText("The previous invitation was canceled.")).not.toBeInTheDocument();
    });
  }

  it("submits an invitation again after visiting another invitation", async () => {
    const fetchMock = vi.fn(async () => Response.json({ message: "Invitation unavailable." }, { status: 400 }));
    vi.stubGlobal("fetch", fetchMock);

    renderInvitation();
    expect(await screen.findByText("Invitation unavailable.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("link", { name: "Open next invitation" }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(await screen.findByText("Invitation unavailable.")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("link", { name: "Open original invitation" }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
    expect(await screen.findByText("Invitation unavailable.")).toBeInTheDocument();
  });

  it("submits once when StrictMode repeats the initial effect", async () => {
    const fetchMock = vi.fn(async () => Response.json({ organization_id: "org-1", organization_slug: "engineering" }));
    vi.stubGlobal("fetch", fetchMock);

    renderInvitation("/invitations/email-token", { strictMode: true });

    expect(await screen.findByText("Organization opened")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
  for (const [reason, message] of Object.entries({
    PAYMENT_PAST_DUE: "This organization cannot add members until its payment is updated. Ask an organization admin.",
    NO_SEAT_AVAILABLE: "No seat is available. Ask an organization admin to free or buy a seat.",
    MEMBER_LIMIT_REACHED: "This organization reached its member limit. Ask an organization admin.",
    CHECKOUT_SEAT_LIMIT:
      "This organization is changing its plan. The new plan has no seat for you yet. Ask an organization admin to finish the plan change and buy a seat for you.",
    BILLING_UNVERIFIED: "SuperPlane could not check the billing state of this organization. Try again later.",
    INVITATION_EXPIRED: "This invitation expired. Ask an organization admin for a new invitation.",
    INVITATION_CANCELED: "This invitation was canceled. Ask an organization admin for a new invitation.",
    INVITATION_ALREADY_ACCEPTED:
      "This invitation was already accepted. Ask an organization admin for a new invitation.",
    INVITATION_REPLACED: "A new invitation replaced this link. Open the latest invitation email.",
    INVITATION_INVITER_REVOKED: "Ask an organization admin to send this invitation again.",
    INVITATION_EMAIL_MISMATCH:
      "This invitation is for j***@example.com. Sign out, then sign in with the invited email address.",
  })) {
    it(`shows the recovery message for ${reason}`, async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn(async () =>
          Response.json(
            {
              code: reason === "INVITATION_EMAIL_MISMATCH" ? 7 : 9,
              message: "Server message",
              details: [
                {
                  "@type": "type.googleapis.com/google.rpc.ErrorInfo",
                  domain: "superplane.com",
                  reason,
                  metadata: { invited_email: "j***@example.com" },
                },
              ],
            },
            { status: 403 },
          ),
        ),
      );
      renderInvitation();
      expect(await screen.findByText(message)).toBeInTheDocument();
      if (reason === "INVITATION_EMAIL_MISMATCH") {
        expect(screen.getByRole("link", { name: "Sign out" })).toHaveAttribute(
          "href",
          "/logout?redirect=%2Finvitations%2Femail-token",
        );
      } else {
        expect(screen.queryByRole("link", { name: "Sign out" })).not.toBeInTheDocument();
      }
    });
  }

  it("identifies an unknown token", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => Response.json({ code: 5, message: "Invitation or organization not found." }, { status: 404 })),
    );
    renderInvitation();
    expect(
      await screen.findByText("This invitation link is invalid. Ask an organization admin for a new invitation."),
    ).toBeInTheDocument();
  });

  for (const reason of ["FUTURE_REASON", "constructor"]) {
    it(`uses the safe server message for unrecognized reason ${reason}`, async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn(async () =>
          Response.json(
            {
              message: "Ask an organization admin for help.",
              details: [{ "@type": "type.googleapis.com/google.rpc.ErrorInfo", domain: "superplane.com", reason }],
            },
            { status: 400 },
          ),
        ),
      );
      renderInvitation();
      expect(await screen.findByText("Ask an organization admin for help.")).toBeInTheDocument();
    });
  }

  it("uses the safe fallback for an unexpected server response", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => new Response("<!doctype html><html>Proxy failure</html>", { status: 502 })),
    );
    renderInvitation();
    expect(await screen.findByText("SuperPlane could not accept this invitation. Try again.")).toBeInTheDocument();
  });

  it("keeps the shared invitation route", async () => {
    const fetchMock = vi.fn(async () => Response.json({ organization_id: "org-1", organization_slug: "engineering" }));
    vi.stubGlobal("fetch", fetchMock);
    renderInvitation("/invite/shared-token");
    expect(await screen.findByText("Organization opened")).toBeInTheDocument();
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/invite-links/shared-token/accept",
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
  });
});
