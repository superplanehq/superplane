import { registerExecutionRenderer, CollapsedViewProps, ExpandedViewProps } from './registry'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Badge } from '../../ui/badge'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'
import { formatTimeAgo } from '../../../utils/date'

interface HTTPConfiguration {
  url?: string
  method?: string
  payload?: Record<string, any>
  headers?: Array<{ name: string; value: string }>
}

interface HTTPResponse {
  status?: number
  headers?: Record<string, string | string[]>
  body?: any
}

const getResultBadge = (result: string) => {
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
    default:
      return null
  }
}

registerExecutionRenderer('http', {
  renderCollapsed: ({ execution, onClick, isExpanded }: CollapsedViewProps) => {
    const configuration = execution.configuration as HTTPConfiguration | undefined
    const response = (execution.outputs?.default as HTTPResponse[] | undefined)?.[0]
    const method = configuration?.method
    const url = configuration?.url
    const statusCode = response?.status

    // Determine status badge styling
    let statusBadgeClasses = 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300'

    if (statusCode) {
      if (statusCode >= 400) {
        statusBadgeClasses = 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300'
      } else if (statusCode >= 300) {
        statusBadgeClasses = 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300'
      }
    }

    return (
      <div
        className="flex items-center gap-3 cursor-pointer"
        onClick={onClick}
      >
        <div className={`flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center`}>
          <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium font-mono ${statusBadgeClasses}`}>
            {statusCode}
          </span>
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            {statusCode && method && url ? (
              <>
                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium text-gray-300 dark:text-zinc-400`}>
                  {method}
                </span>
                <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium text-gray-300 dark:text-zinc-400`}>
                  {url}
                </span>
              </>
            ) : (
              <>
                {execution.result && getResultBadge(execution.result)}
                {execution.state === 'STATE_STARTED' && (
                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">
                    Running
                  </span>
                )}
              </>
            )}
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
    const configuration = execution.configuration as HTTPConfiguration | undefined
    const method = configuration?.method || 'GET'
    const url = configuration?.url || ''
    const payload = configuration?.payload || {}

    const response = (execution.outputs?.default as HTTPResponse[] | undefined)?.[0]
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
                <span className="text-gray-800 dark:text-gray-200 font-mono font-semibold">{method}</span>
                <span className="text-gray-800 dark:text-gray-200 font-mono break-all">{url}</span>
              </div>

              {payload && (
                <div>
                  <pre className="pl-4 text-gray-800 dark:text-gray-200 font-mono text-xs overflow-x-auto bg-white dark:bg-zinc-900 p-2 rounded border border-gray-200 dark:border-zinc-600">
                    {JSON.stringify(payload, null, 2)}
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
