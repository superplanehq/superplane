import { splitBySpaces } from "@/lib/utils";
import { ComponentBase, ComponentBaseSpecValue, type EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";
import { useMemo } from "react";
import { Handle, Position } from "@xyflow/react";

export interface IfProps extends ComponentActionsProps {
  title?: string;
  expression?: string;
  trueEvent?: Omit<EventSection, "title">;
  falseEvent?: Omit<EventSection, "title">;
  trueSectionLabel?: string;
  falseSectionLabel?: string;
  collapsed?: boolean;
  selected?: boolean;
  hideHandle?: boolean;
}

const HANDLE_STYLE = {
  width: 12,
  height: 12,
  border: "3px solid #C9D5E1",
  background: "transparent",
};

const operators = new Set([
  ">=",
  "<=",
  "==",
  "!=",
  ">",
  "<",
  "contains",
  "startswith",
  "endswith",
  "matches",
  "in",
  "!",
  "+",
  "-",
  "*",
  "/",
  "%",
  "**",
  "??",
  "?",
  ":",
]);

const logicalOperators = new Set([
  "and",
  "or",
  "||",
  "&&"
]);

const isStaticValue = (value: string) => {
  if (value === "true" || value === "false") return true;
  if (value === "null" || value === "undefined") return true;
  if (value.startsWith("'") && value.endsWith("'")) return true;
  if (value.startsWith('"') && value.endsWith('"')) return true;
  if (!isNaN(Number(value))) return true;

  return false;
}

export const If: React.FC<IfProps> = ({
  title = "If processed events",
  expression,
  trueEvent,
  falseEvent,
  trueSectionLabel = "TRUE",
  falseSectionLabel = "FALSE",
  collapsed = false,
  selected = false,
  onRun,
  onEdit,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
  hideHandle = false,
}) => {
  const conditions = useMemo(() => {
    if (!expression) return [];
    const result: ComponentBaseSpecValue[] = [];
    const splittedExpression = splitBySpaces(expression);
    let current: ComponentBaseSpecValue = {
      badges: []
    };
    for (const term of splittedExpression) {
      const normalizedTerm = term.trim().toLowerCase();
      if (operators.has(normalizedTerm)) {
        current.badges.push({ label: term, bgColor: "bg-gray-100", textColor: "text-gray-700" });
      } else if (logicalOperators.has(normalizedTerm)) {
        current.badges.push({ label: term, bgColor: "bg-gray-500", textColor: "text-white" });
        result.push(current);
        current = {
          badges: []
        };
      } else if (isStaticValue(normalizedTerm)) {
        current.badges.push({ label: term, bgColor: "bg-green-100", textColor: "text-green-700" });
      } else {
        current.badges.push({ label: term, bgColor: "bg-purple-100", textColor: "text-purple-700" });
      }
    }

    result.push(current);
    return result;
  }, [expression]);

  const spec = expression ? {
    title: "condition",
    tooltipTitle: "conditions applied",
    values: conditions
  } : undefined;

  const eventSections: EventSection[] = [];
  if (trueEvent) {
    eventSections.push({
      title: trueSectionLabel,
      ...trueEvent,
      handleComponent: hideHandle ? undefined : (
        <Handle
          type="source"
          position={Position.Right}
          id="true"
          style={{
            ...HANDLE_STYLE,
            right: -20,
            top: "50%",
            transform: "translateY(-50%)",
          }}
        />
      )
    });
  }
  if (falseEvent) {
    eventSections.push({
      title: falseSectionLabel,
      ...falseEvent,
      handleComponent: hideHandle ? undefined : (
        <Handle
          type="source"
          position={Position.Right}
          id="false"
          style={{
            ...HANDLE_STYLE,
            right: -20,
            top: "50%",
            transform: "translateY(-50%)",
          }}
        />
      )
    });
  }

  return (
    <ComponentBase
      title={title}
      iconSlug="split"
      headerColor="bg-gray-50"
      spec={spec}
      eventSections={eventSections}
      collapsed={collapsed}
      selected={selected}
      onRun={onRun}
      onEdit={onEdit}
      onDuplicate={onDuplicate}
      onDeactivate={onDeactivate}
      onToggleView={onToggleView}
      onDelete={onDelete}
      isCompactView={isCompactView}
    />
  );
};