import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";

const { updateNotificationSettingsMock } = vi.hoisted(() => ({
  updateNotificationSettingsMock: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  meDescribeNotificationSettings: vi.fn(),
  meUpdateNotificationSettings: updateNotificationSettingsMock,
}));

import { meKeys } from "@/hooks/useMe";
import { useUpdateNotificationSettings } from "@/hooks/useNotificationSettings";

describe("useUpdateNotificationSettings", () => {
  beforeEach(() => {
    updateNotificationSettingsMock.mockReset();
  });

  it("updates browser preferences in every organization me cache", async () => {
    const queryClient = new QueryClient();
    const organizationId = "org-1";
    const lastLocation = { path: "/org-1/workspaces/SP" };
    queryClient.setQueryData(meKeys.me(organizationId, true), {
      id: "user-1",
      browserNotificationPreferences: { enabled: false, showWhileViewing: true },
    });
    queryClient.setQueryData(["me", organizationId, "last-location"], lastLocation);
    queryClient.setQueryData(meKeys.me(organizationId, false), {
      id: "user-1",
      browserNotificationPreferences: { enabled: false, showWhileViewing: true },
    });
    updateNotificationSettingsMock.mockResolvedValue({
      data: {
        settings: {
          browser: {
            scope: "WORKSPACE_SCOPE_ALL",
            showWhileViewing: false,
          },
        },
      },
    });

    const { result } = renderHook(() => useUpdateNotificationSettings(organizationId), {
      wrapper: ({ children }: { children: ReactNode }) =>
        createElement(QueryClientProvider, { client: queryClient }, children),
    });

    await act(async () => {
      await result.current.mutateAsync({
        browser: {
          scope: "WORKSPACE_SCOPE_ALL",
          showWhileViewing: false,
        },
      });
    });

    expect(queryClient.getQueryData(meKeys.me(organizationId, true))).toMatchObject({
      browserNotificationPreferences: { enabled: true, showWhileViewing: false },
    });
    expect(queryClient.getQueryData(meKeys.me(organizationId, false))).toMatchObject({
      browserNotificationPreferences: { enabled: true, showWhileViewing: false },
    });
    expect(queryClient.getQueryData(["me", organizationId, "notification-settings"])).toEqual({
      browser: {
        scope: "WORKSPACE_SCOPE_ALL",
        showWhileViewing: false,
      },
    });
    expect(queryClient.getQueryData(["me", organizationId, "last-location"])).toEqual(lastLocation);
  });
});
