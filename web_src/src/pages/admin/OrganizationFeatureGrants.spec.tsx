import { act, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, expect, it, vi } from "bun:test";
import { OrganizationFeatureGrants } from "./OrganizationFeatureGrants";

afterEach(() => vi.unstubAllGlobals());
const definitions = [{ key: "byo_key", label: "Bring your own API key", gated: true }];
const grants = (enabled: boolean) => ({
  grants: [{ feature_key: "byo_key", enabled }],
  effective: { features: { byo_key: enabled } },
});
it("clears the previous organization's grants as soon as the organization changes", async () => {
  const pending = new Map<string, (response: Response) => void>();
  const puts: string[] = [];
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("feature-definitions")) return new Response(JSON.stringify(definitions));
      if (init?.method === "PUT") {
        puts.push(url);
        return new Response(JSON.stringify(grants(false)));
      }
      return new Promise<Response>((resolve) => pending.set(url, resolve));
    }),
  );
  const user = userEvent.setup();
  const { rerender } = render(<OrganizationFeatureGrants orgId="org-1" />);
  await act(async () => {
    pending.get("/admin/api/organizations/org-1/features")?.(new Response(JSON.stringify(grants(true))));
  });
  expect(await screen.findByRole("checkbox", { name: "Bring your own API key" })).toBeChecked();
  rerender(<OrganizationFeatureGrants orgId="org-2" />);
  expect(screen.queryByRole("checkbox")).not.toBeInTheDocument();
  await act(async () => {
    pending.get("/admin/api/organizations/org-2/features")?.(new Response(JSON.stringify(grants(false))));
  });
  const checkbox = await screen.findByRole("checkbox", { name: "Bring your own API key" });
  expect(checkbox).not.toBeChecked();
  await user.click(checkbox);
  await waitFor(() => expect(puts).toEqual(["/admin/api/organizations/org-2/features"]));
});
it("ignores late responses from the previous organization", async () => {
  const pending = new Map<string, (response: Response) => void>();
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url.endsWith("feature-definitions")) return new Response(JSON.stringify(definitions));
      return new Promise<Response>((resolve) => pending.set(url, resolve));
    }),
  );
  const { rerender } = render(<OrganizationFeatureGrants orgId="org-1" />);
  rerender(<OrganizationFeatureGrants orgId="org-2" />);
  await act(async () => {
    pending.get("/admin/api/organizations/org-2/features")?.(new Response(JSON.stringify(grants(false))));
  });
  expect(await screen.findByRole("checkbox", { name: "Bring your own API key" })).not.toBeChecked();
  await act(async () => {
    pending.get("/admin/api/organizations/org-1/features")?.(new Response(JSON.stringify(grants(true))));
  });
  expect(screen.getByRole("checkbox", { name: "Bring your own API key" })).not.toBeChecked();
});
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
