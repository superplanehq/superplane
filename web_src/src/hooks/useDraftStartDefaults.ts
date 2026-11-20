import { useState } from "react";
import type { CanvasesCanvas, FactoriesFactory } from "@/api-client";
import { useFactory } from "./useFactoryData";
import { useCanvas } from "./useCanvasData";
import { primaryAgentNode } from "@/pages/factories/lib/columnCanvasAgent";
import { IMPLEMENT_AGENT_NODE_ID } from "@/pages/factories/lib/columnAutomationHeadline";
import type { DraftStartSelection } from "@/pages/factories/pages/work-order-split-run/draftStartModel";

export function useDraftStartDefaults(organizationId?: string, factoryId?: string, lineName?: string, lineId?: string) {
  const factory = useFactory(organizationId ?? "", factoryId ?? "");
  const appId = implementationAppId(factory.data, lineName, lineId);
  const canvas = useCanvas(organizationId ?? "", appId ?? "");
  return {
    values: savedAgentDefaults(canvas.data),
    isLoading: factory.isPending || Boolean(appId && canvas.isPending),
    isError: factory.isError || canvas.isError || Boolean(factory.data && !appId),
    retry: () => {
      void factory.refetch();
      void canvas.refetch();
    },
  };
}

function implementationAppId(factory: FactoriesFactory | undefined, lineName?: string, lineId?: string) {
  const line = factory?.lines?.find((item) => (lineId ? item.id === lineId : item.name === lineName));
  const steps = line?.steps ?? [];
  return (
    steps.find((step) => step.app?.entrypoint === "start-implementation")?.app?.app ??
    steps.find((step) => step.app?.app)?.app?.app
  );
}

function savedAgentDefaults(canvas?: CanvasesCanvas): DraftStartSelection | undefined {
  if (!canvas) return undefined;
  const configuration = primaryAgentNode(canvas.spec, IMPLEMENT_AGENT_NODE_ID)?.configuration;
  return {
    model: typeof configuration?.model === "string" ? configuration.model : "auto",
    thinkingLevel: typeof configuration?.thinkingLevel === "string" ? configuration.thinkingLevel : "",
  };
}

export function draftStartDefaultsStatus(defaults: DraftStartDefaults): string {
  return defaults.isError ? "Implementation defaults unavailable" : "Loading Implementation defaults";
}

export type DraftStartDefaults = ReturnType<typeof useDraftStartDefaults>;

export function useDraftStartSelection(identity: string) {
  const [state, setState] = useState<{ identity: string; selection: DraftStartSelection }>({ identity, selection: {} });
  if (state.identity !== identity) {
    setState({ identity, selection: {} });
  }
  return {
    selection: state.identity === identity ? state.selection : {},
    onChange: (selection: DraftStartSelection) => setState({ identity, selection }),
  };
}
