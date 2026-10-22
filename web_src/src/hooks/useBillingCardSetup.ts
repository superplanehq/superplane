import { useEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { organizationsCreatePaymentMethodSession } from "@/api-client";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

export function useBillingCardSetup(organizationId: string) {
  const [error, setError] = useState("");
  const [popup, setPopup] = useState<Window | null>(null);
  const activeWindow = useRef<Window | null>(null);
  const { refetch } = useOrganizationBilling(organizationId);
  const session = useMutation({
    mutationFn: async (opened: Window) => {
      setError("");
      const response = await organizationsCreatePaymentMethodSession(
        withOrganizationHeader({ organizationId, path: { id: organizationId }, body: {} }),
      );
      if (!response.data?.portalUrl) throw new Error("Could not open card setup. Try again.");
      const portal = new URL(response.data.portalUrl);
      portal.pathname = portal.pathname.replace(/\/portal(?:\/.*)?$/, "/portal/settings");
      if (!opened.closed) opened.location.replace(portal.toString());
      return opened;
    },
    onSuccess: (opened) => {
      if (!opened.closed) setPopup(opened);
    },
    onError: (cause, opened) => {
      opened.close();
      setError(getApiErrorMessage(cause, "Could not open card setup. Try again."));
    },
  });

  useEffect(() => () => activeWindow.current?.close(), []);

  useEffect(() => {
    if (!popup) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;
    const check = async () => {
      if (popup.closed) {
        setPopup(null);
        return;
      }
      const result = await refetch();
      if (cancelled) return;
      if (result.data?.hasPaymentMethod) {
        setPopup(null);
        return;
      }
      if (result.error) setError("Could not verify the card. Close the Polar window and try again.");
      timer = setTimeout(() => void check(), 2000);
    };
    void check();
    return () => {
      cancelled = true;
      clearTimeout(timer);
      popup.close();
      window.focus();
    };
  }, [popup, refetch]);

  const open = () => {
    const opened = window.open("about:blank", "_blank");
    if (!opened) {
      setError("Allow pop-ups for SuperPlane, then try again.");
      return;
    }
    opened.opener = null;
    activeWindow.current = opened;
    session.mutate(opened);
  };

  return {
    open,
    error,
    isPending: session.isPending || Boolean(popup),
    opening: session.isPending,
    waitingForCard: Boolean(popup),
  };
}
