import { focusManager, QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook, waitFor } from "@testing-library/react";
import { createElement, type ReactNode } from "react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { organizationsUpdateOrganization, usersListUsers, organizationsRemoveUser } = vi.hoisted(() => ({
  organizationsUpdateOrganization: vi.fn(),
  usersListUsers: vi.fn(),
  organizationsRemoveUser: vi.fn(),
}));

vi.mock("@/api-client/sdk.gen", () => ({
  organizationsUpdateOrganization,
  usersListUsers,
  organizationsRemoveUser,
}));

import {
  useUpdateOrganization,
  useOrganizationMembers,
  useOrganizationUsers,
  useRemoveOrganizationSubject,
  organizationKeys,
} from "./useOrganizationData";

function createWrapper(queryClient: QueryClient) {
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

describe("useUpdateOrganization", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    organizationsUpdateOrganization.mockResolvedValue({ data: {} });
    window.history.replaceState(null, "", "/onboarding");
  });

  it("sends the organization header outside organization routes", async () => {
    const queryClient = new QueryClient();
    const { result } = renderHook(() => useUpdateOrganization("org-1"), {
      wrapper: createWrapper(queryClient),
    });

    await act(async () => {
      await result.current.mutateAsync({ name: "GitHub Owner", slug: "github-owner" });
    });

    expect(organizationsUpdateOrganization).toHaveBeenCalledWith(
      expect.objectContaining({
        headers: { "x-organization-id": "org-1" },
        path: { id: "org-1" },
      }),
    );
  });
});

it("shares full metadata, preserves array consumers, and refreshes fresh capacity on focus and removal", async () => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const users = [{ metadata: { id: "member" } }];
  usersListUsers.mockResolvedValue({ data: { users, seatLimit: 1 } });
  organizationsRemoveUser.mockResolvedValue({ data: {} });
  const { result, unmount } = renderHook(
    () => ({
      members: useOrganizationMembers("org-1"),
      users: useOrganizationUsers("org-1", true),
      remove: useRemoveOrganizationSubject("org-1"),
    }),
    { wrapper: createWrapper(queryClient) },
  );
  await waitFor(() => expect(result.current.members.isSuccess).toBe(true));
  expect(result.current.users.data).toEqual(users);
  expect(result.current.members.seatLimit).toBe(1);
  expect(queryClient.getQueryData([...organizationKeys.users("org-1"), true])).toEqual({ users, seatLimit: 1 });
  const calls = usersListUsers.mock.calls.length;
  await act(async () => {
    focusManager.setFocused(false);
    focusManager.setFocused(true);
  });
  await waitFor(() => expect(usersListUsers.mock.calls.length).toBe(calls + 1));
  usersListUsers.mockResolvedValue({ data: { users: [], seatLimit: 1 } });
  await act(async () => {
    await result.current.remove.mutateAsync({ userId: "member" });
  });
  await waitFor(() => expect(result.current.members.users).toEqual([]));
  expect(result.current.members.seatLimit).toBe(1);
  unmount();
  queryClient.clear();
  focusManager.setFocused(undefined);
});
it("does not report unlimited capacity when the member response fails", async () => {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  usersListUsers.mockResolvedValue({ error: { message: "Forbidden" } });
  const { result, unmount } = renderHook(() => useOrganizationMembers("org-failed"), {
    wrapper: createWrapper(queryClient),
  });
  await waitFor(() => expect(result.current.isError).toBe(true));
  expect(result.current.isSuccess).toBe(false);
  unmount();
  queryClient.clear();
});
