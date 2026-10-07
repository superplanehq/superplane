import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { InstallationFactoryTemplates } from "./InstallationFactoryTemplates";

const templates = [
  { id: "backlog", name: "Backlog", description: "Plan new draft tasks.", count: 2 },
  { id: "line-implementation", name: "Implement", description: "Create a branch.", count: 0 },
  { id: "pr-closure", name: "PR Closure", description: "Close the task.", count: 1 },
  { id: "intake", name: "Intake", description: "Import work.", count: 3 },
];

function jsonResponse(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("InstallationFactoryTemplates", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("lists onboarding templates and resets one after confirm", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (input, init) => {
      const url = String(input);
      if (url === "/admin/api/installation/factory-templates") {
        return jsonResponse({ templates });
      }
      if (url === "/admin/api/installation/factory-templates/backlog/reset" && init?.method === "POST") {
        return jsonResponse({ reset: 2, failures: [] });
      }
      return jsonResponse({ error: "unexpected" }, 500);
    });

    render(<InstallationFactoryTemplates />);

    expect(await screen.findByText("Onboarding templates")).toBeInTheDocument();
    expect(screen.getByText("2 automations")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Reset Implement" })).toBeDisabled();

    await user.click(screen.getByRole("button", { name: "Reset Backlog" }));
    expect(await screen.findByRole("heading", { name: "Reset Backlog automations" })).toBeInTheDocument();
    expect(screen.getByText(/You cannot undo this action/)).toBeInTheDocument();
    expect(
      screen.getByText(/replaces every Backlog automation on this installation with the current SuperPlane defaults/),
    ).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Reset Backlog automations" }));

    expect(await screen.findByText("Reset 2 Backlog automations.")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(
      "/admin/api/installation/factory-templates/backlog/reset",
      expect.objectContaining({ method: "POST", credentials: "include" }),
    );
  });

  it("does not reset when the dialog is cancelled", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockResolvedValue(jsonResponse({ templates }));

    render(<InstallationFactoryTemplates />);

    await user.click(await screen.findByRole("button", { name: "Reset Backlog" }));
    await user.click(screen.getByRole("button", { name: "Cancel" }));

    expect(screen.queryByRole("heading", { name: "Reset Backlog automations" })).not.toBeInTheDocument();
    expect(vi.mocked(fetch).mock.calls.some(([, init]) => init?.method === "POST")).toBe(false);
  });

  it("shows an error when the reset request fails", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (input, init) => {
      if (String(input) === "/admin/api/installation/factory-templates") {
        return jsonResponse({ templates });
      }
      if (init?.method === "POST") {
        return jsonResponse({ error: "no" }, 500);
      }
      return jsonResponse({ error: "unexpected" }, 500);
    });

    render(<InstallationFactoryTemplates />);

    await user.click(await screen.findByRole("button", { name: "Reset Intake" }));
    await user.click(screen.getByRole("button", { name: "Reset Intake automations" }));

    expect(await screen.findByText("Could not reset Intake automations.")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Reset Intake automations" })).not.toBeInTheDocument();
  });

  it("names failed automations in the result", async () => {
    const user = userEvent.setup();
    vi.mocked(fetch).mockImplementation(async (input, init) => {
      if (String(input) === "/admin/api/installation/factory-templates") {
        return jsonResponse({ templates });
      }
      if (init?.method === "POST") {
        return jsonResponse({
          reset: 1,
          failures: [{ canvas_id: "c-2", name: "Old Backlog", error: "canvas changed during reset" }],
        });
      }
      return jsonResponse({ error: "unexpected" }, 500);
    });

    render(<InstallationFactoryTemplates />);

    await user.click(await screen.findByRole("button", { name: "Reset Backlog" }));
    await user.click(screen.getByRole("button", { name: "Reset Backlog automations" }));

    expect(
      await screen.findByText("Reset 1 Backlog automation. 1 failed: Old Backlog (canvas changed during reset)."),
    ).toBeInTheDocument();
  });
});
