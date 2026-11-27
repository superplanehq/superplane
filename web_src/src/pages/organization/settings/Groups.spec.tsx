import type { GroupsGroup, RolesRole } from "@/api-client";
import { client } from "@/api-client/client.gen";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { Groups } from "./Groups";

vi.mock("@/hooks/usePageTitle", () => ({ usePageTitle: vi.fn() }));
vi.mock("@/hooks/useReportPageReady", () => ({ useReportPageReady: vi.fn() }));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
const originalConfig = client.getConfig();
afterEach(() => {
  queryClient.clear();
  client.setConfig(originalConfig);
});

function renderGroups(groups: GroupsGroup[], roles: RolesRole[]) {
  queryClient.setQueryData(organizationKeys.groups("org-1"), groups);
  queryClient.setQueryData(organizationKeys.roles("org-1"), roles);
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Groups organizationId="org-1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function rowOrder() {
  return screen
    .getAllByRole("row")
    .slice(1)
    .map((row) => within(row).getByRole("link").textContent);
}

it("sorts each column in both directions with stable ties and defaults without changing query data", () => {
  const groups: GroupsGroup[] = [
    {
      metadata: { name: "Beta", createdAt: "2026-01-01T00:00:00Z" },
      spec: { displayName: "A team", role: "gamma" },
      status: { membersCount: 10 },
    },
    {
      metadata: { name: "alpha", createdAt: "2025-12-31T23:30:00-02:00" },
      spec: { displayName: "Z team", role: "Beta" },
      status: { membersCount: 2 },
    },
    {
      metadata: { name: "ALPHA", createdAt: "2026-01-01T01:30:00Z" },
      spec: { displayName: "Y team", role: "beta" },
      status: { membersCount: 2 },
    },
    { spec: { displayName: "M team" } },
    {
      metadata: { name: "", createdAt: "" },
      spec: { displayName: "N team", role: "" },
      status: { membersCount: 0 },
    },
    {
      metadata: { name: "gamma", createdAt: "2026-01-01T00:30:00Z" },
      spec: { displayName: "B team", role: "alpha" },
      status: { membersCount: 100 },
    },
  ];
  const roles: RolesRole[] = [
    { metadata: { name: "alpha" }, spec: { displayName: "Z role" } },
    { metadata: { name: "Beta" }, spec: { displayName: "M role" } },
    { metadata: { name: "beta" }, spec: { displayName: "N role" } },
    { metadata: { name: "gamma" }, spec: { displayName: "A role" } },
  ];
  const originalGroups = structuredClone(groups);
  renderGroups(groups, roles);
  const cachedGroups = queryClient.getQueryData(organizationKeys.groups("org-1"));

  expect(rowOrder()).toEqual(["A team", "Z team", "Y team", "M team", "N team", "B team"]);
  const columns = [
    {
      name: "Team name",
      ascending: ["M team", "N team", "Z team", "Y team", "A team", "B team"],
      descending: ["B team", "A team", "Z team", "Y team", "M team", "N team"],
    },
    {
      name: "Created",
      ascending: ["M team", "N team", "A team", "B team", "Z team", "Y team"],
      descending: ["Z team", "Y team", "B team", "A team", "M team", "N team"],
    },
    {
      name: "Members",
      ascending: ["M team", "N team", "Z team", "Y team", "A team", "B team"],
      descending: ["B team", "A team", "Z team", "Y team", "M team", "N team"],
    },
    {
      name: "Role",
      ascending: ["M team", "N team", "B team", "Z team", "Y team", "A team"],
      descending: ["A team", "Z team", "Y team", "B team", "M team", "N team"],
    },
  ];
  for (const column of columns) {
    const header = screen.getByRole("columnheader", { name: column.name });
    fireEvent.click(header);
    expect(rowOrder()).toEqual(column.ascending);
    fireEvent.click(header);
    expect(rowOrder()).toEqual(column.descending);
    expect(queryClient.getQueryData(organizationKeys.groups("org-1"))).toBe(cachedGroups);
    expect(cachedGroups).toEqual(originalGroups);
    expect(groups).toEqual(originalGroups);
  }
});

it.each<{ name: string; group: GroupsGroup; roles: RolesRole[]; label: string }>([
  {
    name: "exact role match",
    group: { spec: { role: "admin" } },
    roles: [
      { metadata: { name: "Admin" }, spec: { displayName: "Wrong role" } },
      { metadata: { name: "admin" }, spec: { displayName: "Administrator" } },
    ],
    label: "Administrator",
  },
  { name: "unmatched role", group: { spec: { role: "unknown" } }, roles: [], label: "Select Role" },
  { name: "absent role", group: {}, roles: [], label: "Select Role" },
  {
    name: "missing role spec",
    group: { spec: { role: "admin" } },
    roles: [{ metadata: { name: "admin" } }],
    label: "Select Role",
  },
  {
    name: "missing display name",
    group: { spec: { role: "admin" } },
    roles: [{ metadata: { name: "admin" }, spec: {} }],
    label: "Select Role",
  },
  {
    name: "empty display name",
    group: { spec: { role: "admin" } },
    roles: [{ metadata: { name: "admin" }, spec: { displayName: "" } }],
    label: "Select Role",
  },
  {
    name: "absent role matching incomplete metadata",
    group: {},
    roles: [{ spec: { displayName: "Unnamed role" } }],
    label: "Unnamed role",
  },
])("shows the role label for $name", ({ group, roles, label }) => {
  renderGroups([group], roles);
  expect(screen.getByRole("button", { name: label })).toBeEnabled();
});

it("updates the selected role and disables every role trigger until the response completes", async () => {
  const groups: GroupsGroup[] = [
    { metadata: { name: "engineering" }, spec: { displayName: "Engineering", role: "viewer" } },
    { metadata: { name: "support" }, spec: { displayName: "Support" } },
  ];
  const roles: RolesRole[] = [
    { metadata: { name: "viewer" }, spec: { displayName: "Viewer" } },
    { metadata: { name: "admin" }, spec: { displayName: "Administrator" } },
  ];
  const requests: Request[] = [];
  let completeUpdate!: (response: Response) => void;
  const updateResponse = new Promise<Response>((resolve) => {
    completeUpdate = resolve;
  });
  client.setConfig({
    baseUrl: "http://localhost",
    fetch: async (input, init) => {
      const request = input instanceof Request ? input : new Request(input, init);
      if (request.method === "GET") return Response.json({ groups });
      requests.push(request.clone());
      return updateResponse;
    },
  });
  renderGroups(structuredClone(groups), roles);

  fireEvent.keyDown(screen.getByRole("button", { name: "Viewer" }), { key: "Enter" });
  fireEvent.click(await screen.findByRole("menuitem", { name: "Administrator" }));

  await waitFor(() => {
    const triggers = screen.getAllByRole("button", { name: "Updating..." });
    expect(triggers).toHaveLength(2);
    for (const trigger of triggers) expect(trigger).toBeDisabled();
  });
  expect(requests).toHaveLength(1);
  expect(requests[0].method).toBe("PUT");
  expect(new URL(requests[0].url).pathname).toBe("/api/v1/groups/engineering");
  expect(await requests[0].json()).toEqual({
    group: { metadata: { name: "engineering" }, spec: { role: "admin" } },
    domainId: "org-1",
    domainType: "DOMAIN_TYPE_ORGANIZATION",
  });

  await act(async () => {
    groups[0] = { ...groups[0], spec: { ...groups[0].spec, role: "admin" } };
    completeUpdate(Response.json({ group: groups[0] }));
  });
  await waitFor(() => {
    expect(screen.getByRole("button", { name: "Administrator" })).toBeEnabled();
    expect(screen.getByRole("button", { name: "Select Role" })).toBeEnabled();
    expect(screen.queryByRole("button", { name: "Updating..." })).not.toBeInTheDocument();
  });
});
