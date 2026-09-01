import { useEffect, useRef, useState, type MutableRefObject } from "react";
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
  layoutReady: boolean;
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
  layoutReady,
  getNodeCount,
  getFocusNode,
  fitView,
  getViewport,
  viewportRef,
  reportZoom,
}: UseFactoryConfigureFitViewInput): { ready: boolean } {
  const fittedThisVisitRef = useRef(false);
  const factoryConfigureRef = useRef(factoryConfigure);
  const [ready, setReady] = useState(!factoryConfigure);

  if (factoryConfigure !== factoryConfigureRef.current) {
    factoryConfigureRef.current = factoryConfigure;
    fittedThisVisitRef.current = false;
    setReady(!factoryConfigure);
  }

  useEffect(() => {
    if (
      !shouldFitFactoryConfigureEnter({
        factoryConfigure,
        isEditing,
        hasReactFlowInitialized,
        hasFittedThisVisit: fittedThisVisitRef.current,
        nodeCount,
        layoutReady,
      })
    ) {
      if (factoryConfigure && isEditing && hasReactFlowInitialized && layoutReady && nodeCount === 0) {
        fittedThisVisitRef.current = true;
        setReady(true);
      }
      return;
    }

    const timeoutId = window.setTimeout(() => {
      if (fittedThisVisitRef.current) {
        return;
      }
      if (getNodeCount() === 0) {
        fittedThisVisitRef.current = true;
        setReady(true);
        return;
      }
      fittedThisVisitRef.current = true;
      void fitView(factoryConfigureEnterFitViewOptions(getFocusNode())).finally(() => {
        const nextViewport = getViewport();
        viewportRef.current = nextViewport;
        reportZoom(nextViewport.zoom);
        setReady(true);
      });
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
    layoutReady,
    nodeCount,
    reportZoom,
    viewportRef,
  ]);

  return { ready };
}
