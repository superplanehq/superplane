import { useEffect, useRef, useState } from "react";

type UseFactoryConfigureInitialLayoutOptions = {
  factoryAutoLayout: boolean;
  isEditing: boolean;
  editBootstrapReady: boolean;
  activeCanvasVersionId: string;
  applyLayout: () => Promise<unknown>;
};

/** Apply the mandatory layout once after each Factory Configure session becomes ready. */
export function useFactoryConfigureInitialLayout({
  factoryAutoLayout,
  isEditing,
  editBootstrapReady,
  activeCanvasVersionId,
  applyLayout,
}: UseFactoryConfigureInitialLayoutOptions): { ready: boolean } {
  const [ready, setReady] = useState(!factoryAutoLayout);
  const layoutAppliedRef = useRef(false);
  const applyLayoutRef = useRef(applyLayout);
  applyLayoutRef.current = applyLayout;

  useEffect(() => {
    if (factoryAutoLayout) {
      setReady(false);
      return;
    }
    layoutAppliedRef.current = false;
    setReady(true);
  }, [factoryAutoLayout]);

  useEffect(() => {
    if (!factoryAutoLayout) {
      return;
    }
    if (!isEditing || !editBootstrapReady) {
      return;
    }
    if (!activeCanvasVersionId) {
      setReady(true);
      return;
    }
    if (layoutAppliedRef.current) {
      return;
    }

    layoutAppliedRef.current = true;
    void Promise.resolve(applyLayoutRef.current()).finally(() => {
      setReady(true);
    });
  }, [activeCanvasVersionId, editBootstrapReady, factoryAutoLayout, isEditing]);

  return { ready };
}
