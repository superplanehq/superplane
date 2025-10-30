import { splitBySpaces } from "@/lib/utils";
import { ComponentBase, ComponentBaseSpecValue, type EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";
import { useMemo } from "react";

export interface FilterProps extends ComponentActionsProps {
  title?: string;
  expression: string;
  lastEvent?: Omit<EventSection, "title">;
  collapsed?: boolean;
  selected?: boolean;
}

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
  if (value.startsWith("\"") && value.endsWith("\"")) return true;
  if (!isNaN(Number(value))) return true;

  return false;
}

export const Filter: React.FC<FilterProps> = ({
  title = "Filter events based on branch",
  expression,
  lastEvent,
  collapsed = false,
  selected = false,
  onRun,
  onEdit,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  const filters = useMemo(() => {
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
    title: "filter",
    tooltipTitle: "filters applied",
    values: filters
  } : undefined;

  const eventSections: EventSection[] = [];
  if (lastEvent) {
    eventSections.push({
      title: "Last Event",
      ...lastEvent
    });
  }

  return (
    <ComponentBase
      title={title}
      iconSlug="filter"
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