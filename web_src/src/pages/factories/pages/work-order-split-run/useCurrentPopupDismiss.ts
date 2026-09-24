import { useCallback, useEffect, useRef } from "react";

export function useCurrentPopupDismiss(orderId: string | undefined, onClose: (() => void) | undefined) {
  const activeOrderId = useRef(orderId);
  const mounted = useRef(false);

  useEffect(() => {
    activeOrderId.current = orderId;
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, [orderId]);

  return useCallback(() => {
    if (!mounted.current || activeOrderId.current !== orderId) {
      return;
    }
    onClose?.();
  }, [onClose, orderId]);
}
