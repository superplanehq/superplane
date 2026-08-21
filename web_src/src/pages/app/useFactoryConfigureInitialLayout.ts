import { useEffect, useRef } from "react";

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
}: UseFactoryConfigureInitialLayoutOptions) {
  const layoutAppliedRef = useRef(false);
  const applyLayoutRef = useRef(applyLayout);
  applyLayoutRef.current = applyLayout;

  useEffect(() => {
    if (factoryAutoLayout) {
      return;
    }
    layoutAppliedRef.current = false;
  }, [factoryAutoLayout]);

  useEffect(() => {
    if (!factoryAutoLayout || !isEditing || !editBootstrapReady || !activeCanvasVersionId) {
      return;
    }
    if (layoutAppliedRef.current) {
      return;
    }

    layoutAppliedRef.current = true;
    void applyLayoutRef.current();
  }, [activeCanvasVersionId, editBootstrapReady, factoryAutoLayout, isEditing]);
}
