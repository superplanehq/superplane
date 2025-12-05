import { ComponentBase, type EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";
import { useMemo } from "react";
import { parseExpression } from "@/lib/expressionParser";

export interface FilterProps extends ComponentActionsProps {
  title?: string;
  expression?: string;
  lastEvent?: Omit<EventSection, "title">;
  collapsed?: boolean;
  selected?: boolean;
  collapsedBackground?: string;
}

export const Filter: React.FC<FilterProps> = ({
  title = "Filter events based on branch",
  expression,
  lastEvent,
  collapsed = false,
  selected = false,
  collapsedBackground,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onEdit,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  const filters = useMemo(() => parseExpression(expression || ""), [expression]);

  const specs = expression
    ? [
        {
          title: "filter",
          tooltipTitle: "filters applied",
          values: filters,
        },
      ]
    : undefined;

  const eventSections: EventSection[] = [];
  if (lastEvent) {
    eventSections.push({
      title: "Last Event",
      ...lastEvent,
    });
  }

  return (
    <ComponentBase
      title={title}
      iconSlug="filter"
      headerColor="bg-gray-50"
      specs={specs}
      eventSections={eventSections}
      collapsed={collapsed}
      collapsedBackground={collapsedBackground}
      selected={selected}
      onRun={onRun}
      runDisabled={runDisabled}
      runDisabledTooltip={runDisabledTooltip}
      onEdit={onEdit}
      onDuplicate={onDuplicate}
      onDeactivate={onDeactivate}
      onToggleView={onToggleView}
      onDelete={onDelete}
      isCompactView={isCompactView}
    />
  );
};
