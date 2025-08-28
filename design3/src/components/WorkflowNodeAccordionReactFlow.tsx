import { memo } from 'react'
import { Handle, Position, NodeProps } from '@xyflow/react'
import { WorkflowNodeAccordion, type WorkflowNodeData } from './lib/WorkflowNode/workflow-node-accordion'
import { type AccordionItem } from './lib/Accordion/accordion'

// Error types for workflow nodes
export interface WorkflowNodeError {
  id: string
  type: 'connection' | 'configuration' | 'permission' | 'resource'
  severity: 'error' | 'warning'
  message: string
  description: string
  action?: string
}

export interface WorkflowNodeConnectionError extends WorkflowNodeError {
  type: 'connection'
  connectionType: 'semaphore' | 'github' | 'api'
  resourceName: string
  resourceType: 'project' | 'repository' | 'endpoint'
}

// React Flow node data interface for accordion version
export interface WorkflowNodeAccordionReactFlowData {
  workflowNodeData: WorkflowNodeData
  variant?: 'read' | 'edit'
  sections?: AccordionItem[]
  multiple?: boolean
  partialSave?: boolean
  saveGranular?: boolean
  onUpdate?: (data: Partial<WorkflowNodeData>) => void
  onDelete?: () => void
  onEdit?: () => void
  onSave?: () => void
  onCancel?: () => void
  onSelect?: () => void
  onResolveError?: (errorId: string) => void
  nodes?: any[]
  totalNodesCount?: number
  savedConnectionIndices?: number[]
  errors?: WorkflowNodeError[]
}

/**
 * React Flow wrapper for WorkflowNodeAccordion component
 * Handles React Flow-specific requirements (handles, positioning)
 */
export const WorkflowNodeAccordionReactFlow = memo(({ data, selected }: NodeProps) => {
  const {
    workflowNodeData,
    variant = 'read',
    sections,
    multiple = true,
    partialSave = false,
    saveGranular = false,
    onUpdate,
    onDelete,
    onEdit,
    onSave,
    onCancel,
    onSelect,
    onResolveError,
    nodes = [],
    totalNodesCount = 0,
    savedConnectionIndices = [],
    errors = []
  } = data as unknown as WorkflowNodeAccordionReactFlowData

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

  const handleSelect = () => {
    onSelect?.()
  }

  const handleResolveError = (errorId: string) => {
    onResolveError?.(errorId)
  }

  return (
    <div className="relative">
      {/* Input Handle */}
      <Handle
        type="target"
        position={Position.Left}
        className="w-3 h-3 !bg-zinc-400 !border-2 !border-white dark:!border-zinc-800"
      />
      
      {/* WorkflowNodeAccordion Component */}
      <WorkflowNodeAccordion
        data={workflowNodeData}
        variant={variant}
        selected={selected}
        sections={sections}
        multiple={multiple}
        partialSave={partialSave}
        saveGranular={saveGranular}
        className={selected ? 'ring-2 ring-blue-500 ring-offset-2' : ''}
        onUpdate={handleUpdate}
        onDelete={handleDelete}
        onEdit={handleEdit}
        onSave={handleSave}
        onCancel={handleCancel}
        onSelect={handleSelect}
        onResolveError={handleResolveError}
        nodes={nodes}
        totalNodesCount={totalNodesCount}
        savedConnectionIndices={savedConnectionIndices}
        errors={errors}
      />
      
      {/* Output Handle */}
      <Handle
        type="source"
        position={Position.Right}
        className="w-3 h-3 !bg-zinc-400 !border-2 !border-white dark:!border-zinc-800"
      />
    </div>
  )
})

WorkflowNodeAccordionReactFlow.displayName = 'WorkflowNodeAccordionReactFlow'