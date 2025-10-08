import { useState } from 'react'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'
import { getExecutionRenderer } from './executionRenderers'
import { ComponentActions } from './ComponentActions'

interface ExecutionItemProps {
  execution: any
  isDarkMode: boolean
  workflowId: string
  isBlueprintNode?: boolean
  nodeType?: string
}

const formatTimeAgo = (date: Date): string => {
  const seconds = Math.floor((new Date().getTime() - date.getTime()) / 1000)

  if (seconds < 60) return `${seconds}s ago`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m ago`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
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
              onClick: () => setIsExpanded(true)
            })
          ) : (
            <>
              {customRenderer.renderCollapsed({
                execution,
                onClick: () => setIsExpanded(false)
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

  const getResultBadge = (result: string) => {
    switch (result) {
      case 'RESULT_PASSED':
        return <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">Passed</span>
      case 'RESULT_FAILED':
        return <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300">Failed</span>
      case 'RESULT_CANCELLED':
        return <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 dark:bg-gray-900/30 text-gray-800 dark:text-gray-300">Cancelled</span>
      default:
        return null
    }
  }

  return (
    <div className="border-b border-zinc-200 dark:border-zinc-800">
      <div className="p-4">
        <div
          className="flex items-start gap-3 cursor-pointer"
          onClick={() => setIsExpanded(!isExpanded)}
        >
          <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${getStateColor(execution.state)}`}>
            <MaterialSymbol name={getStateIcon(execution.state)} size="sm" />
          </div>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2 mb-2">
              {getResultBadge(execution.result)}
              {execution.state === 'STATE_WAITING' && (
                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300">
                  Waiting
                </span>
              )}
              {execution.state === 'STATE_STARTED' && (
                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">
                  Running
                </span>
              )}
            </div>

            <p className="text-xs font-mono text-gray-600 dark:text-zinc-400 truncate mb-1">
              {execution.id}
            </p>

            <p className="text-xs text-gray-400 dark:text-zinc-500">
              {formatTimeAgo(new Date(execution.createdAt))}
            </p>
          </div>
          <div className="flex-shrink-0">
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
                      Created {formatTimeAgo(new Date(execution.createdAt))}
                    </span>
                  </div>

                  <div className="flex items-center gap-1">
                    <MaterialSymbol name="info" size="md" className="text-gray-600 dark:text-zinc-400" />
                    <span className="text-xs text-gray-500 dark:text-zinc-400">
                      State: <span className="text-gray-800 dark:text-gray-200 font-medium">{execution.state.replace('STATE_', '').toLowerCase()}</span>
                    </span>
                  </div>

                  <div className="flex items-center gap-1">
                    <MaterialSymbol name="info" size="md" className="text-gray-600 dark:text-zinc-400" />
                    <span className="text-xs text-gray-500 dark:text-zinc-400">
                      Result: <span className="text-gray-800 dark:text-gray-200 font-medium">{execution.result.replace('RESULT_', '').toLowerCase()}</span>
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

                  {execution.event?.blueprintName && (
                    <div className="flex items-center gap-1">
                      <MaterialSymbol name="account_tree" size="md" className="text-gray-600 dark:text-zinc-400" />
                      <span className="text-xs text-gray-500 dark:text-zinc-400">
                        Blueprint: <span className="text-gray-800 dark:text-gray-200 font-medium">{execution.event.blueprintName}</span>
                      </span>
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Actions Section */}
            {!isBlueprintNode && nodeType && (
              <ComponentActions
                executionId={execution.id}
                componentName={nodeType}
                executionState={execution.state}
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
                {execution.inputs && Object.keys(execution.inputs).length > 0 && (
                  <div className="space-y-3">
                    <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
                      Inputs
                    </div>
                    <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs text-left">
                      <JsonView
                        value={execution.inputs}
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
