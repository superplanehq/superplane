import { registerExecutionRenderer, CollapsedViewProps, ExpandedViewProps } from './registry'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Badge } from '../../ui/badge'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'
import { formatTimeAgo } from '../../../utils/date'

// Custom renderer for Filter component executions
registerExecutionRenderer('filter', {
  renderCollapsed: ({ execution, onClick, isExpanded }: CollapsedViewProps) => {
    const outputs = execution.outputs

    // Check if the filter matched (outputs will have data if it matched)
    const hasOutput = outputs?.default && outputs.default.length > 0
    const matched = hasOutput

    // Determine icon, colors, and label based on filter result
    const iconName = matched ? 'fast_forward' : 'filter_alt_off'
    const badgeClassName = matched
      ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300 border-green-200 dark:border-green-800'
      : 'bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300 border-orange-200 dark:border-orange-800'
    const label = matched ? 'Forwarded' : 'Filtered'

    return (
      <div
        className="flex items-center gap-3 cursor-pointer"
        onClick={onClick}
      >
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <Badge className={badgeClassName}>
              <MaterialSymbol name={iconName} size="sm" />
              {label}
            </Badge>
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
    const filterExpression = execution.configuration?.expression

    return (
      <div className="mt-4 space-y-4 text-left">
        {/* Filter Expression Section */}
        {filterExpression && (
          <div className="space-y-3">
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Filter Expression
            </div>
            <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs">
              <code className="text-gray-800 dark:text-gray-200 font-mono text-xs break-all">
                {filterExpression}
              </code>
            </div>
          </div>
        )}

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
      </div>
    )
  }
})
