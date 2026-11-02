import { AccountContext } from "@/contexts/accountContextState";
import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { ChatGPTQuotaNotice } from "./ChatGPTQuotaNotice";

let viewer = "owner-1";
vi.mock("@/contexts/usePermissions", () => ({ usePermissions: () => ({ currentUserId: viewer }) }));

beforeEach(() => {
  viewer = "owner-1";
});

describe("ChatGPT quota recovery", () => {
  it("shows saved quota recovery to the owner and clears the current notice for a new attempt", () => {
    const reset = new Date(Date.now() + 3_600_000).toISOString();
    const notice = render(
      <AccountContext.Provider
        value={{
          account: {
            id: "owner-1",
            name: "Owner",
            email: "owner@example.com",
            avatar_url: "",
            installation_admin: false,
            has_password: false,
            billing_enabled: false,
          },
          loading: false,
          setupRequired: false,
          refreshAccount: async () => undefined,
        }}
      >
        <MemoryRouter>
          <ChatGPTQuotaNotice
            organizationId="org-1"
            factoryKey="sp"
            failure={{ reason: "chatgpt_quota_exhausted", subscriptionOwnerId: "owner-1", resetsAt: reset }}
          />
        </MemoryRouter>
      </AccountContext.Provider>,
    );
    expect(screen.getByRole("alert", { name: "ChatGPT limit reached" })).toHaveTextContent(
      "This run stopped because the ChatGPT subscription reached its usage limit.",
    );
    expect(screen.getByText(new Date(reset).toLocaleString())).toHaveAttribute("dateTime", reset);
    expect(screen.getByRole("link", { name: "View ChatGPT usage" })).toHaveAttribute(
      "href",
      "/org-1/workspaces/sp/settings/account/codex-usage",
    );
    expect(screen.queryByRole("link", { name: "Add credits" })).not.toBeInTheDocument();
    notice.rerender(
      <MemoryRouter>
        <ChatGPTQuotaNotice organizationId="org-1" factoryKey="sp" />
      </MemoryRouter>,
    );
    expect(screen.queryByRole("alert")).not.toBeInTheDocument();
  });

  it("keeps other members out of personal usage and explains an unknown reset", () => {
    viewer = "other-member";
    render(
      <MemoryRouter>
        <ChatGPTQuotaNotice
          organizationId="org-1"
          factoryKey="sp"
          failure={{ reason: "chatgpt_quota_exhausted", subscriptionOwnerId: "owner-1" }}
        />
      </MemoryRouter>,
    );
    expect(screen.getByRole("alert")).toHaveTextContent("The reset time is unavailable.");
    expect(screen.getByRole("alert")).toHaveTextContent("Ask the subscription owner to check ChatGPT usage.");
    expect(screen.queryByRole("link")).not.toBeInTheDocument();
  });
});
