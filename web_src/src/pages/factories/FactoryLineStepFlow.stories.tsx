import type { Meta, StoryObj } from "@storybook/react-vite";

import {
  LineStepAddButton,
  LineStepArrow,
  LineStepDisplayNode,
  LineStepEditorShell,
  LineStepFlow,
} from "./FactoryLineStepFlow";

const meta = {
  title: "Factories/Components/FactoryLineStepFlow",
  parameters: { layout: "centered" },
} satisfies Meta;

export default meta;

type Story = StoryObj;

/** A single line step node — used by both the sidebar preview and the editor. */
export const StepNode: Story = {
  name: "LineStepDisplayNode",
  render: () => (
    <div className="w-64">
      <LineStepDisplayNode stepName="implement" appName="Refund Implementer" entrypoint="start-implementation" />
    </div>
  ),
};

/** The connector arrow between two step nodes. */
export const Arrow: Story = {
  name: "LineStepArrow",
  render: () => (
    <div className="flex flex-col items-center">
      <LineStepArrow />
    </div>
  ),
};

/** The dashed "add step" affordance below the last node. */
export const AddButton: Story = {
  name: "LineStepAddButton",
  render: () => <LineStepAddButton onClick={() => console.log("add step")} />,
};

/** The rounded editor shell that wraps in-place step editing controls. */
export const EditorShell: Story = {
  name: "LineStepEditorShell",
  render: () => (
    <LineStepEditorShell className="w-[420px]">
      <p className="text-sm text-gray-500 dark:text-gray-400">Editor controls slot in here.</p>
    </LineStepEditorShell>
  ),
};

/** Full flow composition: three steps + add button, matching the sidebar preview. */
export const CompactFlow: Story = {
  name: "LineStepFlow (compact)",
  render: () => (
    <LineStepFlow className="w-64">
      <LineStepDisplayNode stepName="plan" appName="Refund Planner" entrypoint="start-plan" />
      <LineStepArrow />
      <LineStepDisplayNode stepName="implement" appName="Refund Implementer" entrypoint="start-implementation" />
      <LineStepArrow />
      <LineStepDisplayNode stepName="verify" appName="Refund Verifier" entrypoint="start-verification" />
      <LineStepAddButton onClick={() => console.log("add step")} />
    </LineStepFlow>
  ),
};
