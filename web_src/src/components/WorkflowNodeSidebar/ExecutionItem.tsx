import { useState } from 'react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { Badge } from '../ui/badge'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'
import { getExecutionRenderer } from './executionRenderers'
import { ComponentActions } from './ComponentActions'
import { formatTimeAgo } from '../../utils/date'
import { WorkflowsWorkflowNodeExecution } from '@/api-client'

interface ExecutionItemProps {
  execution: WorkflowsWorkflowNodeExecution
  isDarkMode: boolean
  workflowId: string
  isBlueprintNode?: boolean
  nodeType?: string
}

export const ExecutionItem = ({ execution, isDarkMode, workflowId, isBlueprintNode = false, nodeType }: ExecutionItemProps) => {
  const [isExpanded, setIsExpanded] = useState(false)

  // Get custom renderer if available
  const customRenderer = nodeType ? getExecutionRenderer(nodeType) : undefined

  // If custom renderer provides full collapsed/expanded views, use them
  if (customRenderer?.renderCollapsed && customRenderer?.renderExpanded) {
    return (
      <div className="border-b border-zinc-200 dark:border-zinc-800">
        <div className="p-4">
          {!isExpanded ? (
            customRenderer.renderCollapsed({
              execution,
              onClick: () => setIsExpanded(true),
              isExpanded: false
            })
          ) : (
            <>
              {customRenderer.renderCollapsed({
                execution,
                onClick: () => setIsExpanded(false),
                isExpanded: true
              })}
              <div className="mt-4">
                {customRenderer.renderExpanded({
                  execution,
                  isDarkMode,
                  workflowId
                })}
              </div>
            </>
          )}
        </div>
      </div>
    )
  }

  const getStateIcon = (state: string) => {
    switch (state) {
      case 'STATE_PENDING': return 'schedule'
      case 'STATE_WAITING': return 'pending'
      case 'STATE_STARTED': return 'play_arrow'
      case 'STATE_FINISHED': return 'check_circle'
      default: return 'help'
    }
  }

  const getStateColor = (state: string) => {
    switch (state) {
      case 'STATE_PENDING': return 'text-yellow-600 dark:text-yellow-400 bg-yellow-100 dark:bg-yellow-900/30'
      case 'STATE_WAITING': return 'text-orange-600 dark:text-orange-400 bg-orange-100 dark:bg-orange-900/30'
      case 'STATE_STARTED': return 'text-blue-600 dark:text-blue-400 bg-blue-100 dark:bg-blue-900/30'
      case 'STATE_FINISHED': return 'text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-900/30'
      default: return 'text-gray-600 dark:text-gray-400 bg-gray-100 dark:bg-gray-900/30'
    }
  }

  const getStateBadge = (state: string, result: string) => {
    // For finished states, show the result
    if (state === 'STATE_FINISHED') {
      switch (result) {
        case 'RESULT_PASSED':
          return (
            <Badge className="bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 border-green-200 dark:border-green-800">
              <MaterialSymbol name="check_circle" size="sm" />
              Passed
            </Badge>
          )
        case 'RESULT_FAILED':
          return (
            <Badge variant="destructive">
              <MaterialSymbol name="cancel" size="sm" />
              Failed
            </Badge>
          )
        case 'RESULT_CANCELLED':
          return (
            <Badge variant="secondary">
              <MaterialSymbol name="block" size="sm" />
              Cancelled
            </Badge>
          )
      }
    }

    // For non-finished states, show the state
    switch (state) {
      case 'STATE_PENDING':
        return (
          <Badge className="bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300 border-yellow-200 dark:border-yellow-800">
            <MaterialSymbol name="schedule" size="sm" />
            Pending
          </Badge>
        )
      case 'STATE_WAITING':
        return (
          <Badge className="bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 border-orange-200 dark:border-orange-800">
            <MaterialSymbol name="pending" size="sm" />
            Waiting
          </Badge>
        )
      case 'STATE_STARTED':
        return (
          <Badge className="bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300 border-blue-200 dark:border-blue-800">
            <MaterialSymbol name="play_arrow" size="sm" />
            Running
          </Badge>
        )
      default:
        return null
    }
  }

  return (
    <div className="border-b border-zinc-200 dark:border-zinc-800">
      <div className="p-4">
        <div
          className="flex items-center gap-3 cursor-pointer"
          onClick={() => setIsExpanded(!isExpanded)}
        >
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              {getStateBadge(execution.state!, execution.result!)}
              <span className="text-xs font-mono text-gray-600 dark:text-zinc-400 truncate">
                {execution.id}
              </span>
            </div>
          </div>
          <div className="flex-shrink-0 flex items-center gap-2">
            <p className="text-xs text-gray-400 dark:text-zinc-500">
              {formatTimeAgo(new Date(execution.createdAt!))}
            </p>
            <MaterialSymbol
              name={isExpanded ? 'expand_less' : 'expand_more'}
              size="xl"
              className="text-gray-600 dark:text-zinc-400"
            />
          </div>
        </div>

        {/* Expanded content */}
        {isExpanded && (
          <div className="mt-4 space-y-4 text-left">
            {/* Execution Details Section */}
            <div className="space-y-3">
              <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
                Execution
              </div>

              <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-4 text-xs">
                <div className="space-y-1">
                  <div className="flex items-center gap-1">
                    <MaterialSymbol name="calendar_today" size="md" className="text-gray-600 dark:text-zinc-400" />
                    <span className="text-xs text-gray-500 dark:text-zinc-400">
                      Created {formatTimeAgo(new Date(execution.createdAt!))}
                    </span>
                  </div>

                  <div className="flex items-center gap-1">
                    <MaterialSymbol name="info" size="md" className="text-gray-600 dark:text-zinc-400" />
                    <span className="text-xs text-gray-500 dark:text-zinc-400">
                      State: <span className="text-gray-800 dark:text-gray-200 font-medium">{execution.state!.replace('STATE_', '').toLowerCase()}</span>
                    </span>
                  </div>

                  <div className="flex items-center gap-1">
                    <MaterialSymbol name="info" size="md" className="text-gray-600 dark:text-zinc-400" />
                    <span className="text-xs text-gray-500 dark:text-zinc-400">
                      Result: <span className="text-gray-800 dark:text-gray-200 font-medium">{execution.result!.replace('RESULT_', '').toLowerCase()}</span>
                    </span>
                  </div>

                  {execution.resultMessage && (
                    <div>
                      <div className="flex items-center gap-1 mb-1">
                        <MaterialSymbol name="message" size="md" className="text-gray-600 dark:text-zinc-400" />
                        <span className="text-xs text-gray-500 dark:text-zinc-400">Message</span>
                      </div>
                      <div className="text-xs text-red-600 dark:text-red-400 pl-6 break-words">
                        {execution.resultMessage}
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Actions Section */}
            {!isBlueprintNode && nodeType && (
              <ComponentActions
                executionId={execution.id!}
                componentName={nodeType}
                executionState={execution.state!}
              />
            )}

            {/* Custom Renderer or Default Sections */}
            {customRenderer ? (
              // Use custom renderer if available
              customRenderer.renderCustomSections?.({ execution, isDarkMode })
            ) : (
              // Default rendering
              <>
                {/* Inputs Section */}
                {execution.input && Object.keys(execution.input).length > 0 && (
                  <div className="space-y-3">
                    <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
                      Inputs
                    </div>
                    <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs text-left">
                      <JsonView
                        value={execution.input}
                        style={{
                          fontSize: '12px',
                          fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                          backgroundColor: 'transparent',
                          textAlign: 'left',
                          ...(isDarkMode ? darkTheme : lightTheme)
                        }}
                        displayDataTypes={false}
                        displayObjectSize={false}
                        enableClipboard={false}
                        collapsed={1}
                      />
                    </div>
                  </div>
                )}

                {/* Outputs Section */}
                {execution.outputs && Object.keys(execution.outputs).length > 0 && (
                  <div className="space-y-3">
                    <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
                      Outputs
                    </div>
                    <div className="space-y-2">
                      {Object.entries(execution.outputs).map(([branch, outputs]: [string, any]) => (
                        <div key={branch} className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs">
                          <div className="flex items-center gap-2 mb-2">
                            <MaterialSymbol name="alt_route" size="md" className="text-gray-600 dark:text-zinc-400" />
                            <span className="text-xs font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide">
                              {branch}
                            </span>
                            {Array.isArray(outputs) && (
                              <span className="text-xs text-gray-500 dark:text-zinc-400">
                                ({outputs.length} {outputs.length === 1 ? 'item' : 'items'})
                              </span>
                            )}
                          </div>
                          <div className="pl-6 text-left">
                            <JsonView
                              value={outputs}
                              style={{
                                fontSize: '12px',
                                fontFamily: 'ui-monospace, SFMono-Regular, "SF Mono", Consolas, "Liberation Mono", Menlo, monospace',
                                backgroundColor: 'transparent',
                                textAlign: 'left',
                                ...(isDarkMode ? darkTheme : lightTheme)
                              }}
                              displayDataTypes={false}
                              displayObjectSize={false}
                              enableClipboard={false}
                              collapsed={1}
                            />
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

              </>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
