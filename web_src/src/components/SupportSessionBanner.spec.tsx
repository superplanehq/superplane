import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, mock, vi } from "bun:test";
import { SupportSessionBanner } from "@/components/SupportSessionBanner";
import { SUPPORT_SESSION_END_PATH } from "@/lib/supportSession";

const accountState: { supportSession?: { active: boolean; user_name: string } } = {};

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({ account: { id: "acc-1", support_session: accountState.supportSession } }),
}));

describe("SupportSessionBanner", () => {
  let originalFetch: typeof globalThis.fetch;
  let locationHref: string;

  beforeEach(() => {
    originalFetch = globalThis.fetch;
    locationHref = "http://localhost/acme";
    accountState.supportSession = undefined;
    Object.defineProperty(window, "location", {
      configurable: true,
      value: {
        get href() {
          return locationHref;
        },
        set href(value: string) {
          locationHref = value;
        },
      },
    });
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("renders nothing outside a support session", () => {
    const { container } = render(<SupportSessionBanner />);

    expect(container).toBeEmptyDOMElement();
  });

  it("names the user and states that the session is read-only", () => {
    accountState.supportSession = { active: true, user_name: "Ada Lovelace" };

    render(<SupportSessionBanner />);

    expect(screen.getByRole("status")).toHaveTextContent(
      "Support session for Ada Lovelace. This session is read-only.",
    );
  });

  it("ends the support session and follows the redirect", async () => {
    accountState.supportSession = { active: true, user_name: "Ada Lovelace" };
    const fetchMock = mock().mockResolvedValue(Response.json({ redirect_url: "/admin/accounts" }));
    globalThis.fetch = fetchMock;

    render(<SupportSessionBanner />);
    fireEvent.click(screen.getByRole("button", { name: "End support session" }));

    await waitFor(() => expect(locationHref).toBe("/admin/accounts"));
    expect(fetchMock).toHaveBeenCalledWith(SUPPORT_SESSION_END_PATH, { method: "POST", credentials: "include" });
  });
});
