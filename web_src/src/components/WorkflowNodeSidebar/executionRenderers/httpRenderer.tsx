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

// Custom renderer for HTTP primitive executions
registerExecutionRenderer('http', {
  renderCollapsed: ({ execution, onClick }: CollapsedViewProps) => {
    // Extract response status from outputs
    const response = execution.outputs?.default?.[0]
    const statusCode = response?.status

    // Determine status badge styling
    let statusBadgeClasses = 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300'
    let iconName = 'check_circle'
    let iconColorClasses = 'text-green-600 dark:text-green-400 bg-green-100 dark:bg-green-900/30'

    if (statusCode) {
      if (statusCode >= 400) {
        statusBadgeClasses = 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300'
        iconName = 'error'
        iconColorClasses = 'text-red-600 dark:text-red-400 bg-red-100 dark:bg-red-900/30'
      } else if (statusCode >= 300) {
        statusBadgeClasses = 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300'
        iconName = 'warning'
        iconColorClasses = 'text-yellow-600 dark:text-yellow-400 bg-yellow-100 dark:bg-yellow-900/30'
      }
    }

    return (
      <div
        className="flex items-start gap-3 cursor-pointer"
        onClick={onClick}
      >
        <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center ${iconColorClasses}`}>
          <MaterialSymbol name={iconName} size="sm" />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-2">
            {statusCode ? (
              <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium font-mono ${statusBadgeClasses}`}>
                {statusCode}
              </span>
            ) : (
              <>
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
              </>
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
    const outputs = execution.outputs

    // Extract HTTP-specific information
    const method = inputs?.method || 'GET'
    const url = inputs?.url || ''
    const headers = inputs?.headers || {}
    const body = inputs?.body

    // Extract response information from outputs
    const response = outputs?.default?.[0]
    const statusCode = response?.status
    const responseHeaders = response?.headers
    const responseBody = response?.body

    return (
      <div className="mt-4 space-y-4 text-left">
        {/* HTTP Request Section */}
        <div className="space-y-3">
          <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
            Request
          </div>
          <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs">
            <div className="space-y-2">
              <div className="flex items-center gap-2">
                <MaterialSymbol name="http" size="md" className="text-gray-600 dark:text-zinc-400" />
                <span className="font-medium text-blue-600 dark:text-blue-400">{method}</span>
                <span className="text-gray-800 dark:text-gray-200 font-mono break-all">{url}</span>
              </div>

              {Object.keys(headers).length > 0 && (
                <div>
                  <div className="text-gray-500 dark:text-zinc-400 mb-1">Headers:</div>
                  <div className="pl-4 space-y-1">
                    {Object.entries(headers).map(([key, value]) => (
                      <div key={key} className="font-mono">
                        <span className="text-gray-600 dark:text-zinc-400">{key}:</span>{' '}
                        <span className="text-gray-800 dark:text-gray-200">{String(value)}</span>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {body && (
                <div>
                  <div className="text-gray-500 dark:text-zinc-400 mb-1">Body:</div>
                  <pre className="pl-4 text-gray-800 dark:text-gray-200 font-mono text-xs overflow-x-auto bg-white dark:bg-zinc-900 p-2 rounded border border-gray-200 dark:border-zinc-600">
                    {typeof body === 'string' ? body : JSON.stringify(body, null, 2)}
                  </pre>
                </div>
              )}
            </div>
          </div>
        </div>

        {/* HTTP Response Headers Section */}
        {statusCode && responseHeaders && Object.keys(responseHeaders).length > 0 && (
          <div className="space-y-3">
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Response Headers
            </div>
            <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs">
              <div className="space-y-1 max-h-32 overflow-y-auto">
                {Object.entries(responseHeaders).map(([key, value]) => (
                  <div key={key} className="font-mono">
                    <span className="text-gray-600 dark:text-zinc-400">{key}:</span>{' '}
                    <span className="text-gray-800 dark:text-gray-200">{String(value)}</span>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}

        {/* HTTP Response Body Section */}
        {statusCode && responseBody && (
          <div className="space-y-3">
            <div className="text-sm font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
              Response Body
            </div>
            <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 text-xs text-left">
              <JsonView
                value={responseBody}
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
