import { useCallback } from "react";
import { BaseEdge, EdgeLabelRenderer, EdgeProps, getBezierPath, useReactFlow } from "@xyflow/react";
import { CircleX } from "lucide-react";

export function CustomEdge({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  style = {},
  selected,
  data,
}: EdgeProps) {
  const { setEdges } = useReactFlow();
  const isHovered = data?.isHovered || false;

  const [edgePath, labelX, labelY] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });

  const onEdgeClick = useCallback(() => {
    setEdges((edges) =>
      edges.map((edge) => ({
        ...edge,
        selected: edge.id === id,
      })),
    );
  }, [id, setEdges]);

  const onDeleteClick = useCallback(
    (event: React.MouseEvent) => {
      event.stopPropagation();
      setEdges((edges) => edges.filter((edge) => edge.id !== id));
    },
    [id, setEdges],
  );

  // Update style based on selection and hover state
  const edgeStyle = {
    ...style,
    stroke: selected || isHovered ? "#3B82F6" : style.stroke || "#C9D5E1",
    strokeWidth: selected ? 3 : style.strokeWidth || 3,
  };
  const isActive = selected || isHovered;

  return (
    <>
      <BaseEdge
        path={edgePath}
        style={edgeStyle}
        onClick={onEdgeClick}
        interactionWidth={20}
        className={isHovered ? "hovered" : undefined}
      />
      <EdgeLabelRenderer>
        <div
          style={{
            position: "absolute",
            transform: `translate(-50%, -50%) translate(${labelX}px,${labelY}px) scale(var(--edge-label-scale, 1))`,
            width: "40px",
            height: "40px",
            zIndex: 1001,
          }}
          className={`edge-label nodrag nopan group flex items-center justify-center${isActive ? " edge-label-visible" : ""}`}
        >
          <button
            className="edge-label-button flex items-center justify-center bg-red-100 rounded-full shadow-lg transition-all cursor-pointer"
            onClick={onDeleteClick}
            aria-label="Delete edge"
          >
            <CircleX size={20} className="text-red-500" />
          </button>
        </div>
      </EdgeLabelRenderer>
    </>
  );
}
