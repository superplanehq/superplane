import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Heading } from '../../../components/Heading/heading'
import { Button } from '../../../components/Button/button'
import { Input, InputGroup } from '../../../components/Input/input'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import debounce from 'lodash.debounce'
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem
} from '../../../components/Dropdown/dropdown'
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell
} from '../../../components/Table/table'
import { useOrganizationRoles, useDeleteRole, useOrganizationCanvases } from '../../../hooks/useOrganizationData'
import { useCanvasRoles } from '../../../hooks/useCanvasData'
import { RolesRole } from '../../../api-client/types.gen'
import { Select, type SelectOption } from '../../../components/Select/index'

interface RolesProps {
  organizationId: string
}


export function Roles({ organizationId }: RolesProps) {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [selectedCanvasId, setSelectedCanvasId] = useState<string>('')
  const [sortConfig, setSortConfig] = useState<{
    key: string | null
    direction: 'asc' | 'desc'
  }>({
    key: null,
    direction: 'asc'
  })

  const setDebouncedSearch = debounce((search: string) => setSearch(search), 500)

  // Use React Query hooks for data fetching
  const { data: roles = [], isLoading: loadingRoles, error } = useOrganizationRoles(organizationId)
  const { data: canvases = [] } = useOrganizationCanvases(organizationId)

  // Mutation for role deletion
  const deleteRoleMutation = useDeleteRole(organizationId)

  const handleCreateRole = () => {
    navigate(`/${organizationId}/settings/create-role`)
  }

  const handleEditRole = (role: RolesRole) => {
    navigate(`/${organizationId}/settings/create-role/${role.metadata?.name}`)
  }

  const handleDeleteRole = async (role: RolesRole) => {
    if (!role.metadata?.name) return

    const confirmed = window.confirm(
      `Are you sure you want to delete the role "${role.metadata?.name}"? This action cannot be undone.`
    )

    if (!confirmed) return

    try {
      await deleteRoleMutation.mutateAsync({
        roleName: role.metadata?.name,
        domainType: 'DOMAIN_TYPE_ORGANIZATION',
        domainId: organizationId
      })
    } catch (err) {
      console.error('Error deleting role:', err)
    }
  }

  const handleSort = (key: string) => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc'
    }))
  }

  const getSortedData = (data: RolesRole[]) => {
    if (!sortConfig.key) return data

    return [...data].sort((a, b) => {
      let aValue: string | number
      let bValue: string | number

      switch (sortConfig.key) {
        case 'name':
          aValue = (a.spec?.displayName || a.metadata?.name || '').toLowerCase()
          bValue = (b.spec?.displayName || b.metadata?.name || '').toLowerCase()
          break
        case 'permissions':
          aValue = a.spec?.permissions?.length || 0
          bValue = b.spec?.permissions?.length || 0
          break
        default:
          return 0
      }

      if (aValue < bValue) {
        return sortConfig.direction === 'asc' ? -1 : 1
      }
      if (aValue > bValue) {
        return sortConfig.direction === 'asc' ? 1 : -1
      }
      return 0
    })
  }

  const isDefaultRole = (roleName: string | undefined) => {
    if (!roleName) return false
    const defaultRoles = ['org_viewer', 'org_admin', 'org_owner']
    return defaultRoles.includes(roleName)
  }

  const getSortIcon = (columnKey: string) => {
    if (sortConfig.key !== columnKey) {
      return 'unfold_more'
    }
    return sortConfig.direction === 'asc' ? 'keyboard_arrow_up' : 'keyboard_arrow_down'
  }



  const filteredAndSortedRoles = useMemo(() => {
    const filtered = roles.filter((role) => {
      if (search === '') {
        return true
      }
      // Search by display name if available, otherwise by name
      const searchText = role.spec?.displayName || role.metadata?.name || ''
      return searchText.toLowerCase().includes(search.toLowerCase())
    })
    return getSortedData(filtered)
  }, [roles, search, getSortedData])

  // Canvas options for the select
  const canvasOptions: SelectOption[] = useMemo(() => {
    return canvases
      .filter((canvas) => canvas.metadata?.id)
      .map((canvas) => ({
        value: canvas.metadata!.id!,
        label: canvas.metadata?.name || 'Unnamed Canvas',
        description: canvas.metadata?.description,
      }))
  }, [canvases])

  // Get canvas roles for selected canvas
  const { data: canvasRoles = [], isLoading: loadingCanvasRoles } = useCanvasRoles(selectedCanvasId)

  const filteredCanvasRoles = useMemo(() => {
    const filtered = canvasRoles.filter((role) => {
      if (search === '') {
        return true
      }
      const searchText = role.spec?.displayName || role.metadata?.name || ''
      return searchText.toLowerCase().includes(search.toLowerCase())
    })
    return getSortedData(filtered)
  }, [canvasRoles, search, getSortedData])

  const selectedCanvas = canvases.find(canvas => canvas.metadata?.id === selectedCanvasId)

  return (
    <div className="space-y-8 pt-6 text-left">
      <div className="flex items-center justify-between">
        <div>
          <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-1">
            Roles
          </Heading>
        </div>
      </div>
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error instanceof Error ? error.message : 'Failed to fetch roles'}</p>
        </div>
      )}

      {/* Search Bar - shared across both sections */}
      <div className="flex justify-center">
        <InputGroup>
          <Input
            name="search"
            placeholder="Search Organization and Canvas Roles…"
            aria-label="Search"
            className="w-96"
            onChange={(e) => setDebouncedSearch(e.target.value)}
          />
        </InputGroup>
      </div>

      {/* Organization Roles Section */}
      <div>
        <div className="mb-4">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-lg font-medium text-zinc-900 dark:text-white">
                Organization Roles
              </h3>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                Roles that apply across the entire organization
              </p>
            </div>
            <Button
              color="blue"
              className='flex items-center'
              onClick={handleCreateRole}
            >
              <MaterialSymbol name="add" />
              New Organization Role
            </Button>
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
          <div className="px-6 pb-6 pt-6">
            {loadingRoles ? (
              <div className="flex justify-center items-center h-32">
                <p className="text-zinc-500 dark:text-zinc-400">Loading organization roles...</p>
              </div>
            ) : (
              <Table dense>
                <TableHead>
                  <TableRow>
                    <TableHeader
                      className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                      onClick={() => handleSort('name')}
                    >
                      <div className="flex items-center gap-2">
                        Role name
                        <MaterialSymbol name={getSortIcon('name')} size="sm" className="text-zinc-400" />
                      </div>
                    </TableHeader>
                    <TableHeader
                      className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                      onClick={() => handleSort('permissions')}
                    >
                      <div className="flex items-center gap-2">
                        Permissions
                        <MaterialSymbol name={getSortIcon('permissions')} size="sm" className="text-zinc-400" />
                      </div>
                    </TableHeader>
                    <TableHeader></TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredAndSortedRoles.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={3} className="text-center py-8 text-zinc-500 dark:text-zinc-400">
                        No organization roles found
                      </TableCell>
                    </TableRow>
                  ) : (
                    filteredAndSortedRoles.map((role, index) => {
                      const isDefault = isDefaultRole(role.metadata?.name)
                      return (
                        <TableRow key={role.metadata?.name || index}>
                          <TableCell className="font-semibold">
                            {role.spec?.displayName || role.metadata?.name}
                          </TableCell>
                          <TableCell>{role.spec?.permissions?.length || 0}</TableCell>
                          <TableCell>
                            <div className="flex justify-end">
                              <Dropdown>
                                <DropdownButton plain disabled={deleteRoleMutation.isPending}>
                                  <MaterialSymbol name="more_vert" size="sm" />
                                </DropdownButton>
                                <DropdownMenu>
                                  {isDefault ? (
                                    <DropdownItem onClick={() => handleEditRole(role)}>
                                      <MaterialSymbol name="visibility" />
                                      View
                                    </DropdownItem>
                                  ) : (
                                    <>
                                      <DropdownItem onClick={() => handleEditRole(role)}>
                                        <MaterialSymbol name="edit" />
                                        Edit
                                      </DropdownItem>
                                      <DropdownItem
                                        onClick={() => handleDeleteRole(role)}
                                        className="text-red-600 dark:text-red-400"
                                      >
                                        <MaterialSymbol name="delete" />
                                        {deleteRoleMutation.isPending ? 'Deleting...' : 'Delete'}
                                      </DropdownItem>
                                    </>
                                  )}
                                </DropdownMenu>
                              </Dropdown>
                            </div>
                          </TableCell>
                        </TableRow>
                      )
                    })
                  )}
                </TableBody>
              </Table>
            )}
          </div>
        </div>
      </div>

      {/* Canvas Roles Section */}
      <div>
        <div className="mb-4">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-lg font-medium text-zinc-900 dark:text-white">
                Canvas Roles
              </h3>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                Roles specific to individual canvases
              </p>
            </div>
            {selectedCanvasId && (
              <Button
                color="blue"
                className='flex items-center'
                onClick={() => navigate(`/${organizationId}/settings/create-role?canvasId=${selectedCanvasId}`)}
              >
                <MaterialSymbol name="add" />
                New Canvas Role
              </Button>
            )}
          </div>
        </div>

        <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
          <div className="px-6 pt-6 pb-4 border-b border-zinc-200 dark:border-zinc-800">
            <div className="flex items-center gap-4">
              <div className="flex-1 max-w-xs">
                <Select
                  options={canvasOptions}
                  value={selectedCanvasId}
                  onChange={setSelectedCanvasId}
                  placeholder="Select a canvas to view roles..."
                />
              </div>
              {selectedCanvas && (
                <div className="text-sm text-zinc-600 dark:text-zinc-400">
                  Showing roles for <span className="font-medium text-zinc-900 dark:text-zinc-100">{selectedCanvas.metadata?.name}</span>
                </div>
              )}
            </div>
          </div>

          <div className="px-6 pb-6 pt-6">
            {!selectedCanvasId ? (
              <div className="flex justify-center items-center h-32">
                <div className="text-center">
                  <MaterialSymbol name="account_circle" className="text-zinc-300 dark:text-zinc-600 text-4xl mb-2" />
                  <p className="text-zinc-500 dark:text-zinc-400">Select a canvas to view its roles</p>
                </div>
              </div>
            ) : loadingCanvasRoles ? (
              <div className="flex justify-center items-center h-32">
                <p className="text-zinc-500 dark:text-zinc-400">Loading canvas roles...</p>
              </div>
            ) : (
              <Table dense>
                <TableHead>
                  <TableRow>
                    <TableHeader
                      className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                      onClick={() => handleSort('name')}
                    >
                      <div className="flex items-center gap-2">
                        Role name
                        <MaterialSymbol name={getSortIcon('name')} size="sm" className="text-zinc-400" />
                      </div>
                    </TableHeader>
                    <TableHeader
                      className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                      onClick={() => handleSort('permissions')}
                    >
                      <div className="flex items-center gap-2">
                        Permissions
                        <MaterialSymbol name={getSortIcon('permissions')} size="sm" className="text-zinc-400" />
                      </div>
                    </TableHeader>
                    <TableHeader></TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filteredCanvasRoles.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={3} className="text-center py-8 text-zinc-500 dark:text-zinc-400">
                        No canvas roles found
                      </TableCell>
                    </TableRow>
                  ) : (
                    filteredCanvasRoles.map((role, index) => {
                      const isDefault = isDefaultRole(role.metadata?.name)
                      return (
                        <TableRow key={role.metadata?.name || index}>
                          <TableCell className="font-semibold">
                            {role.spec?.displayName || role.metadata?.name}
                          </TableCell>
                          <TableCell>{role.spec?.permissions?.length || 0}</TableCell>
                          <TableCell>
                            <div className="flex justify-end">
                              <Dropdown>
                                <DropdownButton plain disabled={deleteRoleMutation.isPending}>
                                  <MaterialSymbol name="more_vert" size="sm" />
                                </DropdownButton>
                                <DropdownMenu>
                                  {isDefault ? (
                                    <DropdownItem onClick={() => navigate(`/${organizationId}/settings/create-role/${role.metadata?.name}?canvasId=${selectedCanvasId}`)}>
                                      <MaterialSymbol name="visibility" />
                                      View
                                    </DropdownItem>
                                  ) : (
                                    <>
                                      <DropdownItem onClick={() => navigate(`/${organizationId}/settings/create-role/${role.metadata?.name}?canvasId=${selectedCanvasId}`)}>
                                        <MaterialSymbol name="edit" />
                                        Edit
                                      </DropdownItem>
                                      <DropdownItem
                                        onClick={() => handleDeleteRole(role)}
                                        className="text-red-600 dark:text-red-400"
                                      >
                                        <MaterialSymbol name="delete" />
                                        {deleteRoleMutation.isPending ? 'Deleting...' : 'Delete'}
                                      </DropdownItem>
                                    </>
                                  )}
                                </DropdownMenu>
                              </Dropdown>
                            </div>
                          </TableCell>
                        </TableRow>
                      )
                    })
                  )}
                </TableBody>
              </Table>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
