import React from "react";
import { BlockContent } from "./content";
import { LeftHandle, RightHandle } from "./handles";
import type { BlockProps } from "./types";

export type { BlockData, BlockProps } from "./types";

export const Block = React.memo(function Block(props: BlockProps) {
  const data = props.data;
  const isHighlighted = data._isHighlighted || false;
  const hasHighlightedNodes = data._hasHighlightedNodes || false;
  const shouldDim = hasHighlightedNodes && !isHighlighted;

  return (
    <div className={`relative w-fit ${shouldDim ? "opacity-30" : ""}`} onClick={(e) => props.onClick?.(e)}>
      <LeftHandle data={data} nodeId={props.nodeId} />
      <BlockContent {...props} />
      <RightHandle data={data} nodeId={props.nodeId} />
    </div>
  );
});
