import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { OrgResetBacklogDefaults } from "./OrgResetBacklogDefaults";

const ORG_ID = "org-1";

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("OrgResetBacklogDefaults", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("resets Backlog automations after confirm", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ reset: 1, failures: [] }));

    render(<OrgResetBacklogDefaults orgId={ORG_ID} />);

    await user.click(screen.getByRole("button", { name: "Reset Backlog defaults" }));
    expect(await screen.findByRole("heading", { name: "Reset Backlog automations" })).toBeInTheDocument();
    expect(screen.getByText(/You cannot undo this action/)).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Reset Backlog automations" }));

    expect(await screen.findByText("Reset 1 Backlog automation.")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(
      `/admin/api/organizations/${ORG_ID}/backlog-defaults/reset`,
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
  });

  it("does not reset when the dialog is cancelled", async () => {
    const user = userEvent.setup();
    render(<OrgResetBacklogDefaults orgId={ORG_ID} />);

    await user.click(screen.getByRole("button", { name: "Reset Backlog defaults" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(screen.queryByText("Reset Backlog automations")).not.toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalled();
  });

  it("shows an error when the reset request fails", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ error: "no" }, 500));

    render(<OrgResetBacklogDefaults orgId={ORG_ID} />);

    await user.click(screen.getByRole("button", { name: "Reset Backlog defaults" }));
    await user.click(screen.getByRole("button", { name: "Reset Backlog automations" }));

    expect(await screen.findByText("Could not reset Backlog automations.")).toBeInTheDocument();
  });
});
