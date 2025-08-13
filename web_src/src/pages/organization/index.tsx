import { useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { Heading } from '../../components/Heading/heading'
import { Text } from '../../components/Text/text'
import { Button } from '../../components/Button/button'
import { MaterialSymbol } from '../../components/MaterialSymbol/material-symbol'
import { Avatar } from '../../components/Avatar/avatar'
import { CreateCanvasModal } from '../../components/CreateCanvasModal'
import { useOrganizationCanvases, useCreateCanvas } from '../../hooks/useOrganizationData'
import { useUserStore } from '../../stores/userStore'
import { SuperplaneCanvas } from '../../api-client'

interface Canvas {
  id: string
  name: string
  description?: string
  createdAt: string
  createdBy: {
    name: string
    avatar?: string
    initials: string
  }
  type: 'canvas'
}

const HomePage = () => {
  const { orgId: organizationId } = useParams<{ orgId: string }>()
  const [searchQuery, setSearchQuery] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const showIcons = false
  const navigate = useNavigate()

  // New canvas modal state
  const [showCreateCanvasModal, setShowCreateCanvasModal] = useState(false)

  // Get user data
  const { user } = useUserStore()

  // Use the new hooks
  const { data: canvasesData = [], isLoading: loading, error: apiError } = useOrganizationCanvases(organizationId || '')
  const createCanvasMutation = useCreateCanvas(organizationId || '')

  const error = apiError ? 'Failed to fetch canvases. Please try again later.' : null

  // Transform API data to match Canvas interface
  const canvases: Canvas[] = canvasesData.map((canvas: SuperplaneCanvas) => ({
    id: canvas.metadata!.id!,
    name: canvas.metadata!.name!,
    description: canvas.metadata!.description,
    createdAt: canvas.metadata!.createdAt ? new Date(canvas.metadata!.createdAt!).toLocaleDateString() : 'Unknown',
    createdBy: {
      name: user?.name || 'Unknown User',
      initials: user?.name ? user.name.split(' ').map(n => n[0]).join('').toUpperCase() : '?',
      avatar: user?.avatar_url,
    },
    type: 'canvas' as const
  }))

  const getCanvasIcon = () => {
    return <MaterialSymbol name="automation" size='md' weight={400} className="text-blue-600" />
  }

  // Filter canvases based on search only
  const filteredCanvases = canvases.filter(canvas => {
    const matchesSearch = canvas.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      canvas.description?.toLowerCase().includes(searchQuery.toLowerCase())
    return matchesSearch
  })

  // New canvas modal handlers
  const handleCreateCanvasClick = () => {
    setShowCreateCanvasModal(true)
  }

  const handleCreateCanvasClose = () => {
    setShowCreateCanvasModal(false)
  }

  const handleCreateCanvasSubmit = async (data: { name: string; description?: string }) => {
    if (organizationId) {
      const result = await createCanvasMutation.mutateAsync({
        canvas: {
          metadata: {
            name: data.name,
            description: data.description,
          },
        },
        organizationId: organizationId
      })

      navigate(`/organization/${organizationId}/canvas/${result.data?.canvas?.metadata?.id}`)
    }
  }

  return (
    <div className="min-h-screen flex flex-col bg-zinc-50 dark:bg-zinc-900 pt-10">
      {/* Main Content */}
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className='bg-zinc-50 dark:bg-zinc-900 w-full flex-grow-1 p-6 mt-3'>
          <div className="p-4">
            {/* Page Header */}
            <div className='flex items-center justify-between mb-8'>
              <Heading level={1} className="!text-3xl mb-2">Canvases</Heading>
              <Button
                color='blue'
                className='flex items-center bg-blue-700 text-white hover:bg-blue-600'
                onClick={handleCreateCanvasClick}
              >
                <MaterialSymbol name="add" className="mr-2" />
                New Canvas
              </Button>
            </div>

            {/* Actions and Filters */}
            <div className="flex flex-col sm:flex-row gap-4 mb-6 justify-between">
              {/* Search */}
              <div className='flex items-center gap-2'>
                <div className="flex-1 w-100">
                  <div className="relative">
                    <MaterialSymbol name="search" className="absolute left-3 top-1/2 transform -translate-y-1/2 text-zinc-400" />
                    <input
                      type="text"
                      placeholder="Search canvases..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="h-9 w-full pl-10 pr-4 py-2 border border-zinc-200 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 placeholder-zinc-500 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                  </div>
                </div>
              </div>

              {/* View Mode Toggle */}
              <div className="flex items-center">
                {viewMode === 'list' ? (
                  <Button
                    plain
                    onClick={() => setViewMode('grid')}
                    title="Grid view"
                  >
                    <MaterialSymbol name="grid_view" />
                  </Button>
                ) : (
                  <Button
                    color='light'
                    onClick={() => setViewMode('grid')}
                    title="Grid view"
                  >
                    <MaterialSymbol name="grid_view" />
                  </Button>
                )}
                {viewMode === 'grid' ? (
                  <Button
                    plain
                    onClick={() => setViewMode('list')}
                    title="List view"
                  >
                    <MaterialSymbol name="view_list" />
                  </Button>
                ) : (
                  <Button
                    color='light'
                    onClick={() => setViewMode('list')}
                    title="List view"
                  >
                    <MaterialSymbol name="view_list" />
                  </Button>
                )}
              </div>
            </div>

            {/* Loading State */}
            {loading ? (
              <div className="flex justify-center items-center h-40">
                <Text className="text-zinc-500">Loading canvases...</Text>
              </div>
            ) : error ? (
              <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                <Text>{error}</Text>
              </div>
            ) : (
              <>
                {/* Canvases Display */}
                {viewMode === 'grid' ? (
                  /* Grid View */
                  <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6">
                    {filteredCanvases.map((canvas) => (
                      <div key={canvas.id} className="max-h-45 bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-md transition-shadow group">
                        <div className="p-6 flex flex-col justify-between h-full">
                          <div>
                            {/* Header */}
                            <div className="flex items-start mb-4">
                              <div className="flex items-start justify-between space-x-3 flex-1">
                                {showIcons && (
                                  <div className="p-1 bg-zinc-100 dark:bg-zinc-700 rounded-md h-8 w-8 text-center">
                                    {getCanvasIcon()}
                                  </div>
                                )}
                                <div className='flex flex-col flex-1'>
                                  <Link
                                    to={`/organization/${organizationId}/canvas/${canvas.id}`}
                                    className="block text-left w-full"
                                  >
                                    <Heading level={3} className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-0 !leading-6">
                                      {canvas.name}
                                    </Heading>
                                  </Link>
                                </div>
                              </div>
                            </div>

                            {/* Content */}
                            <div className="mb-4">

                              <Text className="text-sm text-left text-zinc-600 dark:text-zinc-400 line-clamp-2 mt-2">
                                {canvas.description || ''}
                              </Text>
                            </div>
                          </div>

                          {/* Footer */}
                          <div className="flex justify-between items-center">
                            <div className='flex items-center space-x-2'>
                              <Avatar
                                src={canvas.createdBy.avatar}
                                initials={canvas.createdBy.initials}
                                alt={canvas.createdBy.name}
                                className="w-6 h-6 bg-blue-700 dark:bg-blue-900 text-blue-100 dark:text-blue-100"
                              />
                              <div className="text-zinc-500 text-left">
                                <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none mb-1">
                                  Created by <strong>{canvas.createdBy.name}</strong>
                                </p>
                                <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none">
                                  Created at {canvas.createdAt}
                                </p>
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : (
                  /* List View */
                  <div className="space-y-2">
                    {filteredCanvases.map((canvas) => (
                      <div key={canvas.id} className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-sm transition-shadow group">
                        <div className="p-4 pl-6">
                          <div className="flex items-center justify-between">
                            <div className="flex items-center space-x-4 flex-1">
                              {/* Icon */}
                              {showIcons && (
                                <div className="p-2 bg-zinc-100 dark:bg-zinc-700 rounded-lg flex-shrink-0">
                                  {getCanvasIcon()}
                                </div>
                              )}

                              {/* Content */}
                              <div className="flex-1 min-w-0">
                                <div className="flex items-center space-x-3 mb-1">
                                  <Link
                                    to={`/organization/${organizationId}/canvas/${canvas.id}`}
                                    className="block text-left"
                                  >
                                    <Heading level={3} className="text-base font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors truncate">
                                      {canvas.name}
                                    </Heading>
                                  </Link>
                                </div>

                                <Text className="text-sm text-left text-zinc-600 dark:text-zinc-400 mb-2 line-clamp-1 !mb-0">
                                  {canvas.description || ''}
                                </Text>
                              </div>
                            </div>

                            {/* Actions */}
                            <div className="flex items-center space-x-2 flex-shrink-0">
                              <div className='flex items-center space-x-2'>
                                <div className="text-zinc-500 text-right">
                                  <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none mb-1">
                                    Created by <strong>{canvas.createdBy.name}</strong>
                                  </p>
                                  <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none">
                                    Created at {canvas.createdAt}
                                  </p>
                                </div>
                                <Avatar
                                  src={canvas.createdBy.avatar}
                                  initials={canvas.createdBy.initials}
                                  alt={canvas.createdBy.name}
                                  className="w-6 h-6 bg-blue-700 dark:bg-blue-900 text-blue-100 dark:text-blue-100"
                                />
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {/* Empty State */}
                {filteredCanvases.length === 0 && (
                  <div className="text-center py-12">
                    <MaterialSymbol name="automation" className="mx-auto text-zinc-400 mb-4" size="xl" />
                    <Heading level={3} className="text-lg text-zinc-900 dark:text-white mb-2">
                      {searchQuery ? 'No canvases found' : 'No canvases yet'}
                    </Heading>
                    <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
                      {searchQuery
                        ? 'Try adjusting your search criteria.'
                        : 'Get started by creating your first canvas.'}
                    </Text>
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      </main>

      {/* Create Canvas Modal */}
      <CreateCanvasModal
        isOpen={showCreateCanvasModal}
        onClose={handleCreateCanvasClose}
        onSubmit={handleCreateCanvasSubmit}
        isLoading={createCanvasMutation.isPending}
      />
    </div>
  )
}

export default HomePage 