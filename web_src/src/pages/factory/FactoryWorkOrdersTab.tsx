import { CheckCircle2 } from "lucide-react";

import { EmptyState } from "@/ui/emptyState";

import { countPillClassName, factoryPanelClassName, sectionTitleClassName } from "./factoryStyles";
import type { WorkOrder, WorkOrderGroup } from "./factoryTypes";
import { workOrderGroup } from "./factoryTypes";
import { WorkOrderListItem } from "./WorkOrderListItem";

interface FactoryWorkOrdersTabProps {
  workOrders: WorkOrder[];
  onOpenWorkOrder: (workOrder: WorkOrder) => void;
}

/**
 * PRD: the complete operational queue, grouped into Needs attention → Running →
 * Recently done → Unsuccessful. The order is fixed: it runs from "blocked on a
 * human" to "finished", so the top of the page is always the actionable end.
 */
const GROUPS: { id: WorkOrderGroup; title: string; empty: string }[] = [
  {
    id: "needs-attention",
    title: "Needs attention",
    empty: "Nothing is waiting on a decision, approval, or clarification.",
  },
  { id: "running", title: "Running", empty: "No Work Orders are moving through Automations." },
  { id: "recently-done", title: "Recently done", empty: "No Work Orders have been marked successful yet." },
  { id: "unsuccessful", title: "Unsuccessful", empty: "No Work Orders have been marked unsuccessful." },
];

export function FactoryWorkOrdersTab({ workOrders, onOpenWorkOrder }: FactoryWorkOrdersTabProps) {
  return (
    <div className="flex flex-col gap-6">
      {GROUPS.map((group) => {
        const items = workOrders.filter((workOrder) => workOrderGroup(workOrder) === group.id);
        return (
          <section key={group.id} className={factoryPanelClassName}>
            <div className="mb-3 flex items-center gap-2">
              <h2 className={sectionTitleClassName}>{group.title}</h2>
              {items.length > 0 && <span className={countPillClassName}>{items.length}</span>}
            </div>
            {items.length === 0 ? (
              <EmptyState
                compact
                tone={group.id === "needs-attention" ? "neutral" : "accent"}
                icon={CheckCircle2}
                title="Nothing here"
                description={group.empty}
              />
            ) : (
              <ul className="flex flex-col gap-2.5">
                {items.map((workOrder) => (
                  <WorkOrderListItem key={workOrder.id} workOrder={workOrder} onOpen={onOpenWorkOrder} />
                ))}
              </ul>
            )}
          </section>
        );
      })}
    </div>
  );
}
