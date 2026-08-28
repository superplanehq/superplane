import { useEffect, useRef, type MutableRefObject } from "react";
import {
  FACTORY_CONFIGURE_FIT_SETTLE_MS,
  factoryConfigureEnterFitViewOptions,
  shouldFitFactoryConfigureEnter,
} from "./factoryConfigureFitView";

type Viewport = { x: number; y: number; zoom: number };

type UseFactoryConfigureFitViewInput = {
  factoryConfigure: boolean;
  isEditing: boolean;
  hasReactFlowInitialized: boolean;
  nodeCount: number;
  getNodeCount: () => number;
  getFocusNode: () => { id: string } | undefined;
  fitView: (options: Record<string, unknown>) => Promise<unknown>;
  getViewport: () => Viewport;
  viewportRef: MutableRefObject<Viewport | undefined>;
  reportZoom: (zoom: number) => void;
};

/** Fit the factory automation canvas once when Edit opens Configure. */
export function useFactoryConfigureFitView({
  factoryConfigure,
  isEditing,
  hasReactFlowInitialized,
  nodeCount,
  getNodeCount,
  getFocusNode,
  fitView,
  getViewport,
  viewportRef,
  reportZoom,
}: UseFactoryConfigureFitViewInput) {
  const fittedThisVisitRef = useRef(false);

  useEffect(() => {
    if (factoryConfigure) {
      return;
    }
    fittedThisVisitRef.current = false;
  }, [factoryConfigure]);

  useEffect(() => {
    if (
      !shouldFitFactoryConfigureEnter({
        factoryConfigure,
        isEditing,
        hasReactFlowInitialized,
        hasFittedThisVisit: fittedThisVisitRef.current,
        nodeCount,
      })
    ) {
      return;
    }

    const timeoutId = window.setTimeout(() => {
      if (fittedThisVisitRef.current) {
        return;
      }
      if (getNodeCount() === 0) {
        return;
      }
      fittedThisVisitRef.current = true;
      void fitView(factoryConfigureEnterFitViewOptions(getFocusNode())).then(
        () => {
          const nextViewport = getViewport();
          viewportRef.current = nextViewport;
          reportZoom(nextViewport.zoom);
        },
        () => undefined,
      );
    }, FACTORY_CONFIGURE_FIT_SETTLE_MS);

    return () => window.clearTimeout(timeoutId);
  }, [
    factoryConfigure,
    fitView,
    getFocusNode,
    getNodeCount,
    getViewport,
    hasReactFlowInitialized,
    isEditing,
    nodeCount,
    reportZoom,
    viewportRef,
  ]);
}
