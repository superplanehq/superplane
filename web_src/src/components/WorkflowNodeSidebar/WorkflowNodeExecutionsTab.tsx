import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { useState, useEffect } from 'react'
import { ExecutionItem } from './ExecutionItem'
import { useNodeExecutions } from '../../hooks/useWorkflowData'
import { WorkflowsWorkflowNodeExecution } from '@/api-client'

interface WorkflowNodeExecutionsTabProps {
  workflowId: string
  nodeId: string
  isBlueprintNode?: boolean
  componentName?: string
  organizationId: string
  blueprintId?: string
}

export const WorkflowNodeExecutionsTab = ({ workflowId, nodeId, isBlueprintNode, componentName, organizationId, blueprintId }: WorkflowNodeExecutionsTabProps) => {
  const [isDarkMode, setIsDarkMode] = useState(false)
  const [expandedExecutionIds, setExpandedExecutionIds] = useState<Set<string>>(new Set())

  // Detect dark mode
  useEffect(() => {
    const checkDarkMode = () => {
      setIsDarkMode(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches)
    }

    checkDarkMode()

    const observer = new MutationObserver(checkDarkMode)
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class']
    })

    return () => observer.disconnect()
  }, [])

  const { data, isLoading, error } = useNodeExecutions(workflowId, nodeId, {
    states: ['STATE_STARTED', 'STATE_FINISHED']
  })

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mb-3"></div>
        <p className="text-sm text-gray-500 dark:text-zinc-400">Loading executions...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6">
        <MaterialSymbol name="error" className="text-red-500 mb-3" size="xl" />
        <p className="text-sm text-gray-900 dark:text-zinc-100 font-medium mb-1">Failed to load executions</p>
        <p className="text-xs text-gray-500 dark:text-zinc-400">{error.message}</p>
      </div>
    )
  }

  const executions = data?.executions || []

  const handleToggleChildExecutions = (executionId: string) => {
    const newSet = new Set(expandedExecutionIds)
    if (newSet.has(executionId)) {
      newSet.delete(executionId)
    } else {
      newSet.add(executionId)
    }
    setExpandedExecutionIds(newSet)
  }

  if (executions.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6">
        <MaterialSymbol name="history" className="text-gray-400 dark:text-zinc-500 mb-3" size="xl" />
        <p className="text-sm text-gray-600 dark:text-zinc-400 font-medium mb-1">No executions yet</p>
        <p className="text-xs text-gray-500 dark:text-zinc-500 text-center">
          Execution history will appear here once this node starts running
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex-shrink-0 px-4 py-3 border-b border-zinc-200 dark:border-zinc-800">
        <p className="text-xs text-gray-500 dark:text-zinc-400">
          {executions.length} {executions.length === 1 ? 'execution' : 'executions'}
        </p>
      </div>
      <div className="flex-1 overflow-y-auto">
        {executions.map((execution: WorkflowsWorkflowNodeExecution) => (
          <ExecutionItem
            key={execution.id}
            execution={execution}
            isDarkMode={isDarkMode}
            workflowId={workflowId}
            isBlueprintNode={isBlueprintNode}
            componentName={componentName}
            organizationId={organizationId}
            blueprintId={blueprintId}
            onToggleChildExecutions={handleToggleChildExecutions}
            isChildExpanded={expandedExecutionIds.has(execution.id!)}
          />
        ))}
      </div>
    </div>
  )
}
