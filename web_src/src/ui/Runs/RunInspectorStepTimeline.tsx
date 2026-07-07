import { useState, type CSSProperties } from "react";
import { getJsonViewStyle } from "@/lib/jsonViewTheme";
import { Accordion } from "@/ui/accordion";
import { useTheme } from "@/contexts/useTheme";
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
