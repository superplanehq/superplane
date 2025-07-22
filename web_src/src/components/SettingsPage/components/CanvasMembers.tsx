import { useState, useMemo } from 'react'
import { Text } from '../../Text/text'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Avatar } from '../../Avatar/avatar'
import { Input, InputGroup } from '../../Input/input'
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from '../../Dropdown/dropdown'
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell
} from '../../Table/table'
import { AddCanvasMembersSection } from './AddCanvasMembersSection'
import {
  useCanvasRoles,
  useCanvasUsers,
  useAssignCanvasRole,
  useRemoveCanvasRole
} from '../../../hooks/useCanvasData'

interface CanvasMembersProps {
  canvasId: string
  organizationId: string
}

interface CanvasMember {
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

export function CanvasMembers({ canvasId, organizationId }: CanvasMembersProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [sortConfig, setSortConfig] = useState<{
    key: keyof CanvasMember | null
    direction: 'asc' | 'desc'
  }>({ key: null, direction: 'asc' })

  // Use new hooks for data fetching
  const { data: canvasUsers = [], isLoading: loadingMembers, error: usersError } = useCanvasUsers(canvasId)
  const { data: canvasRoles = [], isLoading: loadingRoles, error: rolesError } = useCanvasRoles(canvasId)

  // Mutations
  const assignRoleMutation = useAssignCanvasRole(canvasId)
  const removeRoleMutation = useRemoveCanvasRole(canvasId)

  const error = usersError || rolesError

  // Transform users to CanvasMember interface format
  const members = useMemo(() => {
    return canvasUsers.map((user) => {
      const name = user.displayName || user.userId || 'Unknown User'
      const initials = name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)

      // Get primary role name and display name from role assignments
      const primaryRoleName = user.roleAssignments?.[0]?.roleName || 'Member'
      const primaryRoleDisplayName = user.roleAssignments?.[0]?.roleDisplayName || primaryRoleName

      return {
        id: user.userId || '',
        name: name,
        email: user.email || `${user.userId}@email.placeholder`,
        role: primaryRoleDisplayName,
        status: user.isActive ? 'Active' as const : 'Pending' as const,
        lastActive: user.isActive ? 'Recently' : 'Never',
        initials: initials,
        avatar: user.avatarUrl,
        roleName: primaryRoleName // Keep track of the actual role name for mutations
      }
    })
  }, [canvasUsers])

  const handleSort = (key: keyof CanvasMember) => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc'
    }))
  }

  const getSortIcon = (columnKey: keyof CanvasMember) => {
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

  const handleMemberAdded = () => {
    // No need to manually refresh - React Query will handle cache invalidation
  }

  const handleRoleChange = async (memberId: string, newRoleName: string) => {
    try {
      await assignRoleMutation.mutateAsync({
        userId: memberId,
        roleAssignment: {
          role: newRoleName,
          domainType: 'DOMAIN_TYPE_CANVAS',
          domainId: canvasId
        }
      })
    } catch (err) {
      console.error('Error updating role:', err)
    }
  }

  const handleRemoveMember = async (userId: string, roleName: string) => {
    try {
      await removeRoleMutation.mutateAsync({
        userId,
        roleAssignment: {
          role: roleName,
          domainType: 'DOMAIN_TYPE_CANVAS',
          domainId: canvasId
        }
      })
    } catch (err) {
      console.error('Error removing member:', err)
    }
  }

  return (
    <div className="space-y-6">
      <AddCanvasMembersSection
        canvasId={canvasId}
        organizationId={organizationId}
        onMemberAdded={handleMemberAdded}
      />

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <Text>{error instanceof Error ? error.message : 'Failed to fetch canvas members'}</Text>
        </div>
      )}

      {/* Members list section */}
      <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
        <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <div className="max-w-sm">
                <div className="relative">
                  <InputGroup>
                    <MaterialSymbol
                      name="search"
                      className="absolute left-3 top-1/2 transform -translate-y-1/2 text-zinc-400"
                      size="sm"
                    />
                    <Input
                      type="text"
                      placeholder="Search members..."
                      value={searchTerm}
                      onChange={(e) => setSearchTerm(e.target.value)}
                      className="pl-10 w-full"
                    />
                  </InputGroup>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div className="p-6">
          {loadingMembers ? (
            <div className="flex justify-center items-center h-32">
              <Text className="text-zinc-500">Loading members...</Text>
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
                          className="size-8 bg-blue-700 dark:bg-blue-900 text-blue-100 dark:text-blue-100"
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
                          {canvasRoles.map((role) => (
                            <DropdownItem
                              key={role.name}
                              onClick={() => handleRoleChange(member.id, role.name || '')}
                              disabled={assignRoleMutation.isPending}
                            >
                              <DropdownLabel>{role.displayName || role.name}</DropdownLabel>
                              {role.description && (
                                <DropdownDescription>{role.description}</DropdownDescription>
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
                            <DropdownItem
                              onClick={() => handleRemoveMember(member.id, member.roleName)}
                              disabled={removeRoleMutation.isPending}
                            >
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
                        <MaterialSymbol name="person" className="h-12 w-12 mx-auto mb-4 text-zinc-300" />
                        <p className="text-lg font-medium text-zinc-900 dark:text-white mb-2">
                          {searchTerm ? 'No members found' : 'No members yet'}
                        </p>
                        <p className="text-sm">
                          {searchTerm ? 'Try adjusting your search criteria' : 'Invite members to get started'}
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