import { useQuery } from '@tanstack/react-query'
import { workflowsListNodeExecutions } from '../../api-client/sdk.gen'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { withOrganizationHeader } from '../../utils/withOrganizationHeader'
import { useState, useEffect } from 'react'
import { ExecutionItem } from './ExecutionItem'

interface WorkflowNodeExecutionsTabProps {
  workflowId: string
  nodeId: string
  isBlueprintNode?: boolean
  nodeType?: string
}

export const WorkflowNodeExecutionsTab = ({ workflowId, nodeId, isBlueprintNode, nodeType }: WorkflowNodeExecutionsTabProps) => {
  const [isDarkMode, setIsDarkMode] = useState(false)

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

  const { data, isLoading, error } = useQuery({
    queryKey: ['workflow-node-executions', workflowId, nodeId],
    queryFn: async () => {
      const response = await workflowsListNodeExecutions(
        withOrganizationHeader({
          path: {
            workflowId: workflowId,
            nodeId: nodeId,
          },
        })
      )
      return response.data
    },
    refetchInterval: 5000, // Refresh every 5 seconds
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
        {executions.map((execution: any) => (
          <ExecutionItem
            key={execution.id}
            execution={execution}
            isDarkMode={isDarkMode}
            workflowId={workflowId}
            isBlueprintNode={isBlueprintNode}
            nodeType={nodeType}
          />
        ))}
      </div>
    </div>
  )
}
