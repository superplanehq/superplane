import { useQuery } from '@tanstack/react-query'
import { workflowsListNodeQueueItems } from '../../api-client/sdk.gen'
import { MaterialSymbol } from '../MaterialSymbol/material-symbol'
import { withOrganizationHeader } from '../../utils/withOrganizationHeader'
import { useState, useEffect } from 'react'
import JsonView from '@uiw/react-json-view'
import { lightTheme } from '@uiw/react-json-view/light'
import { darkTheme } from '@uiw/react-json-view/dark'

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

interface WorkflowNodeQueueTabProps {
  workflowId: string
  nodeId: string
}

export const WorkflowNodeQueueTab = ({ workflowId, nodeId }: WorkflowNodeQueueTabProps) => {
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

  const { data, isLoading, error } = useQuery({
    queryKey: ['workflow-node-queue', workflowId, nodeId],
    queryFn: async () => {
      const response = await workflowsListNodeQueueItems(
        withOrganizationHeader({
          path: {
            workflowId: workflowId,
            nodeId: nodeId,
          },
        })
      )
      return response.data
    },
    refetchInterval: 5000, // Refresh every 5 seconds
  })

  if (isLoading) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600 mb-3"></div>
        <p className="text-sm text-gray-500 dark:text-zinc-400">Loading queue...</p>
      </div>
    )
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6">
        <MaterialSymbol name="error" className="text-red-500 mb-3" size="xl" />
        <p className="text-sm text-gray-900 dark:text-zinc-100 font-medium mb-1">Failed to load queue</p>
        <p className="text-xs text-gray-500 dark:text-zinc-400">{error.message}</p>
      </div>
    )
  }

  const queueItems = data?.queueItems || []

  if (queueItems.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full p-6">
        <MaterialSymbol name="inbox" className="text-gray-400 dark:text-zinc-500 mb-3" size="xl" />
        <p className="text-sm text-gray-600 dark:text-zinc-400 font-medium mb-1">No items in queue</p>
        <p className="text-xs text-gray-500 dark:text-zinc-500 text-center">
          Events will appear here when they are queued for execution
        </p>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex-shrink-0 px-4 py-3 border-b border-zinc-200 dark:border-zinc-800">
        <p className="text-xs text-gray-500 dark:text-zinc-400">
          {queueItems.length} {queueItems.length === 1 ? 'event' : 'events'} in queue
        </p>
      </div>
      <div className="flex-1 overflow-y-auto">
        <div className="divide-y divide-zinc-200 dark:divide-zinc-800">
          {queueItems.map((item: any) => (
            <div key={item.eventId} className="p-4 hover:bg-gray-50 dark:hover:bg-zinc-800/50 transition-colors">
              <div className="flex items-start gap-3">
                <div className="flex-shrink-0 w-8 h-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                  <MaterialSymbol name="schedule" size="sm" className="text-blue-600 dark:text-blue-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <p className="text-xs font-mono text-gray-600 dark:text-zinc-400 truncate">
                      {item.eventId}
                    </p>
                  </div>

                  {item.event?.state && (
                    <div className="mb-2">
                      <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                        item.event.state === 'routing' ? 'bg-yellow-100 dark:bg-yellow-900/30 text-yellow-800 dark:text-yellow-300' :
                        item.event.state === 'processing' ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-800 dark:text-blue-300' :
                        item.event.state === 'completed' ? 'bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300' :
                        item.event.state === 'failed' ? 'bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300' :
                        'bg-gray-100 dark:bg-gray-900/30 text-gray-800 dark:text-gray-300'
                      }`}>
                        {item.event.state}
                      </span>
                    </div>
                  )}

                  {item.event?.blueprintName && (
                    <p className="text-xs text-gray-500 dark:text-zinc-400 mb-1">
                      Blueprint: <span className="font-medium">{item.event.blueprintName}</span>
                    </p>
                  )}

                  <p className="text-xs text-gray-400 dark:text-zinc-500">
                    Queued {formatTimeAgo(new Date(item.createdAt))}
                  </p>

                  {item.event?.data && Object.keys(item.event.data).length > 0 && (
                    <div className="mt-3 space-y-2">
                      <div className="text-xs font-semibold text-gray-700 dark:text-zinc-300 uppercase tracking-wide border-b border-gray-200 dark:border-zinc-700 pb-1">
                        Event Data
                      </div>
                      <div className="bg-zinc-50 dark:bg-zinc-800 border border-gray-200 dark:border-zinc-700 p-3 rounded text-left">
                        <JsonView
                          value={item.event.data}
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
