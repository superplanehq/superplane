import {
  ArrowRight,
  ChevronRight,
  Clock,
  Database,
  Globe,
  LayoutTemplate,
  Loader2,
  Mail,
  Plus,
  Plug,
  Timer,
} from "lucide-react";

import { Heading } from "@/components/Heading/heading";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Badge } from "@/components/ui/badge";

const FLOW_STEPS = [
  {
    icon: Clock,
    label: "Schedule",
    bg: "bg-indigo-100 dark:bg-indigo-900/50",
    iconColor: "text-indigo-600 dark:text-indigo-400",
  },
  {
    icon: Globe,
    label: "HTTP Check",
    bg: "bg-sky-100 dark:bg-sky-900/50",
    iconColor: "text-sky-600 dark:text-sky-400",
  },
  {
    icon: Database,
    label: "Memory Check",
    bg: "bg-violet-100 dark:bg-violet-900/50",
    iconColor: "text-violet-600 dark:text-violet-400",
  },
  {
    icon: Mail,
    label: "Email Alert",
    bg: "bg-amber-100 dark:bg-amber-900/50",
    iconColor: "text-amber-600 dark:text-amber-400",
  },
];

interface QuickStartUiPanelProps {
  canCreateCanvases: boolean;
  permissionAllowed: boolean;
  templatesLoading: boolean;
  isLaunchingQuickStart: boolean;
  isCreatingBlankCanvas: boolean;
  onQuickStart: () => void;
  onBrowseTemplates: () => void;
  onCreateBlankCanvas: () => void;
}

function QuickStartHeroCard({
  canCreateCanvases,
  permissionAllowed,
  templatesLoading,
  isLaunchingQuickStart,
  onQuickStart,
}: Pick<
  QuickStartUiPanelProps,
  "canCreateCanvases" | "permissionAllowed" | "templatesLoading" | "isLaunchingQuickStart" | "onQuickStart"
>) {
  return (
    <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
      <button
        type="button"
        disabled={!canCreateCanvases || isLaunchingQuickStart || templatesLoading}
        onClick={onQuickStart}
        className="group w-full text-left bg-white dark:bg-gray-800 rounded-xl outline outline-slate-950/10 dark:outline-gray-700 p-5 mb-5 hover:shadow-lg hover:outline-primary/30 transition-all duration-200 cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
      >
        <div className="flex items-start justify-between gap-3 mb-3">
          <div>
            <div className="flex items-center gap-2 mb-1">
              <Heading level={3} className="!text-[15px]">
                Health Check Monitor
              </Heading>
              <Badge variant="secondary" className="text-[10px] font-medium">
                Quick Start
              </Badge>
            </div>
            <p className="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed">
              Pings your endpoint every ten minutes and alerts only on healthy-to-failing transitions, including
              approximately how long it stayed healthy.
            </p>
          </div>
          {isLaunchingQuickStart || templatesLoading ? (
            <Loader2 size={18} className="mt-0.5 text-gray-400 shrink-0 animate-spin" />
          ) : (
            <ArrowRight
              size={18}
              className="mt-0.5 text-gray-300 dark:text-gray-600 shrink-0 group-hover:text-primary group-hover:translate-x-0.5 transition-all"
            />
          )}
        </div>

        <div className="flex items-center gap-1.5 mb-3">
          {FLOW_STEPS.map((step, index) => (
            <div key={step.label} className="flex items-center gap-1.5">
              <span className={`inline-flex items-center gap-1.5 rounded-full ${step.bg} px-2.5 py-1`}>
                <step.icon size={12} className={step.iconColor} />
                <span className="text-[11px] font-medium text-gray-600 dark:text-gray-300">{step.label}</span>
              </span>
              {index < FLOW_STEPS.length - 1 && (
                <ChevronRight size={12} className="text-gray-400 dark:text-gray-500 shrink-0" />
              )}
            </div>
          ))}
        </div>

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <span className="inline-flex items-center gap-1 text-[11px] text-gray-500 dark:text-gray-400">
              <Timer size={12} />1 min setup
            </span>
            <span className="text-gray-300 dark:text-gray-600">|</span>
            <span className="inline-flex items-center gap-1 text-[11px] text-gray-500 dark:text-gray-400">
              <Plug size={12} />
              No integrations
            </span>
          </div>
          {isLaunchingQuickStart || templatesLoading ? (
            <span className="text-[11px] text-gray-400">
              {templatesLoading ? "Loading template..." : "Setting up..."}
            </span>
          ) : (
            <span className="text-[11px] text-primary group-hover:underline">Get started</span>
          )}
        </div>
      </button>
    </PermissionTooltip>
  );
}

function SecondaryCards({
  canCreateCanvases,
  permissionAllowed,
  isCreatingBlankCanvas,
  onBrowseTemplates,
  onCreateBlankCanvas,
}: Pick<
  QuickStartUiPanelProps,
  "canCreateCanvases" | "permissionAllowed" | "isCreatingBlankCanvas" | "onBrowseTemplates" | "onCreateBlankCanvas"
>) {
  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
        <button
          type="button"
          disabled={!canCreateCanvases}
          onClick={onBrowseTemplates}
          className="w-full text-left bg-white dark:bg-gray-800 rounded-xl outline outline-slate-950/10 dark:outline-gray-700 p-5 hover:shadow-md transition-shadow cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <div className="flex items-start gap-3">
            <div className="mt-0.5 rounded-lg bg-gray-100 dark:bg-gray-700 p-2">
              <LayoutTemplate size={18} className="text-gray-600 dark:text-gray-300" />
            </div>
            <div>
              <Heading level={3} className="!text-sm mb-1">
                Browse Templates
              </Heading>
              <p className="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed">
                Incident routing, CI/CD, rollbacks, and more. Ready to go.
              </p>
            </div>
          </div>
        </button>
      </PermissionTooltip>

      <PermissionTooltip allowed={permissionAllowed} message="You don't have permission to create canvases.">
        <button
          type="button"
          disabled={!canCreateCanvases || isCreatingBlankCanvas}
          onClick={onCreateBlankCanvas}
          className="w-full text-left bg-white dark:bg-gray-800 rounded-xl outline outline-slate-950/10 dark:outline-gray-700 p-5 hover:shadow-md transition-shadow cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <div className="flex items-start gap-3">
            <div className="mt-0.5 rounded-lg bg-gray-100 dark:bg-gray-700 p-2">
              {isCreatingBlankCanvas ? (
                <Loader2 size={18} className="text-gray-600 dark:text-gray-300 animate-spin" />
              ) : (
                <Plus size={18} className="text-gray-600 dark:text-gray-300" />
              )}
            </div>
            <div>
              <Heading level={3} className="!text-sm mb-1">
                Blank Canvas
              </Heading>
              <p className="text-[13px] text-gray-500 dark:text-gray-400 leading-relaxed">
                Start from scratch. You know what you&rsquo;re doing.
              </p>
            </div>
          </div>
        </button>
      </PermissionTooltip>
    </div>
  );
}

export function QuickStartUiPanel(props: QuickStartUiPanelProps) {
  return (
    <>
      <QuickStartHeroCard {...props} />

      <div className="flex items-center gap-3 mb-5">
        <div className="flex-1 h-px bg-gray-200 dark:bg-gray-700" />
        <span className="text-xs text-gray-400 dark:text-gray-500 font-medium uppercase tracking-wider">
          or pick your own path
        </span>
        <div className="flex-1 h-px bg-gray-200 dark:bg-gray-700" />
      </div>

      <SecondaryCards {...props} />
    </>
  );
}
