import React from "react";
import { ComponentHeader } from "../componentHeader";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { CollapsedComponent } from "../collapsedComponent";
import { StickyNote } from "lucide-react";
import { parseBasicMarkdown } from "@/utils/markdown";

export interface AnnotationComponentProps extends ComponentActionsProps {
  title: string;
  annotationText?: string;
  collapsed?: boolean;
  selected?: boolean;
  hideActionsButton?: boolean;
}

export const AnnotationComponent: React.FC<AnnotationComponentProps> = ({
  title,
  annotationText = "",
  collapsed = false,
  selected = false,
  onToggleCollapse,
  onEdit,
  onDuplicate,
  onDelete,
  hideActionsButton,
}) => {
  if (collapsed) {
    return (
      <SelectionWrapper selected={selected} fullRounded>
        <CollapsedComponent
          iconSlug="sticky-note"
          iconColor="text-yellow-600"
          iconBackground="bg-yellow-100"
          title={title}
          collapsedBackground="bg-yellow-50"
          shape="circle"
          onDoubleClick={onToggleCollapse}
          onEdit={onEdit}
          onDuplicate={onDuplicate}
          onDelete={onDelete}
          hideActionsButton={hideActionsButton}
        >
          <div className="flex flex-col items-center gap-1">
            <StickyNote size={16} className="text-yellow-600" />
            <span className="text-xs text-gray-600 truncate max-w-[150px]">
              {annotationText ? "Has content" : "Empty note"}
            </span>
          </div>
        </CollapsedComponent>
      </SelectionWrapper>
    );
  }

  const parsedContent = annotationText ? parseBasicMarkdown(annotationText) : "";

  return (
    <SelectionWrapper selected={selected}>
      <div className="relative flex flex-col outline-1 outline-slate-400 rounded-md w-[23rem] bg-yellow-50 border border-yellow-200">
        <ComponentHeader
          iconSlug="sticky-note"
          iconBackground="bg-yellow-100"
          iconColor="text-yellow-600"
          headerColor="bg-yellow-100"
          title={title}
          onDoubleClick={onToggleCollapse}
          onEdit={onEdit}
          onDuplicate={onDuplicate}
          onDelete={onDelete}
          hideActionsButton={hideActionsButton}
        />

        <div className="px-3 py-3 pt-1 min-h-[80px] text-sm text-gray-800">
          {annotationText ? (
            <div className="text-left prose prose-sm max-w-none" dangerouslySetInnerHTML={{ __html: parsedContent }} />
          ) : (
            <div className="text-gray-500 italic flex items-center gap-2">
              <StickyNote size={16} />
              Click Edit to add content to this note
            </div>
          )}
        </div>
      </div>
    </SelectionWrapper>
  );
};
