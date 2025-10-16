import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useWorkflow, useWorkflowEvents, useEventExecutions } from '../../hooks/useWorkflowData'
import { Button } from '../../components/ui/button'
import { MaterialSymbol } from '../../components/MaterialSymbol/material-symbol'
import { Heading } from '../../components/Heading/heading'
import { Text } from '../../components/Text/text'
import { Item, ItemMedia, ItemContent, ItemTitle, ItemDescription } from '../../components/ui/item'
import { formatTimeAgo } from '../../utils/date'
import { ExecutionItem } from '../../components/WorkflowNodeSidebar/ExecutionItem'
import { ChildExecutions } from '../../components/ChildExecutions'
import { WorkflowsWorkflowNodeExecution } from '../../api-client'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'

export const WorkflowEvents = () => {
  const { organizationId, workflowId } = useParams<{ organizationId: string; workflowId: string }>()
  const navigate = useNavigate()
  const [expandedEventId, setExpandedEventId] = useState<string | null>(null)
  const [expandedExecutionIds, setExpandedExecutionIds] = useState<Set<string>>(new Set())

  const { data: workflow, isLoading: workflowLoading } = useWorkflow(organizationId!, workflowId!)
  const { data: eventsData, isLoading: eventsLoading } = useWorkflowEvents(workflowId!)
  const { data: executionsData } = useEventExecutions(workflowId!, expandedEventId)

  const events = eventsData?.events || []
  const executions = executionsData?.executions || []

  // Create a map of nodeId to node information for workflow nodes only
  const [nodeMap, setNodeMap] = useState<Map<string, { blockName: string; isBlueprint: boolean; nodeName: string; blueprintId?: string }>>(new Map())

  useEffect(() => {
    const map = new Map<string, { blockName: string; isBlueprint: boolean; nodeName: string; blueprintId?: string }>()

    // Add workflow nodes
    if (workflow?.nodes) {
      workflow.nodes.forEach((node: any) => {
        const isComponent = node.type === 'TYPE_COMPONENT'
        const blockName = isComponent ? node.component?.name : node.blueprint?.name
        const blueprintId = !isComponent ? node.blueprint?.id : undefined
        map.set(node.id, {
          blockName,
          isBlueprint: !isComponent,
          nodeName: node.name,
          blueprintId,
        })
      })
    }

    setNodeMap(map)
  }, [workflow])

  // Build a simple execution chain by following previousExecutionId
  const buildExecutionChain = (executions: WorkflowsWorkflowNodeExecution[]): WorkflowsWorkflowNodeExecution[] => {
    if (!executions || executions.length === 0) return []

    // Find root execution (no previousExecutionId)
    const rootExecution = executions.find((exec) => !exec.previousExecutionId)
    if (!rootExecution) return executions // Return all if no root found

    const chain: WorkflowsWorkflowNodeExecution[] = []
    const visited = new Set<string>()
    let current: WorkflowsWorkflowNodeExecution | undefined = rootExecution

    // Build the chain by following previousExecutionId links
    while (current && !visited.has(current.id!)) {
      chain.push(current)
      visited.add(current.id!)

      // Find the next execution in the chain
      current = executions.find((exec) => exec.previousExecutionId === current!.id)
    }

    // Add any unvisited executions at the end
    executions.forEach((exec) => {
      if (!visited.has(exec.id!)) {
        chain.push(exec)
      }
    })

    return chain
  }

  const handleToggleChildExecutions = (executionId: string) => {
    const newSet = new Set(expandedExecutionIds)
    if (newSet.has(executionId)) {
      newSet.delete(executionId)
    } else {
      newSet.add(executionId)
    }
    setExpandedExecutionIds(newSet)
  }

  const handleEventClick = (eventId: string) => {
    if (expandedEventId === eventId) {
      setExpandedEventId(null)
    } else {
      setExpandedEventId(eventId)
    }
  }

  if (workflowLoading || eventsLoading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <p className="ml-3 text-gray-500">Loading events...</p>
      </div>
    )
  }

  if (!workflow) {
    return (
      <div className="flex flex-col items-center justify-center h-screen">
        <MaterialSymbol name="error" className="text-red-500 mb-4" size="xl" />
        <Heading level={2}>Workflow not found</Heading>
        <Button variant="outline" onClick={() => navigate(`/${organizationId}`)} className="mt-4">
          Go back to home
        </Button>
      </div>
    )
  }

  return (
    <div className="h-screen flex flex-col bg-zinc-50 dark:bg-zinc-950">
      {/* Header */}
      <div className="bg-white dark:bg-zinc-900 border-b border-zinc-200 dark:border-zinc-800 p-4 flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button variant="ghost" size="icon" onClick={() => navigate(`/${organizationId}/workflows/${workflowId}`)}>
            <MaterialSymbol name="arrow_back" />
          </Button>
          <div>
            <Heading level={2} className="!text-xl !mb-0">Event Execution Chains</Heading>
            <Text className="text-sm text-zinc-600 dark:text-zinc-400">{workflow.name}</Text>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onClick={() => navigate(`/${organizationId}/workflows/${workflowId}`)}
          >
            <MaterialSymbol name="edit" />
            Edit Workflow
          </Button>
        </div>
      </div>

      {/* Main content */}
      <div className="flex-1 overflow-auto p-6">
        <div className="max-w-4xl mx-auto">
          {events.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12">
              <MaterialSymbol name="event" size="xl" className="text-zinc-400 dark:text-zinc-600 mb-4" />
              <Heading level={3} className="!text-zinc-600 dark:text-zinc-400">No events yet</Heading>
              <Text className="text-zinc-500 dark:text-zinc-500 text-center mt-2">
                Events will appear here once the workflow starts processing
              </Text>
            </div>
          ) : (
            <div className="space-y-3">
              {events.map((event: any) => {
                const isExpanded = expandedEventId === event.id
                const executionChain = isExpanded ? buildExecutionChain(executions) : []

                const renderExecution = (execution: WorkflowsWorkflowNodeExecution, isLast: boolean) => {
                  const nodeInfo = nodeMap.get(execution.nodeId!)
                  const isBlueprint = nodeInfo?.isBlueprint || false
                  const isChildExpanded = expandedExecutionIds.has(execution.id!)

                  return (
                    <div key={execution.id}>
                      {/* Node name badge */}
                      <div className="mb-2">
                        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 border border-zinc-200 dark:border-zinc-700">
                          <MaterialSymbol name={isBlueprint ? 'account_tree' : 'widgets'} size="sm" />
                          {nodeInfo?.nodeName || execution.nodeId}
                        </span>
                      </div>

                      {/* Execution item */}
                      <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg overflow-hidden">
                        <ExecutionItem
                          execution={execution}
                          isDarkMode={document.documentElement.classList.contains('dark')}
                          workflowId={workflowId!}
                          isBlueprintNode={isBlueprint}
                          nodeType={nodeInfo?.blockName}
                        />
                      </div>

                      {/* See child executions button for blueprint nodes */}
                      {isBlueprint && (
                        <div className="mt-3">
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => handleToggleChildExecutions(execution.id!)}
                            className="text-xs"
                          >
                            <MaterialSymbol name={isChildExpanded ? 'expand_less' : 'expand_more'} size="sm" />
                            {isChildExpanded ? 'Hide' : 'See'} child executions
                          </Button>
                        </div>
                      )}

                      {/* Child executions */}
                      {isBlueprint && isChildExpanded && nodeInfo?.blueprintId && (
                        <ChildExecutions
                          workflowId={workflowId!}
                          executionId={execution.id!}
                          organizationId={organizationId!}
                          blueprintId={nodeInfo.blueprintId}
                        />
                      )}

                      {/* Output flow connector - show outputs flowing to next execution */}
                      {!isLast && execution.outputs && Object.keys(execution.outputs).length > 0 && (
                        <div className="my-6">
                          {/* Arrow down */}
                          <div className="flex justify-center mb-3">
                            <MaterialSymbol name="arrow_downward" size="lg" className="text-zinc-400 dark:text-zinc-600" />
                          </div>

                          {/* Outputs preview */}
                          <div className="bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-700 rounded-lg p-4">
                            <div className="space-y-3">
                              {Object.entries(execution.outputs).map(([channel, outputs]: [string, any]) => (
                                <div key={channel}>
                                  <div className="flex items-center gap-2 mb-2">
                                    <MaterialSymbol name="alt_route" size="sm" className="text-zinc-500 dark:text-zinc-400" />
                                    <span className="font-medium text-zinc-700 dark:text-zinc-300 text-xs uppercase">
                                      {channel}
                                    </span>
                                    {Array.isArray(outputs) && (
                                      <span className="text-zinc-500 dark:text-zinc-400 text-xs">
                                        ({outputs.length} {outputs.length === 1 ? 'item' : 'items'})
                                      </span>
                                    )}
                                  </div>
                                  <div className="pl-6 bg-white dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded p-2">
                                    <JsonView
                                      value={outputs}
                                      style={{
                                        fontSize: '11px',
                                        fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                                        backgroundColor: 'transparent',
                                        textAlign: 'left',
                                        ...(document.documentElement.classList.contains('dark') ? darkTheme : lightTheme)
                                      }}
                                      displayDataTypes={false}
                                      displayObjectSize={false}
                                      enableClipboard={false}
                                      collapsed={2}
                                    />
                                  </div>
                                </div>
                              ))}
                            </div>
                          </div>

                          {/* Arrow down */}
                          <div className="flex justify-center mt-3">
                            <MaterialSymbol name="arrow_downward" size="lg" className="text-zinc-400 dark:text-zinc-600" />
                          </div>
                        </div>
                      )}
                    </div>
                  )
                }

                return (
                  <div key={event.id} className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg overflow-hidden">
                    <Item
                      onClick={() => handleEventClick(event.id)}
                      className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                    >
                      <ItemMedia>
                        <MaterialSymbol
                          name={isExpanded ? 'expand_less' : 'expand_more'}
                          size="xl"
                          className="text-zinc-600 dark:text-zinc-400"
                        />
                      </ItemMedia>
                      <ItemContent>
                        <div className="flex items-center justify-between w-full">
                          <div className="flex items-center gap-3">
                            <ItemTitle>Event #{event.id.substring(0, 8)}</ItemTitle>
                            {event.data && (
                              <span className="text-xs text-zinc-500 dark:text-zinc-400">
                                {Object.keys(event.data).length} field{Object.keys(event.data).length !== 1 ? 's' : ''}
                              </span>
                            )}
                          </div>
                          <ItemDescription className="!line-clamp-1">
                            Triggered {formatTimeAgo(new Date(event.createdAt))}
                          </ItemDescription>
                        </div>
                      </ItemContent>
                    </Item>

                    {isExpanded && (
                      <div className="border-t border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/50 p-6">
                        {/* Event Data Section */}
                        {event.data && Object.keys(event.data).length > 0 && (
                          <div className="mb-6 p-4 bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-700 rounded">
                            <div className="text-sm font-semibold text-zinc-700 dark:text-zinc-300 uppercase tracking-wide mb-3">
                              Event Data
                            </div>
                            <div className="bg-zinc-50 dark:bg-zinc-800 border border-zinc-200 dark:border-zinc-700 rounded p-3">
                              <JsonView
                                value={event.data}
                                style={{
                                  fontSize: '12px',
                                  fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                                  backgroundColor: 'transparent',
                                  textAlign: 'left',
                                  ...(document.documentElement.classList.contains('dark') ? darkTheme : lightTheme)
                                }}
                                displayDataTypes={false}
                                displayObjectSize={false}
                                enableClipboard={false}
                                collapsed={1}
                              />
                            </div>
                          </div>
                        )}

                        {/* Executions Timeline */}
                        <div className="text-sm font-semibold text-zinc-700 dark:text-zinc-300 uppercase tracking-wide mb-4">
                          Execution Chain
                        </div>

                        {executionChain.length === 0 ? (
                          <div className="text-center py-8 text-zinc-500 dark:text-zinc-400">
                            <MaterialSymbol name="pending" size="xl" className="mb-2" />
                            <p className="text-sm">No executions for this event</p>
                          </div>
                        ) : (
                          <div className="space-y-0">
                            {executionChain.map((execution, index) =>
                              renderExecution(execution, index === executionChain.length - 1)
                            )}
                          </div>
                        )}
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
