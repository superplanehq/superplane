import { useEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { organizationsCreatePaymentMethodSession } from "@/api-client";
import { useOrganizationBilling } from "@/hooks/useOrganizationBilling";
import { getApiErrorMessage } from "@/lib/errors";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";

/** Points a Polar portal session URL at the portal settings page, where payment methods are managed. */
export function polarPortalSettingsUrl(portalUrl: string): string {
  const portal = new URL(portalUrl);
  portal.pathname = portal.pathname.replace(/\/portal(?:\/.*)?$/, "/portal/settings");
  return portal.toString();
}

interface CardSetupPopup {
  opened: Window;
  hadPaymentMethod: boolean;
}

export function useBillingCardSetup(organizationId: string) {
  const [error, setError] = useState("");
  const [popup, setPopup] = useState<CardSetupPopup | null>(null);
  const activeWindow = useRef<Window | null>(null);
  const onCardSaved = useRef<(() => void) | undefined>(undefined);
  const { data: billing, refetch } = useOrganizationBilling(organizationId);
  const session = useMutation({
    retry: false,
    mutationFn: async (popup: CardSetupPopup) => {
      setError("");
      const response = await organizationsCreatePaymentMethodSession(
        withOrganizationHeader({ organizationId, path: { id: organizationId }, body: {} }),
      );
      if (!response.data?.portalUrl) throw new Error("Could not open card setup. Try again.");
      if (!popup.opened.closed) popup.opened.location.replace(polarPortalSettingsUrl(response.data.portalUrl));
      return popup;
    },
    onSuccess: setPopup,
    onError: (cause, popup) => {
      popup.opened.close();
      onCardSaved.current = undefined;
      setError(getApiErrorMessage(cause, "Could not open card setup. Try again."));
    },
  });

  useEffect(() => () => activeWindow.current?.close(), []);

  useEffect(() => {
    if (!popup) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout>;
    const check = async () => {
      if (cancelled) return;
      if (popup.hadPaymentMethod && !popup.opened.closed) {
        timer = setTimeout(() => void check(), 2000);
        return;
      }
      const result = await refetch();
      if (cancelled) return;
      if (result.data?.hasPaymentMethod) {
        const onSaved = onCardSaved.current;
        onCardSaved.current = undefined;
        popup.opened.close();
        window.focus();
        setPopup(null);
        onSaved?.();
        return;
      }
      if (result.error) setError("Could not verify the card. Close the Polar window and try again.");
      if (popup.opened.closed) {
        onCardSaved.current = undefined;
        setPopup(null);
        return;
      }
      timer = setTimeout(() => void check(), 2000);
    };
    void check();
    return () => {
      cancelled = true;
      clearTimeout(timer);
      popup.opened.close();
      window.focus();
    };
  }, [popup, refetch]);

  const open = (onSaved?: () => void) => {
    setError("");
    const width = 600;
    const height = 750;
    const left = Math.max(0, window.screenX + (window.outerWidth - width) / 2);
    const top = Math.max(0, window.screenY + (window.outerHeight - height) / 2);
    const opened = window.open(
      "about:blank",
      "_blank",
      `popup=yes,width=${width},height=${height},left=${Math.round(left)},top=${Math.round(top)}`,
    );
    if (!opened) {
      setError("Allow pop-ups for SuperPlane, then try again.");
      return;
    }
    onCardSaved.current = onSaved;
    opened.opener = null;
    activeWindow.current = opened;
    session.mutate({ opened, hadPaymentMethod: Boolean(billing?.hasPaymentMethod) });
  };

  return {
    open,
    error,
    isPending: session.isPending || Boolean(popup),
    opening: session.isPending,
    waitingForCard: Boolean(popup),
  };
}
