import { Pencil } from "lucide-react";
import { useMemo, useState, type CSSProperties } from "react";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { SEGMENTED_NAV_CLASSES, segmentedNavTabClassName } from "@/lib/segmentedNav";
import { ConfigurationFieldRenderer } from "@/ui/configurationFieldRenderer";
import { cn } from "@/lib/utils";
import { EmptySectionText, JsonPayload, TimelineAccordionCard } from "./RunInspectorTimelineCard";
import { HeaderIconButton } from "@/ui/HeaderIconButton";
import { hasObjectValue } from "./runNodeDetailModel";
import type { RunInspectorNodeSection } from "./types";
import { buildRuntimeExpressionContext } from "./runInspectorExpressionContext";

export function RuntimeTimelineCard({
  section,
  jsonViewStyle,
  organizationId,
  canShowExpressionTemplates = false,
  onEditNode,
}: {
  section: RunInspectorNodeSection;
  jsonViewStyle: CSSProperties;
  organizationId?: string;
  canShowExpressionTemplates?: boolean;
  onEditNode?: (nodeId: string) => void;
}) {
  const [mode, setMode] = useState<"form" | "json">("form");
  const configuration = section.tabData?.configuration;

  return (
    <TimelineAccordionCard
      value="runtime"
      status={{ dotClassName: "bg-blue-500", label: "Running" }}
      title="Runtime Config"
      trailing={
        <RuntimeHeaderActions mode={mode} nodeId={section.nodeId} onModeChange={setMode} onEditNode={onEditNode} />
      }
      jsonViewStyle={jsonViewStyle}
    >
      {mode === "form" ? (
        <RuntimeConfigForm
          section={section}
          value={configuration}
          jsonViewStyle={jsonViewStyle}
          organizationId={organizationId}
          canShowExpressionTemplates={canShowExpressionTemplates}
        />
      ) : (
        <JsonPayload value={configuration} jsonViewStyle={jsonViewStyle} />
      )}
    </TimelineAccordionCard>
  );
}

function RuntimeHeaderActions({
  mode,
  nodeId,
  onModeChange,
  onEditNode,
}: {
  mode: "form" | "json";
  nodeId: string;
  onModeChange: (mode: "form" | "json") => void;
  onEditNode?: (nodeId: string) => void;
}) {
  return (
    <span className="inline-flex items-center gap-2">
      <RuntimeViewToggle mode={mode} onChange={onModeChange} />
      {onEditNode ? (
        <HeaderIconButton
          label="Edit runtime config"
          icon={<Pencil className="h-3.5 w-3.5" />}
          onClick={() => onEditNode(nodeId)}
        />
      ) : null}
    </span>
  );
}

function RuntimeViewToggle({ mode, onChange }: { mode: "form" | "json"; onChange: (mode: "form" | "json") => void }) {
  return (
    <nav aria-label="Runtime config view" className={SEGMENTED_NAV_CLASSES}>
      {(["form", "json"] as const).map((item) => (
        <button
          key={item}
          type="button"
          aria-label={item === "form" ? "Form" : "JSON"}
          aria-pressed={mode === item}
          className={cn(segmentedNavTabClassName(mode === item), "text-xs")}
          onClick={(event) => {
            event.stopPropagation();
            onChange(item);
          }}
        >
          {item === "form" ? "Form" : "JSON"}
        </button>
      ))}
    </nav>
  );
}

function RuntimeConfigForm({
  section,
  value,
  jsonViewStyle,
  organizationId,
  canShowExpressionTemplates,
}: {
  section: RunInspectorNodeSection;
  value: unknown;
  jsonViewStyle: CSSProperties;
  organizationId?: string;
  canShowExpressionTemplates: boolean;
}) {
  const expressionPreviewContext = useMemo(() => buildRuntimeExpressionContext(section), [section]);

  if (!hasObjectValue(value)) {
    return <EmptySectionText>No runtime configuration for this step.</EmptySectionText>;
  }

  if (section.configurationFields.length > 0) {
    return (
      <RuntimeSchemaConfigForm
        fields={section.configurationFields}
        value={value}
        templateValues={canShowExpressionTemplates ? section.workflowNode?.configuration : undefined}
        organizationId={organizationId}
        expressionPreviewContext={expressionPreviewContext}
        expressionErrorMessage={section.errorMessage}
      />
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
  templateValues,
  organizationId,
  expressionPreviewContext,
  expressionErrorMessage,
}: {
  fields: RunInspectorNodeSection["configurationFields"];
  value: Record<string, unknown>;
  templateValues?: Record<string, unknown>;
  organizationId?: string;
  expressionPreviewContext?: Record<string, unknown> | null;
  expressionErrorMessage?: string;
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
            organizationId={organizationId}
            readOnly
            expressionPreviewContext={expressionPreviewContext}
            expressionErrorMessage={expressionErrorMessage}
            expressionTemplateValue={templateValues?.[field.name]}
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
    const displayValue = value === null ? "" : String(value);

    return (
      <label className="block space-y-1.5">
        <span className="text-sm font-medium text-slate-800 dark:text-gray-100">{label}</span>
        {displayValue.includes("\n") ? (
          <Textarea
            aria-label={label}
            readOnly
            value={displayValue}
            className="min-h-24 resize-y border-slate-300 bg-white text-slate-900 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100"
          />
        ) : (
          <Input
            aria-label={label}
            readOnly
            value={displayValue}
            className="h-9 border-slate-300 bg-white text-slate-900 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-100"
          />
        )}
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
