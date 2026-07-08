import { SquareCheckBig } from "lucide-react";
import type { ReactNode } from "react";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "./RunNodeIcon";
import type { RunInspectorNodeSection, RunInspectorUpstreamSection } from "./runNodeDetailModel";
import type { TimelineStepType } from "./RunInspectorTimelineTypes";

export function TimelineRail({
  marker,
  isLast,
  children,
}: {
  marker: ReactNode;
  isLast: boolean;
  children: ReactNode;
}) {
  return (
    <div className="flex gap-3">
      <div className="flex flex-col items-center">
        {marker}
        {!isLast ? <div className="min-h-4 w-px flex-1 bg-slate-200 dark:bg-gray-800" /> : null}
      </div>
      <div className="min-w-0 flex-1 pb-3">{children}</div>
    </div>
  );
}

export function StepMarker({ type }: { type: TimelineStepType }) {
  const Icon = stepMarkerIcons[type];

  return (
    <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white text-slate-500 ring-1 ring-slate-200 dark:bg-gray-950 dark:ring-gray-800">
      <Icon className="h-3.5 w-3.5" />
    </span>
  );
}

export function NodeMarker({
  section,
  fallbackLabel,
  componentIconMap,
}: {
  section?: RunInspectorUpstreamSection | RunInspectorNodeSection;
  fallbackLabel: string;
  componentIconMap: Record<string, string>;
}) {
  const workflowNode = section?.workflowNode;
  const component = workflowNode?.component;

  return (
    <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white text-slate-500 ring-1 ring-slate-200 dark:bg-gray-950 dark:ring-gray-800">
      <RunNodeIcon
        iconSrc={getHeaderIconSrc(component)}
        iconSlug={component ? componentIconMap[component] : undefined}
        alt={workflowNode?.name || fallbackLabel}
        size={RUN_NODE_ICON_SIZE}
        className="h-3.5 w-3.5"
      />
    </span>
  );
}

function InputStepIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      <path d="m10 16 4-4-4-4" />
      <path d="M3 12h11" />
      <path d="M3 8V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-3" />
    </svg>
  );
}

function OutputStepIcon({ className }: { className?: string }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      className={className}
      aria-hidden="true"
    >
      <path d="m14 16 4-4-4-4" />
      <path d="M3 12h15" />
      <path d="M3 8V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-3" />
    </svg>
  );
}

const stepMarkerIcons = {
  input: InputStepIcon,
  runtime: SquareCheckBig,
  output: OutputStepIcon,
} satisfies Record<TimelineStepType, ({ className }: { className?: string }) => ReactNode>;
