import { AlertTriangle, Factory, PauseCircle, ShieldCheck } from "lucide-react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";

import { FactoryAutomationsTab } from "./FactoryAutomationsTab";
import { FactoryOverviewTab } from "./FactoryOverviewTab";
import { FactoryVelocityTab } from "./FactoryVelocityTab";
import { FactoryWorkOrdersTab } from "./FactoryWorkOrdersTab";
import { factoryPageWidthClassName, mutedTextClassName } from "./factoryStyles";
import type { Automation, SoftwareFactory, SoftwareFactoryPageData, WorkOrder } from "./factoryTypes";

/** PRD: Overview is the default of four tabs. */
export type FactoryTab = "overview" | "work-orders" | "automations" | "velocity";

const TABS: { value: FactoryTab; label: string }[] = [
  { value: "overview", label: "Overview" },
  { value: "work-orders", label: "Work Orders" },
  { value: "automations", label: "Automations" },
  { value: "velocity", label: "Velocity" },
];

/**
 * The shared TabsTrigger renders inactive tabs with `dark:text-muted-foreground`,
 * which measures 4.38:1 on the dark surface and fails WCAG AA. Corrected here
 * rather than in the shared component, whose blast radius is every Tabs usage.
 */
const inactiveTabContrastClassName = "dark:data-[state=inactive]:text-gray-300";

interface SoftwareFactoryPageProps {
  data: SoftwareFactoryPageData;
  /** Uncontrolled default; the real page will read this from the route. */
  defaultTab?: FactoryTab;
  onOpenWorkOrder: (workOrder: WorkOrder) => void;
  onOpenAutomation: (automation: Automation) => void;
  onCreateWorkOrder: () => void;
  onCreateAutomation: () => void;
  onSelectRepository: (repository: string) => void;
}

/**
 * Dedicated page for a Software Factory.
 *
 * The PRD is emphatic that a Factory is not a renamed App, so this is its own
 * page shell rather than a reuse of the App detail experience: Factory header,
 * then the four tabs — Overview, Work Orders, Automations, Velocity.
 */
export function SoftwareFactoryPage({
  data,
  defaultTab = "overview",
  onOpenWorkOrder,
  onOpenAutomation,
  onCreateWorkOrder,
  onCreateAutomation,
  onSelectRepository,
}: SoftwareFactoryPageProps) {
  const [tab, setTab] = useState<FactoryTab>(defaultTab);
  const { factory, summary, workOrders, automations, velocity } = data;

  return (
    <div className={cn(factoryPageWidthClassName, "py-8")}>
      <FactoryHeader factory={factory} onCreateWorkOrder={onCreateWorkOrder} />

      <Tabs value={tab} onValueChange={(value) => setTab(value as FactoryTab)} className="mt-6">
        <TabsList>
          {TABS.map((entry) => (
            <TabsTrigger key={entry.value} value={entry.value} className={inactiveTabContrastClassName}>
              {entry.label}
            </TabsTrigger>
          ))}
        </TabsList>

        <TabsContent value="overview" className="mt-6">
          <FactoryOverviewTab
            summary={summary}
            workOrders={workOrders}
            onOpenWorkOrder={onOpenWorkOrder}
            onSeeAllWorkOrders={() => setTab("work-orders")}
          />
        </TabsContent>
        <TabsContent value="work-orders" className="mt-6">
          <FactoryWorkOrdersTab workOrders={workOrders} onOpenWorkOrder={onOpenWorkOrder} />
        </TabsContent>
        <TabsContent value="automations" className="mt-6">
          <FactoryAutomationsTab
            automations={automations}
            onOpenAutomation={onOpenAutomation}
            onCreateAutomation={onCreateAutomation}
          />
        </TabsContent>
        <TabsContent value="velocity" className="mt-6">
          <FactoryVelocityTab velocity={velocity} onSelectRepository={onSelectRepository} />
        </TabsContent>
      </Tabs>
    </div>
  );
}

const STATUS_META = {
  healthy: {
    label: "Healthy",
    icon: ShieldCheck,
    className: "bg-emerald-100 text-emerald-800 dark:bg-emerald-950/70 dark:text-emerald-300",
  },
  degraded: {
    label: "Needs attention",
    icon: AlertTriangle,
    className: "bg-amber-100 text-amber-900 dark:bg-amber-950/70 dark:text-amber-300",
  },
  paused: {
    label: "Paused",
    icon: PauseCircle,
    className: "bg-slate-200 text-gray-700 dark:bg-slate-900 dark:text-gray-300",
  },
} as const satisfies Record<SoftwareFactory["status"], { label: string; icon: typeof ShieldCheck; className: string }>;

/** PRD: name and optional description sit above the tab navigation. */
function FactoryHeader({ factory, onCreateWorkOrder }: { factory: SoftwareFactory; onCreateWorkOrder: () => void }) {
  const status = STATUS_META[factory.status];
  const StatusIcon = status.icon;

  return (
    <header>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2.5">
            {/* The Factory badge is what distinguishes this from an App at a glance. */}
            <span
              aria-hidden
              className="flex size-8 items-center justify-center rounded-md bg-slate-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300"
            >
              <Factory className="size-4" />
            </span>
            <h1 className="text-2xl font-medium text-slate-900 dark:text-gray-100">{factory.name}</h1>
            <span
              className={cn(
                "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
                status.className,
              )}
            >
              <StatusIcon className="size-3.5" aria-hidden />
              {status.label}
            </span>
          </div>
          {factory.description && (
            <p className={cn("mt-2 max-w-3xl text-sm leading-normal", mutedTextClassName)}>{factory.description}</p>
          )}
          <p className={cn("mt-2 text-xs", mutedTextClassName)}>
            Software Factory · {factory.automationCount} {factory.automationCount === 1 ? "Automation" : "Automations"}
          </p>
        </div>
        <Button type="button" onClick={onCreateWorkOrder}>
          New Work Order
        </Button>
      </div>

      {factory.status !== "healthy" && factory.statusDetail && (
        <div
          className={cn(
            "mt-5 flex items-center gap-2 rounded-lg px-4 py-3 text-sm",
            "bg-amber-50 text-amber-900 outline outline-amber-200",
            "dark:bg-amber-950/40 dark:text-amber-200 dark:outline-amber-900/60",
          )}
        >
          <AlertTriangle className="size-4 shrink-0" aria-hidden />
          {factory.statusDetail}
        </div>
      )}
    </header>
  );
}
