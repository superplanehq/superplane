import React from "react";
import { ComponentHeader } from "../componentHeader";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";
import { CollapsedComponent } from "../collapsedComponent";
import { StickyNote } from "lucide-react";
import ReactMarkdown from "react-markdown";

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
            <div className="text-left prose prose-sm max-w-none prose-headings:mt-3 prose-headings:mb-1 prose-p:my-1 prose-ul:my-1 prose-ol:my-1 prose-li:my-0">
              <ReactMarkdown
                disallowedElements={["script", "iframe", "object", "embed"]}
                unwrapDisallowed={true}
                components={{
                  h1: ({ children }) => <h1 className="text-lg font-bold text-yellow-800">{children}</h1>,
                  h2: ({ children }) => <h2 className="text-base font-bold text-yellow-800">{children}</h2>,
                  h3: ({ children }) => <h3 className="text-sm font-bold text-yellow-800">{children}</h3>,
                  a: ({ href, children }) => {
                    const safeHref = href && (href.startsWith("http://") || href.startsWith("https://")) ? href : "#";
                    return (
                      <a
                        href={safeHref}
                        className="text-yellow-700 underline hover:text-yellow-900"
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        {children}
                      </a>
                    );
                  },
                  code: ({ children }) => (
                    <code className="bg-yellow-100 px-1 py-0.5 rounded text-xs font-mono text-yellow-900">
                      {children}
                    </code>
                  ),
                  pre: ({ children }) => (
                    <pre className="bg-yellow-100 p-2 rounded text-xs overflow-x-auto">{children}</pre>
                  ),
                }}
              >
                {annotationText}
              </ReactMarkdown>
            </div>
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
