import { useState, type CSSProperties } from "react";
import { getJsonViewStyle } from "@/lib/jsonViewTheme";
import { Accordion } from "@/ui/accordion";
import { useTheme } from "@/contexts/useTheme";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { RunNodeDetailDetailsView } from "./RunNodeDetailDetailsView";
import { InputChainMoreChip } from "./RunInspectorInputChainModal";
import {
  DetailBox,
  EmptySectionText,
  ErrorOutputCard,
  JsonPayload,
  TimelineAccordionCard,
} from "./RunInspectorTimelineCard";
import { StepMarker, TimelineRail } from "./RunInspectorTimelineMarkers";
import {
  ACCORDION_STORAGE_KEY,
  buildTimelineItems,
  readAccordionPreferences,
  type InternalAccordionPreferences,
  type StatusPill,
} from "./RunInspectorTimelineTypes";
import { hasObjectValue, type RunInspectorNodeSection, type RunInspectorUpstreamSection } from "./runNodeDetailModel";

export function RunInspectorStepTimeline({
  section,
  componentIconMap,
  organizationId,
}: {
  section: RunInspectorNodeSection;
  componentIconMap: Record<string, string>;
  organizationId?: string;
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
              <RuntimeTimelineCard section={section} jsonViewStyle={jsonViewStyle} organizationId={organizationId} />
            ) : (
              <OutputTimelineCard section={section} jsonViewStyle={jsonViewStyle} />
            )}
          </TimelineRail>
        ))}
      </Accordion>
    </div>
  );
}

function RuntimeTimelineCard({
  section,
  jsonViewStyle,
  organizationId,
}: {
  section: RunInspectorNodeSection;
  jsonViewStyle: CSSProperties;
  organizationId?: string;
}) {
  const [mode, setMode] = useState<"form" | "json">("form");
  const configuration = section.tabData?.configuration;

  return (
    <TimelineAccordionCard
      value="runtime"
      status={{ dotClassName: "bg-blue-500", label: "Running" }}
      title="Runtime Config"
      trailing={<RuntimeViewToggle mode={mode} onChange={setMode} />}
      jsonViewStyle={jsonViewStyle}
    >
      {mode === "form" ? (
        <RuntimeConfigForm
          section={section}
          value={configuration}
          jsonViewStyle={jsonViewStyle}
          organizationId={organizationId}
        />
      ) : (
        <JsonPayload value={configuration} jsonViewStyle={jsonViewStyle} />
      )}
    </TimelineAccordionCard>
  );
}

function RuntimeViewToggle({ mode, onChange }: { mode: "form" | "json"; onChange: (mode: "form" | "json") => void }) {
  return (
    <span className="inline-flex h-7 items-center rounded-md border border-slate-200 bg-white p-0.5 text-xs font-medium dark:border-gray-700 dark:bg-gray-950">
      {(["form", "json"] as const).map((item) => (
        <button
          key={item}
          type="button"
          aria-label={item === "form" ? "Form" : "JSON"}
          aria-pressed={mode === item}
          className={
            mode === item
              ? "h-6 rounded bg-slate-100 px-2 text-slate-900 shadow-sm dark:bg-gray-800 dark:text-gray-50"
              : "h-6 rounded px-2 text-slate-500 hover:bg-slate-50 hover:text-slate-800 dark:text-gray-400 dark:hover:bg-gray-900 dark:hover:text-gray-100"
          }
          onClick={(event) => {
            event.stopPropagation();
            onChange(item);
          }}
        >
          {item === "form" ? "Form" : "JSON"}
        </button>
      ))}
    </span>
  );
}

function RuntimeConfigForm({
  section,
  value,
  jsonViewStyle,
  organizationId,
}: {
  section: RunInspectorNodeSection;
  value: unknown;
  jsonViewStyle: CSSProperties;
  organizationId?: string;
}) {
  if (!hasObjectValue(value)) {
    return <EmptySectionText>No runtime configuration for this step.</EmptySectionText>;
  }

  if (section.configurationFields.length > 0) {
    return (
      <RuntimeSchemaConfigForm fields={section.configurationFields} value={value} organizationId={organizationId} />
    );
  }

  return (
    <div className="space-y-3">
      {Object.entries(value).map(([key, fieldValue]) => (
        <RuntimeFallbackConfigField key={key} name={key} value={fieldValue} jsonViewStyle={jsonViewStyle} />
      ))}
    </div>
  );
}

function RuntimeSchemaConfigForm({
  fields,
  value,
  organizationId,
}: {
  fields: RunInspectorNodeSection["configurationFields"];
  value: Record<string, unknown>;
  organizationId?: string;
}) {
  return (
    <div className="space-y-4">
      {fields.map((field) => {
        if (!field.name) return null;

        return (
          <ConfigurationFieldRenderer
            key={field.name}
            field={field}
            value={value[field.name]}
            allValues={value}
            onChange={() => {}}
            domainId={organizationId}
            organizationId={organizationId}
            readOnly
          />
        );
      })}
    </div>
  );
}

function RuntimeFallbackConfigField({
  name,
  value,
  jsonViewStyle,
}: {
  name: string;
  value: unknown;
  jsonViewStyle: CSSProperties;
}) {
  const label = formatRuntimeFieldLabel(name);

  if (typeof value === "boolean") {
    return (
      <label className="flex items-center gap-3 text-sm font-medium text-slate-800 dark:text-gray-100">
        <span
          className={
            value
              ? "relative inline-flex h-5 w-9 rounded-full bg-blue-500"
              : "relative inline-flex h-5 w-9 rounded-full bg-slate-200 dark:bg-gray-700"
          }
        >
          <span
            className={
              value
                ? "absolute right-0.5 top-0.5 h-4 w-4 rounded-full bg-white"
                : "absolute left-0.5 top-0.5 h-4 w-4 rounded-full bg-white"
            }
          />
        </span>
        {label}
      </label>
    );
  }

  if (typeof value === "string" || typeof value === "number" || value === null) {
    return (
      <label className="block space-y-1.5">
        <span className="text-sm font-medium text-slate-800 dark:text-gray-100">{label}</span>
        <input
          aria-label={label}
          readOnly
          value={value === null ? "" : String(value)}
          className="h-9 w-full rounded-md border border-slate-300 bg-white px-3 text-sm text-slate-900 outline-none dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100"
        />
      </label>
    );
  }

  return (
    <div className="space-y-1.5">
      <div className="text-sm font-medium text-slate-800 dark:text-gray-100">{label}</div>
      <div className="rounded-md border border-slate-200 bg-white p-2 dark:border-gray-800 dark:bg-gray-950">
        <JsonPayload value={value} jsonViewStyle={jsonViewStyle} />
      </div>
    </div>
  );
}

function formatRuntimeFieldLabel(value: string): string {
  return value
    .replace(/[_-]+/g, " ")
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/\w\S*/g, (word) => word.charAt(0).toUpperCase() + word.slice(1));
}

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

function outputActionPayload(section: RunInspectorNodeSection): unknown {
  if (section.outputSections.length === 1) return section.outputSections[0].value;
  if (section.outputSections.length > 1) {
    return Object.fromEntries(section.outputSections.map((output) => [output.channel, output.value]));
  }

  return section.errorMessage ? { error: section.errorMessage } : undefined;
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
