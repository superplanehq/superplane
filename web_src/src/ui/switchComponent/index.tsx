import { ComponentBase, type EventSection } from "../componentBase";

export interface SwitchStage {
  pathName: string;
  field: string;
  operator: string;
  value: string;
  receivedAt?: Date;
  eventState?: "success" | "failed";
  eventTitle?: string;
}

export interface SwitchComponentProps {
  title?: string;
  stages: SwitchStage[];
  collapsed?: boolean;
}

export const SwitchComponent: React.FC<SwitchComponentProps> = ({
  title = "Branch processed events",
  stages,
  collapsed = false,
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
    eventTitle: stage.eventTitle
  }));

  return (
    <ComponentBase
      title={title}
      iconSlug="git-branch"
      headerColor="bg-gray-50"
      spec={spec}
      eventSections={eventSections}
      collapsed={collapsed}
    />
  );
};