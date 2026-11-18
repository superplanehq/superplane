import { render, screen } from "@testing-library/react";
import { expect, it } from "bun:test";
import { ComponentBase, type ComponentBaseProps } from "@/ui/componentBase";
import type { CanvasNode } from "@/ui/CanvasPage";
import { configuredAgentSteps } from "@/lib/agentRunnerSteps";
import { prepareMergeConfidenceDisplayGraph } from "./mergeConfidenceDisplayGraph";

it("renders legacy checks from the selected graph and preserves saved step names", () => {
  const prompt = [
    "Report each enabled check with the report_merge_check tool.",
    "Enabled checks: risk, performance.",
    "Risk. Use this section only when risk is enabled.",
    "Check the blast radius.",
    "Performance. Use this section only when performance is enabled.",
    "Check hot paths.",
  ].join("\n");
  const component = {
    title: "Historical assessment",
    isFactoryApp: true,
    factorySteps: configuredAgentSteps({ steps: [{ name: "Review Pull Request", prompt }] }) ?? [],
  };
  const node: CanvasNode = { id: "agent", position: { x: 0, y: 0 }, data: { type: "component", component } };
  const graph = prepareMergeConfidenceDisplayGraph({
    nodes: [node],
    edges: [],
    specNodes: [{ id: "agent", configuration: { steps: [{ name: "Current custom name" }] } }],
  });
  const prepared = graph.nodes[0].data.component as ComponentBaseProps;
  const view = render(<ComponentBase {...component} {...prepared} />);
  expect(screen.getByText("Blast radius")).toBeInTheDocument();
  expect(screen.getByText("Performance")).toBeInTheDocument();
  expect(screen.queryByText("Current custom name")).not.toBeInTheDocument();

  view.rerender(<ComponentBase {...component} />);
  expect(screen.getByText("Review Pull Request")).toBeInTheDocument();
  expect(screen.queryByText("Blast radius")).not.toBeInTheDocument();

  const split = prepareMergeConfidenceDisplayGraph({
    nodes: [
      {
        ...node,
        data: {
          ...node.data,
          component: {
            ...component,
            factorySteps: [
              { name: "Saved custom label", type: "prompt", prompt: "Merge check: risk.\nAssess the diff." },
            ],
          },
        },
      },
    ],
    edges: [],
  });
  view.rerender(<ComponentBase {...component} {...(split.nodes[0].data.component as ComponentBaseProps)} />);
  expect(screen.getByText("Saved custom label")).toBeInTheDocument();
});
