import { registerExecutionRenderer, CollapsedViewProps, ExpandedViewProps } from './registry'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'
import { formatTimeAgo } from '../../../utils/date'

// Custom renderer for IF component executions
registerExecutionRenderer('if', {
  renderCollapsed: ({ execution, onClick, isExpanded }: CollapsedViewProps) => {
    const outputs = execution.outputs
    const isFailed = execution.result === 'RESULT_FAILED'

    // Extract condition expression from configuration
    const conditionExpression = execution.configuration?.expression || 'No expression'

    // Determine which channel was taken
    const trueChannel = outputs?.true
    const falseChannel = outputs?.false
    const channelTaken = trueChannel && trueChannel.length > 0 ? 'TRUE' :
                        falseChannel && falseChannel.length > 0 ? 'FALSE' :
                        'NONE'

    return (
      <div
        className="flex items-center gap-3 cursor-pointer"
        onClick={onClick}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 text-sm">
            <span className="font-mono text-gray-700 dark:text-gray-300 bg-gray-100 dark:bg-gray-800 px-2 py-1 rounded truncate">{conditionExpression}</span>
            <MaterialSymbol name="arrow_forward" size="sm" className="text-gray-400 dark:text-gray-500" />
            <span className="font-semibold text-blue-600 dark:text-blue-400">{isFailed ? 'FAILED' : channelTaken}</span>
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
    )
  },

  renderExpanded: ({ execution, isDarkMode }: ExpandedViewProps) => {
    const input = execution.input
    const isFailed = execution.result === 'RESULT_FAILED'

    return (
      <div className="mt-4 space-y-4 text-left">
        {/* Error Section - Show when execution failed */}
        {isFailed && (
          <div className="space-y-3">
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Error
            </div>
            <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 p-3 text-xs">
              <div className="space-y-2">
                <div className="flex items-start gap-2">
                  <MaterialSymbol
                    name="error"
                    size="md"
                    className="text-red-600 dark:text-red-400"
                  />
                  <div className="flex-1">
                    {execution.resultReason && (
                      <div className="font-medium text-red-700 dark:text-red-300 mb-1">
                        {execution.resultReason.replace('RESULT_REASON_', '')}
                      </div>
                    )}
                    {execution.resultMessage && (
                      <div className="text-red-600 dark:text-red-400 break-words">
                        {execution.resultMessage}
                      </div>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Input Data Section */}
        {input && Object.keys(input).length > 0 && (
          <div className="space-y-3">
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Input Data
            </div>
            <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs text-left">
              <JsonView
                value={input}
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
      </div>
    )
  }
})
