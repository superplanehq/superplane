import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useWorkflow, useWorkflowEvents, useEventExecutions } from '../../hooks/useWorkflowData'
import { useBlueprints } from '../../hooks/useBlueprintData'
import { Button } from '../../components/ui/button'
import { MaterialSymbol } from '../../components/MaterialSymbol/material-symbol'
import { Heading } from '../../components/Heading/heading'
import { Text } from '../../components/Text/text'
import { Item, ItemMedia, ItemContent, ItemTitle, ItemDescription } from '../../components/ui/item'
import { formatTimeAgo } from '../../utils/date'
import { ExecutionItem } from '../../components/WorkflowNodeSidebar/ExecutionItem'
import { WorkflowsWorkflowNodeExecution } from '../../api-client'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'

export const WorkflowEvents = () => {
  const { organizationId, workflowId } = useParams<{ organizationId: string; workflowId: string }>()
  const navigate = useNavigate()
  const [expandedEventId, setExpandedEventId] = useState<string | null>(null)

  const { data: workflow, isLoading: workflowLoading } = useWorkflow(organizationId!, workflowId!)
  const { data: eventsData, isLoading: eventsLoading } = useWorkflowEvents(workflowId!)
  const { data: executionsData } = useEventExecutions(workflowId!, expandedEventId)
  const { data: blueprints = [] } = useBlueprints(organizationId!)

  const events = eventsData?.events || []
  const executions = executionsData?.executions || []

  // Create a comprehensive map of nodeId to node information
  // This includes workflow nodes AND blueprint internal nodes
  const [nodeMap, setNodeMap] = useState<Map<string, { blockName: string; isBlueprint: boolean; nodeName: string }>>(new Map())

  useEffect(() => {
    const map = new Map<string, { blockName: string; isBlueprint: boolean; nodeName: string }>()

    // Add workflow nodes
    if (workflow?.nodes) {
      workflow.nodes.forEach((node: any) => {
        const isComponent = node.refType === 'REF_TYPE_COMPONENT'
        const blockName = isComponent ? node.component?.name : node.blueprint?.name
        map.set(node.id, {
          blockName,
          isBlueprint: !isComponent,
          nodeName: node.name,
        })
      })
    }

    // Add blueprint internal nodes
    if (blueprints && blueprints.length > 0) {
      blueprints.forEach((blueprint: any) => {
        if (blueprint.nodes) {
          blueprint.nodes.forEach((node: any) => {
            const isComponent = node.refType === 'REF_TYPE_COMPONENT'
            const blockName = isComponent ? node.component?.name : node.blueprint?.name
            map.set(node.id, {
              blockName,
              isBlueprint: !isComponent,
              nodeName: node.name,
            })
          })
        }
      })
    }

    setNodeMap(map)
  }, [workflow, blueprints])

  // Build execution chain from the executions list, grouping blueprint internals
  interface ExecutionNode {
    execution: WorkflowsWorkflowNodeExecution
    children: ExecutionNode[]  // For blueprint internal executions
  }

  const buildExecutionChain = (executions: WorkflowsWorkflowNodeExecution[]): ExecutionNode[] => {
    if (!executions || executions.length === 0) return []

    // Separate top-level executions from blueprint internal executions
    const topLevelExecutions = executions.filter((exec) => !exec.parentExecutionId)
    const blueprintInternalExecutions = executions.filter((exec) => exec.parentExecutionId)

    // Build a map of parent execution ID to its internal executions
    const internalsByParent = new Map<string, WorkflowsWorkflowNodeExecution[]>()
    blueprintInternalExecutions.forEach((exec) => {
      const parentId = exec.parentExecutionId!
      if (!internalsByParent.has(parentId)) {
        internalsByParent.set(parentId, [])
      }
      internalsByParent.get(parentId)!.push(exec)
    })

    // Build chains for blueprint internal executions recursively
    const buildInternalChain = (executions: WorkflowsWorkflowNodeExecution[]): ExecutionNode[] => {
      const rootInternals = executions.filter((exec) => !exec.previousExecutionId)
      const visited = new Set<string>()

      const buildNode = (execution: WorkflowsWorkflowNodeExecution): ExecutionNode => {
        visited.add(execution.id!)

        const node: ExecutionNode = {
          execution,
          children: []
        }

        // Add blueprint internal executions if this is a blueprint node
        const internals = internalsByParent.get(execution.id!) || []
        if (internals.length > 0) {
          node.children = buildInternalChain(internals)
        }

        return node
      }

      const chains: ExecutionNode[] = []

      const buildChainFromRoot = (root: WorkflowsWorkflowNodeExecution): ExecutionNode[] => {
        const chain: ExecutionNode[] = []
        let current: WorkflowsWorkflowNodeExecution | undefined = root

        while (current && !visited.has(current.id!)) {
          const node = buildNode(current)
          chain.push(node)

          // Find the next execution in the chain
          current = executions.find((exec) => exec.previousExecutionId === current!.id)
        }

        return chain
      }

      rootInternals.forEach((root) => {
        chains.push(...buildChainFromRoot(root))
      })

      // Add any unvisited executions
      executions.forEach((exec) => {
        if (!visited.has(exec.id!)) {
          chains.push(buildNode(exec))
        }
      })

      return chains
    }

    // Build the top-level chain
    const visited = new Set<string>()
    const topLevelChain: ExecutionNode[] = []

    const buildTopLevelNode = (execution: WorkflowsWorkflowNodeExecution): ExecutionNode => {
      visited.add(execution.id!)

      const node: ExecutionNode = {
        execution,
        children: []
      }

      // Add blueprint internal executions if this is a blueprint node
      const internals = internalsByParent.get(execution.id!) || []
      if (internals.length > 0) {
        node.children = buildInternalChain(internals)
      }

      return node
    }

    const buildTopLevelChain = (root: WorkflowsWorkflowNodeExecution): ExecutionNode[] => {
      const chain: ExecutionNode[] = []
      let current: WorkflowsWorkflowNodeExecution | undefined = root

      while (current && !visited.has(current.id!)) {
        const node = buildTopLevelNode(current)
        chain.push(node)

        // Find the next execution in the top-level chain
        current = topLevelExecutions.find((exec) => exec.previousExecutionId === current!.id)
      }

      return chain
    }

    // Find root top-level executions
    const rootTopLevel = topLevelExecutions.filter((exec) => !exec.previousExecutionId)

    rootTopLevel.forEach((root) => {
      topLevelChain.push(...buildTopLevelChain(root))
    })

    // Add any unvisited top-level executions
    topLevelExecutions.forEach((exec) => {
      if (!visited.has(exec.id!)) {
        topLevelChain.push(buildTopLevelNode(exec))
      }
    })

    return topLevelChain
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

                const renderExecutionNode = (node: ExecutionNode, index: number, isLast: boolean, depth: number = 0) => {
                  const nodeInfo = nodeMap.get(node.execution.nodeId!)
                  const isBlueprint = nodeInfo?.isBlueprint || false

                  return (
                    <div key={node.execution.id}>
                      {/* Node name badge */}
                      <div className="mb-2">
                        <span className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs font-medium bg-zinc-100 dark:bg-zinc-800 text-zinc-700 dark:text-zinc-300 border border-zinc-200 dark:border-zinc-700">
                          <MaterialSymbol name={isBlueprint ? 'account_tree' : 'widgets'} size="sm" />
                          {nodeInfo?.nodeName || node.execution.nodeId}
                        </span>
                      </div>

                      {/* Execution item */}
                      <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg overflow-hidden">
                        <ExecutionItem
                          execution={node.execution}
                          isDarkMode={document.documentElement.classList.contains('dark')}
                          workflowId={workflowId!}
                          isBlueprintNode={isBlueprint}
                          nodeType={nodeInfo?.blockName}
                        />
                      </div>

                      {/* Blueprint internal executions */}
                      {node.children.length > 0 && (
                        <div className="mt-6 ml-8 pl-4 border-l-2 border-zinc-300 dark:border-zinc-700">
                          <div className="mb-4 text-xs font-semibold text-zinc-600 dark:text-zinc-400 uppercase tracking-wide">
                            Blueprint Internal Executions
                          </div>
                          <div className="space-y-6">
                            {node.children.map((childNode, childIndex) => (
                              <div key={childNode.execution.id}>
                                {renderExecutionNode(childNode, childIndex, childIndex === node.children.length - 1, depth + 1)}
                              </div>
                            ))}
                          </div>
                        </div>
                      )}

                      {/* Output flow connector - show outputs flowing to next execution */}
                      {!isLast && node.execution.outputs && Object.keys(node.execution.outputs).length > 0 && (
                        <div className="my-6">
                          {/* Arrow down */}
                          <div className="flex justify-center mb-3">
                            <MaterialSymbol name="arrow_downward" size="lg" className="text-zinc-400 dark:text-zinc-600" />
                          </div>

                          {/* Outputs preview */}
                          <div className="bg-zinc-50 dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-700 rounded-lg p-4">
                            <div className="space-y-3">
                              {Object.entries(node.execution.outputs).map(([branch, outputs]: [string, any]) => (
                                <div key={branch}>
                                  <div className="flex items-center gap-2 mb-2">
                                    <MaterialSymbol name="alt_route" size="sm" className="text-zinc-500 dark:text-zinc-400" />
                                    <span className="font-medium text-zinc-700 dark:text-zinc-300 text-xs uppercase">
                                      {branch}
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
                            {executionChain.map((node, index) =>
                              renderExecutionNode(node, index, index === executionChain.length - 1)
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
