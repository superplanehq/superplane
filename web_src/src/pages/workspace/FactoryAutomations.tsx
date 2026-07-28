import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Bot, ChevronRight, CirclePause, CirclePlay, Clock3, GitFork, Plus } from "lucide-react";

import type { AutomationStatus, FactoryAutomation } from "./types";

interface FactoryAutomationsProps {
  automations: FactoryAutomation[];
  onOpenAutomation: (automation: FactoryAutomation) => void;
}

const statusPresentation: Record<AutomationStatus, { label: string; className: string; icon: typeof CirclePlay }> = {
  active: {
    label: "Active",
    className:
      "border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-800 dark:bg-emerald-950/40 dark:text-emerald-300",
    icon: CirclePlay,
  },
  paused: {
    label: "Paused",
    className:
      "border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-800 dark:bg-amber-950/40 dark:text-amber-300",
    icon: CirclePause,
  },
  draft: {
    label: "Draft",
    className: "border-slate-200 bg-slate-50 text-slate-600 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300",
    icon: GitFork,
  },
};

export function FactoryAutomations({ automations, onOpenAutomation }: FactoryAutomationsProps) {
  return (
    <section className="overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70 dark:bg-gray-900">
      <div className="flex min-h-16 items-center justify-between gap-4 border-b border-slate-200 px-4 py-3 sm:px-5 dark:border-gray-700/70">
        <div className="min-w-0">
          <h2 className="text-sm font-semibold text-slate-900 dark:text-gray-100">Factory automations</h2>
          <p className="mt-0.5 text-xs text-slate-500 dark:text-gray-400">
            Canvas workflows that create, route, and complete work orders.
          </p>
        </div>
        <Button type="button" size="sm">
          <Plus />
          New automation
        </Button>
      </div>

      {automations.length === 0 ? (
        <div className="flex min-h-56 flex-col items-center justify-center px-6 text-center">
          <span className="flex size-10 items-center justify-center rounded-md bg-slate-100 text-slate-600 dark:bg-gray-800 dark:text-gray-300">
            <Bot className="size-5" />
          </span>
          <h3 className="mt-3 text-sm font-medium text-slate-900 dark:text-gray-100">No automations yet</h3>
          <p className="mt-1 max-w-sm text-xs text-slate-500 dark:text-gray-400">
            Start from a template, build with AI, or create a blank Canvas.
          </p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[900px] border-collapse text-left">
            <thead>
              <tr className="border-b border-slate-200 text-[11px] font-medium text-slate-500 dark:border-gray-700/70 dark:text-gray-400">
                <th scope="col" className="w-[31%] px-4 py-2 font-medium sm:px-5">
                  Automation
                </th>
                <th scope="col" className="w-[23%] px-3 py-2 font-medium">
                  Starts when
                </th>
                <th scope="col" className="w-[12%] px-3 py-2 font-medium">
                  Status
                </th>
                <th scope="col" className="w-[12%] px-3 py-2 text-right font-medium">
                  Active work
                </th>
                <th scope="col" className="w-[10%] px-3 py-2 text-right font-medium">
                  Success
                </th>
                <th scope="col" className="w-[12%] px-4 py-2 text-right font-medium sm:px-5">
                  Last run
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200 dark:divide-gray-700/70">
              {automations.map((automation) => (
                <AutomationRow key={automation.id} automation={automation} onOpen={onOpenAutomation} />
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function AutomationRow({
  automation,
  onOpen,
}: {
  automation: FactoryAutomation;
  onOpen: (automation: FactoryAutomation) => void;
}) {
  const presentation = statusPresentation[automation.status];
  const StatusIcon = presentation.icon;

  return (
    <tr className="group hover:bg-slate-50 dark:hover:bg-gray-800/60">
      <th scope="row" className="px-4 py-3.5 font-normal sm:px-5">
        <button
          type="button"
          onClick={() => onOpen(automation)}
          className="flex w-full min-w-0 items-start gap-3 text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-400"
        >
          <span className="flex size-8 shrink-0 items-center justify-center rounded-md border border-slate-200 bg-slate-50 text-slate-600 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-300">
            <GitFork className="size-4" />
          </span>
          <span className="min-w-0 flex-1">
            <span className="flex items-center gap-2">
              <span className="truncate text-xs font-medium text-slate-900 dark:text-gray-100">{automation.name}</span>
              <ChevronRight className="size-3.5 shrink-0 text-slate-400 opacity-0 transition-all group-hover:translate-x-0.5 group-hover:opacity-100" />
            </span>
            <span className="mt-0.5 block truncate text-[11px] font-normal text-slate-500 dark:text-gray-400">
              {automation.description}
            </span>
          </span>
        </button>
      </th>
      <td className="px-3 py-3.5 text-xs text-slate-600 dark:text-gray-400">{automation.trigger}</td>
      <td className="px-3 py-3.5">
        <Badge variant="outline" className={cn("text-[10px]", presentation.className)}>
          <StatusIcon />
          {presentation.label}
        </Badge>
      </td>
      <td className="px-3 py-3.5 text-right text-xs text-slate-700 tabular-nums dark:text-gray-300">
        {automation.activeWorkItems}
      </td>
      <td className="px-3 py-3.5 text-right text-xs text-slate-700 tabular-nums dark:text-gray-300">
        {automation.successRate}
      </td>
      <td className="px-4 py-3.5 text-right text-xs text-slate-500 sm:px-5 dark:text-gray-400">
        <span className="inline-flex items-center gap-1">
          <Clock3 className="size-3" />
          {automation.lastRun}
        </span>
      </td>
    </tr>
  );
}
