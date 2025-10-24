import { useChildExecutions } from '../../hooks/useWorkflowData'
import { useBlueprints } from '../../hooks/useBlueprintData'
import { ExecutionItem } from '../WorkflowNodeSidebar/ExecutionItem'
import { WorkflowsWorkflowNodeExecution } from '../../api-client'
import { useMemo } from 'react'

interface ChildExecutionsProps {
  workflowId: string
  executionId: string
  organizationId: string
  blueprintId: string
}

export const ChildExecutions = ({ workflowId, executionId, organizationId, blueprintId }: ChildExecutionsProps) => {
  const { data: childExecutionsData, isLoading } = useChildExecutions(workflowId, executionId)
  const { data: blueprints = [] } = useBlueprints(organizationId)
  const childExecutions = childExecutionsData?.executions || []

  // Build a map of nodeId to node information from the blueprint
  const nodeMap = useMemo(() => {
    const map = new Map<string, { blockName: string; isBlueprint: boolean }>()

    const blueprint = blueprints.find((bp: any) => bp.id === blueprintId)

    if (blueprint?.nodes) {
      blueprint.nodes.forEach((node: any) => {
        const isComponent = node.type === 'TYPE_COMPONENT'
        const blockName = isComponent ? node.component?.name : node.blueprint?.name
        map.set(node.id, {
          blockName,
          isBlueprint: !isComponent,
        })
      })
    }

    return map
  }, [blueprints, blueprintId])

  if (isLoading) {
    return (
      <div className="py-4 px-4">
        <div className="flex items-center gap-2 text-zinc-500 dark:text-zinc-400 text-sm">
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-zinc-600"></div>
          Loading child executions...
        </div>
      </div>
    )
  }

  if (childExecutions.length === 0) {
    return (
      <div className="py-4 px-4">
        <div className="text-zinc-500 dark:text-zinc-400 text-sm">
          No child executions found
        </div>
      </div>
    )
  }

  // Build execution chain for child executions
  const buildChildChain = (execs: WorkflowsWorkflowNodeExecution[]): WorkflowsWorkflowNodeExecution[] => {
    if (!execs || execs.length === 0) return []

    const rootExec = execs.find((exec) => !exec.previousExecutionId)
    if (!rootExec) return execs

    const chain: WorkflowsWorkflowNodeExecution[] = []
    const visited = new Set<string>()
    let current: WorkflowsWorkflowNodeExecution | undefined = rootExec

    while (current && !visited.has(current.id!)) {
      chain.push(current)
      visited.add(current.id!)
      current = execs.find((exec) => exec.previousExecutionId === current!.id)
    }

    execs.forEach((exec) => {
      if (!visited.has(exec.id!)) {
        chain.push(exec)
      }
    })

    return chain
  }

  const childChain = buildChildChain(childExecutions)

  return (
    <div className="py-4 px-4 bg-zinc-50 dark:bg-zinc-900/50">
      <div className="mb-3 text-xs font-semibold text-zinc-600 dark:text-zinc-400 uppercase tracking-wide">
        Child Executions
      </div>
      <div className="space-y-3">
        {childChain.map((childExec) => {
          // The nodeId format is: <parent-node-id>:<blueprint-node-id>
          // We need to extract just the blueprint-node-id part
          const blueprintNodeId = childExec.nodeId!.split(':')[1] || childExec.nodeId!
          const nodeInfo = nodeMap.get(blueprintNodeId)

          return (
            <div key={childExec.id}>
              <div className="bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 rounded-lg overflow-hidden">
                <ExecutionItem
                  execution={childExec}
                  isDarkMode={document.documentElement.classList.contains('dark')}
                  workflowId={workflowId}
                  isBlueprintNode={nodeInfo?.isBlueprint || false}
                  componentName={nodeInfo?.blockName}
                />
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
