import { usePermissions } from "@/contexts/usePermissions";
import {
  currentBrowserNotificationPermission,
  parseUserNotificationEvent,
  raiseUserBrowserNotification,
  shouldRaiseBrowserNotification,
} from "@/lib/browserNotifications";
import { useWebSocket } from "@/lib/reactUseWebsocket";
import { useCallback } from "react";
import { useLocation, useNavigate } from "react-router";

const SOCKET_SERVER_URL = `${window.location.protocol === "https:" ? "wss:" : "ws:"}//${window.location.host}/ws/users/notifications`;

export function UserNotificationsListener({ organizationId }: { organizationId: string }) {
  useUserNotificationsWebsocket(organizationId);
  return null;
}

export function useUserNotificationsWebsocket(organizationId: string): void {
  const { currentUserId, browserNotificationPreferences } = usePermissions();
  const location = useLocation();
  const navigate = useNavigate();
  const enabled = Boolean(currentUserId && organizationId && browserNotificationPreferences.enabled);

  const onMessage = useCallback(
    (event: MessageEvent<unknown>) => {
      const data = parseUserNotificationEvent(event);
      if (!data || data.event !== "user_notification" || !data.payload) {
        return;
      }
      const payload = data.payload;
      if (
        !shouldRaiseBrowserNotification({
          permission: currentBrowserNotificationPermission(),
          showWhileViewing: browserNotificationPreferences.showWhileViewing,
          tabVisible: document.visibilityState === "visible",
          pathname: location.pathname,
          factoryKey: payload.factoryKey ?? "",
        })
      ) {
        return;
      }
      if (!payload.title) {
        return;
      }
      raiseUserBrowserNotification(payload, navigate);
    },
    [browserNotificationPreferences.showWhileViewing, location.pathname, navigate],
  );

  const url = organizationId ? `${SOCKET_SERVER_URL}?organization_id=${organizationId}` : null;

  useWebSocket(
    url,
    {
      shouldReconnect: () => true,
      reconnectAttempts: Number.POSITIVE_INFINITY,
      reconnectInterval: 3000,
      heartbeat: false,
      share: false,
      onMessage,
    },
    enabled && url !== null,
  );
}
