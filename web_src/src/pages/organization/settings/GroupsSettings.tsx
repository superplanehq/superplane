import { useState, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Heading } from '../../../components/Heading/heading'
import { Button } from '../../../components/Button/button'
import { Input, InputGroup } from '../../../components/Input/input'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Avatar } from '../../../components/Avatar/avatar'
import { Link } from '../../../components/Link/link'
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from '../../../components/Dropdown/dropdown'
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell
} from '../../../components/Table/table'
import { useOrganizationGroups, useOrganizationRoles, useUpdateGroup, useDeleteGroup } from '../../../hooks/useOrganizationData'
import debounce from 'lodash.debounce'
import { formatRelativeTime } from '@/pages/canvas/utils/stageEventUtils'

interface GroupsSettingsProps {
  organizationId: string
}

export function GroupsSettings({ organizationId }: GroupsSettingsProps) {
  const navigate = useNavigate()
  const [search, setSearch] = useState('')
  const [sortConfig, setSortConfig] = useState<{
    key: string | null
    direction: 'asc' | 'desc'
  }>({
    key: null,
    direction: 'asc'
  })

  const setDebouncedSearch = debounce(setSearch, 300)

  // Use React Query hooks for data fetching
  const { data: groups = [], isLoading: loadingGroups, error: groupsError } = useOrganizationGroups(organizationId)
  const { data: roles = [], error: rolesError } = useOrganizationRoles(organizationId)

  // Mutations
  const updateGroupMutation = useUpdateGroup(organizationId)
  const deleteGroupMutation = useDeleteGroup(organizationId)

  const error = groupsError || rolesError

  const handleCreateGroup = () => {
    navigate(`/organization/${organizationId}/settings/create-group`)
  }

  const handleViewMembers = (groupName: string) => {
    navigate(`/organization/${organizationId}/settings/groups/${groupName}/members`)
  }

  const handleDeleteGroup = async (groupName: string) => {
    const confirmed = window.confirm(
      `Are you sure you want to delete the group "${groupName}"? This action cannot be undone.`
    )

    if (!confirmed) return

    try {
      await deleteGroupMutation.mutateAsync({
        groupName,
        organizationId
      })
    } catch (err) {
      console.error('Error deleting group:', err)
    }
  }

  const handleRoleUpdate = async (groupName: string, newRoleName: string) => {
    try {
      await updateGroupMutation.mutateAsync({
        groupName,
        organizationId,
        role: newRoleName
      })
    } catch (err) {
      console.error('Error updating group role:', err)
    }
  }

  const handleSort = (key: string) => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc'
    }))
  }


  const getSortIcon = (columnKey: string) => {
    if (sortConfig.key !== columnKey) {
      return 'unfold_more'
    }
    return sortConfig.direction === 'asc' ? 'keyboard_arrow_up' : 'keyboard_arrow_down'
  }

  const filteredAndSortedGroups = useMemo(() => {
    const filtered = groups.filter((group) => {
      if (search === '') {
        return true
      }
      return group.metadata?.name?.toLowerCase().includes(search.toLowerCase())
    })

    if (!sortConfig.key) return filtered

    return [...filtered].sort((a, b) => {
      let aValue: string | number
      let bValue: string | number

      switch (sortConfig.key) {
        case 'name':
          aValue = (a.metadata?.name || '').toLowerCase()
          bValue = (b.metadata?.name || '').toLowerCase()
          break
        case 'role':
          aValue = (a.spec?.role || '').toLowerCase()
          bValue = (b.spec?.role || '').toLowerCase()
          break
        case 'created':
          aValue = a.metadata?.createdAt ? new Date(a.metadata.createdAt).getTime() : 0
          bValue = b.metadata?.createdAt ? new Date(b.metadata.createdAt).getTime() : 0
          break
        case 'members':
          aValue = a.status?.membersCount || 0
          bValue = b.status?.membersCount || 0
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
  }, [groups, search, sortConfig])

  return (
    <div className="space-y-6 pt-6">
      <div className="flex items-center justify-between">
        <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
          Groups
        </Heading>
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error instanceof Error ? error.message : 'Failed to fetch data'}</p>
        </div>
      )}

      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4 flex items-center justify-between">
          <InputGroup>
            <Input name="search" placeholder="Search Groups…" aria-label="Search" className="w-xs" onChange={(e) => setDebouncedSearch(e.target.value)} />
          </InputGroup>
          <Button
            color="blue"
            className='flex items-center'
            onClick={handleCreateGroup}
          >
            <MaterialSymbol name="add" />
            Create New Group
          </Button>
        </div>
        <div className="px-6 pb-6">
          {loadingGroups ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-zinc-500 dark:text-zinc-400">Loading groups...</p>
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
                      Team name
                      <MaterialSymbol name={getSortIcon('name')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('created')}
                  >
                    <div className="flex items-center gap-2">
                      Created
                      <MaterialSymbol name={getSortIcon('created')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('members')}
                  >
                    <div className="flex items-center gap-2">
                      Members
                      <MaterialSymbol name={getSortIcon('members')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('role')}
                  >
                    <div className="flex items-center gap-2">
                      Role
                      <MaterialSymbol name={getSortIcon('role')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredAndSortedGroups.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center py-8 text-zinc-500 dark:text-zinc-400">
                      No groups found
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredAndSortedGroups.map((group, index) => (
                    <TableRow key={index}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <Avatar
                            className='w-9'
                            square
                            initials={group.spec?.displayName?.charAt(0).toUpperCase() || 'G'}
                          />
                          <div>
                            <Link
                              href={`/organization/${organizationId}/settings/groups/${group.metadata?.name}/members`}
                              className="cursor-pointer text-sm font-medium text-blue-600 dark:text-blue-400"
                            >
                              {group.spec?.displayName}
                            </Link>
                            <p className="text-xs text-zinc-500 dark:text-zinc-400">{group.spec?.description || 'No description available'}</p>
                          </div>
                        </div>
                      </TableCell>

                      <TableCell>
                        <span className="text-sm text-zinc-600 dark:text-zinc-400">
                          {formatRelativeTime(group.metadata?.createdAt)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-zinc-600 dark:text-zinc-400">
                          {group.status?.membersCount || 0} member{group.status?.membersCount === 1 ? '' : 's'}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Dropdown>
                          <DropdownButton
                            outline
                            className="flex items-center gap-2 text-sm justify-between"
                            disabled={updateGroupMutation.isPending}
                          >
                            {updateGroupMutation.isPending ? (
                              'Updating...'
                            ) : (
                              roles.find(r => r?.metadata?.name === group.spec?.role)?.spec?.displayName || 'Select Role'
                            )}
                            <MaterialSymbol name="keyboard_arrow_down" />
                          </DropdownButton>
                          <DropdownMenu>
                            {roles.map((role) => (
                              <DropdownItem
                                key={role.metadata?.name}
                                onClick={() => handleRoleUpdate(group.metadata!.name!, role.metadata!.name!)}
                              >
                                <DropdownLabel>{role.spec?.displayName || role.metadata!.name}</DropdownLabel>
                                <DropdownDescription>
                                  {role.spec?.description || 'No description available'}
                                </DropdownDescription>
                              </DropdownItem>
                            ))}
                          </DropdownMenu>
                        </Dropdown>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end">
                          <Dropdown>
                            <DropdownButton
                              plain
                              disabled={deleteGroupMutation.isPending}
                            >
                              <MaterialSymbol name="more_vert" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem onClick={() => handleViewMembers(group.metadata!.name!)}>
                                <MaterialSymbol name="group" />
                                View Members
                              </DropdownItem>
                              <DropdownItem
                                onClick={() => handleDeleteGroup(group.metadata!.name!)}
                                className="text-red-600 dark:text-red-400"
                              >
                                <MaterialSymbol name="delete" />
                                {deleteGroupMutation.isPending ? 'Deleting...' : 'Delete Group'}
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </div>
    </div>
  )
}