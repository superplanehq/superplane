import { useState, ComponentType, useRef } from 'react'
import { NodeProps } from '@xyflow/react'
import { NodeActionButtons } from './NodeActionButtons'

interface WorkflowNodeWrapperProps extends NodeProps {
  BaseNodeComponent: ComponentType<NodeProps>
  onEdit?: () => void
  onEmit?: () => void
}

export const WorkflowNodeWrapper = ({
  BaseNodeComponent,
  onEdit,
  onEmit,
  ...nodeProps
}: WorkflowNodeWrapperProps) => {
  const [isHovered, setIsHovered] = useState(false)
  const hoverTimeoutRef = useRef<NodeJS.Timeout | null>(null)

  const handleMouseEnter = () => {
    if (hoverTimeoutRef.current) {
      clearTimeout(hoverTimeoutRef.current)
      hoverTimeoutRef.current = null
    }
    setIsHovered(true)
  }

  const handleMouseLeave = () => {
    // Add a small delay before hiding to allow moving to the buttons
    hoverTimeoutRef.current = setTimeout(() => {
      setIsHovered(false)
    }, 100)
  }

  return (
    <div
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      className="relative"
    >
      {isHovered && (
        <div onMouseEnter={handleMouseEnter} onMouseLeave={handleMouseLeave}>
          <NodeActionButtons onEdit={onEdit} onEmit={onEmit} />
        </div>
      )}
      <BaseNodeComponent {...nodeProps} />
    </div>
  )
}
