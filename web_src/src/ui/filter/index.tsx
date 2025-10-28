import { ComponentBase, type EventSection } from "../componentBase";

export interface FilterCondition {
  field: string;
  operator: string;
  value: string;
  logicalOperator?: "AND" | "OR";
}

export interface FilterProps {
  title?: string;
  filters: FilterCondition[];
  lastEvent?: Omit<EventSection, "title">;
  collapsed?: boolean;
  selected?: boolean;
}

export const Filter: React.FC<FilterProps> = ({
  title = "Filter events based on branch",
  filters,
  lastEvent,
  collapsed = false,
  selected = false,
}) => {
  const spec = filters.length > 0 ? {
    title: "filter",
    tooltipTitle: "filters applied",
    values: filters.map(filter => ({
      badges: [
        { label: filter.field, bgColor: "bg-purple-100", textColor: "text-purple-700" },
        { label: filter.operator, bgColor: "bg-gray-100", textColor: "text-gray-700" },
        { label: filter.value, bgColor: "bg-green-100", textColor: "text-green-700" },
        ...(filter.logicalOperator ? [{ label: filter.logicalOperator, bgColor: "bg-gray-500", textColor: "text-white" }] : [])
      ]
    }))
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
    />
  );
};