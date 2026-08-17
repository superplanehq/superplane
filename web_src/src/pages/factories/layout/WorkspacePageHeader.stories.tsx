import type { Meta, StoryObj } from "@storybook/react-vite";
import { Copy, Ellipsis, Funnel, Pencil, Plus, RefreshCw, Search, Settings2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import { factoryWorkOrdersHeaderClassName } from "../pages/factoryPageLayoutStyles";
import { WorkspacePageHeader } from "./WorkspacePageHeader";

const meta = {
  title: "Factories/Layout/WorkspacePageHeader",
  component: WorkspacePageHeader,
  parameters: { layout: "fullscreen" },
  decorators: [
    withFactoriesTheme,
    (Story) => (
      <ComponentStoryShell className="min-h-screen bg-background">
        <Story />
      </ComponentStoryShell>
    ),
  ],
} satisfies Meta<typeof WorkspacePageHeader>;

export default meta;

type Story = StoryObj<typeof meta>;

export const SectionTitleOnly: Story = {
  args: {
    title: "Overview",
  },
};

export const SectionWithSubtitle: Story = {
  args: {
    title: "Overview",
    subtitle: "Your workspace at a glance.",
  },
};

export const SectionWithPrimaryAction: Story = {
  args: {
    title: "Lines",
    subtitle:
      "Factory lines specialize how work moves through the workspace. Each phase is backed by a canvas that runs work orders.",
    actions: (
      <Button type="button" size="sm">
        <Plus className="size-3.5" aria-hidden />
        New line
      </Button>
    ),
  },
};

export const SectionWithToolbarAndChips: Story = {
  args: {
    className: factoryWorkOrdersHeaderClassName,
    title: "Work Orders",
    leading: (
      <>
        <div className="flex items-center rounded-md border border-border p-0.5" role="group" aria-label="Scope">
          <span className="inline-flex h-7 items-center rounded-[5px] bg-accent px-2.5 text-[12px] font-medium text-foreground">
            All
          </span>
          <span className="inline-flex h-7 items-center rounded-[5px] px-2.5 text-[12px] font-medium text-muted-foreground">
            Active
          </span>
          <span className="inline-flex h-7 items-center rounded-[5px] px-2.5 text-[12px] font-medium text-muted-foreground">
            My
          </span>
        </div>
        <Button type="button" variant="ghost" size="sm" className="text-muted-foreground">
          <Funnel className="size-3.5" aria-hidden />
          Filter
        </Button>
      </>
    ),
    actions: (
      <>
        <Button type="button" variant="ghost" size="icon-xs" aria-label="Search work orders">
          <Search className="size-3.5" aria-hidden />
        </Button>
        <Button type="button" variant="ghost" size="sm" className="text-muted-foreground">
          <Settings2 className="size-3.5" aria-hidden />
          Display
        </Button>
        <Button type="button" size="sm">
          <Plus className="size-3.5" aria-hidden />
          New work order
        </Button>
      </>
    ),
    belowRow: (
      <div className="flex flex-wrap items-center gap-1.5 pt-1 text-[12px] text-muted-foreground">
        <span className="inline-flex h-7 items-center gap-1 rounded-md border border-border bg-background px-2">
          Status: Active
        </span>
        <span className="inline-flex h-7 items-center gap-1 rounded-md border border-border bg-background px-2">
          Owner: You
        </span>
      </div>
    ),
  },
};

export const SectionWithSecondaryAction: Story = {
  args: {
    title: "Wiki",
    subtitle: "Shared product context — intent, architecture, and delivery notes.",
    actions: (
      <Button type="button" variant="outline" size="sm">
        <RefreshCw className="size-3.5" aria-hidden />
        Refresh knowledge
      </Button>
    ),
  },
};

export const EntityWithKicker: Story = {
  args: {
    variant: "entity",
    backHref: "#",
    backLabel: "Work Orders",
    kicker: "SP-42",
    title: "Reconcile duplicate refunds in ledger",
    actions: (
      <>
        <Button type="button" variant="ghost" size="icon-xs" aria-label="Copy link">
          <Copy className="size-3.5" aria-hidden />
        </Button>
        <Button type="button" variant="ghost" size="icon-xs" aria-label="More actions">
          <Ellipsis className="size-3.5" aria-hidden />
        </Button>
      </>
    ),
  },
};

export const EntityWithSubtitleAndAction: Story = {
  args: {
    variant: "entity",
    backHref: "#",
    backLabel: "Lines",
    title: "Refunds daily sweep",
    subtitle: "3 phases",
    actions: (
      <Button type="button" variant="outline" size="sm">
        <Pencil className="size-3.5" aria-hidden />
        Edit
      </Button>
    ),
  },
};
