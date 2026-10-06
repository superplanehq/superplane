import { GitBranch, Info, LayoutGrid, Ticket, type LucideIcon } from "lucide-react";

import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { INTAKE_CATEGORIES, INTAKE_STATUSES, type IntakeCategory } from "@/lib/intakeCatalog";
import { intakeSurfaces, type IntakeSurface } from "@/lib/intakePresentation";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";

import { INTAKE_CATEGORY_LABELS, INTAKE_STATUS_INFO, INTAKE_SURFACE_LABELS } from "./intakeCatalogModel";
import { IntakeStatusPill } from "./IntakeStatusPill";

export function StatusGuide() {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          className="flex items-center gap-1 text-sm font-medium text-slate-600 hover:text-slate-900 dark:text-gray-300 dark:hover:text-gray-100"
        >
          <Info size={14} />
          About statuses
        </button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[34rem] p-0">
        <div className="border-b border-slate-200 px-4 py-3 dark:border-gray-700">
          <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">Statuses</p>
          <p className="text-xs text-slate-500 dark:text-gray-400">
            Each intake moves from Planned to Generally available.
          </p>
        </div>
        <table className="w-full text-left text-xs">
          <thead className="text-slate-500 dark:text-gray-400">
            <tr>
              <th className="px-4 py-2 font-medium">Status</th>
              <th className="px-4 py-2 font-medium">Meaning</th>
            </tr>
          </thead>
          <tbody>
            {INTAKE_STATUSES.map((status) => (
              <tr key={status} className="border-t border-slate-100 align-top dark:border-gray-800">
                <td className="px-4 py-2">
                  <IntakeStatusPill status={status} withTooltip={false} />
                </td>
                <td className="px-4 py-2 text-slate-700 dark:text-gray-200">{INTAKE_STATUS_INFO[status].summary}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </PopoverContent>
    </Popover>
  );
}

interface CategoryRailProps {
  counts: Record<IntakeCategory, number>;
  total: number;
  selected: IntakeCategory | null;
  onSelect: (category: IntakeCategory | null) => void;
}

export function CategoryRail({ counts, total, selected, onSelect }: CategoryRailProps) {
  const items: { value: IntakeCategory | null; label: string; count: number }[] = [
    { value: null, label: "All intakes", count: total },
    ...INTAKE_CATEGORIES.map((category) => ({
      value: category,
      label: INTAKE_CATEGORY_LABELS[category],
      count: counts[category],
    })),
  ];

  return (
    <nav aria-label="Categories" className="flex flex-col gap-0.5">
      <p className="px-2 pb-2 text-[11px] font-semibold tracking-wide text-slate-400 uppercase dark:text-gray-500">
        Categories
      </p>
      {items.map((item) => {
        const active = selected === item.value;
        return (
          <button
            key={item.label}
            type="button"
            aria-current={active ? "true" : undefined}
            onClick={() => onSelect(item.value)}
            className={cn(
              "flex items-center justify-between rounded-md px-2 py-1.5 text-left text-sm transition-colors",
              active
                ? "bg-white font-medium text-slate-900 shadow-xs ring-1 ring-slate-200 dark:bg-gray-800 dark:text-gray-100 dark:ring-gray-700"
                : "text-slate-600 hover:bg-white/70 dark:text-gray-300 dark:hover:bg-gray-800/60",
            )}
          >
            <span>{item.label}</span>
            <span className="text-xs text-slate-400 tabular-nums dark:text-gray-500">{item.count}</span>
          </button>
        );
      })}
    </nav>
  );
}

const SURFACE_ICONS: Record<IntakeSurface, LucideIcon> = {
  addIntake: LayoutGrid,
  onboardingTickets: Ticket,
  onboardingRepository: GitBranch,
};

export function SurfaceIcons({ intakeKey, category }: { intakeKey: string; category: string }) {
  const surfaces = intakeSurfaces(intakeKey, category);
  if (surfaces.length === 0) {
    return (
      <Tooltip>
        <TooltipTrigger asChild>
          <span className="text-xs text-slate-400 dark:text-gray-500">Not shown</span>
        </TooltipTrigger>
        <TooltipContent>No screen lists this intake. The code does not support it on any screen.</TooltipContent>
      </Tooltip>
    );
  }

  return (
    <span className="flex items-center gap-1" aria-label="Shown on">
      {surfaces.map((surface) => {
        const Icon = SURFACE_ICONS[surface];
        return (
          <Tooltip key={surface}>
            <TooltipTrigger asChild>
              <span
                aria-label={INTAKE_SURFACE_LABELS[surface]}
                className="flex size-6 items-center justify-center rounded border border-slate-200 text-slate-500 dark:border-gray-700 dark:text-gray-400"
              >
                <Icon size={12} />
              </span>
            </TooltipTrigger>
            <TooltipContent>Shown in {INTAKE_SURFACE_LABELS[surface]}</TooltipContent>
          </Tooltip>
        );
      })}
    </span>
  );
}
