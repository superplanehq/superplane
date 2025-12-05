import { ComponentBase, type EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";
import { useMemo } from "react";
import { Handle, Position } from "@xyflow/react";
import { parseExpression } from "@/lib/expressionParser";

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
  collapsedBackground?: string;
}

const HANDLE_STYLE = {
  width: 12,
  height: 12,
  border: "3px solid #C9D5E1",
  background: "transparent",
};

export const If: React.FC<IfProps> = ({
  title = "If processed events",
  expression,
  trueEvent,
  falseEvent,
  trueSectionLabel = "TRUE",
  falseSectionLabel = "FALSE",
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
  hideHandle = false,
}) => {
  const conditions = useMemo(() => parseExpression(expression || ""), [expression]);

  const specs = expression
    ? [
        {
          title: "condition",
          tooltipTitle: "conditions applied",
          values: conditions,
        },
      ]
    : undefined;

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
      ),
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
      ),
    });
  }

  return (
    <ComponentBase
      title={title}
      iconSlug="split"
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
