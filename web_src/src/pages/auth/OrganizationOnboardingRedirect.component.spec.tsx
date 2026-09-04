import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { OrganizationOnboardingRedirect } from "./OrganizationOnboardingRedirect";

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: {
      id: "account-1",
      linked_accounts: [{ provider: "github", username: "dev-user" }],
      providers: [],
    },
  }),
}));

describe("OrganizationOnboardingRedirect", () => {
  beforeEach(() => {
    window.history.replaceState(null, "", "/onboarding");
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
        json: () => Promise.resolve({ organizationSlug: "dev-user", workspaceKey: "NEWWO" }),
      }),
    );
  });

  it("keeps the provisional workspace behind the onboarding route", async () => {
    render(
      <OrganizationOnboardingRedirect
        renderWorkspace={(workspace, entryPath) => (
          <div data-entry-path={entryPath} data-testid="internal-workspace-key">
            {workspace.workspaceKey}
          </div>
        )}
      />,
    );

    const workspace = await screen.findByTestId("internal-workspace-key");
    expect(workspace).toHaveTextContent("NEWWO");
    expect(workspace.dataset.entryPath).toMatch(/^\/onboarding\?attempt=[0-9a-f-]+$/);
    expect(window.location.pathname).toBe("/onboarding");
  });
});
