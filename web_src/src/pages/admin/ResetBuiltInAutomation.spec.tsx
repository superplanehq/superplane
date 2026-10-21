import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { ResetBuiltInAutomation } from "./ResetBuiltInAutomation";

type CatalogAutomation = {
  id: string;
  name: string;
  last_reset_at?: string;
  last_reset_by?: string;
};

const automations: CatalogAutomation[] = [
  { id: "intake:github-issues", name: "GitHub issues" },
  { id: "pr-feedback:discussion", name: "Pull request discussion" },
];

function catalogResponse(items: CatalogAutomation[] = automations) {
  return new Response(JSON.stringify({ automations: items }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

afterEach(() => vi.unstubAllGlobals());

describe("Reset one built-in automation", () => {
  it("resets the selected automation in every organization", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      if (init?.method === "POST") {
        return new Response(
          JSON.stringify({
            reset: 2,
            organizations: 2,
            failed: [],
            last_reset_at: "2026-10-21T15:04:00.000Z",
            last_reset_by: "Ada Admin",
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        );
      }
      return catalogResponse();
    });
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    render(<ResetBuiltInAutomation />);

    await user.click(await screen.findByRole("button", { name: "Reset GitHub issues" }));
    const dialog = await screen.findByRole("dialog", { name: "Reset GitHub issues" });
    await user.click(within(dialog).getByRole("button", { name: "Reset GitHub issues" }));

    expect(await screen.findByText("Reset 2 GitHub issues automations in 2 organizations.")).toBeInTheDocument();
    const row = screen.getByRole("row", { name: /GitHub issues/ });
    expect(within(row).getByText("Ada Admin")).toBeInTheDocument();
    expect(within(row).queryByText("—")).not.toBeInTheDocument();
    const resetCall = fetchMock.mock.calls.find(([, init]) => init?.method === "POST");
    expect(String(resetCall?.[0])).toBe("/admin/api/built-in-automations/reset");
    expect(resetCall?.[1]?.body).toBe(JSON.stringify({ id: "intake:github-issues" }));
  });

  it("shows organizations that were not changed", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "POST") {
          return new Response(
            JSON.stringify({
              reset: 1,
              organizations: 1,
              failed: [
                {
                  organization_id: "org-1",
                  organization_name: "Acme",
                  error:
                    "Reset stopped: executionTimeoutSeconds must be at least 0. This organization was not changed.",
                },
              ],
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
          );
        }
        return catalogResponse();
      }),
    );
    const user = userEvent.setup();
    render(<ResetBuiltInAutomation />);

    await user.click(await screen.findByRole("button", { name: "Reset GitHub issues" }));
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Reset GitHub issues" }));

    expect(
      await screen.findByText("Reset 1 GitHub issues automation in 1 organization. 1 organization was not changed."),
    ).toBeInTheDocument();
    expect(
      screen.getByText(
        "Acme: Reset stopped: executionTimeoutSeconds must be at least 0. This organization was not changed.",
      ),
    ).toBeInTheDocument();
  });

  it("keeps internal server errors private", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        if (init?.method === "POST") {
          return new Response("database password", { status: 500 });
        }
        return catalogResponse();
      }),
    );
    const user = userEvent.setup();
    render(<ResetBuiltInAutomation />);

    await user.click(await screen.findByRole("button", { name: "Reset Pull request discussion" }));
    const dialog = await screen.findByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "Reset Pull request discussion" }));

    expect(await within(dialog).findByText("Could not reset this automation.")).toBeInTheDocument();
    expect(screen.queryByText("database password")).not.toBeInTheDocument();
  });

  it("does not reset when the administrator cancels", async () => {
    const fetchMock = vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
      void init;
      return catalogResponse();
    });
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    render(<ResetBuiltInAutomation />);

    await user.click(await screen.findByRole("button", { name: "Reset GitHub issues" }));
    await user.click(within(await screen.findByRole("dialog")).getByRole("button", { name: "Cancel" }));

    expect(fetchMock.mock.calls.some(([, init]) => init?.method === "POST")).toBe(false);
    expect(screen.queryByRole("dialog")).not.toBeInTheDocument();
  });

  it("shows who last reset each automation", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        catalogResponse([
          {
            id: "intake:github-issues",
            name: "GitHub issues",
            last_reset_at: "2026-10-21T12:00:00.000Z",
            last_reset_by: "Ada Admin",
          },
          { id: "pr-feedback:discussion", name: "Pull request discussion" },
          { id: "line-implementation", name: "Implement" },
          { id: "pr-closure", name: "Pull request closure" },
        ]),
      ),
    );

    render(<ResetBuiltInAutomation />);

    const intake = await screen.findByRole("rowgroup", { name: "Intake" });
    const implementation = screen.getByRole("rowgroup", { name: "Implementation" });
    const verify = screen.getByRole("rowgroup", { name: "Verify" });
    const done = screen.getByRole("rowgroup", { name: "Done" });
    expect(within(intake).getByRole("row", { name: /GitHub issues/ })).toBeInTheDocument();
    expect(within(intake).queryByRole("row", { name: /Pull request discussion/ })).not.toBeInTheDocument();
    expect(within(implementation).getByRole("button", { name: "Reset Implement" })).toBeInTheDocument();
    expect(within(verify).getByRole("row", { name: /Pull request discussion/ })).toBeInTheDocument();
    expect(within(verify).queryByRole("button", { name: "Reset Pull request closure" })).not.toBeInTheDocument();
    expect(within(done).getByRole("button", { name: "Reset Pull request closure" })).toBeInTheDocument();

    const resetRow = within(intake).getByRole("row", { name: /GitHub issues/ });
    expect(within(resetRow).getByText("Ada Admin")).toBeInTheDocument();
    expect(within(resetRow).queryByText("—")).not.toBeInTheDocument();

    const untouchedRow = within(verify).getByRole("row", { name: /Pull request discussion/ });
    expect(within(untouchedRow).getAllByText("—")).toHaveLength(2);
    expect(screen.queryByRole("combobox", { name: "Automation" })).not.toBeInTheDocument();
  });
});
