import { AlertTriangle, ArrowRight, CircleDot, PauseCircle } from "lucide-react";

import { Button } from "@/components/ui/button";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { EmptyState } from "@/ui/emptyState";

import {
  factoryCardClassName,
  factoryPanelClassName,
  mutedTextClassName,
  sectionTitleClassName,
} from "./factoryStyles";
import type { Automation } from "./factoryTypes";

const STATUS_META = {
  active: { label: "Active", icon: CircleDot, className: "text-emerald-700 dark:text-emerald-400" },
  paused: { label: "Paused", icon: PauseCircle, className: "text-gray-600 dark:text-gray-300" },
  error: { label: "Error", icon: AlertTriangle, className: "text-red-700 dark:text-red-400" },
} as const satisfies Record<Automation["status"], { label: string; icon: typeof CircleDot; className: string }>;

interface FactoryAutomationsTabProps {
  automations: Automation[];
  /** PRD: opening an Automation opens the existing Canvas editing experience. */
  onOpenAutomation: (automation: Automation) => void;
  onCreateAutomation: () => void;
}

/**
 * PRD: a finite operational list, not a canvas gallery. Each row carries enough
 * context to pick the right Automation — name, description, trigger, status,
 * current activity, recent success, last run.
 */
export function FactoryAutomationsTab({
  automations,
  onOpenAutomation,
  onCreateAutomation,
}: FactoryAutomationsTabProps) {
  return (
    <section className={factoryPanelClassName}>
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <h2 className={sectionTitleClassName}>Automations</h2>
        <Button type="button" size="sm" onClick={onCreateAutomation}>
          New Automation
        </Button>
      </div>

      {automations.length === 0 ? (
        <EmptyState
          title="No Automations yet"
          description="A Factory starts empty. Add an Automation to give it something to run."
        />
      ) : (
        <ul className="flex flex-col gap-2.5">
          {automations.map((automation) => {
            const status = STATUS_META[automation.status];
            const StatusIcon = status.icon;
            return (
              <li key={automation.id} className={cn("px-4 py-3", factoryCardClassName)}>
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2.5">
                      <button
                        type="button"
                        onClick={() => onOpenAutomation(automation)}
                        className="text-left text-sm font-medium text-slate-900 underline-offset-4 hover:underline dark:text-gray-100"
                      >
                        {automation.name}
                      </button>
                      <span className={cn("inline-flex items-center gap-1 text-xs font-medium", status.className)}>
                        <StatusIcon className="size-3.5" aria-hidden />
                        {status.label}
                      </span>
                    </div>
                    <p className={cn("mt-1 text-sm", mutedTextClassName)}>{automation.description}</p>
                    {automation.currentActivity && (
                      <p className="mt-1 text-sm text-blue-800 dark:text-blue-300">{automation.currentActivity}</p>
                    )}
                    <div
                      className={cn("mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs", mutedTextClassName)}
                    >
                      <span>Trigger: {automation.trigger}</span>
                      <span>{Math.round(automation.recentSuccess * 100)}% recent success</span>
                      <span>Last run {formatTimeAgo(new Date(automation.lastRunAt))}</span>
                      {/* PRD: Automations in one Factory may span repositories. */}
                      <span>{automation.repositories.join(", ")}</span>
                    </div>
                  </div>
                  <Button type="button" size="sm" variant="outline" onClick={() => onOpenAutomation(automation)}>
                    Open canvas
                    <ArrowRight />
                  </Button>
                </div>
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
