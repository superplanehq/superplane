"use client";

import * as CollapsiblePrimitive from "@radix-ui/react-collapsible";
import React from "react";
import { SidebarActionsDropdown } from "../componentSidebar/SidebarActionsDropdown";
import { ComponentActionsProps } from "../types/componentActions";

const Collapsible = CollapsiblePrimitive.Root;

const CollapsibleTrigger = CollapsiblePrimitive.Trigger;

const CollapsibleContent = CollapsiblePrimitive.Content;

export interface CollapsibleWithActionsProps extends ComponentActionsProps {
  children: React.ReactNode;
  className?: string;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
  disabled?: boolean;
}

const CollapsibleWithActions: React.FC<CollapsibleWithActionsProps> = ({
  children,
  className,
  open,
  onOpenChange,
  disabled,
  onRun,
  onDuplicate,
  onDeactivate,
  onToggleView,
  onDelete,
  isCompactView,
}) => {
  return (
    <div className="relative">
      <Collapsible open={open} onOpenChange={onOpenChange} disabled={disabled} className={className}>
        {children}
      </Collapsible>
      <div className="absolute top-0 left-1/2 transform -translate-x-1/2 -translate-y-2 z-10">
        <SidebarActionsDropdown
          onRun={onRun}
          onDuplicate={onDuplicate}
          onDeactivate={onDeactivate}
          onToggleView={onToggleView}
          onDelete={onDelete}
          isCompactView={isCompactView}
        />
      </div>
    </div>
  );
};

export { Collapsible, CollapsibleContent, CollapsibleTrigger, CollapsibleWithActions };
