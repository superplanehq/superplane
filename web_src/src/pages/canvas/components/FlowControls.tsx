import React from "react";
import { Controls, ControlButton, Edge } from "@xyflow/react";
import { AllNodeType } from "../types/flow";
import { MaterialSymbol } from "@/components/MaterialSymbol/material-symbol";

interface FlowControlsProps {
  onAutoLayout: (nodes: AllNodeType[], edges: Edge[]) => void;
  nodes: AllNodeType[];
  edges: Edge[];
  isLocked: boolean;
  onLockToggle: (isLocked: boolean) => void;
}

export const FlowControls: React.FC<FlowControlsProps> = ({
  onAutoLayout,
  nodes,
  edges,
  isLocked,
  onLockToggle
}) => {
  return (
    <Controls position="bottom-right" showInteractive={false}  >
      <ControlButton
        onClick={() => onLockToggle(!isLocked)}
        title="Lock"
      >
        <MaterialSymbol name={isLocked ? "lock" : "lock_open"} />
      </ControlButton>

      <ControlButton
        onClick={() => onAutoLayout(nodes, edges)}
        title="ELK Auto Layout"
      >
        <MaterialSymbol name="account_tree" />
      </ControlButton>
    </Controls>
  );
};