import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import { OrgCanvasesTable } from "./OrgCanvasesTable";

afterEach(() => vi.unstubAllGlobals());

describe("organization automation reset failures", () => {
  for (const scenario of [
    {
      name: "shows the saved-setting error so an administrator can correct it",
      status: 400,
      response: 'Reset "Implement": executionTimeoutSeconds must be at least 0. No automations were changed.',
      expected: 'Reset "Implement": executionTimeoutSeconds must be at least 0. No automations were changed.',
    },
    {
      name: "keeps internal server errors private",
      status: 500,
      response: "internal database connection details",
      expected: "Could not reset built-in automations.",
    },
    {
      name: "uses the fallback when a bad-request response has no details",
      status: 400,
      response: "",
      expected: "Could not reset built-in automations.",
    },
  ]) {
    it(scenario.name, async () => {
      vi.stubGlobal(
        "fetch",
        vi.fn(async (input: RequestInfo | URL) => {
          if (String(input).endsWith("/built-in-automations/reset")) {
            return new Response(scenario.response, { status: scenario.status });
          }
          return new Response(JSON.stringify({ items: [], total: 0 }), {
            headers: { "Content-Type": "application/json" },
          });
        }),
      );
      const user = userEvent.setup();
      render(<OrgCanvasesTable orgId="org-1" />);
      await user.click(screen.getByRole("button", { name: "Reset built-in automations" }));
      const dialog = await screen.findByRole("dialog");
      await user.click(within(dialog).getByRole("button", { name: "Reset built-in automations" }));
      expect(await within(dialog).findByText(scenario.expected)).toBeInTheDocument();
      expect(within(dialog).getByRole("button", { name: "Reset built-in automations" })).toBeEnabled();
      if (scenario.status === 500) {
        expect(screen.queryByText(scenario.response)).not.toBeInTheDocument();
      }
    });
  }
});
