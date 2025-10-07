import { registerExecutionRenderer, CollapsedViewProps, ExpandedViewProps } from './registry'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'

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

// Custom renderer for Filter primitive executions
registerExecutionRenderer('filter', {
  renderCollapsed: ({ execution, onClick }: CollapsedViewProps) => {
    const outputs = execution.outputs

    // Check if the filter matched (outputs will have data if it matched)
    const hasOutput = outputs?.default && outputs.default.length > 0
    const matched = hasOutput

    // Determine icon, colors, and label based on filter result
    const iconName = matched ? 'fast_forward' : 'filter_alt_off'
    const colorClasses = matched
      ? 'text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-900/30'
      : 'text-orange-600 dark:text-orange-400 bg-orange-100 dark:bg-orange-900/30'
    const badgeClasses = matched
      ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300'
      : 'bg-orange-100 dark:bg-orange-900/30 text-orange-800 dark:text-orange-300'
    const label = matched ? 'Forwarded' : 'Filtered'

    return (
      <div
        className="flex items-start gap-3 cursor-pointer"
        onClick={onClick}
      >
        <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${colorClasses}`}>
          <MaterialSymbol name={iconName} size="sm" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2">
            <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${badgeClasses}`}>
              {label}
            </span>
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
            name="expand_more"
            size="xl"
            className="text-gray-600 dark:text-zinc-400"
          />
        </div>
      </div>
    )
  },

  renderExpanded: ({ execution, isDarkMode }: ExpandedViewProps) => {
    const inputs = execution.inputs

    return (
      <div className="mt-4 space-y-4 text-left">
        {/* Inputs Section */}
        {inputs && Object.keys(inputs).length > 0 && (
          <div className="space-y-3">
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Inputs
            </div>
            <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs text-left">
              <JsonView
                value={inputs}
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
