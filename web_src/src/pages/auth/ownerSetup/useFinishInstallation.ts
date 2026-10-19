import { useCallback, useRef, useState } from "react";

export function useFinishInstallation({
  organizationSlug,
  saveSetupProgress,
  onClosed,
}: {
  organizationSlug: string | null;
  saveSetupProgress: (step: string) => Promise<boolean>;
  onClosed: (organizationSlug: string) => void;
}) {
  const closingSetup = useRef(false);
  const [closeError, setCloseError] = useState(false);
  const [isClosing, setIsClosing] = useState(false);

  const finishInstallation = useCallback(() => {
    if (!organizationSlug || closingSetup.current) return;
    closingSetup.current = true;
    setIsClosing(true);
    setCloseError(false);
    void saveSetupProgress("").then((saved) => {
      closingSetup.current = false;
      setIsClosing(false);
      if (!saved) {
        setCloseError(true);
        return;
      }
      onClosed(organizationSlug);
    });
  }, [onClosed, organizationSlug, saveSetupProgress]);

  return { closeError, isClosing, finishInstallation };
}
