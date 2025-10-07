import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Heading } from '../../components/Heading/heading'
import { Text } from '../../components/Text/text'
import { Button } from '../../components/Button/button'
import { MaterialSymbol } from '../../components/MaterialSymbol/material-symbol'
import { CreateCanvasModal } from '../../components/CreateCanvasModal'
import { CreateBlueprintModal } from '../../components/CreateBlueprintModal'
import { CreateWorkflowModal } from '../../components/CreateWorkflowModal'
import { CanvasCard, CanvasCardData } from '../../components/CanvasCard'
import { useOrganizationCanvases, useCreateCanvas, useOrganizationUsers } from '../../hooks/useOrganizationData'
import { useBlueprints, useCreateBlueprint } from '../../hooks/useBlueprintData'
import { useWorkflows, useCreateWorkflow } from '../../hooks/useWorkflowData'
import { SuperplaneCanvas } from '../../api-client'
import { useAccount } from '../../contexts/AccountContext'

interface UserData {
  metadata?: {
    id?: string
    email?: string
  }
  spec?: {
    displayName?: string
    accountProviders?: Array<{
      avatarUrl?: string
      displayName?: string
      email?: string
    }>
  }
}

const createUserDisplayNames = (orgUsers: UserData[]) => {
  const map: Record<string, { name: string; initials: string; avatar?: string }> = {}
  orgUsers.forEach(user => {
    if (user.metadata?.id) {
      const name = user.spec?.displayName || user.metadata?.email || user.metadata.id
      const initials = name.split(' ').map(n => n[0]).join('').toUpperCase()
      const avatar = user.spec?.accountProviders?.[0]?.avatarUrl
      map[user.metadata.id] = { name, initials, avatar }
    }
  })
  return map
}

type TabType = 'canvases' | 'blueprints' | 'workflows'

interface BlueprintCardData {
  id: string
  name: string
  description?: string
  createdAt: string
  type: 'blueprint'
}

interface WorkflowCardData {
  id: string
  name: string
  description?: string
  createdAt: string
  type: 'workflow'
}

// Home page component - displays canvases and blueprints for the current user's organization
const HomePage = () => {
  const [searchQuery, setSearchQuery] = useState('')
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid')
  const [activeTab, setActiveTab] = useState<TabType>('canvases')
  const [showCreateCanvasModal, setShowCreateCanvasModal] = useState(false)
  const [showCreateBlueprintModal, setShowCreateBlueprintModal] = useState(false)
  const [showCreateWorkflowModal, setShowCreateWorkflowModal] = useState(false)
  const { organizationId } = useParams<{ organizationId: string }>()
  const { account } = useAccount()
  const navigate = useNavigate()

  // Use the organization canvases hook with organization ID from URL
  const { data: canvasesData = [], isLoading: canvasesLoading, error: canvasApiError } = useOrganizationCanvases(organizationId || '')
  const { data: orgUsers = [], isLoading: usersLoading } = useOrganizationUsers(organizationId || '')
  const { data: blueprintsData = [], isLoading: blueprintsLoading, error: blueprintApiError } = useBlueprints(organizationId || '')
  const { data: workflowsData = [], isLoading: workflowsLoading, error: workflowApiError } = useWorkflows(organizationId || '')
  const createCanvasMutation = useCreateCanvas(organizationId || '')
  const createBlueprintMutation = useCreateBlueprint(organizationId || '')
  const createWorkflowMutation = useCreateWorkflow(organizationId || '')

  const canvasError = canvasApiError ? 'Failed to fetch canvases. Please try again later.' : null
  const blueprintError = blueprintApiError ? 'Failed to fetch blueprints. Please try again later.' : null
  const workflowError = workflowApiError ? 'Failed to fetch workflows. Please try again later.' : null

  // Create user display names mapping for organization users
  const userDisplayNames = createUserDisplayNames(orgUsers)

  // Transform API data to match CanvasCardData interface
  const canvases: CanvasCardData[] = canvasesData.map((canvas: SuperplaneCanvas) => {
    const createdById = canvas.metadata?.createdBy
    const creator = createdById ? userDisplayNames[createdById] : null

    return {
      id: canvas.metadata!.id!,
      name: canvas.metadata!.name!,
      description: canvas.metadata!.description,
      createdAt: canvas.metadata!.createdAt ? new Date(canvas.metadata!.createdAt!).toLocaleDateString() : 'Unknown',
      createdBy: {
        name: creator?.name || 'Unknown User',
        initials: creator?.initials || '?',
        avatar: creator?.avatar,
      },
      type: 'canvas' as const
    }
  })

  // Transform blueprint data
  const blueprints: BlueprintCardData[] = (blueprintsData || []).map((blueprint: any) => ({
    id: blueprint.id!,
    name: blueprint.name!,
    description: blueprint.description,
    createdAt: blueprint.createdAt ? new Date(blueprint.createdAt).toLocaleDateString() : 'Unknown',
    type: 'blueprint' as const
  }))

  // Transform workflow data
  const workflows: WorkflowCardData[] = (workflowsData || []).map((workflow: any) => ({
    id: workflow.id!,
    name: workflow.name!,
    description: workflow.description,
    createdAt: workflow.createdAt ? new Date(workflow.createdAt).toLocaleDateString() : 'Unknown',
    type: 'workflow' as const
  }))

  // Filter items based on search and active tab
  const filteredCanvases = canvases.filter(canvas => {
    const matchesSearch = canvas.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      canvas.description?.toLowerCase().includes(searchQuery.toLowerCase())
    return matchesSearch
  })

  const filteredBlueprints = blueprints.filter(blueprint => {
    const matchesSearch = blueprint.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      blueprint.description?.toLowerCase().includes(searchQuery.toLowerCase())
    return matchesSearch
  })

  const filteredWorkflows = workflows.filter(workflow => {
    const matchesSearch = workflow.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      workflow.description?.toLowerCase().includes(searchQuery.toLowerCase())
    return matchesSearch
  })

  // Modal handlers
  const handleCreateCanvasClick = () => {
    setShowCreateCanvasModal(true)
  }

  const handleCreateBlueprintClick = () => {
    setShowCreateBlueprintModal(true)
  }

  const handleCreateWorkflowClick = () => {
    setShowCreateWorkflowModal(true)
  }

  const handleCreateCanvasClose = () => {
    setShowCreateCanvasModal(false)
  }

  const handleCreateBlueprintClose = () => {
    setShowCreateBlueprintModal(false)
  }

  const handleCreateWorkflowClose = () => {
    setShowCreateWorkflowModal(false)
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

      if (result) {
        setShowCreateCanvasModal(false)
        navigate(`/${organizationId}/canvas/${result.data?.canvas?.metadata?.id}`)
      }
    }
  }

  const handleCreateBlueprintSubmit = async (data: { name: string; description?: string }) => {
    if (organizationId) {
      const result = await createBlueprintMutation.mutateAsync({
        name: data.name,
        description: data.description,
      })

      if (result?.data?.blueprint?.id) {
        setShowCreateBlueprintModal(false)
        navigate(`/${organizationId}/blueprints/${result.data.blueprint.id}`)
      }
    }
  }

  const handleCreateWorkflowSubmit = async (data: { name: string; description?: string }) => {
    if (organizationId) {
      const result = await createWorkflowMutation.mutateAsync({
        name: data.name,
        description: data.description,
      })

      if (result?.data?.workflow?.id) {
        setShowCreateWorkflowModal(false)
        navigate(`/${organizationId}/workflows/${result.data.workflow.id}`)
      }
    }
  }

  const isLoading = (activeTab === 'canvases' && (canvasesLoading || usersLoading)) ||
                     (activeTab === 'blueprints' && blueprintsLoading) ||
                     (activeTab === 'workflows' && workflowsLoading)

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-40">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <p className="ml-3 text-gray-500">Loading...</p>
      </div>
    )
  }

  if (!account || !organizationId) {
    return (
      <div className="text-center py-8">
        <p className="text-gray-500">Unable to load user information</p>
      </div>
    )
  }

  const error = activeTab === 'canvases' ? canvasError : activeTab === 'blueprints' ? blueprintError : workflowError
  const currentItems = activeTab === 'canvases' ? filteredCanvases : activeTab === 'blueprints' ? filteredBlueprints : filteredWorkflows

  return (
    <div className="min-h-screen flex flex-col bg-zinc-50 dark:bg-zinc-900 pt-10">
      {/* Main Content */}
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className='bg-zinc-50 dark:bg-zinc-900 w-full flex-grow-1 p-6'>
          <div className="p-4">
            {/* Page Header */}
            <div className='flex items-center justify-between mb-8'>
              <Heading level={2} className="!text-2xl mb-2">
                {activeTab === 'canvases' ? 'Canvases' : activeTab === 'blueprints' ? 'Blueprints' : 'Workflows'}
              </Heading>
              <Button
                color="blue"
                className='flex items-center bg-blue-700 text-white hover:bg-blue-600'
                onClick={activeTab === 'canvases' ? handleCreateCanvasClick : activeTab === 'blueprints' ? handleCreateBlueprintClick : handleCreateWorkflowClick}
              >
                <MaterialSymbol name="add" className="mr-2" />
                New {activeTab === 'canvases' ? 'Canvas' : activeTab === 'blueprints' ? 'Blueprint' : 'Workflow'}
              </Button>
            </div>

            {/* Tabs */}
            <div className="flex border-b border-zinc-200 dark:border-zinc-700 mb-6">
              <button
                onClick={() => setActiveTab('canvases')}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === 'canvases'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300'
                }`}
              >
                Canvases ({canvases.length})
              </button>
              <button
                onClick={() => setActiveTab('blueprints')}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === 'blueprints'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300'
                }`}
              >
                Blueprints ({blueprints.length})
              </button>
              <button
                onClick={() => setActiveTab('workflows')}
                className={`px-4 py-2 text-sm font-medium border-b-2 transition-colors ${
                  activeTab === 'workflows'
                    ? 'border-blue-600 text-blue-600'
                    : 'border-transparent text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300'
                }`}
              >
                Workflows ({workflows.length})
              </button>
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
                      placeholder={`Search ${activeTab}...`}
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      className="h-9 w-full pl-10 pr-4 py-2 border border-zinc-200 dark:border-zinc-700 rounded-lg bg-white dark:bg-zinc-800 text-zinc-900 dark:text-zinc-100 placeholder-zinc-500 focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                    />
                  </div>
                </div>
              </div>

              {/* View Mode Toggle */}
              <div className="flex items-center">
                <Button
                  {...(viewMode === 'grid' ? { color: 'light' as const } : { plain: true })}
                  onClick={() => setViewMode('grid')}
                  title="Grid view"
                >
                  <MaterialSymbol name="grid_view" />
                </Button>
                <Button
                  {...(viewMode === 'list' ? { color: 'light' as const } : { plain: true })}
                  onClick={() => setViewMode('list')}
                  title="List view"
                >
                  <MaterialSymbol name="view_list" />
                </Button>
              </div>
            </div>

            {/* Loading State */}
            {isLoading ? (
              <div className="flex justify-center items-center h-40">
                <Text className="text-zinc-500">Loading {activeTab}...</Text>
              </div>
            ) : error ? (
              <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
                <Text>{error}</Text>
              </div>
            ) : (
              <>
                {/* Items Display */}
                {activeTab === 'canvases' ? (
                  viewMode === 'grid' ? (
                    <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6">
                      {filteredCanvases.map((canvas) => (
                        <CanvasCard
                          key={canvas.id}
                          canvas={canvas}
                          organizationId={organizationId!}
                          variant="grid"
                        />
                      ))}
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {filteredCanvases.map((canvas) => (
                        <CanvasCard
                          key={canvas.id}
                          canvas={canvas}
                          organizationId={organizationId!}
                          variant="list"
                        />
                      ))}
                    </div>
                  )
                ) : activeTab === 'blueprints' ? (
                  // Blueprints
                  viewMode === 'grid' ? (
                    <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6">
                      {filteredBlueprints.map((blueprint) => (
                        <div key={blueprint.id} className="max-h-45 bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-md transition-shadow">
                          <div className="p-6 flex flex-col justify-between h-full">
                            <div>
                              <div className="flex items-start mb-4">
                                <div className="flex items-start justify-between space-x-3 flex-1">
                                  <div className='flex flex-col flex-1 min-w-0'>
                                    <button
                                      onClick={() => navigate(`/${organizationId}/blueprints/${blueprint.id}`)}
                                      className="block text-left w-full"
                                    >
                                      <Heading level={3} className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-0 !leading-6 line-clamp-2 max-w-[15vw] truncate">
                                        {blueprint.name}
                                      </Heading>
                                    </button>
                                  </div>
                                </div>
                              </div>

                              <div className="mb-4">
                                <Text className="text-sm text-left text-zinc-600 dark:text-zinc-400 line-clamp-2 mt-2">
                                  {blueprint.description || 'No description'}
                                </Text>
                              </div>
                            </div>

                            <div className="flex justify-between items-center">
                              <div className="text-zinc-500 text-left">
                                <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none">
                                  Created at {blueprint.createdAt}
                                </p>
                              </div>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {filteredBlueprints.map((blueprint) => (
                        <div key={blueprint.id} className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-sm transition-shadow p-4">
                          <button
                            onClick={() => navigate(`/${organizationId}/blueprints/${blueprint.id}`)}
                            className="block text-left w-full"
                          >
                            <Heading level={3} className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-1">
                              {blueprint.name}
                            </Heading>
                            <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                              {blueprint.description || 'No description'}
                            </Text>
                            <Text className="text-xs text-zinc-500 mt-2">
                              Created at {blueprint.createdAt}
                            </Text>
                          </button>
                        </div>
                      ))}
                    </div>
                  )
                ) : (
                  // Workflows
                  viewMode === 'grid' ? (
                    <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-4 gap-6">
                      {filteredWorkflows.map((workflow) => (
                        <div key={workflow.id} className="max-h-45 bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-md transition-shadow">
                          <div className="p-6 flex flex-col justify-between h-full">
                            <div>
                              <div className="flex items-start mb-4">
                                <div className="flex items-start justify-between space-x-3 flex-1">
                                  <div className='flex flex-col flex-1 min-w-0'>
                                    <button
                                      onClick={() => navigate(`/${organizationId}/workflows/${workflow.id}`)}
                                      className="block text-left w-full"
                                    >
                                      <Heading level={3} className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-0 !leading-6 line-clamp-2 max-w-[15vw] truncate">
                                        {workflow.name}
                                      </Heading>
                                    </button>
                                  </div>
                                </div>
                              </div>

                              <div className="mb-4">
                                <Text className="text-sm text-left text-zinc-600 dark:text-zinc-400 line-clamp-2 mt-2">
                                  {workflow.description || 'No description'}
                                </Text>
                              </div>
                            </div>

                            <div className="flex justify-between items-center">
                              <div className="text-zinc-500 text-left">
                                <p className="text-xs text-zinc-600 dark:text-zinc-400 leading-none">
                                  Created at {workflow.createdAt}
                                </p>
                              </div>
                            </div>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <div className="space-y-2">
                      {filteredWorkflows.map((workflow) => (
                        <div key={workflow.id} className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 hover:shadow-sm transition-shadow p-4">
                          <button
                            onClick={() => navigate(`/${organizationId}/workflows/${workflow.id}`)}
                            className="block text-left w-full"
                          >
                            <Heading level={3} className="!text-md font-semibold text-zinc-900 dark:text-white hover:text-blue-600 dark:hover:text-blue-400 transition-colors mb-1">
                              {workflow.name}
                            </Heading>
                            <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                              {workflow.description || 'No description'}
                            </Text>
                            <Text className="text-xs text-zinc-500 mt-2">
                              Created at {workflow.createdAt}
                            </Text>
                          </button>
                        </div>
                      ))}
                    </div>
                  )
                )}

                {/* Empty State */}
                {currentItems.length === 0 && (
                  <div className="text-center py-12">
                    <MaterialSymbol name={activeTab === 'canvases' ? 'automation' : activeTab === 'blueprints' ? 'architecture' : 'account_tree'} className="mx-auto text-zinc-400 mb-4" size="xl" />
                    <Heading level={3} className="text-lg text-zinc-900 dark:text-white mb-2">
                      {searchQuery ? `No ${activeTab} found` : `No ${activeTab} yet`}
                    </Heading>
                    <Text className="text-zinc-600 dark:text-zinc-400 mb-6">
                      {searchQuery
                        ? 'Try adjusting your search criteria.'
                        : `Get started by creating your first ${activeTab === 'canvases' ? 'canvas' : activeTab === 'blueprints' ? 'blueprint' : 'workflow'}.`}
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

      {/* Create Blueprint Modal */}
      <CreateBlueprintModal
        isOpen={showCreateBlueprintModal}
        onClose={handleCreateBlueprintClose}
        onSubmit={handleCreateBlueprintSubmit}
        isLoading={createBlueprintMutation.isPending}
      />

      {/* Create Workflow Modal */}
      <CreateWorkflowModal
        isOpen={showCreateWorkflowModal}
        onClose={handleCreateWorkflowClose}
        onSubmit={handleCreateWorkflowSubmit}
        isLoading={createWorkflowMutation.isPending}
      />
    </div>
  )
}

export default HomePage
