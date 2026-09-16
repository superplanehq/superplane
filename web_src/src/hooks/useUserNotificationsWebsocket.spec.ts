import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";
import { createElement, type ReactNode } from "react";
import { MemoryRouter } from "react-router";

const { useWebSocketMock, useAccountMock, useNotificationSettingsMock } = vi.hoisted(() => ({
  useWebSocketMock: vi.fn(),
  useAccountMock: vi.fn(),
  useNotificationSettingsMock: vi.fn(),
}));

vi.mock("@/lib/reactUseWebsocket", () => ({
  useWebSocket: useWebSocketMock,
}));

vi.mock("@/contexts/useAccount", () => ({
  useAccount: useAccountMock,
}));

vi.mock("@/hooks/useNotificationSettings", () => ({
  useNotificationSettings: useNotificationSettingsMock,
  useUpdateNotificationSettings: vi.fn(),
}));

import { useUserNotificationsWebsocket } from "@/hooks/useUserNotificationsWebsocket";

afterEach(() => {
  vi.clearAllMocks();
});

function lastCall() {
  const call = useWebSocketMock.mock.calls.at(-1);
  if (!call) throw new Error("useWebSocket was not invoked");
  return call;
}

function renderNotificationsHook(pathname = "/org-1/settings") {
  const queryClient = new QueryClient();
  renderHook(() => useUserNotificationsWebsocket("org-1"), {
    wrapper: ({ children }: { children: ReactNode }) =>
      createElement(
        QueryClientProvider,
        { client: queryClient },
        createElement(MemoryRouter, { initialEntries: [pathname] }, children),
      ),
  });
}

function emit(payload: unknown) {
  const [, options] = lastCall();
  const onMessage = options.onMessage as (e: MessageEvent<unknown>) => void;
  act(() => {
    onMessage(
      new MessageEvent("message", {
        data: JSON.stringify(payload),
      }),
    );
  });
}

const browserOnSettings = {
  workspaces: { scope: "WORKSPACE_SCOPE_NONE" as const, eventTypes: [], filters: [] },
  browser: {
    scope: "WORKSPACE_SCOPE_ALL" as const,
    eventTypes: [],
    filters: [],
    showWhileViewing: true,
  },
};

describe("useUserNotificationsWebsocket", () => {
  const notificationSpy = vi.fn<(title: string, options?: NotificationOptions) => void>();
  let permission: NotificationPermission;

  beforeEach(() => {
    permission = "granted";
    notificationSpy.mockReset();
    class FakeNotification {
      static get permission() {
        return permission;
      }
      static requestPermission = vi.fn(async () => permission);
      onclick: (() => void) | null = null;
      close = vi.fn();
      constructor(
        public title: string,
        public options?: NotificationOptions,
      ) {
        notificationSpy(title, options);
      }
    }
    Object.defineProperty(window, "Notification", {
      configurable: true,
      value: FakeNotification,
    });
    useAccountMock.mockReturnValue({ account: { id: "user-1", email: "ada@example.com" } });
    useNotificationSettingsMock.mockReturnValue({ data: browserOnSettings, isPending: false });
    Object.defineProperty(document, "visibilityState", { configurable: true, value: "visible" });
  });

  it("connects on any organization page when the browser channel is on", () => {
    renderNotificationsHook("/org-1/settings/account");
    const [url, , enabled] = lastCall();
    expect(url).toContain("/ws/users/notifications");
    expect(url).toContain("organization_id=org-1");
    expect(enabled).toBe(true);
  });

  it("does not connect when the browser channel is off", () => {
    useNotificationSettingsMock.mockReturnValue({
      data: {
        workspaces: { scope: "WORKSPACE_SCOPE_ALL" },
        browser: { scope: "WORKSPACE_SCOPE_NONE" },
      },
      isPending: false,
    });
    renderNotificationsHook();
    const [, , enabled] = lastCall();
    expect(enabled).toBe(false);
  });

  it("raises a notification outside workspace routes", () => {
    renderNotificationsHook("/org-1/settings");
    emit({
      event: "user_notification",
      payload: {
        factoryKey: "SP",
        orderKey: "SP-1",
        title: "[SP-1] New comment",
        body: "Ada commented on SP-1.",
        urlPath: "/org-1/workspaces/SP/work-order/1",
      },
    });
    expect(notificationSpy).toHaveBeenCalledWith("[SP-1] New comment", {
      body: "Ada commented on SP-1.",
      tag: "SP-1",
    });
  });

  it("raises a notification while the source board is visible by default", () => {
    renderNotificationsHook("/org-1/workspaces/sp/lines/line-1");
    emit({
      event: "user_notification",
      payload: {
        factoryKey: "SP",
        orderKey: "SP-1",
        title: "[SP-1] New comment",
        body: "Ada commented on SP-1.",
      },
    });
    expect(notificationSpy).toHaveBeenCalled();
  });

  it("skips an alert on the visible source board when that setting is off", () => {
    useNotificationSettingsMock.mockReturnValue({
      data: {
        ...browserOnSettings,
        browser: { ...browserOnSettings.browser, showWhileViewing: false },
      },
      isPending: false,
    });
    renderNotificationsHook("/org-1/workspaces/sp/lines/line-1");
    emit({
      event: "user_notification",
      payload: {
        factoryKey: "SP",
        orderKey: "SP-1",
        title: "[SP-1] New comment",
        body: "Ada commented on SP-1.",
      },
    });
    expect(notificationSpy).not.toHaveBeenCalled();
  });

  it("skips the alert when permission is not granted", () => {
    permission = "denied";
    renderNotificationsHook("/org-1/settings");
    emit({
      event: "user_notification",
      payload: {
        factoryKey: "SP",
        orderKey: "SP-1",
        title: "[SP-1] New comment",
        body: "Ada commented on SP-1.",
      },
    });
    expect(notificationSpy).not.toHaveBeenCalled();
  });
});
