import { useState, useMemo } from 'react'
import { Heading } from '../../../components/Heading/heading'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Avatar } from '../../../components/Avatar/avatar'
import { Input, InputGroup } from '../../../components/Input/input'
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
import { AddMembersSection } from './AddMembersSection'
import { useOrganizationUsers, useOrganizationRoles, useAssignRole, useRemoveRole } from '../../../hooks/useOrganizationData'

interface Member {
  id: string
  name: string
  email: string
  role: string
  roleName: string
  status: 'Active' | 'Pending' | 'Inactive'
  lastActive: string
  initials: string
  avatar?: string
}

interface MembersSettingsProps {
  organizationId: string
}

export function MembersSettings({ organizationId }: MembersSettingsProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [sortConfig, setSortConfig] = useState<{
    key: keyof Member | null
    direction: 'asc' | 'desc'
  }>({ key: null, direction: 'asc' })

  // Use React Query hooks for data fetching
  const { data: users = [], isLoading: loadingMembers, error: usersError } = useOrganizationUsers(organizationId)
  const { data: organizationRoles = [], isLoading: loadingRoles, error: rolesError } = useOrganizationRoles(organizationId)

  // Mutations for role assignment
  const assignRoleMutation = useAssignRole(organizationId)
  const removeRoleMutation = useRemoveRole(organizationId)

  const error = usersError || rolesError

  // Transform users to Member interface format
  const members = useMemo(() => {
    return users.map((user) => {
      // Generate initials from displayName or userId
      const name = user.spec?.displayName || 'Unknown User'
      const initials = name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)

      // Get primary role name and display name from role assignments
      const primaryRoleName = user.status?.roleAssignments?.[0]?.roleName || 'Member'
      const primaryRoleDisplayName = user.status?.roleAssignments?.[0]?.roleDisplayName || primaryRoleName

      // Calculate last active time
      const lastLoginAt = user.status?.isActive ? Date.now() : null
      const lastActive = lastLoginAt ? new Date(lastLoginAt).toLocaleDateString() : 'Never'

      return {
        id: user.metadata?.id || '',
        name: name,
        email: user.metadata?.email || '',
        role: primaryRoleDisplayName,
        roleName: primaryRoleName,
        status: user.status?.isActive ? 'Active' : 'Pending',
        lastActive: lastActive,
        initials: initials,
        avatar: user.spec?.avatarUrl
      }
    })
  }, [users])

  const handleSort = (key: keyof Member) => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc'
    }))
  }

  const getSortIcon = (columnKey: keyof Member) => {
    if (sortConfig.key !== columnKey) {
      return 'unfold_more'
    }
    return sortConfig.direction === 'asc' ? 'keyboard_arrow_up' : 'keyboard_arrow_down'
  }

  const getSortedMembers = () => {
    if (!sortConfig.key) return members

    return [...members].sort((a, b) => {
      const aValue = a[sortConfig.key!]
      const bValue = b[sortConfig.key!]

      if (aValue == null && bValue == null) return 0
      if (aValue == null) return sortConfig.direction === 'asc' ? -1 : 1
      if (bValue == null) return sortConfig.direction === 'asc' ? 1 : -1

      if (aValue < bValue) {
        return sortConfig.direction === 'asc' ? -1 : 1
      }
      if (aValue > bValue) {
        return sortConfig.direction === 'asc' ? 1 : -1
      }
      return 0
    })
  }

  const getFilteredMembers = () => {
    const sorted = getSortedMembers()
    if (!searchTerm) return sorted

    return sorted.filter(member =>
      member.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      member.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
      member.role.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }

  const handleRoleChange = async (memberId: string, newRoleName: string) => {
    try {
      await assignRoleMutation.mutateAsync({
        userId: memberId,
        roleAssignment: {
          role: newRoleName,
          domainType: 'DOMAIN_TYPE_ORGANIZATION',
          domainId: organizationId
        }
      })
    } catch (err) {
      console.error('Error updating role:', err)
    }
  }

  const handleMemberRemove = async (memberId: string) => {
    try {
      // Find the member to get their current role
      const member = members.find(m => m.id === memberId)
      if (!member) {
        return
      }

      await removeRoleMutation.mutateAsync({
        userId: memberId,
        roleAssignment: {
          role: member.roleName,
          domainType: 'DOMAIN_TYPE_ORGANIZATION',
          domainId: organizationId
        }
      })
    } catch (err) {
      console.error('Error removing member:', err)
    }
  }

  const handleMemberAdded = () => {
    // No need to manually refresh - React Query will handle cache invalidation
  }

  return (
    <div className="space-y-6 pt-6">
      <div className="flex items-center justify-between">
        <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
          Members
        </Heading>
      </div>

      <AddMembersSection
        organizationId={organizationId}
        onMemberAdded={handleMemberAdded}
      />

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error instanceof Error ? error.message : 'Failed to fetch data'}</p>
        </div>
      )}

      {/* Members List */}
      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4">
          <div className="flex items-center justify-between">
            <InputGroup>
              <Input
                name="search"
                placeholder="Search members…"
                aria-label="Search"
                className="w-xs"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </InputGroup>
          </div>
        </div>

        <div className="px-6 pb-6">
          {loadingMembers ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-zinc-500 dark:text-zinc-400">Loading members...</p>
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
                    onClick={() => handleSort('role')}
                  >
                    <div className="flex items-center gap-2">
                      Role
                      <MaterialSymbol name={getSortIcon('role')} size="sm" className="text-zinc-400" />
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
                {getFilteredMembers().map((member) => (
                  <TableRow key={member.id}>
                    <TableCell>
                      <div className="flex items-center gap-3">
                        <Avatar
                          src={member.avatar}
                          initials={member.initials}
                          className="size-8"
                        />
                        <div>
                          <div className="text-sm font-medium text-zinc-900 dark:text-white">
                            {member.name}
                          </div>
                          <div className="text-xs text-zinc-500 dark:text-zinc-400">
                            Last active: {member.lastActive}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      {member.email}
                    </TableCell>
                    <TableCell>
                      <Dropdown>
                        <DropdownButton outline className="flex items-center gap-2 text-sm">
                          {member.role}
                          <MaterialSymbol name="keyboard_arrow_down" />
                        </DropdownButton>
                        <DropdownMenu>
                          {organizationRoles.map((role) => (
                            <DropdownItem
                              key={role.metadata?.name}
                              onClick={() => handleRoleChange(member.id, role.metadata?.name || '')}
                              disabled={loadingRoles}
                            >
                              <DropdownLabel>{role.spec?.displayName || role.metadata?.name}</DropdownLabel>
                              {role.spec?.description && (
                                <DropdownDescription>{role.spec?.description}</DropdownDescription>
                              )}
                            </DropdownItem>
                          ))}
                          {loadingRoles && (
                            <DropdownItem disabled>
                              <DropdownLabel>Loading roles...</DropdownLabel>
                            </DropdownItem>
                          )}
                        </DropdownMenu>
                      </Dropdown>
                    </TableCell>
                    <TableCell>
                      <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${member.status === 'Active'
                        ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                        : member.status === 'Pending'
                          ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
                          : 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400'
                        }`}>
                        {member.status}
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Dropdown>
                          <DropdownButton plain className="flex items-center gap-2 text-sm">
                            <MaterialSymbol name="more_vert" size="sm" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem onClick={() => handleMemberRemove(member.id)}>
                              <MaterialSymbol name="delete" />
                              Remove
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {getFilteredMembers().length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center py-8">
                      <div className="text-zinc-500 dark:text-zinc-400">
                        <MaterialSymbol name="search" className="h-12 w-12 mx-auto mb-4 text-zinc-300" />
                        <p className="text-lg font-medium text-zinc-900 dark:text-white mb-2">
                          {searchTerm ? 'No members found' : 'No members yet'}
                        </p>
                        <p className="text-sm">
                          {searchTerm ? 'Try adjusting your search criteria' : 'Add members to get started'}
                        </p>
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </div>
    </div>
  )
}