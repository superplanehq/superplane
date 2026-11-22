import type { GroupsGroup, RolesRole } from "@/api-client";
import { organizationKeys } from "@/hooks/useOrganizationData";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, expect, it, vi } from "bun:test";
import { MemoryRouter } from "react-router";
import { Groups } from "./Groups";

vi.mock("@/hooks/usePageTitle", () => ({ usePageTitle: vi.fn() }));
vi.mock("@/hooks/useReportPageReady", () => ({ useReportPageReady: vi.fn() }));
vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: () => ({ canAct: () => true, isLoading: false }),
}));

const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
afterEach(() => queryClient.clear());

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
  queryClient.setQueryData(organizationKeys.groups("org-1"), groups);
  queryClient.setQueryData(organizationKeys.roles("org-1"), roles);
  const cachedGroups = queryClient.getQueryData(organizationKeys.groups("org-1"));
  render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter>
        <Groups organizationId="org-1" />
      </MemoryRouter>
    </QueryClientProvider>,
  );

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
