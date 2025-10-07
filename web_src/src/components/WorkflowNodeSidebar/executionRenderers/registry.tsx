import { ReactNode } from 'react'

export interface ExecutionRendererProps {
  execution: any
  isDarkMode: boolean
  workflowId: string
}

export interface CollapsedViewProps {
  execution: any
  onClick: () => void
}

export interface ExpandedViewProps {
  execution: any
  isDarkMode: boolean
  workflowId: string
}

export interface ExecutionRenderer {
  // Render the collapsed (non-expanded) view of the execution
  renderCollapsed?: (props: CollapsedViewProps) => ReactNode

  // Render the expanded view of the execution
  renderExpanded?: (props: ExpandedViewProps) => ReactNode

  // Render the execution details section (replaces the default execution details)
  renderDetails?: (props: ExecutionRendererProps) => ReactNode

  // Render the inputs section (replaces the default inputs display)
  renderInputs?: (props: ExecutionRendererProps) => ReactNode

  // Render the outputs section (replaces the default outputs display)
  renderOutputs?: (props: ExecutionRendererProps) => ReactNode

  // Render additional custom sections
  renderCustomSections?: (props: ExecutionRendererProps) => ReactNode
}

// Registry to map node types to their custom renderers
const executionRenderers: Record<string, ExecutionRenderer> = {}

export function registerExecutionRenderer(nodeType: string, renderer: ExecutionRenderer) {
  executionRenderers[nodeType] = renderer
}

export function getExecutionRenderer(nodeType: string): ExecutionRenderer | undefined {
  return executionRenderers[nodeType]
}

export function hasCustomRenderer(nodeType: string): boolean {
  return nodeType in executionRenderers
}
