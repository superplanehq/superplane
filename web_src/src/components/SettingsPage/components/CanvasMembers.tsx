import { useState, useMemo } from 'react'
import { Text } from '../../Text/text'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Avatar } from '../../Avatar/avatar'
import { Input } from '../../Input/input'
import { AddMembersSection } from '../../AddMembersSection/add-members-section'
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
import {
  useCanvasRoles,
  useCanvasUsers,
  useAssignCanvasRole,
  useRemoveCanvasSubject,
  useAddCanvasUser
} from '../../../hooks/useCanvasData'
import {
  useOrganizationUsers,
  useOrganizationInvitations,
  useCreateInvitation
} from '../../../hooks/useOrganizationData'

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
  initials: string
  avatar?: string
}

interface CanvasInvitation {
  id: string
  email: string
  name: string
  initials: string
  createdAt?: string
  type: 'invitation'
}

interface User {
  id: string
  name: string
  email: string
  username?: string
  avatar?: string
  initials: string
  type: 'member' | 'invitation' | 'custom'
}

export function CanvasMembers({ canvasId, organizationId }: CanvasMembersProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [sortConfig, setSortConfig] = useState<{
    key: keyof CanvasMember | null
    direction: 'asc' | 'desc'
  }>({ key: null, direction: 'asc' })

  // Data fetching hooks
  const { data: canvasUsers = [], isLoading: loadingMembers, error: usersError } = useCanvasUsers(canvasId)
  const { data: canvasRoles = [], isLoading: loadingRoles, error: rolesError } = useCanvasRoles(canvasId)
  const { data: orgUsers = [], isLoading: loadingOrgUsers } = useOrganizationUsers(organizationId)
  const { data: orgInvitations = [], isLoading: loadingOrgInvitations } = useOrganizationInvitations(organizationId)

  // Mutations
  const assignRoleMutation = useAssignCanvasRole(canvasId)
  const removeUserMutation = useRemoveCanvasSubject(canvasId)
  const addUserMutation = useAddCanvasUser(canvasId)
  const createInvitationMutation = useCreateInvitation(organizationId)

  const error = usersError || rolesError

  // Transform canvas users to CanvasMember interface format
  const members = useMemo(() => {
    return canvasUsers.map((user) => {
      const name = user.spec?.displayName || user.metadata?.id || 'Unknown User'
      const initials = name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)

      // Get primary role name and display name from role assignments
      const primaryRoleName = user.status?.roleAssignments?.[0]?.roleName || 'Member'
      const primaryRoleDisplayName = user.status?.roleAssignments?.[0]?.roleDisplayName || primaryRoleName

      return {
        id: user.metadata?.id || '',
        name: name,
        email: user.metadata?.email || `${user.metadata?.id}@email.placeholder`,
        role: primaryRoleDisplayName,
        initials: initials,
        avatar: user.spec?.accountProviders?.[0]?.avatarUrl,
        roleName: primaryRoleName // Keep track of the actual role name for mutations
      }
    })
  }, [canvasUsers])

  // Transform pending canvas invitations - only invitations that have been assigned to this canvas
  const canvasInvitations = useMemo(() => {
    return orgInvitations
      .filter(invitation => invitation.canvasIds?.includes(canvasId))
      .map((invitation): CanvasInvitation => {
        const name = invitation.email?.split('@')[0] || 'Unknown'
        const initials = name.charAt(0).toUpperCase()

        return {
          id: invitation.id || '',
          email: invitation.email || '',
          name: name,
          initials: initials,
          createdAt: invitation.createdAt,
          type: 'invitation'
        }
      })
  }, [orgInvitations, canvasId])

  // Create user options from organization members and invitations not already in canvas
  const availableUsers = useMemo(() => {
    const canvasUserIds = new Set(canvasUsers.map(user => user.metadata?.id))

    const availableOrgUsers = orgUsers
      .filter(user => !canvasUserIds.has(user.metadata?.id))
      .map((user): User => {
        const name = user.spec?.displayName || user.metadata?.id || 'Unknown User'
        const initials = name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)

        return {
          id: user.metadata?.id || '',
          name: name,
          email: user.metadata?.email || '',
          username: user.metadata?.id || '',
          type: 'member',
          avatar: user.spec?.accountProviders?.[0]?.avatarUrl,
          initials: initials
        }
      })

    // Include org invitations that are NOT already assigned to this canvas
    const availableOrgInvitations = orgInvitations
      .filter(invitation => !invitation.canvasIds?.includes(canvasId))
      .map((invitation): User => {
        const name = invitation.email?.split('@')[0] || 'Unknown'
        const initials = name.charAt(0).toUpperCase()

        return {
          id: invitation.id || '',
          name: name,
          email: invitation.email || '',
          username: name,
          type: 'invitation',
          initials: initials
        }
      })

    return [...availableOrgUsers, ...availableOrgInvitations]
  }, [orgUsers, orgInvitations, canvasUsers, canvasId])

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

  // Handler for adding members from the AddMembersSection
  const handleAddMembers = async (users: User[]) => {
    for (const user of users) {
      try {
        if (user.type === 'member') {
          // Add existing org member to canvas
          await handleAddMember(user.id)
        } else if (user.type === 'invitation') {
          // For invitations, assign canvas viewer role to the invitation
          await handleAssignInvitationToCanvas(user.id)
        } else if (user.type === 'custom') {
          // It's a new email - invite to org and add to canvas
          await handleInviteNewUser(user.email)
        }
      } catch (err) {
        console.error(`Error adding user ${user.email}:`, err)
      }
    }
  }

  // Add existing org member to canvas
  const handleAddMember = async (userId: string) => {
    try {
      await addUserMutation.mutateAsync({ userId })
    } catch (err) {
      console.error('Error adding member to canvas:', err)
    }
  }

  // Assign pending invitation to canvas by giving it canvas viewer role
  const handleAssignInvitationToCanvas = async (invitationId: string) => {
    try {
      // Get the default viewer role for canvas
      const viewerRole = canvasRoles.find(role =>
        role.metadata?.name?.toLowerCase().includes('viewer') ||
        role.spec?.displayName?.toLowerCase().includes('viewer')
      )
      const roleName = viewerRole?.metadata?.name || 'canvas-viewer' // fallback to default role name

      // Use the updated assignRoleMutation with subjectId/subjectType support
      await assignRoleMutation.mutateAsync({
        subjectIdentifier: invitationId,
        subjectIdentifierType: 'INVITATION_ID',
        role: roleName,
      })
    } catch (err) {
      console.error('Error assigning invitation to canvas:', err)
    }
  }

  // Invite new user to organization and add to canvas
  const handleInviteNewUser = async (email: string) => {
    try {
      // Create the organization invitation
      const result = await createInvitationMutation.mutateAsync(email)
      // Assign canvas viewer role to the invitation
      await handleAssignInvitationToCanvas(result.invitation?.id || '')

    } catch (err) {
      console.error('Error creating invitation:', err)
    }
  }

  const handleRoleChange = async (memberId: string, newRoleName: string) => {
    try {
      await assignRoleMutation.mutateAsync({
        subjectIdentifier: memberId,
        subjectIdentifierType: 'USER_ID',
        role: newRoleName,
      })
    } catch (err) {
      console.error('Error updating role:', err)
    }
  }

  const handleRemoveMember = async (userId: string) => {
    try {
      await removeUserMutation.mutateAsync({
        subjectId: userId,
        subjectType: 'USER_ID',
      })
    } catch (err) {
      console.error('Error removing member:', err)
    }
  }

  const handleRemoveInvitation = async (invitationId: string) => {
    try {
      await removeUserMutation.mutateAsync({
        subjectId: invitationId,
        subjectType: 'INVITATION_ID',
      })
    } catch (err) {
      console.error('Error removing invitation:', err)
    }
  }

  return (
    <div className="h-full overflow-y-auto">

      <div className="max-w-4xl mx-auto space-y-6 py-6">
        {/* Header */}
        <div className="flex items-center justify-between text-left">
          <div>
            <Text className="text-2xl font-semibold text-zinc-900 dark:text-white">Canvas Members</Text>
            <Text className="text-sm text-zinc-600 dark:text-zinc-400">
              Manage members and invitations for this canvas
            </Text>
          </div>
        </div>

        {/* Add Members Section */}
        <AddMembersSection
          orgUsers={availableUsers}
          onAddMembers={handleAddMembers}
          isLoading={addUserMutation.isPending || createInvitationMutation.isPending}
          disabled={loadingOrgUsers || loadingOrgInvitations}
        />

        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
            <Text>{error instanceof Error ? error.message : 'Failed to fetch canvas members'}</Text>
          </div>
        )}

        {/* Pending Invitations Section - Only show if there are pending invitations */}
        {canvasInvitations.length > 0 && (
          <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800">
            <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
              <div className="flex items-center justify-between">
                <Text className="font-semibold text-zinc-900 dark:text-white">
                  Pending Invitations ({canvasInvitations.length})
                </Text>
              </div>
            </div>

            <div className="p-4">
              <Table dense>
                <TableHead>
                  <TableRow>
                    <TableHeader>Email</TableHeader>
                    <TableHeader></TableHeader>
                    <TableHeader></TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {canvasInvitations.map((invitation) => (
                    <TableRow key={invitation.id}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <Avatar
                            initials={invitation.initials}
                            className="size-8"
                          />
                          <div>
                            <div className="text-sm font-medium text-zinc-900 dark:text-white">
                              {invitation.name}
                            </div>
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">
                              {invitation.email}
                            </div>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <Text className="text-sm text-zinc-500 dark:text-zinc-400">
                          Pending invitation
                        </Text>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end">
                          <Dropdown>
                            <DropdownButton plain className="flex items-center gap-2 text-sm">
                              <MaterialSymbol name="more_vert" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem
                                onClick={() => handleRemoveInvitation(invitation.id)}
                                disabled={removeUserMutation.isPending}
                              >
                                <MaterialSymbol name="delete" />
                                Cancel Invitation
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </div>
        )}

        {/* Active Members Section - Always visible */}
        <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800">
          <div className="px-6 py-2 border-b border-zinc-200 dark:border-zinc-700">
            <div className="flex items-center justify-between">
              <Text className="font-semibold text-zinc-900 dark:text-white">
                Active Members ({members.length})
              </Text>
              <div className="max-w-sm">
                <Input
                  type="text"
                  placeholder="Search members..."
                  value={searchTerm}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearchTerm(e.target.value)}
                  className="w-full"
                />
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
                                key={role.metadata?.name}
                                onClick={() => handleRoleChange(member.id, role.metadata?.name || '')}
                                disabled={assignRoleMutation.isPending}
                              >
                                <DropdownLabel>{role.spec?.displayName || role?.metadata?.name}</DropdownLabel>
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
                        <div className="flex justify-end">
                          <Dropdown>
                            <DropdownButton plain className="flex items-center gap-2 text-sm">
                              <MaterialSymbol name="more_vert" size="sm" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem
                                onClick={() => handleRemoveMember(member.id)}
                                disabled={removeUserMutation.isPending}
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
                      <TableCell colSpan={4} className="text-center py-8">
                        <div className="text-zinc-500 dark:text-zinc-400">
                          <MaterialSymbol name="person" size="4xl" className="mx-auto text-zinc-300" />
                          <p className="text-lg font-medium text-zinc-900 dark:text-white mb-2">
                            {searchTerm ? 'No members found' : 'No members yet'}
                          </p>
                          <p className="text-sm">
                            {searchTerm ? 'Try adjusting your search criteria' : 'Add members using the search above'}
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
    </div>
  )
}