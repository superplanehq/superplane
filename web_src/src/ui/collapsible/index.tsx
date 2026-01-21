"use client";

import * as CollapsiblePrimitive from "@radix-ui/react-collapsible";
import React from "react";
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
}) => {
  return (
    <div className="relative">
      <Collapsible open={open} onOpenChange={onOpenChange} disabled={disabled} className={className}>
        {children}
      </Collapsible>
    </div>
  );
};

export { Collapsible, CollapsibleContent, CollapsibleTrigger, CollapsibleWithActions };
