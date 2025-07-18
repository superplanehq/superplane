import { useState, useEffect, useMemo } from 'react'
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
import {
  authorizationListOrganizationGroups,
  authorizationListRoles,
  authorizationUpdateOrganizationGroup,
  authorizationDeleteOrganizationGroup
} from '../../../api-client/sdk.gen'
import { AuthorizationGroup, AuthorizationRole } from '../../../api-client/types.gen'
import { capitalizeFirstLetter } from '@/utils/text'
import debounce from 'lodash.debounce'

// Utility function to format relative time
const formatRelativeTime = (dateString: string | undefined) => {
  if (!dateString) return 'Unknown'

  const now = new Date()
  const date = new Date(dateString)
  const diffInMs = now.getTime() - date.getTime()
  const diffInMinutes = Math.floor(diffInMs / (1000 * 60))
  const diffInHours = Math.floor(diffInMinutes / 60)
  const diffInDays = Math.floor(diffInHours / 24)
  const diffInWeeks = Math.floor(diffInDays / 7)
  const diffInMonths = Math.floor(diffInDays / 30)
  const diffInYears = Math.floor(diffInDays / 365)

  if (diffInMinutes < 1) return 'Just now'
  if (diffInMinutes < 60) return `${diffInMinutes}m ago`
  if (diffInHours < 24) return `${diffInHours}h ago`
  if (diffInDays < 7) return `${diffInDays}d ago`
  if (diffInWeeks < 4) return `${diffInWeeks}w ago`
  if (diffInMonths < 12) return `${diffInMonths}mo ago`
  return `${diffInYears}y ago`
}

interface GroupsSettingsProps {
  organizationId: string
}

export function GroupsSettings({ organizationId }: GroupsSettingsProps) {
  const navigate = useNavigate()
  const [groups, setGroups] = useState<AuthorizationGroup[]>([])
  const [roles, setRoles] = useState<AuthorizationRole[]>([])
  const [loadingGroups, setLoadingGroups] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [updatingGroupRoles, setUpdatingGroupRoles] = useState<Set<string>>(new Set())
  const [deletingGroups, setDeletingGroups] = useState<Set<string>>(new Set())
  const [sortConfig, setSortConfig] = useState<{
    key: string | null
    direction: 'asc' | 'desc'
  }>({
    key: null,
    direction: 'asc'
  })

  const setDebouncedSearch = debounce(setSearch, 300)

  useEffect(() => {
    const fetchGroups = async () => {
      try {
        setLoadingGroups(true)
        setError(null)
        const response = await authorizationListOrganizationGroups({
          query: { organizationId }
        })
        if (response.data?.groups) {
          setGroups(response.data.groups)
        }
      } catch (err) {
        console.error('Error fetching groups:', err)
        setError('Failed to fetch groups')
      } finally {
        setLoadingGroups(false)
      }
    }

    const fetchRoles = async () => {
      try {
        setLoadingGroups(true)
        setError(null)
        const response = await authorizationListRoles({
          query: { domainType: 'DOMAIN_TYPE_ORGANIZATION', domainId: organizationId }
        })
        if (response.data?.roles) {
          setRoles(response.data.roles)
        }
      } catch (err) {
        console.error('Error fetching roles:', err)
        setError('Failed to fetch roles')
      } finally {
        setLoadingGroups(false)
      }
    }

    fetchGroups()
    fetchRoles()
  }, [organizationId])

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
      setError(null)
      setDeletingGroups(prev => new Set(prev).add(groupName))

      await authorizationDeleteOrganizationGroup({
        path: { groupName },
        query: { organizationId }
      })

      // Remove the group from local state
      setGroups(prev => prev.filter(group => group.name !== groupName))
    } catch (err) {
      console.error('Error deleting group:', err)
      setError(`Failed to delete group "${groupName}": ${err instanceof Error ? err.message : 'Unknown error'}`)
    } finally {
      setDeletingGroups(prev => {
        const newSet = new Set(prev)
        newSet.delete(groupName)
        return newSet
      })
    }
  }

  const handleRoleUpdate = async (groupName: string, newRoleName: string) => {
    try {
      setError(null)
      setUpdatingGroupRoles(prev => new Set(prev).add(groupName))

      await authorizationUpdateOrganizationGroup({
        path: { groupName },
        body: {
          organizationId,
          role: newRoleName
        }
      })

      // Update the group's role in the local state
      setGroups(prev => prev.map(group =>
        group.name === groupName ? { ...group, role: newRoleName } : group
      ))
    } catch (err) {
      console.error('Error updating group role:', err)
      setError(`Failed to update role for group "${groupName}": ${err instanceof Error ? err.message : 'Unknown error'}`)
    } finally {
      setUpdatingGroupRoles(prev => {
        const newSet = new Set(prev)
        newSet.delete(groupName)
        return newSet
      })
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
      return group.name?.toLowerCase().includes(search.toLowerCase())
    })

    if (!sortConfig.key) return filtered

    return [...filtered].sort((a, b) => {
      let aValue: string | number
      let bValue: string | number

      switch (sortConfig.key) {
        case 'name':
          aValue = (a.name || '').toLowerCase()
          bValue = (b.name || '').toLowerCase()
          break
        case 'role':
          aValue = (a.role || '').toLowerCase()
          bValue = (b.role || '').toLowerCase()
          break
        case 'created':
          aValue = a.createdAt ? new Date(a.createdAt).getTime() : 0
          bValue = b.createdAt ? new Date(b.createdAt).getTime() : 0
          break
        case 'members':
          aValue = a.membersCount || 0
          bValue = b.membersCount || 0
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
          <p>{error}</p>
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
                            initials={group.displayName?.charAt(0).toUpperCase() || 'G'}
                          />
                          <div>
                            <Link
                              href={`/organization/${organizationId}/settings/groups/${group.name}/members`}
                              className="cursor-pointer text-sm font-medium text-blue-600 dark:text-blue-400"
                            >
                              {group.displayName}
                            </Link>
                            <p className="text-xs text-zinc-500 dark:text-zinc-400">{group.description || 'No description available'}</p>
                          </div>
                        </div>
                      </TableCell>

                      <TableCell>
                        <span className="text-sm text-zinc-600 dark:text-zinc-400">
                          {formatRelativeTime(group.createdAt)}
                        </span>
                      </TableCell>
                      <TableCell>
                        <span className="text-sm text-zinc-600 dark:text-zinc-400">
                          {group.membersCount || 0} member{group.membersCount === 1 ? '' : 's'}
                        </span>
                      </TableCell>
                      <TableCell>
                        <Dropdown>
                          <DropdownButton
                            outline
                            className="flex items-center gap-2 text-sm justify-between"
                            disabled={updatingGroupRoles.has(group.name!)}
                          >
                            {updatingGroupRoles.has(group.name!) ? (
                              'Updating...'
                            ) : (
                              capitalizeFirstLetter(group.role?.split('_').at(-1) || '') || 'Select Role'
                            )}
                            <MaterialSymbol name="keyboard_arrow_down" />
                          </DropdownButton>
                          <DropdownMenu>
                            {roles.map((role) => (
                              <DropdownItem
                                key={role.name}
                                onClick={() => handleRoleUpdate(group.name!, role.name!)}
                              >
                                <DropdownLabel>{capitalizeFirstLetter(role.name?.split('_').at(-1) || '')}</DropdownLabel>
                                <DropdownDescription>
                                  {role.description || 'No description available'}
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
                              disabled={deletingGroups.has(group.name!)}
                            >
                              <MaterialSymbol name="more_vert" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem onClick={() => handleViewMembers(group.name!)}>
                                <MaterialSymbol name="group" />
                                View Members
                              </DropdownItem>
                              <DropdownItem
                                onClick={() => handleDeleteGroup(group.name!)}
                                className="text-red-600 dark:text-red-400"
                              >
                                <MaterialSymbol name="delete" />
                                {deletingGroups.has(group.name!) ? 'Deleting...' : 'Delete Group'}
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