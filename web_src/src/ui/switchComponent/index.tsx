import { Handle, Position } from "@xyflow/react";
import { ComponentBase, type EventSection } from "../componentBase";
import { ComponentActionsProps } from "../types/componentActions";

export interface SwitchStage {
  pathName: string;
  field: string;
  operator: string;
  value: string;
  receivedAt?: Date;
  eventState?: "success" | "failed";
  eventTitle?: string;
}

export interface SwitchComponentProps extends ComponentActionsProps {
  title?: string;
  stages: SwitchStage[];
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

export const SwitchComponent: React.FC<SwitchComponentProps> = ({
  title = "Branch processed events",
  stages,
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
  const spec = stages.length > 0 ? {
    title: "path",
    tooltipTitle: "paths applied",
    values: stages.map(stage => ({
      badges: [
        { label: stage.pathName, bgColor: "bg-gray-500", textColor: "text-white" },
        { label: stage.field, bgColor: "bg-purple-100", textColor: "text-purple-700" },
        { label: stage.operator, bgColor: "bg-gray-100", textColor: "text-gray-700" },
        { label: stage.value, bgColor: "bg-green-100", textColor: "text-green-700" }
      ]
    }))
  } : undefined;

  const eventSections: EventSection[] = stages.map(stage => ({
    title: stage.pathName,
    receivedAt: stage.receivedAt,
    eventState: stage.eventState,
    eventTitle: stage.eventTitle,
    handleComponent: hideHandle ? undefined : (
      <Handle
        type="source"
        position={Position.Right}
        id={stage.pathName}
        style={{
          ...HANDLE_STYLE,
          right: -20,
          top: "50%",
          transform: "translateY(-50%)",
        }}
      />
    )
  }));

  return (
    <ComponentBase
      title={title}
      iconSlug="git-branch"
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