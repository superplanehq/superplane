import type { Meta, StoryObj } from "@storybook/react-vite";
import { Copy, Ellipsis, Funnel, Pencil, Plus, RefreshCw, Search, Settings2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { SegmentedNav } from "@/ui/SegmentedNav";
import { ComponentStoryShell } from "../__fixtures__/ComponentStoryShell";
import { withFactoriesTheme } from "../__fixtures__/factoriesStoryTheme";
import {
  EMPTY_FACTORY,
  FACTORIES_ORGANIZATION_ID,
  REFUND_FACTORY,
  REFUND_LINE_PLAN_ID,
} from "../__fixtures__/factoryPageResponses";
import { factorySectionHeaderClassName } from "../pages/factoryPageLayoutStyles";
import { FactoriesSidebarNav } from "./FactoriesSidebarNav";
import { WorkspacePageHeader } from "./WorkspacePageHeader";
import { WorkspaceSwitcher } from "./WorkspaceSwitcher";

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

const sectionHeader = { className: factorySectionHeaderClassName };

export const SectionTitleOnly: Story = {
  args: {
    ...sectionHeader,
    title: "Overview",
  },
};

export const SectionWithSubtitle: Story = {
  args: {
    ...sectionHeader,
    title: "Overview",
    subtitle: "Your workspace at a glance. Content for this page comes next.",
  },
};

export const SectionWithPrimaryAction: Story = {
  args: {
    ...sectionHeader,
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

export const SectionWithPeriodPills: Story = {
  args: {
    ...sectionHeader,
    title: "Velocity",
    subtitle: "Merged pull requests from SuperPlane, waste, and cost.",
    actions: (
      <SegmentedNav
        ariaLabel="Velocity period in days"
        size="xs"
        value="7"
        onValueChange={() => undefined}
        options={[
          { value: "7", label: "7d" },
          { value: "30", label: "30d" },
        ]}
      />
    ),
  },
};

export const SectionWithToolbarAndChips: Story = {
  args: {
    ...sectionHeader,
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
    ...sectionHeader,
    title: "Wiki",
    subtitle: "Shared product context — intent, architecture, and delivery notes for people and Planner.",
    actions: (
      <Button type="button" variant="outline" size="sm">
        <RefreshCw className="size-3.5" aria-hidden />
        Refresh knowledge
      </Button>
    ),
  },
};

/**
 * Settings and some entity pages still use the large centered title.
 * Section pages (Overview, Work Orders, Lines, Automations, Wiki, Velocity)
 * use the compact class instead.
 */
export const SettingsLargeTitle: Story = {
  args: {
    title: "General",
    subtitle: "Workspace name, key, and description.",
  },
};

export const EntityLineDetail: Story = {
  args: {
    ...sectionHeader,
    variant: "entity",
    backHref: "#",
    backLabel: "Lines",
    title: "Plan and Implement",
    subtitle: "3 phases",
    actions: (
      <Button type="button" variant="outline" size="sm">
        <Pencil className="size-3.5" aria-hidden />
        Edit
      </Button>
    ),
  },
};

export const EntityWorkOrder: Story = {
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

/** Compact section title lines up with the sidebar workspace control. */
export const AlignedWithSidebar: Story = {
  name: "Aligned with sidebar",
  render: () => (
    <div className="flex min-h-screen bg-background">
      <aside className="flex w-[var(--workspace-navigation-width)] shrink-0 flex-col items-center border-r border-sidebar-border bg-sidebar text-sidebar-foreground">
        <WorkspaceSwitcher
          organizationId={FACTORIES_ORGANIZATION_ID}
          factory={REFUND_FACTORY}
          factories={[REFUND_FACTORY, EMPTY_FACTORY]}
          canCreateFactory
          permissionsLoading={false}
          onCreateFactory={() => console.log("create workspace")}
        />
        <FactoriesSidebarNav
          organizationId={FACTORIES_ORGANIZATION_ID}
          factoryKey={REFUND_FACTORY.key!}
          lineId={REFUND_LINE_PLAN_ID}
          canOpenSettings
          canCreateWorkOrder
          permissionsLoading={false}
          onCreateWorkOrder={() => console.log("create work order")}
        />
      </aside>
      <div className="min-w-0 flex-1">
        <WorkspacePageHeader
          className={factorySectionHeaderClassName}
          title="Plan and Implement"
          subtitle="Refund Planner → Refund Implementer"
        />
      </div>
    </div>
  ),
};
