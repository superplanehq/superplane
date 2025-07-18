import { memo } from 'react'
import { Handle, Position, NodeProps } from '@xyflow/react'
import { WorkflowNodeAccordion, type WorkflowNodeData } from './lib/WorkflowNode/workflow-node-accordion'
import { type AccordionItem } from './lib/Accordion/accordion'

// React Flow node data interface for accordion version
export interface WorkflowNodeAccordionReactFlowData {
  workflowNodeData: WorkflowNodeData
  variant?: 'read' | 'edit'
  sections?: AccordionItem[]
  multiple?: boolean
  partialSave?: boolean
  saveGranular?: boolean
  modalEdit?: boolean
  onConnectionModalOpen?: () => void
  onUpdate?: (data: Partial<WorkflowNodeData>) => void
  onDelete?: () => void
  onEdit?: () => void
  onSave?: () => void
  onCancel?: () => void
  onSelect?: () => void
  nodes?: any[]
  totalNodesCount?: number
  savedConnectionIndices?: number[]
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
    modalEdit = false,
    onConnectionModalOpen,
    onUpdate,
    onDelete,
    onEdit,
    onSave,
    onCancel,
    onSelect,
    nodes = [],
    totalNodesCount = 0,
    savedConnectionIndices = []
  } = data as unknown as WorkflowNodeAccordionReactFlowData
  
  console.log('WorkflowNodeAccordionReactFlow received nodes:', nodes);
  console.log('WorkflowNodeAccordionReactFlow received totalNodesCount:', totalNodesCount);
  console.log('WorkflowNodeAccordionReactFlow received modalEdit:', modalEdit);
  console.log('WorkflowNodeAccordionReactFlow full data:', data);
  console.log('Data keys:', Object.keys(data));

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

  return (
    <div className="relative">
      {/* Input Handle */}
      <Handle
        type="target"
        position={Position.Top}
        className="w-3 h-3 !bg-zinc-400 !border-2 !border-white dark:!border-zinc-800"
      />
      
      {/* WorkflowNodeAccordion Component */}
      <WorkflowNodeAccordion
        data={workflowNodeData}
        variant={variant}
        sections={sections}
        multiple={multiple}
        partialSave={partialSave}
        saveGranular={saveGranular}
        modalEdit={modalEdit}
        onConnectionModalOpen={onConnectionModalOpen}
        className={selected ? 'ring-2 ring-blue-500 ring-offset-2' : ''}
        onUpdate={handleUpdate}
        onDelete={handleDelete}
        onEdit={handleEdit}
        onSave={handleSave}
        onCancel={handleCancel}
        onSelect={handleSelect}
        nodes={nodes}
        totalNodesCount={totalNodesCount}
        savedConnectionIndices={savedConnectionIndices}
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

WorkflowNodeAccordionReactFlow.displayName = 'WorkflowNodeAccordionReactFlow'