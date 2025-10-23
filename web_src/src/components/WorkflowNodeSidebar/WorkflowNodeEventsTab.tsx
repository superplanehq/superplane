import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { useState, useEffect } from 'react'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'
import { useNodeEvents } from '../../hooks/useWorkflowData'
import { formatTimeAgo } from '../../utils/date'

interface WorkflowNodeEventsTabProps {
  workflowId: string
  nodeId: string
}

export const WorkflowNodeEventsTab = ({ workflowId, nodeId }: WorkflowNodeEventsTabProps) => {
  const [isDarkMode, setIsDarkMode] = useState(false)

  // Detect dark mode
  useEffect(() => {
    const checkDarkMode = () => {
      setIsDarkMode(window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches)
    }

    checkDarkMode()

    const observer = new MutationObserver(checkDarkMode)
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ['class']
    })

    return () => observer.disconnect()
  }, [])

  const { data, isLoading, error } = useNodeEvents(workflowId, nodeId)

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mb-3"></div>
        <p className="text-sm text-gray-500 dark:text-zinc-400">Loading events...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6">
        <MaterialSymbol name="error" className="text-red-500 mb-3" size="xl" />
        <p className="text-sm text-gray-900 dark:text-zinc-100 font-medium mb-1">Failed to load events</p>
        <p className="text-xs text-gray-500 dark:text-zinc-400">{error.message}</p>
      </div>
    )
  }

  const events = data?.events || []

  if (events.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6">
        <MaterialSymbol name="event" className="text-gray-400 dark:text-zinc-500 mb-3" size="xl" />
        <p className="text-sm text-gray-600 dark:text-zinc-400 font-medium mb-1">No events yet</p>
        <p className="text-xs text-gray-500 dark:text-zinc-500 text-center">
          Events will appear here once this trigger node emits them
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex-shrink-0 px-4 py-3 border-b border-zinc-200 dark:border-zinc-800">
        <p className="text-xs text-gray-500 dark:text-zinc-400">
          {events.length} {events.length === 1 ? 'event' : 'events'}
        </p>
      </div>
      <div className="flex-1 overflow-y-auto">
        <div className="divide-y divide-zinc-200 dark:divide-zinc-800">
          {events.map((event: any) => (
            <div key={event.id} className="p-4 hover:bg-gray-50 dark:hover:bg-zinc-800/50 transition-colors">
              <div className="flex items-start gap-3">
                <div className="flex-shrink-0 w-8 h-8 rounded-full bg-purple-100 dark:bg-purple-900/30 flex items-center justify-center">
                  <MaterialSymbol name="bolt" size="sm" className="text-purple-600 dark:text-purple-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <p className="text-xs font-mono text-gray-600 dark:text-zinc-400 truncate">
                      {event.id}
                    </p>
                  </div>

                  {event.channel && (
                    <div className="flex items-center gap-1 mb-1">
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300">
                        {event.channel}
                      </span>
                    </div>
                  )}

                  <p className="text-xs text-gray-400 dark:text-zinc-500">
                    Created {formatTimeAgo(new Date(event.createdAt))}
                  </p>

                  {event.data && Object.keys(event.data).length > 0 && (
                    <div className="mt-3 space-y-2">
                      <div className="text-xs font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
                        Event Data
                      </div>
                      <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 rounded text-left">
                        <JsonView
                          value={event.data}
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
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
