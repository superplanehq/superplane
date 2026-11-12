import type { ComponentBaseProps } from "@/ui/componentBase";
import { expandMergeConfidenceSteps } from "./mergeConfidenceSteps";
import type { IntakeAutomationGraph } from "./useIntakeAutomationCanvas";

/** Apply display expansion to the selected version, including historical runs. */
export function prepareMergeConfidenceDisplayGraph(graph: IntakeAutomationGraph): IntakeAutomationGraph {
  return {
    ...graph,
    nodes: graph.nodes.map((node) => {
      const component = node.data.component as ComponentBaseProps | undefined;
      if (!component?.factorySteps) return node;
      return {
        ...node,
        data: {
          ...node.data,
          component: { ...component, factorySteps: expandMergeConfidenceSteps(component.factorySteps) },
        },
      };
    }),
  };
}
