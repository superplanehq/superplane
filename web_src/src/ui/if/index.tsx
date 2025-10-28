import { ComponentBase, type EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";

export interface IfCondition {
  field: string;
  operator: string;
  value: string;
  logicalOperator?: "AND" | "OR";
}

export interface IfProps extends ComponentActionsProps {
  title?: string;
  conditions: IfCondition[];
  trueEvent?: Omit<EventSection, "title">;
  falseEvent?: Omit<EventSection, "title">;
  trueSectionLabel?: string;
  falseSectionLabel?: string;
  collapsed?: boolean;
  selected?: boolean;
}

export const If: React.FC<IfProps> = ({
  title = "If processed events",
  conditions,
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
}) => {
  const spec = conditions.length > 0 ? {
    title: "condition",
    tooltipTitle: "conditions applied",
    values: conditions.map(condition => ({
      badges: [
        { label: condition.field, bgColor: "bg-purple-100", textColor: "text-purple-700" },
        { label: condition.operator, bgColor: "bg-gray-100", textColor: "text-gray-700" },
        { label: condition.value, bgColor: "bg-green-100", textColor: "text-green-700" },
        ...(condition.logicalOperator ? [{ label: condition.logicalOperator, bgColor: "bg-gray-500", textColor: "text-white" }] : [])
      ]
    }))
  } : undefined;

  const eventSections: EventSection[] = [];
  if (trueEvent) {
    eventSections.push({
      title: trueSectionLabel,
      ...trueEvent
    });
  }
  if (falseEvent) {
    eventSections.push({
      title: falseSectionLabel,
      ...falseEvent
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