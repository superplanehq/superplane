import type { Meta, StoryObj } from "@storybook/react-vite";
import { useEffect, useRef, type ComponentProps } from "react";

import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { ColumnLaneMenu } from "./ColumnLaneMenu";

/**
 * Phase-column three-dots menu. Archive Step is last, after Set color.
 * Backlog, Verify, and Done omit the item. A one-step line also omits it.
 */
const meta = {
  title: "Factories/Components/ColumnLaneMenu",
  component: ColumnLaneMenu,
  parameters: { layout: "centered" },
  decorators: [
    (Story) => (
      <ComponentStoryShell className="flex min-h-[320px] items-start justify-end bg-gray-50 p-8 dark:bg-gray-950">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof ColumnLaneMenu>;

export default meta;

type Story = StoryObj<typeof meta>;

function OpenedColumnLaneMenu(props: ComponentProps<typeof ColumnLaneMenu>) {
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const id = window.setTimeout(() => {
      const button = containerRef.current?.querySelector<HTMLButtonElement>(`[data-testid="${props.testId}"]`);
      if (!button) {
        return;
      }
      const opts = { bubbles: true, cancelable: true };
      button.dispatchEvent(new PointerEvent("pointerdown", opts));
      button.dispatchEvent(new MouseEvent("mousedown", opts));
      button.dispatchEvent(new PointerEvent("pointerup", opts));
      button.dispatchEvent(new MouseEvent("mouseup", opts));
      button.dispatchEvent(new MouseEvent("click", opts));
    }, 50);
    return () => window.clearTimeout(id);
  }, [props.testId]);

  return (
    <div ref={containerRef}>
      <ColumnLaneMenu {...props} />
    </div>
  );
}

/** Phase menu with Archive Step last. */
export const PhaseWithArchive: Story = {
  args: {
    title: "Plan",
    testId: "lines-phase-menu-0",
    colorId: null,
    onColorChange: () => undefined,
  },
  render: () => (
    <OpenedColumnLaneMenu
      title="Plan"
      testId="lines-phase-menu-0"
      editHref="/plan-automation"
      editLabel="Edit Automation"
      onEditAgent={() => console.log("edit agent")}
      onSetParallelism={() => console.log("set parallelism")}
      onArchiveStep={() => console.log("archive step")}
      colorId="sky"
      onColorChange={(colorId) => console.log("color", colorId)}
    />
  ),
};

/** One remaining step: Archive Step is hidden. */
export const PhaseWithoutArchive: Story = {
  args: {
    title: "Implement",
    testId: "lines-phase-menu-0",
    colorId: null,
    onColorChange: () => undefined,
  },
  render: () => (
    <OpenedColumnLaneMenu
      title="Implement"
      testId="lines-phase-menu-0"
      editHref="/implement-automation"
      editLabel="Edit Automation"
      onSetParallelism={() => console.log("set parallelism")}
      colorId="teal"
      onColorChange={(colorId) => console.log("color", colorId)}
    />
  ),
};

/** Backlog never offers Archive Step. */
export const Backlog: Story = {
  args: {
    title: "Backlog",
    testId: "lines-backlog-menu",
    colorId: null,
    onColorChange: () => undefined,
  },
  render: () => (
    <OpenedColumnLaneMenu
      title="Backlog"
      testId="lines-backlog-menu"
      onEdit={() => console.log("edit backlog")}
      onAddIntake={() => console.log("add intake")}
      colorId="lime"
      onColorChange={(colorId) => console.log("color", colorId)}
    />
  ),
};
