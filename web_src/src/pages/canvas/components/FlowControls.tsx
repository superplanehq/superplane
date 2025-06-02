import React from "react";
import { Controls, ControlButton, Edge } from "@xyflow/react";
import { AllNodeType } from "../types/flow";

interface FlowControlsProps {
  onAutoLayout: (nodes: AllNodeType[], edges: Edge[]) => void;
  nodes: AllNodeType[];
  edges: Edge[];
}

export const FlowControls: React.FC<FlowControlsProps> = ({
  onAutoLayout,
  nodes,
  edges
}) => {
  return (
    <Controls>
      <ControlButton
        onClick={() => onAutoLayout(nodes, edges)}
        title="ELK Auto Layout"
      >
        <span className="material-icons" style={{ fontSize: 20 }}>
          account_tree
        </span>
      </ControlButton>
    </Controls>
  );
};