import { memo } from 'react'
import { Handle, Position, NodeProps } from '@xyflow/react'
import { WorkflowNode, type WorkflowNodeData } from './lib/WorkflowNode/workflow-node'

// React Flow node data interface
export interface WorkflowNodeReactFlowData {
  workflowNodeData: WorkflowNodeData
  variant?: 'read' | 'edit'
  onUpdate?: (data: Partial<WorkflowNodeData>) => void
  onDelete?: () => void
  onEdit?: () => void
  onSave?: () => void
  onCancel?: () => void
}

/**
 * React Flow wrapper for WorkflowNode component
 * Handles React Flow-specific requirements (handles, positioning)
 */
export const WorkflowNodeReactFlow = memo(({ data, selected }: NodeProps) => {
  const {
    workflowNodeData,
    variant = 'read',
    onUpdate,
    onDelete,
    onEdit,
    onSave,
    onCancel
  } = data as unknown as WorkflowNodeReactFlowData

  const handleUpdate = (updates: Partial<WorkflowNodeData>) => {
    onUpdate?.(updates)
  }

  const handleDelete = () => {
    onDelete?.()
  }

  const handleEdit = () => {
    onEdit?.()
  }

  const handleSave = () => {
    onSave?.()
  }

  const handleCancel = () => {
    onCancel?.()
  }

  return (
    <div className="relative">
      {/* Input Handle */}
      <Handle
        type="target"
        position={Position.Top}
        className="w-3 h-3 !bg-zinc-400 !border-2 !border-white dark:!border-zinc-800"
      />
      
      {/* WorkflowNode Component */}
      <WorkflowNode
        data={workflowNodeData}
        variant={variant}
        className={selected ? 'ring-2 ring-blue-500 ring-offset-2' : ''}
        onUpdate={handleUpdate}
        onDelete={handleDelete}
        onEdit={handleEdit}
        onSave={handleSave}
        onCancel={handleCancel}
      />
      
      {/* Output Handle */}
      <Handle
        type="source"
        position={Position.Bottom}
        className="w-3 h-3 !bg-zinc-400 !border-2 !border-white dark:!border-zinc-800"
      />
    </div>
  )
})

WorkflowNodeReactFlow.displayName = 'WorkflowNodeReactFlow'