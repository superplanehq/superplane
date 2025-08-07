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
import { useOrganizationRoles, useDeleteRole } from '../../../hooks/useOrganizationData'
import { RolesRole } from '../../../api-client/types.gen'

interface RolesSettingsProps {
  organizationId: string
}


export function RolesSettings({ organizationId }: RolesSettingsProps) {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
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

  // Mutation for role deletion
  const deleteRoleMutation = useDeleteRole(organizationId)

  const handleCreateRole = () => {
    navigate(`/settings/create-role`)
  }

  const handleEditRole = (role: RolesRole) => {
    navigate(`/settings/create-role/${role.metadata?.name}`)
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
  }, [roles, search, sortConfig])

  return (
    <div className="space-y-6 pt-6">
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

      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4 flex items-center justify-between">
          <InputGroup>
            <Input name="search" placeholder="Search Roles…" aria-label="Search" className="w-xs" onChange={(e) => setDebouncedSearch(e.target.value)} />
          </InputGroup>
          <Button
            color="blue"
            className='flex items-center'
            onClick={handleCreateRole}
          >
            <MaterialSymbol name="add" />
            New Organization Role
          </Button>
        </div>
        <div className="px-6 pb-6">
          {loadingRoles ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-zinc-500 dark:text-zinc-400">Loading roles...</p>
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
                      No roles found
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
                            {isDefault ? (
                              <span className="text-xs text-zinc-500 dark:text-zinc-400 px-2 py-1 bg-zinc-100 dark:bg-zinc-800 rounded">
                                Default Role
                              </span>
                            ) : (
                              <Dropdown>
                                <DropdownButton plain disabled={deleteRoleMutation.isPending}>
                                  <MaterialSymbol name="more_vert" size="sm" />
                                </DropdownButton>
                                <DropdownMenu>
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
                                </DropdownMenu>
                              </Dropdown>
                            )}
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
  )
}
