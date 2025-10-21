import * as React from "react"
import ReactFlow, { Background, BackgroundVariant, Controls, MiniMap } from "reactflow"

import "reactflow/dist/style.css"

import { cn } from "@/lib/utils"

type CanvasProps = React.ComponentProps<typeof ReactFlow> & {
  containerClassName?: string
  containerStyle?: React.CSSProperties
  showControls?: boolean
  showMiniMap?: boolean
}

const Canvas: React.FC<CanvasProps> = ({
  containerClassName,
  containerStyle,
  nodes,
  edges,
  fitView = true,
  children,
  showControls = false,
  showMiniMap = false,
  ...props
}) => {
  const safeNodes = React.useMemo(() => nodes ?? [], [nodes])
  const safeEdges = React.useMemo(() => edges ?? [], [edges])

  return (
    <div
      className={cn("h-[480px] w-full overflow-hidden rounded-lg border", containerClassName)}
      style={containerStyle}
    >
      <ReactFlow nodes={safeNodes} edges={safeEdges} fitView={fitView} {...props}>
        {children}
        <Background variant={BackgroundVariant.Dots} gap={24} size={1} />
        {showMiniMap ? <MiniMap pannable zoomable /> : null}
        {showControls ? <Controls /> : null}
      </ReactFlow>
    </div>
  )
}

export { Canvas }
