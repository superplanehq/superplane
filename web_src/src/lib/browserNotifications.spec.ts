import { describe, expect, it, vi } from "bun:test";

import {
  browserNotificationIconUrl,
  isWorkspaceBoardPath,
  raiseUserBrowserNotification,
  shouldRaiseBrowserNotification,
} from "./browserNotifications";

describe("isWorkspaceBoardPath", () => {
  it("matches the workspace home and line board", () => {
    expect(isWorkspaceBoardPath("/org-1/workspaces/sp", "SP")).toBe(true);
    expect(isWorkspaceBoardPath("/org-1/workspaces/sp/lines/line-1", "SP")).toBe(true);
  });

  it("does not match settings or task pages", () => {
    expect(isWorkspaceBoardPath("/org-1/settings", "SP")).toBe(false);
    expect(isWorkspaceBoardPath("/org-1/workspaces/sp/task/1", "SP")).toBe(false);
    expect(isWorkspaceBoardPath("/org-1/workspaces/sp/lines/new", "SP")).toBe(false);
  });
});

describe("shouldRaiseBrowserNotification", () => {
  it("skips the alert when permission is not granted", () => {
    expect(
      shouldRaiseBrowserNotification({
        permission: "denied",
        showWhileViewing: true,
        tabVisible: true,
        pathname: "/org-1/settings",
        factoryKey: "SP",
      }),
    ).toBe(false);
  });

  it("raises outside workspace routes", () => {
    expect(
      shouldRaiseBrowserNotification({
        permission: "granted",
        showWhileViewing: false,
        tabVisible: true,
        pathname: "/org-1/settings",
        factoryKey: "SP",
      }),
    ).toBe(true);
  });

  it("skips an alert on the visible source board when that setting is off", () => {
    expect(
      shouldRaiseBrowserNotification({
        permission: "granted",
        showWhileViewing: false,
        tabVisible: true,
        pathname: "/org-1/workspaces/sp/lines/line-1",
        factoryKey: "SP",
      }),
    ).toBe(false);
  });
});

describe("raiseUserBrowserNotification", () => {
  it("navigates to the task when the notification is clicked", () => {
    const navigate = vi.fn();
    const close = vi.fn();
    let clickHandler: (() => void) | undefined;
    class FakeNotification {
      constructor(
        public title: string,
        public options?: NotificationOptions,
      ) {}
      set onclick(handler: (() => void) | null) {
        clickHandler = handler ?? undefined;
      }
      close = close;
    }
    Object.defineProperty(window, "Notification", {
      configurable: true,
      value: FakeNotification,
    });

    raiseUserBrowserNotification(
      {
        title: "[SP-1] New comment",
        body: "Ada commented on SP-1.",
        orderKey: "SP-1",
        urlPath: "/org-1/workspaces/SP/work-order/1",
      },
      navigate,
    );
    clickHandler?.();
    expect(navigate).toHaveBeenCalledWith("/org-1/workspaces/SP/work-order/1");
    expect(close).toHaveBeenCalled();
  });

  it("sets the SuperPlane icon", () => {
    const constructed: NotificationOptions[] = [];
    class FakeNotification {
      constructor(
        public title: string,
        public options?: NotificationOptions,
      ) {
        constructed.push(options ?? {});
      }
      set onclick(_handler: (() => void) | null) {}
      close = vi.fn();
    }
    Object.defineProperty(window, "Notification", {
      configurable: true,
      value: FakeNotification,
    });

    raiseUserBrowserNotification({ title: "[SP-1] New comment" }, vi.fn());
    expect(constructed[0]?.icon).toBe(browserNotificationIconUrl());
    expect(constructed[0]?.icon).toContain("/favicon.ico");
  });
});
