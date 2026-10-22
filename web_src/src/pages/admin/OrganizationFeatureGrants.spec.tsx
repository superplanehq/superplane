import { act, render, screen, waitFor, within } from "@testing-library/react";
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
  expect(await screen.findByRole("button", { name: "Remove grant for Bring your own API key" })).toBeInTheDocument();
  rerender(<OrganizationFeatureGrants orgId="org-2" />);
  expect(screen.queryByRole("table")).not.toBeInTheDocument();
  await act(async () => {
    pending.get("/admin/api/organizations/org-2/features")?.(new Response(JSON.stringify(grants(false))));
  });
  const addGrant = await screen.findByRole("button", { name: "Add grant for Bring your own API key" });
  await user.click(addGrant);
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
  expect(await screen.findByRole("button", { name: "Add grant for Bring your own API key" })).toBeInTheDocument();
  await act(async () => {
    pending.get("/admin/api/organizations/org-1/features")?.(new Response(JSON.stringify(grants(true))));
  });
  expect(screen.getByRole("button", { name: "Add grant for Bring your own API key" })).toBeInTheDocument();
});

it("lists all features with their access source and manages manual grants independently of plan access", async () => {
  const definitions = [
    { key: "byo_key", label: "Bring your own API key", gated: false },
    { key: "byos", label: "Bring your own subscription", gated: true },
    { key: "extra", label: "Extra feature", gated: true },
  ];
  const requests: unknown[] = [];
  let granted = true;
  let extraGranted = false;
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      if (String(input).endsWith("feature-definitions")) return new Response(JSON.stringify(definitions));
      if (init?.method === "PUT") {
        const body = JSON.parse(String(init.body));
        requests.push(body);
        if (body.feature_key === "byos") granted = body.enabled;
        if (body.feature_key === "extra") extraGranted = body.enabled;
      }
      return new Response(
        JSON.stringify({
          grants: [
            { feature_key: "byos", enabled: granted },
            { feature_key: "extra", enabled: extraGranted },
          ],
          effective: { features: { byo_key: true, byos: true, extra: extraGranted } },
        }),
      );
    }),
  );
  const user = userEvent.setup();
  render(<OrganizationFeatureGrants orgId="org-1" />);
  const table = await screen.findByRole("table", { name: "Organization features" });
  expect(within(table).queryByRole("checkbox")).not.toBeInTheDocument();
  const globalRow = within(table).getByRole("row", { name: /Bring your own API key/ });
  expect(within(globalRow).getByText("Enabled")).toBeInTheDocument();
  expect(within(globalRow).getByText("All organizations")).toBeInTheDocument();
  expect(within(globalRow).queryByRole("button")).not.toBeInTheDocument();
  const planRow = within(table).getByRole("row", { name: /Bring your own subscription/ });
  expect(within(planRow).getByText("Manual grant")).toBeInTheDocument();
  await user.click(within(planRow).getByRole("button", { name: /Remove grant/ }));
  await waitFor(() => expect(within(planRow).getByText("Plan")).toBeInTheDocument());
  expect(within(planRow).getByText("Enabled")).toBeInTheDocument();
  expect(within(planRow).queryByRole("button")).not.toBeInTheDocument();
  const extraRow = within(table).getByRole("row", { name: /Extra feature/ });
  expect(within(extraRow).getByText("Disabled")).toBeInTheDocument();
  expect(within(extraRow).getByText("No access")).toBeInTheDocument();
  await user.click(within(extraRow).getByRole("button", { name: /Add grant/ }));
  await waitFor(() => expect(within(extraRow).getByText("Manual grant")).toBeInTheDocument());
  expect(within(extraRow).getByText("Enabled")).toBeInTheDocument();
  expect(requests).toEqual([
    { feature_key: "byos", enabled: false },
    { feature_key: "extra", enabled: true },
  ]);
});
