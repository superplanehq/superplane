import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Avatar } from '../../../components/Avatar/avatar'
import { Heading, Subheading } from '../../../components/Heading/heading'
import { Button } from '../../../components/Button/button'
import { Input, InputGroup } from '../../../components/Input/input'
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from '../../../components/Dropdown/dropdown'
import { AddMembersSection } from './AddMembersSection'
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell
} from '../../../components/Table/table'
import {
  authorizationGetOrganizationGroup,
  authorizationGetOrganizationGroupUsers,
  authorizationRemoveUserFromOrganizationGroup,
  authorizationListRoles,
  authorizationUpdateOrganizationGroup
} from '../../../api-client/sdk.gen'
import { AuthorizationGroup, AuthorizationUser, AuthorizationRole } from '../../../api-client/types.gen'

export function GroupMembersPage() {
  const { orgId, groupName: encodedGroupName } = useParams<{ orgId: string; groupName: string }>()
  const groupName = encodedGroupName ? decodeURIComponent(encodedGroupName) : undefined
  const [group, setGroup] = useState<AuthorizationGroup | null>(null)
  const [members, setMembers] = useState<AuthorizationUser[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [roles, setRoles] = useState<AuthorizationRole[]>([])
  const [loadingRoles, setLoadingRoles] = useState(true)
  const [searchQuery, setSearchQuery] = useState('')
  const [isEditingGroupName, setIsEditingGroupName] = useState(false)
  const [isEditingGroupDescription, setIsEditingGroupDescription] = useState(false)
  const [editedGroupName, setEditedGroupName] = useState('')
  const [editedGroupDescription, setEditedGroupDescription] = useState('')
  const [sortConfig, setSortConfig] = useState<{
    key: string | null
    direction: 'asc' | 'desc'
  }>({
    key: null,
    direction: 'asc'
  })

  useEffect(() => {
    const fetchGroupData = async () => {
      if (!orgId || !groupName) return

      try {
        setLoading(true)
        setError(null)

        console.log('Fetching group data for:', { orgId, groupName })

        // Fetch group details, members, and roles in parallel
        const [groupResponse, membersResponse, rolesResponse] = await Promise.all([
          authorizationGetOrganizationGroup({
            path: { groupName },
            query: { organizationId: orgId }
          }),
          authorizationGetOrganizationGroupUsers({
            path: { groupName },
            query: { organizationId: orgId }
          }),
          authorizationListRoles({
            query: { domainType: 'DOMAIN_TYPE_ORGANIZATION', domainId: orgId }
          })
        ])

        if (groupResponse.data?.group) {
          setGroup(groupResponse.data.group)
        } else {
          console.warn('No group data in response')
          // Set a fallback group with the name from URL params
          setGroup({
            name: groupName,
            description: 'Group details could not be loaded',
            role: 'member'
          })
        }

        if (membersResponse.data?.users) {
          setMembers(membersResponse.data.users)
        } else {
          setMembers([])
        }

        if (rolesResponse.data?.roles) {
          setRoles(rolesResponse.data.roles)
        } else {
          setRoles([])
        }
      } catch (err) {
        console.error('Error fetching group data:', err)
        setError(`Failed to load group data: ${err instanceof Error ? err.message : 'Unknown error'}`)
      } finally {
        setLoading(false)
        setLoadingRoles(false)
      }
    }

    fetchGroupData()
  }, [orgId, groupName])

  const handleBackToGroups = () => {
    window.history.back()
  }

  const handleEditGroupName = () => {
    if (group) {
      setEditedGroupName(group.name || '')
      setIsEditingGroupName(true)
    }
  }

  const handleSaveGroupName = () => {
    if (group && editedGroupName.trim()) {
      // Update the group name - in a real app this would call an API
      setGroup({ ...group, name: editedGroupName.trim() })
      setIsEditingGroupName(false)
      console.log('Saving group name:', editedGroupName)
    }
  }

  const handleCancelGroupName = () => {
    setIsEditingGroupName(false)
    setEditedGroupName('')
  }

  const handleEditGroupDescription = () => {
    if (group) {
      setEditedGroupDescription(group.description || '')
      setIsEditingGroupDescription(true)
    }
  }

  const handleSaveGroupDescription = () => {
    if (group && editedGroupDescription.trim()) {
      // Update the group description - in a real app this would call an API
      setGroup({ ...group, description: editedGroupDescription.trim() })
      setIsEditingGroupDescription(false)
      console.log('Saving group description:', editedGroupDescription)
    }
  }

  const handleCancelGroupDescription = () => {
    setIsEditingGroupDescription(false)
    setEditedGroupDescription('')
  }

  const handleSort = (key: string) => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc'
    }))
  }

  const getSortedData = (data: AuthorizationUser[]) => {
    if (!sortConfig.key) return data

    return [...data].sort((a, b) => {
      const aValue = a[sortConfig.key as keyof AuthorizationUser]
      const bValue = b[sortConfig.key as keyof AuthorizationUser]

      if (aValue && bValue && aValue < bValue) {
        return sortConfig.direction === 'asc' ? -1 : 1
      }
      if (aValue && bValue && aValue > bValue) {
        return sortConfig.direction === 'asc' ? 1 : -1
      }
      return 0
    })
  }

  const getSortIcon = (columnKey: string) => {
    if (sortConfig.key !== columnKey) {
      return 'unfold_more'
    }
    return sortConfig.direction === 'asc' ? 'keyboard_arrow_up' : 'keyboard_arrow_down'
  }

  const handleRemoveMember = async (userId: string) => {
    if (!groupName || !orgId) return

    try {
      await authorizationRemoveUserFromOrganizationGroup({
        path: { groupName },
        body: { userId }
      })

      // Remove member from local state
      setMembers(prev => prev.filter(member => member.userId !== userId))
    } catch (err) {
      console.error('Error removing member:', err)
    }
  }

  const handleMemberAdded = () => {
    // Refresh the members list after adding a new member
    window.location.reload()
  }

  const handleRoleUpdate = async (newRoleName: string) => {
    if (!orgId || !group || !groupName) return

    try {
      setError(null)
      await authorizationUpdateOrganizationGroup({
        path: { groupName },
        body: {
          organizationId: orgId,
          role: newRoleName
        }
      })

      // Update the group's role in the local state
      setGroup(prev => prev ? { ...prev, role: newRoleName } : null)
    } catch (err) {
      console.error('Error updating group role:', err)
      setError(`Failed to update group role: ${err instanceof Error ? err.message : 'Unknown error'}`)
    }
  }

  const filteredMembers = members.filter(member =>
    member.displayName?.toLowerCase().includes(searchQuery.toLowerCase()) ||
    member.email?.toLowerCase().includes(searchQuery.toLowerCase())
  )

  if (loading) {
    return (
      <div className="flex justify-center items-center h-screen">
        <p className="text-zinc-500 dark:text-zinc-400">Loading group...</p>
      </div>
    )
  }

  if (error && !group) {
    return (
      <div className="space-y-6 pt-6">
        <div className="flex items-center gap-2 mb-4">
          <button
            onClick={handleBackToGroups}
            className="flex items-center gap-2 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white transition-colors"
          >
            <MaterialSymbol name="arrow_back" size="sm" />
            <span className="text-sm">Back to Groups</span>
          </button>
        </div>
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6 pt-6">
      {/* Back navigation */}
      <div className="flex items-center gap-2 mb-4">
        <button
          onClick={handleBackToGroups}
          className="flex items-center gap-2 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white transition-colors"
        >
          <MaterialSymbol name="arrow_back" size="sm" />
          <span className="text-sm">Back to Groups</span>
        </button>
      </div>

      <div className="bg-zinc-100 dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 space-y-6">
        {/* Group header */}
        <div className='flex items-start justify-between'>
          <div className='flex items-start gap-3'>
            <Avatar
              className='w-12 bg-blue-200 dark:bg-blue-800 border border-blue-300 dark:border-blue-700'
              square
              initials={group?.name?.charAt(0) || 'G'}
            />
            <div className='flex flex-col space-y-2'>
              {/* Group Name - Inline Edit */}
              <div className="group">
                {isEditingGroupName ? (
                  <div className="flex items-center gap-2">
                    <Input
                      type="text"
                      value={editedGroupName}
                      onChange={(e) => setEditedGroupName(e.target.value)}
                      className="text-2xl font-semibold bg-white dark:bg-zinc-800 border-zinc-300 dark:border-zinc-600"
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleSaveGroupName()
                        if (e.key === 'Escape') handleCancelGroupName()
                      }}
                      autoFocus
                    />
                    <Button plain onClick={handleSaveGroupName} className="text-green-600 hover:text-green-700">
                      <MaterialSymbol name="check" size="sm" />
                    </Button>
                    <Button plain onClick={handleCancelGroupName} className="text-red-600 hover:text-red-700">
                      <MaterialSymbol name="close" size="sm" />
                    </Button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                      {group?.name}
                    </Heading>
                    <Button
                      plain
                      onClick={handleEditGroupName}
                      className="opacity-0 group-hover:opacity-100 transition-opacity text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                    >
                      <MaterialSymbol name="edit" size="sm" />
                    </Button>
                  </div>
                )}
              </div>

              {/* Group Description - Inline Edit */}
              <div className="group">
                {isEditingGroupDescription ? (
                  <div className="flex items-center gap-2">
                    <Input
                      type="text"
                      value={editedGroupDescription}
                      onChange={(e) => setEditedGroupDescription(e.target.value)}
                      className="text-lg bg-white dark:bg-zinc-800 border-zinc-300 dark:border-zinc-600"
                      onKeyDown={(e) => {
                        if (e.key === 'Enter') handleSaveGroupDescription()
                        if (e.key === 'Escape') handleCancelGroupDescription()
                      }}
                      autoFocus
                    />
                    <Button plain onClick={handleSaveGroupDescription} className="text-green-600 hover:text-green-700">
                      <MaterialSymbol name="check" size="sm" />
                    </Button>
                    <Button plain onClick={handleCancelGroupDescription} className="text-red-600 hover:text-red-700">
                      <MaterialSymbol name="close" size="sm" />
                    </Button>
                  </div>
                ) : (
                  <div className="flex items-center gap-2">
                    <Subheading level={3} className="text-lg !font-normal text-zinc-600 dark:text-zinc-400">
                      {group?.description || 'No description'}
                    </Subheading>
                    <Button
                      plain
                      onClick={handleEditGroupDescription}
                      className="opacity-0 group-hover:opacity-100 transition-opacity text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                    >
                      <MaterialSymbol name="edit" size="sm" />
                    </Button>
                  </div>
                )}
              </div>
            </div>
          </div>
          <div className='flex items-center gap-2'>
            <Dropdown>
              <DropdownButton color='white'
                className="flex items-center gap-2 text-sm"
                disabled={loadingRoles}
              >
                {loadingRoles ? 'Loading...' : roles.find(role => role.name === group?.role)?.displayName || 'Member'}
                <MaterialSymbol name="keyboard_arrow_down" />
              </DropdownButton>
              <DropdownMenu>
                {roles.map((role) => (
                  <DropdownItem
                    key={role.name}
                    onClick={() => handleRoleUpdate(role.name!)}
                  >
                    <DropdownLabel>{role.displayName}</DropdownLabel>
                    <DropdownDescription>{role.description}</DropdownDescription>
                  </DropdownItem>
                ))}
              </DropdownMenu>
            </Dropdown>
            <Dropdown>
              <DropdownButton plain aria-label="More options">
                <MaterialSymbol name="more_vert" size="sm" />
              </DropdownButton>
              <DropdownMenu>
                <DropdownItem>Delete group</DropdownItem>
              </DropdownMenu>
            </Dropdown>
          </div>
        </div>

        {/* Add Members Section */}
        <AddMembersSection
          organizationId={orgId!}
          groupName={groupName!}
          showRoleSelection={false}
          onMemberAdded={handleMemberAdded}
        />

        {/* Group members table */}
        <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
          <div className="px-6 pt-6 pb-4">
            <div className="flex items-center justify-between">
              <InputGroup>
                <Input
                  name="search"
                  placeholder="Search team members…"
                  aria-label="Search"
                  className="w-xs"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                />
              </InputGroup>
            </div>
          </div>
          <div className="px-6 pb-6">
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('name')}
                  >
                    <div className="flex items-center gap-2">
                      Name
                      <MaterialSymbol name={getSortIcon('name')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('email')}
                  >
                    <div className="flex items-center gap-2">
                      Email
                      <MaterialSymbol name={getSortIcon('email')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('status')}
                  >
                    <div className="flex items-center gap-2">
                      Status
                      <MaterialSymbol name={getSortIcon('status')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {getSortedData(filteredMembers).map((member) => (
                  <TableRow key={member.userId}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <Avatar
                          src={member.avatarUrl}
                          initials={member.displayName?.charAt(0) || 'U'}
                          className="size-8"
                        />
                        <div>
                          <div className="text-sm font-medium text-zinc-900 dark:text-white">
                            {member.displayName}
                          </div>
                          <div className="text-xs text-zinc-500 dark:text-zinc-400">
                            Member since {new Date().toLocaleDateString()}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      {member.email}
                    </TableCell>
                    <TableCell>
                      {
                        member.isActive ?
                          <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400">
                            Active
                          </span>
                          :
                          <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400">
                            Pending
                          </span>
                      }
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Dropdown>
                          <DropdownButton plain className="flex items-center gap-2 text-sm">
                            <MaterialSymbol name="more_vert" size="sm" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem>
                              <MaterialSymbol name="edit" />
                              Edit Member
                            </DropdownItem>
                            <DropdownItem>
                              <MaterialSymbol name="security" />
                              Change Role
                            </DropdownItem>
                            <DropdownItem onClick={() => handleRemoveMember(member.userId!)}>
                              <MaterialSymbol name="person_remove" />
                              Remove from Group
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {filteredMembers.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center h-[200px] py-6">
                      {searchQuery ? `No members found matching "${searchQuery}"` : 'No group members yet'}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </div>
    </div>
  )
}