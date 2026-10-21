import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { OrganizationFeatureGrants } from "./OrganizationFeatureGrants";

afterEach(() => vi.unstubAllGlobals());
it("removes an extra grant and shows the effective access returned by the server", async () => {
  const requests: unknown[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("feature-definitions"))
        return new Response(JSON.stringify([{ key: "byo_key", label: "Bring your own API key", gated: true }]));
      if (init?.method === "PUT") requests.push(JSON.parse(String(init.body)));
      return new Response(
        JSON.stringify({
          grants: [{ feature_key: "byo_key", enabled: init?.method !== "PUT" }],
          effective: { features: { byo_key: true } },
        }),
      );
    }),
  );
  const user = userEvent.setup();
  render(<OrganizationFeatureGrants orgId="org-1" />);
  await user.click(await screen.findByRole("checkbox", { name: "Bring your own API key" }));
  await waitFor(() => expect(requests).toEqual([{ feature_key: "byo_key", enabled: false }]));
  expect(screen.getByText("Enabled")).toBeInTheDocument();
});
