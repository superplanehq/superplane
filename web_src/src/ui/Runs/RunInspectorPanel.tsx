import JsonView from "@uiw/react-json-view";
import { AlertTriangle, Loader2, RefreshCw, Square, X } from "lucide-react";
import { useMemo, useState, type CSSProperties, type ReactNode } from "react";
import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { Timestamp } from "@/components/Timestamp";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useTheme } from "@/contexts/useTheme";
import { useEventExecutions } from "@/hooks/useCanvasData";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { formatDuration } from "@/lib/duration";
import { withEventStatusBadgeClasses } from "@/lib/eventStatusBadge";
import { getJsonViewStyle, jsonViewClassName } from "@/lib/jsonViewTheme";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";
import { RunNodeDetailDetailsView } from "./RunNodeDetailDetailsView";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "./RunNodeIcon";
import { buildNodeMap, buildRunPresentation, getRunStatus, RUN_STATUS_META } from "./runPresentation";
import {
  buildRunInspectorNodeSections,
  calculateRunDuration,
  findRunInspectorErrorSummaries,
  hasObjectValue,
  type RunInspectorNodeSection,
  type RunInspectorUpstreamSection,
} from "./runNodeDetailModel";

const ACCORDION_STORAGE_KEY = "superplane.runInspector.internalAccordions";

type InternalAccordionKey = "input" | "runtime" | "output";

type InternalAccordionPreferences = Record<InternalAccordionKey, boolean>;

const defaultAccordionPreferences: InternalAccordionPreferences = {
  input: true,
  runtime: false,
  output: true,
};

export interface RunInspectorPanelProps {
  canvasId: string;
  run: CanvasesCanvasRun;
  workflowNodes: ComponentsNode[];
  componentIconMap?: Record<string, string>;
  selectedNodeId?: string | null;
  onSelectNode: (nodeId: string) => void;
  onClearSelectedNode?: () => void;
  onClose: () => void;
}

export function RunInspectorPanel({
  canvasId,
  run,
  workflowNodes,
  componentIconMap = {},
  selectedNodeId = null,
  onSelectNode,
  onClearSelectedNode,
  onClose,
}: RunInspectorPanelProps) {
  const rootEventId = run.rootEvent?.id || null;
  const executionsQuery = useEventExecutions(canvasId, rootEventId);
  const executions = useMemo(() => executionsQuery.data?.executions || [], [executionsQuery.data?.executions]);
  const nodeMap = useMemo(() => buildNodeMap(workflowNodes), [workflowNodes]);
  const presentation = useMemo(() => buildRunPresentation(run, nodeMap), [nodeMap, run]);
  const sections = useMemo(
    () => buildRunInspectorNodeSections({ run, executions, workflowNodes }),
    [executions, run, workflowNodes],
  );
  const errorSummaries = useMemo(() => findRunInspectorErrorSummaries(sections), [sections]);
  const selectedValue = selectedNodeId ?? "";

  const handleValueChange = (value: string) => {
    if (value) {
      onSelectNode(value);
      return;
    }

    onClearSelectedNode?.();
  };

  return (
    <aside
      className={cn(
        "z-20 flex h-full w-[480px] max-w-[42vw] shrink-0 flex-col border-l bg-white shadow-sm dark:bg-gray-950",
        appDarkModeClasses.sidebarEdge,
      )}
      data-testid="run-inspector-panel"
      aria-label="Run inspector"
    >
      <RunInspectorHeader
        run={run}
        title={presentation.title}
        stepCount={sections.length || run.executions?.length || 0}
        onClose={onClose}
      />

      <div className="min-h-0 flex-1 overflow-y-auto">
        {errorSummaries.length > 0 ? (
          <div className="space-y-2 px-4 py-3">
            {errorSummaries.map((summary) => (
              <ErrorSummaryCard
                key={summary.nodeId}
                nodeName={summary.nodeName}
                message={summary.message}
                onJump={() => onSelectNode(summary.nodeId)}
              />
            ))}
          </div>
        ) : null}

        <div className="border-y border-slate-200 px-4 py-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:border-gray-800 dark:text-gray-400">
          Steps
          <span className="ml-2 font-medium normal-case tracking-normal text-slate-400">
            {RUN_STATUS_META[presentation.status].label}
          </span>
        </div>

        {executionsQuery.isLoading ? (
          <div className="flex items-center justify-center gap-2 px-4 py-8 text-sm text-slate-500 dark:text-gray-400">
            <Loader2 className="h-4 w-4 animate-spin" />
            Loading run steps...
          </div>
        ) : sections.length === 0 ? (
          <div className="px-4 py-8 text-sm text-slate-500 dark:text-gray-400">No executed nodes in this run.</div>
        ) : (
          <Accordion type="single" collapsible value={selectedValue} onValueChange={handleValueChange}>
            {sections.map((section) => (
              <RunInspectorNodeAccordion
                key={section.nodeId}
                section={section}
                componentIconMap={componentIconMap}
                isOpen={selectedValue === section.nodeId}
              />
            ))}
          </Accordion>
        )}
      </div>
    </aside>
  );
}

function RunInspectorHeader({
  run,
  title,
  stepCount,
  onClose,
}: {
  run: CanvasesCanvasRun;
  title: string;
  stepCount: number;
  onClose: () => void;
}) {
  const status = getRunStatus(run);
  const meta = RUN_STATUS_META[status];
  const Icon = meta.icon;
  const duration = calculateRunDuration(run);
  const actionLabel = status === "running" ? "Stop" : "Rerun";
  const ActionIcon = status === "running" ? Square : RefreshCw;
  const actionTooltip =
    status === "running"
      ? "Stop all running steps and cancel queued ones"
      : "Restart this whole run from trigger event";

  return (
    <div className="sticky top-0 z-10 border-b border-slate-200 bg-white px-4 py-4 dark:border-gray-800 dark:bg-gray-950">
      <div className="flex items-start gap-3">
        <span
          className={cn(
            "inline-flex shrink-0 items-center gap-1 rounded-full px-2 py-1 text-xs font-medium ring-1",
            meta.badgeClassName,
          )}
        >
          <Icon className="h-3.5 w-3.5" />
          {meta.label}
        </span>
        <div className="min-w-0 flex-1">
          <h2 className="truncate text-base font-semibold text-slate-950 dark:text-gray-100">{title}</h2>
          <div className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-1 text-sm text-slate-500 dark:text-gray-400">
            {run.createdAt ? <Timestamp date={run.createdAt} display="relative" relativeStyle="abbreviated" /> : null}
            {duration !== null ? (
              <>
                <span aria-hidden>·</span>
                <span>{formatDuration(duration)}</span>
              </>
            ) : null}
            <span aria-hidden>·</span>
            <span>
              {stepCount} {stepCount === 1 ? "step" : "steps"}
            </span>
          </div>
        </div>
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8 shrink-0" onClick={onClose}>
          <X className="h-4 w-4" />
          <span className="sr-only">Close run inspector</span>
        </Button>
      </div>
      <div className="mt-3 flex justify-end">
        <Tooltip>
          <TooltipTrigger asChild>
            <span>
              <Button type="button" size="sm" variant="outline" disabled className="gap-1.5">
                <ActionIcon className="h-3.5 w-3.5" />
                {actionLabel}
              </Button>
            </span>
          </TooltipTrigger>
          <TooltipContent>{actionTooltip}</TooltipContent>
        </Tooltip>
      </div>
    </div>
  );
}

function ErrorSummaryCard({ nodeName, message, onJump }: { nodeName: string; message: string; onJump: () => void }) {
  return (
    <div className="rounded-md border border-red-200 bg-red-50 p-3 text-red-700 dark:border-red-900/70 dark:bg-red-950/30 dark:text-red-300">
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
        <div className="min-w-0 flex-1">
          <p className="font-semibold">Errored at &quot;{nodeName}&quot;</p>
          <p className="mt-1 line-clamp-3 break-words text-sm">{message}</p>
        </div>
        <Button
          type="button"
          variant="outline"
          size="sm"
          className="shrink-0 border-red-300 text-red-700 hover:bg-red-100 dark:border-red-800 dark:text-red-300 dark:hover:bg-red-950"
          onClick={onJump}
        >
          Jump to error
        </Button>
      </div>
    </div>
  );
}

function RunInspectorNodeAccordion({
  section,
  componentIconMap,
  isOpen,
}: {
  section: RunInspectorNodeSection;
  componentIconMap: Record<string, string>;
  isOpen: boolean;
}) {
  const iconSrc = getHeaderIconSrc(section.workflowNode?.component);
  const iconSlug = section.workflowNode?.component ? componentIconMap[section.workflowNode.component] : undefined;

  return (
    <AccordionItem value={section.nodeId} className="border-slate-200 dark:border-gray-800">
      <AccordionTrigger
        className={cn(
          "min-h-14 gap-3 px-4 py-3 hover:no-underline",
          isOpen && "bg-sky-50 text-slate-950 dark:bg-gray-900 dark:text-gray-100",
        )}
      >
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <RunNodeIcon
            iconSrc={iconSrc}
            iconSlug={iconSlug}
            alt={section.nodeName}
            size={RUN_NODE_ICON_SIZE}
            className="text-slate-500 dark:text-gray-400"
          />
          <span className="min-w-0 truncate text-sm font-medium text-slate-900 dark:text-gray-100">
            {section.nodeName}
          </span>
        </div>
        <NodeMetadata section={section} />
      </AccordionTrigger>
      <AccordionContent className="bg-slate-50 px-4 pb-4 pt-3 dark:bg-gray-950">
        <RunInspectorNodeContent section={section} />
      </AccordionContent>
    </AccordionItem>
  );
}

function NodeMetadata({ section }: { section: RunInspectorNodeSection }) {
  return (
    <div className="ml-auto flex shrink-0 items-center gap-2 text-xs text-slate-500 dark:text-gray-400">
      {section.isTrigger && section.createdAt ? (
        <Timestamp date={section.createdAt} withHint={false} />
      ) : section.durationMs !== undefined ? (
        <span>{formatDuration(section.durationMs)}</span>
      ) : null}
      {section.badge ? <EventStatusBadge badgeColor={section.badge.badgeColor} label={section.badge.label} /> : null}
    </div>
  );
}

function RunInspectorNodeContent({ section }: { section: RunInspectorNodeSection }) {
  const { resolvedTheme } = useTheme();
  const jsonViewStyle = getJsonViewStyle(resolvedTheme);
  const [preferences, setPreferences] = useState(readAccordionPreferences);
  const openValues = Object.entries(preferences)
    .filter(([, isOpen]) => isOpen)
    .map(([key]) => key);

  const handlePreferenceChange = (values: string[]) => {
    const next: InternalAccordionPreferences = {
      input: values.includes("input"),
      runtime: values.includes("runtime"),
      output: values.includes("output"),
    };
    setPreferences(next);
    localStorage.setItem(ACCORDION_STORAGE_KEY, JSON.stringify(next));
  };

  const hasDetails = !!section.tabData?.details && Object.keys(section.tabData.details).length > 0;
  const hasRuntimeConfig = hasObjectValue(section.tabData?.configuration);
  const hasOutputs = section.outputSections.length > 0;
  const shouldShowError = !!section.errorMessage && !hasOutputs;

  return (
    <div className="space-y-3">
      {hasDetails || section.badge || section.createdAt ? (
        <div className="rounded-md border border-slate-200 bg-white p-3 dark:border-gray-800 dark:bg-gray-900">
          <RunNodeDetailDetailsView
            details={section.tabData?.details ?? {}}
            statusBadge={section.badge}
            relativeTime={section.createdAt}
          />
        </div>
      ) : null}

      <Accordion type="multiple" value={openValues} onValueChange={handlePreferenceChange} className="space-y-2">
        <InternalAccordionItem value="input" title="Input">
          <InputSection upstreamSections={section.upstreamSections} jsonViewStyle={jsonViewStyle} />
        </InternalAccordionItem>
        <InternalAccordionItem value="runtime" title="Runtime config">
          {hasRuntimeConfig ? (
            <JsonView
              value={section.tabData?.configuration as object}
              collapsed={2}
              style={jsonViewStyle}
              className={jsonViewClassName}
              displayObjectSize={false}
              enableClipboard={false}
            />
          ) : (
            <EmptySectionText>No runtime config for this step.</EmptySectionText>
          )}
        </InternalAccordionItem>
        <InternalAccordionItem value="output" title={outputTitle(section)}>
          {hasOutputs ? (
            <div className="space-y-3">
              {section.outputSections.map((output) => (
                <div key={output.channel} className="space-y-2">
                  <div className="text-xs font-medium text-slate-500 dark:text-gray-400">{output.channel}</div>
                  <JsonView
                    value={output.value as object}
                    collapsed={2}
                    style={jsonViewStyle}
                    className={jsonViewClassName}
                    displayObjectSize={false}
                    enableClipboard={false}
                  />
                </div>
              ))}
            </div>
          ) : shouldShowError ? (
            <p className="break-words text-sm text-red-600 dark:text-red-300">{section.errorMessage}</p>
          ) : (
            <EmptySectionText>No output for this step.</EmptySectionText>
          )}
        </InternalAccordionItem>
      </Accordion>
    </div>
  );
}

function InternalAccordionItem({
  value,
  title,
  children,
}: {
  value: InternalAccordionKey;
  title: string;
  children: ReactNode;
}) {
  return (
    <AccordionItem
      value={value}
      className="rounded-md border border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900"
    >
      <AccordionTrigger className="px-3 py-2 text-xs font-semibold uppercase tracking-wide text-slate-500 hover:no-underline dark:text-gray-400">
        {title}
      </AccordionTrigger>
      <AccordionContent className="px-3 pb-3">{children}</AccordionContent>
    </AccordionItem>
  );
}

function InputSection({
  upstreamSections,
  jsonViewStyle,
}: {
  upstreamSections: RunInspectorUpstreamSection[];
  jsonViewStyle: CSSProperties;
}) {
  const [showAll, setShowAll] = useState(false);

  if (upstreamSections.length === 0) {
    return <EmptySectionText>No upstream input for this step.</EmptySectionText>;
  }

  const visibleSections = showAll ? upstreamSections : upstreamSections.slice(0, 1);
  const hiddenCount = upstreamSections.length - visibleSections.length;

  return (
    <div className="space-y-3">
      {visibleSections.map((upstream) => (
        <div key={upstream.nodeId} className="space-y-2">
          <div className="flex items-center gap-2 text-sm">
            <span className="min-w-0 truncate font-medium text-slate-800 dark:text-gray-100">{upstream.nodeName}</span>
            {upstream.badge ? (
              <EventStatusBadge badgeColor={upstream.badge.badgeColor} label={upstream.badge.label} />
            ) : null}
          </div>
          {hasObjectValue(upstream.output) ? (
            <JsonView
              value={upstream.output as object}
              collapsed={2}
              style={jsonViewStyle}
              className={jsonViewClassName}
              displayObjectSize={false}
              enableClipboard={false}
            />
          ) : (
            <EmptySectionText>No output from this upstream step.</EmptySectionText>
          )}
        </div>
      ))}
      {hiddenCount > 0 ? (
        <Button type="button" variant="ghost" size="sm" className="h-7 px-2 text-xs" onClick={() => setShowAll(true)}>
          +{hiddenCount} more
        </Button>
      ) : null}
    </div>
  );
}

function outputTitle(section: RunInspectorNodeSection): string {
  if (section.errorMessage && section.outputSections.length === 0) return "Error";
  if (section.outputSections.length === 1) {
    const [output] = section.outputSections;
    return `Output · ${output.channel} · ${output.sizeKb}`;
  }
  if (section.outputSections.length > 1) return `Output · ${section.outputSections.length} channels`;
  return "Output";
}

function EventStatusBadge({ badgeColor, label }: { badgeColor: string; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded px-[5px] py-[1.5px] text-[10px] font-semibold uppercase tracking-wide text-white",
        withEventStatusBadgeClasses(badgeColor),
      )}
    >
      {label}
    </span>
  );
}

function EmptySectionText({ children }: { children: ReactNode }) {
  return <p className="text-sm text-slate-500 dark:text-gray-400">{children}</p>;
}

function readAccordionPreferences(): InternalAccordionPreferences {
  const storedValue = localStorage.getItem(ACCORDION_STORAGE_KEY);
  if (!storedValue) return defaultAccordionPreferences;

  try {
    const parsed = JSON.parse(storedValue) as Partial<InternalAccordionPreferences>;
    return {
      input: parsed.input ?? defaultAccordionPreferences.input,
      runtime: parsed.runtime ?? defaultAccordionPreferences.runtime,
      output: parsed.output ?? defaultAccordionPreferences.output,
    };
  } catch {
    return defaultAccordionPreferences;
  }
}
