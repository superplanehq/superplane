import JsonView from "@uiw/react-json-view";
import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { Check, ChevronDown, Copy, Maximize2, SquareCheckBig } from "lucide-react";
import { useEffect, useMemo, useState, type CSSProperties, type MouseEvent, type ReactNode } from "react";
import { Dialog, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { getJsonViewStyle, jsonViewClassName } from "@/lib/jsonViewTheme";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { Accordion, AccordionContent, AccordionItem } from "@/ui/accordion";
import { useTheme } from "@/contexts/useTheme";
import { RunNodeDetailDetailsView } from "./RunNodeDetailDetailsView";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "./RunNodeIcon";
import { hasObjectValue, type RunInspectorNodeSection, type RunInspectorUpstreamSection } from "./runNodeDetailModel";

const ACCORDION_STORAGE_KEY = "superplane.runInspector.internalAccordions";

type InternalAccordionKey = "input" | "runtime" | "output";
type InternalAccordionPreferences = Record<InternalAccordionKey, boolean>;

const defaultAccordionPreferences: InternalAccordionPreferences = {
  input: true,
  runtime: true,
  output: true,
};

type StatusPill = {
  dotClassName: string;
  label: string;
  tone?: "default" | "error";
};

type TimelineStepType = "input" | "runtime" | "output";

export function RunInspectorStepTimeline({
  section,
  componentIconMap,
}: {
  section: RunInspectorNodeSection;
  componentIconMap: Record<string, string>;
}) {
  const { resolvedTheme } = useTheme();
  const jsonViewStyle = getJsonViewStyle(resolvedTheme);
  const [preferences, setPreferences] = useState(readAccordionPreferences);
  const openValues = Object.entries(preferences)
    .filter(([, isOpen]) => isOpen)
    .map(([key]) => key);

  const hasDetails = !!section.tabData?.details && Object.keys(section.tabData.details).length > 0;
  const hasRuntimeConfig = hasObjectValue(section.tabData?.configuration);
  const timelineItems = buildTimelineItems(section, hasRuntimeConfig);

  const handlePreferenceChange = (values: string[]) => {
    const next: InternalAccordionPreferences = {
      input: values.includes("input"),
      runtime: values.includes("runtime"),
      output: values.includes("output"),
    };
    setPreferences(next);
    localStorage.setItem(ACCORDION_STORAGE_KEY, JSON.stringify(next));
  };

  return (
    <div className="space-y-2">
      {hasDetails || section.badge || section.createdAt ? (
        <DetailBox title="Summary">
          <RunNodeDetailDetailsView
            details={section.tabData?.details ?? {}}
            statusBadge={section.badge}
            relativeTime={section.createdAt}
          />
        </DetailBox>
      ) : null}

      <Accordion type="multiple" value={openValues} onValueChange={handlePreferenceChange}>
        {timelineItems.map((item, index) => (
          <TimelineRail
            key={item.value}
            marker={<StepMarker type={item.value} />}
            isLast={index === timelineItems.length - 1}
          >
            {item.value === "input" ? (
              <InputTimelineCard section={section} jsonViewStyle={jsonViewStyle} componentIconMap={componentIconMap} />
            ) : item.value === "runtime" ? (
              <TimelineAccordionCard
                value="runtime"
                status={{ dotClassName: "bg-blue-500", label: "Running" }}
                title="Runtime Config"
                trailing="JSON"
                jsonViewStyle={jsonViewStyle}
              >
                <JsonPayload value={section.tabData?.configuration} jsonViewStyle={jsonViewStyle} />
              </TimelineAccordionCard>
            ) : (
              <OutputTimelineCard section={section} jsonViewStyle={jsonViewStyle} />
            )}
          </TimelineRail>
        ))}
      </Accordion>
    </div>
  );
}

function OutputTimelineCard({
  section,
  jsonViewStyle,
}: {
  section: RunInspectorNodeSection;
  jsonViewStyle: CSSProperties;
}) {
  const hasOutputs = section.outputSections.length > 0;

  if (section.errorMessage && !hasOutputs) {
    return <ErrorOutputCard nodeId={section.nodeId} message={section.errorMessage} />;
  }

  return (
    <TimelineAccordionCard
      value="output"
      status={outputStatus(section)}
      title={outputTitle(section)}
      actionPayload={outputActionPayload(section)}
      jsonViewStyle={jsonViewStyle}
    >
      {hasOutputs ? (
        <OutputSection section={section} jsonViewStyle={jsonViewStyle} />
      ) : (
        <EmptySectionText>No output for this step.</EmptySectionText>
      )}
    </TimelineAccordionCard>
  );
}

function buildTimelineItems(section: RunInspectorNodeSection, hasRuntimeConfig: boolean) {
  const items: Array<{
    value: TimelineStepType;
  }> = [];

  if (!section.isTrigger) {
    items.push({ value: "input" });
  }

  if (hasRuntimeConfig) {
    items.push({ value: "runtime" });
  }

  items.push({ value: "output" });

  return items;
}

function TimelineRail({ marker, isLast, children }: { marker: ReactNode; isLast: boolean; children: ReactNode }) {
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

function StepMarker({ type }: { type: TimelineStepType }) {
  const Icon = stepMarkerIcons[type];

  return (
    <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-white text-slate-500 ring-1 ring-slate-200 dark:bg-gray-950 dark:ring-gray-800">
      <Icon className="h-3.5 w-3.5" />
    </span>
  );
}

function NodeMarker({
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

function InputTimelineCard({
  section,
  jsonViewStyle,
  componentIconMap,
}: {
  section: RunInspectorNodeSection;
  jsonViewStyle: CSSProperties;
  componentIconMap: Record<string, string>;
}) {
  const leadInput =
    section.upstreamSections.find((upstreamSection) => upstreamSection.nodeId === section.primaryInputNodeId) ??
    section.upstreamSections.at(-1);
  const hiddenInputCount = Math.max(0, section.upstreamSections.length - 1);

  return (
    <TimelineAccordionCard
      value="input"
      status={inputStatus(leadInput)}
      title="Input"
      sourceName={leadInput?.nodeName}
      actionPayload={leadInput ? (leadInput.output ?? {}) : undefined}
      jsonViewStyle={jsonViewStyle}
      sourceTrailing={
        hiddenInputCount > 0 ? (
          <InputChainMoreChip
            count={hiddenInputCount}
            sections={section.upstreamSections}
            initialSelectedNodeId={leadInput?.nodeId}
            componentIconMap={componentIconMap}
            jsonViewStyle={jsonViewStyle}
          />
        ) : null
      }
    >
      {leadInput ? (
        <InputPayload section={leadInput} jsonViewStyle={jsonViewStyle} />
      ) : (
        <EmptySectionText>No upstream input for this step.</EmptySectionText>
      )}
    </TimelineAccordionCard>
  );
}

function TimelineAccordionCard({
  value,
  status,
  title,
  sourceName,
  sourceTrailing,
  trailing,
  actionPayload,
  jsonViewStyle,
  children,
}: {
  value: InternalAccordionKey;
  status: StatusPill;
  title: string;
  sourceName?: string;
  sourceTrailing?: ReactNode;
  trailing?: ReactNode;
  actionPayload?: unknown;
  jsonViewStyle: CSSProperties;
  children: ReactNode;
}) {
  const [modalOpen, setModalOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [modalCopied, setModalCopied] = useState(false);
  const canUsePayloadActions = isDisplayablePayload(actionPayload);
  const payloadString = useMemo(() => JSON.stringify(actionPayload ?? {}, null, 2), [actionPayload]);

  const copyPayload = (markCopied: (value: boolean) => void) => {
    void navigator.clipboard?.writeText(payloadString).catch(() => {});
    markCopied(true);
    setTimeout(() => markCopied(false), 1500);
  };

  return (
    <>
      <AccordionItem
        value={value}
        className="overflow-hidden rounded border border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900"
      >
        <AccordionPrimitive.Header className="flex items-center gap-1 border-b border-slate-200 bg-slate-50 py-1.5 pr-2 dark:border-gray-800 dark:bg-gray-900">
          <AccordionPrimitive.Trigger className="flex min-w-0 items-center gap-1.5 px-3 text-left hover:no-underline">
            <EventStatusPill {...status} />
            <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-400 dark:text-gray-500">
              {title}
            </span>
            {sourceName ? (
              <span className="min-w-0 truncate text-[12px] font-medium text-slate-600 dark:text-gray-200">
                {sourceName}
              </span>
            ) : null}
          </AccordionPrimitive.Trigger>
          {sourceTrailing}
          <span className="ml-auto flex shrink-0 items-center gap-0.5 text-xs text-slate-700 dark:text-gray-200">
            {trailing ? <span className="pr-1">{trailing}</span> : null}
            {canUsePayloadActions ? (
              <>
                <HeaderIconButton
                  label={copied ? "Copied" : "Copy"}
                  icon={copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
                  onClick={() => copyPayload(setCopied)}
                />
                <HeaderIconButton
                  label="Open fullscreen"
                  icon={<Maximize2 className="h-3.5 w-3.5" />}
                  onClick={() => setModalOpen(true)}
                />
              </>
            ) : null}
          </span>
          <AccordionPrimitive.Trigger
            aria-label="Toggle section"
            className="flex h-6 w-6 shrink-0 items-center justify-center rounded text-slate-400 transition-colors hover:bg-slate-200 hover:text-slate-700 data-[state=open]:text-slate-600 dark:hover:bg-gray-800 dark:hover:text-gray-100 dark:data-[state=open]:text-gray-300 [&[data-state=open]>svg]:rotate-180"
          >
            <ChevronDown className="h-4 w-4 transition-transform duration-200" />
          </AccordionPrimitive.Trigger>
        </AccordionPrimitive.Header>
        <AccordionContent className="px-3 py-2.5">{children}</AccordionContent>
      </AccordionItem>

      <Dialog open={modalOpen} onOpenChange={setModalOpen}>
        <DialogContent
          size="large"
          className="flex h-[80vh] w-[60vw] max-w-[60vw] flex-col gap-0 overflow-hidden p-0"
          onClick={(event) => event.stopPropagation()}
        >
          <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 pr-10 dark:border-gray-800 dark:bg-gray-900">
            <DialogTitle className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
              {title}
            </DialogTitle>
            <span className="flex items-center gap-0.5">
              <HeaderIconButton
                label={modalCopied ? "Copied" : "Copy"}
                icon={
                  modalCopied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />
                }
                onClick={() => copyPayload(setModalCopied)}
              />
            </span>
          </div>
          <div className="min-h-0 flex-1 overflow-auto p-3">
            <JsonPayload value={actionPayload} jsonViewStyle={jsonViewStyle} />
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

function EventStatusPill({ dotClassName, label, tone = "default" }: StatusPill) {
  return (
    <span
      className={cn(
        "flex shrink-0 items-center gap-1.5 rounded-full bg-white px-2 py-0.5 ring-1 dark:bg-gray-950",
        tone === "error" ? "ring-red-200 dark:ring-red-900/70" : "ring-slate-200 dark:ring-gray-800",
      )}
    >
      <span className={cn("h-2 w-2 shrink-0 rounded-full", dotClassName)} />
      <span
        className={cn(
          "text-[11px] font-medium capitalize",
          tone === "error" ? "text-red-600" : "text-slate-700 dark:text-gray-200",
        )}
      >
        {label}
      </span>
    </span>
  );
}

function InputPayload({
  section,
  jsonViewStyle,
}: {
  section: RunInspectorUpstreamSection;
  jsonViewStyle: CSSProperties;
}) {
  if (!hasObjectValue(section.output)) {
    return <EmptySectionText>No output from this upstream step.</EmptySectionText>;
  }

  return <JsonPayload value={section.output} jsonViewStyle={jsonViewStyle} />;
}

function OutputSection({ section, jsonViewStyle }: { section: RunInspectorNodeSection; jsonViewStyle: CSSProperties }) {
  if (section.outputSections.length === 1) {
    return <JsonPayload value={section.outputSections[0].value} jsonViewStyle={jsonViewStyle} />;
  }

  return (
    <div className="space-y-3">
      {section.outputSections.map((output) => (
        <div key={output.channel} className="space-y-2">
          <div className="text-xs font-medium text-slate-500 dark:text-gray-400">{output.channel}</div>
          <JsonPayload value={output.value} jsonViewStyle={jsonViewStyle} />
        </div>
      ))}
    </div>
  );
}

function JsonPayload({ value, jsonViewStyle }: { value: unknown; jsonViewStyle: CSSProperties }) {
  return (
    <JsonView
      value={(value ?? {}) as object}
      collapsed={2}
      style={jsonViewStyle}
      className={jsonViewClassName}
      displayObjectSize={false}
      enableClipboard={false}
    />
  );
}

function isDisplayablePayload(value: unknown): boolean {
  if (typeof value === "string") return value.length > 0;
  return !!value && typeof value === "object";
}

function outputActionPayload(section: RunInspectorNodeSection): unknown {
  if (section.outputSections.length === 1) return section.outputSections[0].value;
  if (section.outputSections.length > 1) {
    return Object.fromEntries(section.outputSections.map((output) => [output.channel, output.value]));
  }

  return section.errorMessage ? { error: section.errorMessage } : undefined;
}

function InputChainMoreChip({
  count,
  sections,
  initialSelectedNodeId,
  componentIconMap,
  jsonViewStyle,
}: {
  count: number;
  sections: RunInspectorUpstreamSection[];
  initialSelectedNodeId?: string;
  componentIconMap: Record<string, string>;
  jsonViewStyle: CSSProperties;
}) {
  const [open, setOpen] = useState(false);

  return (
    <>
      <button
        type="button"
        title="Open input chain"
        onClick={(event) => {
          event.stopPropagation();
          setOpen(true);
        }}
        className="flex shrink-0 items-center rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-600 transition-colors hover:bg-slate-200 hover:text-slate-700 dark:bg-gray-800 dark:text-gray-300 dark:hover:bg-gray-700"
      >
        +{count} more
      </button>
      <InputChainModal
        open={open}
        onOpenChange={setOpen}
        sections={sections}
        initialSelectedNodeId={initialSelectedNodeId}
        componentIconMap={componentIconMap}
        jsonViewStyle={jsonViewStyle}
      />
    </>
  );
}

function InputChainModal({
  open,
  onOpenChange,
  sections,
  initialSelectedNodeId,
  componentIconMap,
  jsonViewStyle,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  sections: RunInspectorUpstreamSection[];
  initialSelectedNodeId?: string;
  componentIconMap: Record<string, string>;
  jsonViewStyle: CSSProperties;
}) {
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const selected =
    sections.find((section) => section.nodeId === selectedNodeId) ??
    sections.find((section) => section.nodeId === initialSelectedNodeId) ??
    sections.at(-1);
  const payloadString = useMemo(() => JSON.stringify(selected?.output ?? {}, null, 2), [selected?.output]);

  useEffect(() => {
    if (!open) return;

    setSelectedNodeId(initialSelectedNodeId ?? sections.at(-1)?.nodeId ?? null);
  }, [initialSelectedNodeId, open, sections]);

  const copyPayload = () => {
    void navigator.clipboard?.writeText(payloadString).catch(() => {});
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        size="large"
        className="flex h-[80vh] w-[70vw] max-w-[70vw] flex-col gap-0 overflow-hidden p-0"
        onClick={(event) => event.stopPropagation()}
      >
        <DialogTitle className="sr-only">Input chain</DialogTitle>
        <div className="flex min-h-0 flex-1">
          <div className="flex w-56 shrink-0 flex-col gap-0.5 overflow-y-auto border-r border-slate-200 bg-slate-50 p-2 dark:border-gray-800 dark:bg-gray-900">
            <div className="px-2 py-1 text-[11px] font-semibold uppercase tracking-wide text-slate-400">
              Input chain
            </div>
            {sections.map((section) => (
              <button
                key={section.nodeId}
                type="button"
                onClick={() => setSelectedNodeId(section.nodeId)}
                className={cn(
                  "flex items-center gap-2 rounded px-2 py-1.5 text-left text-[12px] transition-colors",
                  selected?.nodeId === section.nodeId
                    ? "bg-white font-medium text-slate-900 shadow-sm ring-1 ring-slate-200 dark:bg-gray-950 dark:text-gray-100 dark:ring-gray-800"
                    : "text-slate-600 hover:bg-slate-100 dark:text-gray-300 dark:hover:bg-gray-800",
                )}
              >
                <NodeMarker section={section} fallbackLabel={section.nodeName} componentIconMap={componentIconMap} />
                <span className="min-w-0 truncate">{section.nodeName}</span>
              </button>
            ))}
          </div>
          <div className="flex min-w-0 flex-1 flex-col">
            <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 pr-10 dark:border-gray-800 dark:bg-gray-900">
              <div className="flex min-w-0 items-center gap-1.5">
                {selected ? (
                  <NodeMarker
                    section={selected}
                    fallbackLabel={selected.nodeName}
                    componentIconMap={componentIconMap}
                  />
                ) : null}
                <span className="truncate text-[12px] font-medium text-slate-700 dark:text-gray-200">
                  {selected?.nodeName}
                </span>
                <span className="shrink-0 text-[11px] font-semibold uppercase tracking-wide text-slate-500">
                  Output
                </span>
              </div>
              <div className="flex items-center gap-0.5">
                <HeaderIconButton
                  label={copied ? "Copied" : "Copy"}
                  icon={copied ? <Check className="h-3.5 w-3.5 text-emerald-600" /> : <Copy className="h-3.5 w-3.5" />}
                  onClick={copyPayload}
                />
              </div>
            </div>
            <div className="min-h-0 flex-1 overflow-auto p-3">
              <JsonPayload value={selected?.output} jsonViewStyle={jsonViewStyle} />
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function HeaderIconButton({
  label,
  icon,
  onClick,
  active,
}: {
  label: string;
  icon: ReactNode;
  onClick?: (event: MouseEvent<HTMLButtonElement>) => void;
  active?: boolean;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button
          type="button"
          aria-label={label}
          aria-pressed={active}
          onClick={(event) => {
            event.stopPropagation();
            onClick?.(event);
          }}
          className={cn(
            "flex h-6 w-6 items-center justify-center rounded transition-colors",
            active
              ? "bg-blue-100 text-blue-700 hover:bg-blue-200"
              : "text-slate-400 hover:bg-slate-200 hover:text-slate-700 dark:hover:bg-gray-800 dark:hover:text-gray-100",
          )}
        >
          {icon}
        </button>
      </TooltipTrigger>
      <TooltipContent side="top">{label}</TooltipContent>
    </Tooltip>
  );
}

function DetailBox({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="overflow-hidden rounded border border-slate-200 bg-white dark:border-gray-800 dark:bg-gray-900">
      <div className="flex items-center justify-between gap-2 border-b border-slate-200 bg-slate-50 px-3 py-1.5 dark:border-gray-800 dark:bg-gray-900">
        <span className="text-[11px] font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
          {title}
        </span>
      </div>
      <div className="px-3 py-2.5">{children}</div>
    </div>
  );
}

function ErrorOutputCard({ nodeId, message }: { nodeId: string; message?: string }) {
  return (
    <div
      className="overflow-hidden rounded border border-red-200 bg-red-50 text-red-700 dark:border-red-900/70 dark:bg-red-950/30 dark:text-red-300"
      data-run-error-output-node-id={nodeId}
    >
      <div className="flex items-center justify-between gap-1.5 border-b border-red-200 px-3 py-1.5 dark:border-red-900/70">
        <span className="truncate text-[11px] font-semibold uppercase tracking-wide text-red-600">
          Error - Output not emitted
        </span>
      </div>
      <div className="px-3 py-2.5 text-[13px]">
        <span className="min-w-0 break-all font-medium">{message}</span>
      </div>
    </div>
  );
}

function inputStatus(section?: RunInspectorUpstreamSection): StatusPill {
  return {
    dotClassName: section?.badge?.badgeColor ?? "bg-violet-400",
    label: section?.badge?.label ?? "Triggered",
  };
}

function outputStatus(section: RunInspectorNodeSection): StatusPill {
  if (section.errorMessage && section.outputSections.length === 0) {
    return { dotClassName: "bg-red-500", label: "Error", tone: "error" };
  }

  return {
    dotClassName: section.badge?.badgeColor ?? "bg-slate-400",
    label: section.badge?.label ?? "Output",
  };
}

function outputTitle(section: RunInspectorNodeSection): string {
  if (section.errorMessage && section.outputSections.length === 0) return "Output";
  if (section.outputSections.length === 1) {
    const [output] = section.outputSections;
    return `Output · ${output.channel} · ${output.sizeKb}`.toUpperCase();
  }
  if (section.outputSections.length > 1) return `Output · ${section.outputSections.length} channels`.toUpperCase();
  return "Output";
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
