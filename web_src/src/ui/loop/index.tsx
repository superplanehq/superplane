import React from "react";
import { Button } from "../button";
import { CollapsedComponent } from "../collapsedComponent";
import { ComponentHeader } from "../componentHeader";
import { SelectionWrapper } from "../selectionWrapper";
import { ComponentActionsProps } from "../types/componentActions";

export interface LoopProps extends ComponentActionsProps {
  title: string;
  width: number;
  height: number;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  iconBackground?: string;
  headerColor?: string;
  collapsed?: boolean;
  selected?: boolean;
  childCount?: number;
  onAddChild?: () => void;
}

export const Loop: React.FC<LoopProps> = ({
  title,
  width,
  height,
  iconSrc,
  iconSlug,
  iconColor,
  iconBackground,
  headerColor = "bg-slate-100",
  selected = false,
  childCount = 0,
  onAddChild,
  onRun,
  runDisabled,
  runDisabledTooltip,
  onEdit,
  onConfigure,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  const hasChildren = childCount > 0;

  if (collapsed) {
    return (
      <SelectionWrapper selected={selected} fullRounded>
        <CollapsedComponent
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconColor={iconColor}
          iconBackground={iconBackground}
          title={title}
          collapsedBackground={headerColor}
          onRun={onRun}
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onEdit={onEdit}
          onConfigure={onConfigure}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        />
      </SelectionWrapper>
    );
  }

  return (
    <SelectionWrapper selected={selected} fullRounded>
      <div
        className="relative flex flex-col rounded-xl border border-dashed border-slate-300 bg-slate-50/80 text-left"
        style={{ width, height }}
      >
        <ComponentHeader
          iconSrc={iconSrc}
          iconSlug={iconSlug}
          iconColor={iconColor}
          iconBackground={iconBackground}
          headerColor={headerColor}
          title={title}
          onRun={onRun}
          runDisabled={runDisabled}
          runDisabledTooltip={runDisabledTooltip}
          onEdit={onEdit}
          onConfigure={onConfigure}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        />
        <div className="flex flex-1 items-start justify-between px-4 py-3">
          <div className="text-xs text-slate-500">
            {hasChildren ? `${childCount} node${childCount === 1 ? "" : "s"} in loop` : "Drop nodes here"}
          </div>
          {onAddChild && (
            <Button size="sm" variant="secondary" onClick={onAddChild}>
              Add Node
            </Button>
          )}
        </div>
      </div>
    </SelectionWrapper>
  );
};
