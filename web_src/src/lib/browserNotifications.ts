export type BrowserNotificationPermission = NotificationPermission | "unsupported";

export type UserNotificationPayload = {
  userId?: string;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderKey?: string;
  eventType?: string;
  title?: string;
  body?: string;
  urlPath?: string;
};

export type UserNotificationWebsocketMessage = {
  event?: string;
  payload?: UserNotificationPayload;
};

export function parseUserNotificationEvent(event: MessageEvent<unknown>): UserNotificationWebsocketMessage | null {
  try {
    return JSON.parse(event.data as string) as UserNotificationWebsocketMessage;
  } catch (error) {
    console.warn("user notifications ws: failed to parse message", error);
    return null;
  }
}

export function currentBrowserNotificationPermission(): BrowserNotificationPermission {
  if (typeof Notification === "undefined") {
    return "unsupported";
  }
  return Notification.permission;
}

export function isWorkspaceBoardPath(pathname: string, factoryKey: string): boolean {
  const key = factoryKey.toLowerCase();
  const parts = pathname.split("/").filter(Boolean);
  if (parts.length < 3 || parts[1] !== "workspaces") {
    return false;
  }
  if (parts[2].toLowerCase() !== key) {
    return false;
  }
  if (parts.length === 3) {
    return true;
  }
  return parts.length === 5 && parts[3] === "lines" && parts[4] !== "new";
}

export function shouldRaiseBrowserNotification({
  permission,
  showWhileViewing,
  tabVisible,
  pathname,
  factoryKey,
}: {
  permission: BrowserNotificationPermission;
  showWhileViewing: boolean;
  tabVisible: boolean;
  pathname: string;
  factoryKey: string;
}): boolean {
  if (permission !== "granted") {
    return false;
  }
  if (showWhileViewing || !tabVisible) {
    return true;
  }
  return !isWorkspaceBoardPath(pathname, factoryKey);
}

export function browserNotificationIconUrl(): string {
  return new URL("/favicon.ico", window.location.origin).href;
}

export function raiseUserBrowserNotification(payload: UserNotificationPayload, navigate: (path: string) => void): void {
  if (!payload.title || typeof Notification === "undefined") {
    return;
  }
  const notification = new Notification(payload.title, {
    body: payload.body,
    icon: browserNotificationIconUrl(),
    tag: payload.orderKey,
  });
  notification.onclick = () => {
    window.focus();
    if (payload.urlPath) {
      navigate(payload.urlPath);
    }
    notification.close();
  };
}
