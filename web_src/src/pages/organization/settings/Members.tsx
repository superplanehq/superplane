import { useState, useMemo } from 'react'
import { Heading } from '../../../components/Heading/heading'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Avatar } from '../../../components/Avatar/avatar'
import { Input, InputGroup } from '../../../components/Input/input'
import { Button } from '../../../components/Button/button'
import { Textarea } from '../../../components/Textarea/textarea'
import { Text } from '../../../components/Text/text'
import { Badge } from '../../../components/Badge/badge'
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
  useOrganizationUsers,
  useOrganizationRoles,
  useAssignRole,
  useRemoveOrganizationSubject,
  useOrganizationInvitations,
  useCreateInvitation
} from '../../../hooks/useOrganizationData'

interface Member {
  id: string
  name: string
  email: string
  role: string
  roleName: string
  initials: string
  avatar?: string
  type: 'member'
  status: 'active'
}

interface PendingInvitation {
  id: string
  name: string
  email: string
  role: string
  roleName: string
  initials: string
  type: 'invitation'
  status: 'pending'
  createdAt?: string
  state?: string
}

type UnifiedMember = Member | PendingInvitation

interface MembersProps {
  organizationId: string
}

export function Members({ organizationId }: MembersProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [sortConfig, setSortConfig] = useState<{
    key: keyof UnifiedMember | null
    direction: 'asc' | 'desc'
  }>({ key: null, direction: 'asc' })
  const [emailsInput, setEmailsInput] = useState('')
  const [invitationError, setInvitationError] = useState<string | null>(null)
  const [activeTab, setActiveTab] = useState<'all' | 'active' | 'invited'>('all')


  // Use React Query hooks for data fetching
  const { data: users = [], isLoading: loadingMembers, error: usersError } = useOrganizationUsers(organizationId)
  const { data: organizationRoles = [], isLoading: loadingRoles, error: rolesError } = useOrganizationRoles(organizationId)

  // Fetch pending invitations
  const { data: invitations = [], isLoading: loadingInvitations, error: invitationsError } = useOrganizationInvitations(organizationId)

  // Mutations for role assignment and user removal
  const assignRoleMutation = useAssignRole(organizationId)
  const removeUserMutation = useRemoveOrganizationSubject(organizationId)

  // Create invitation mutation
  const createInvitationMutation = useCreateInvitation(organizationId, {
    onError: (error: Error) => {
      setInvitationError(error.message)
    }
  })


  const error = usersError || rolesError || invitationsError
  const isInviting = createInvitationMutation.isPending

  // Transform users to Member interface format
  const members = useMemo(() => {
    return users.map((user): Member => {
      // Generate initials from displayName or userId
      const name = user.spec?.displayName || 'Unknown User'
      const initials = name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)

      // Get primary role name and display name from role assignments
      const primaryRoleName = user.status?.roleAssignments?.[0]?.roleName || 'Member'
      const primaryRoleDisplayName = user.status?.roleAssignments?.[0]?.roleDisplayName || primaryRoleName

      return {
        id: user.metadata?.id || '',
        name: name,
        email: user.metadata?.email || '',
        role: primaryRoleDisplayName,
        roleName: primaryRoleName,
        initials: initials,
        avatar: user.spec?.accountProviders?.[0]?.avatarUrl,
        type: 'member',
        status: 'active'
      }
    })
  }, [users])

  // Transform invitations to PendingInvitation interface format
  const pendingInvitations = useMemo(() => {
    return invitations.map((invitation): PendingInvitation => {
      const name = invitation.email?.split('@')[0] || 'Unknown'
      const initials = name.charAt(0).toUpperCase()

      return {
        id: invitation.id || '',
        name: name,
        email: invitation.email || '',
        role: 'Invited',
        roleName: 'pending',
        initials: initials,
        type: 'invitation',
        status: 'pending',
        createdAt: invitation.createdAt,
        state: invitation.state
      }
    })
  }, [invitations])

  // Combine members and invitations
  const unifiedMembers = useMemo(() => {
    return [...members, ...pendingInvitations]
  }, [members, pendingInvitations])

  const handleSort = (key: keyof UnifiedMember) => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc'
    }))
  }

  const getSortIcon = (columnKey: keyof UnifiedMember) => {
    if (sortConfig.key !== columnKey) {
      return 'unfold_more'
    }
    return sortConfig.direction === 'asc' ? 'keyboard_arrow_up' : 'keyboard_arrow_down'
  }

  const getSortedMembers = () => {
    if (!sortConfig.key) return unifiedMembers

    return [...unifiedMembers].sort((a, b) => {
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
    let filtered = sorted

    // Apply tab filter
    if (activeTab === 'active') {
      filtered = sorted.filter(member => member.type === 'member')
    } else if (activeTab === 'invited') {
      filtered = sorted.filter(member => member.type === 'invitation')
    }

    // Apply search filter
    if (!searchTerm) return filtered

    return filtered.filter(member =>
      member.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      member.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
      member.role.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }

  const handleRoleChange = async (memberId: string, newRoleName: string) => {
    try {
      await assignRoleMutation.mutateAsync({
        userId: memberId,
        roleName: newRoleName,
      })
    } catch (err) {
      console.error('Error updating role:', err)
    }
  }

  const handleMemberRemove = async (member: UnifiedMember) => {
    try {
      if (member.type === 'member') {
        await removeUserMutation.mutateAsync({
          subjectId: member.id,
          subjectType: 'USER_ID',
        })
      } else {
        await removeUserMutation.mutateAsync({
          subjectId: member.id,
          subjectType: 'INVITATION_ID',
        })
      }
    } catch (err) {
      console.error('Error removing member:', err)
    }
  }

  const isEmailValid = (email: string) => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    return emailRegex.test(email)
  }

  const handleEmailsSubmit = async () => {
    if (!emailsInput.trim()) return

    try {
      const emails = emailsInput.split(',').map(email => email.trim()).filter(email => email.length > 0 && isEmailValid(email))

      // Process each email
      for (const email of emails) {
        await createInvitationMutation.mutateAsync(email)
      }

      setEmailsInput('')
      setInvitationError(null)
    } catch {
      console.error('Failed to send invitations')
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleEmailsSubmit()
    }
  }

  const getStateBadge = (member: UnifiedMember) => {
    if (member.type === 'member') {
      return <Badge color="green">Active</Badge>
    } else {
      return <Badge color="yellow">Invited</Badge>
    }
  }

  const formatDate = (dateString?: string) => {
    if (!dateString) return 'N/A'

    const date = new Date(dateString)

    if (isNaN(date.getTime())) {
      console.error('Invalid date string:', dateString)
      return 'Invalid Date'
    }

    return date.toLocaleDateString(undefined, {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    })
  }

  const getTabCounts = () => {
    const activeCount = members.length
    const invitedCount = pendingInvitations.length
    const totalCount = activeCount + invitedCount

    return { activeCount, invitedCount, totalCount }
  }

  const { activeCount, invitedCount, totalCount } = getTabCounts()

  return (
    <div className="space-y-6 pt-6">
      <div className="flex items-center justify-between">
        <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
          Members
        </Heading>
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error instanceof Error ? error.message : 'Failed to fetch data'}</p>
        </div>
      )}

      {/* Send Invitations Section */}
      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <Text className="font-semibold text-zinc-900 dark:text-white mb-1">
              Invite new members
            </Text>
            <Text className="text-sm text-zinc-500 dark:text-zinc-400">
              Add people to your organization by sending them an invitation
            </Text>
          </div>
        </div>

        {invitationError && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            <p className="text-sm">{invitationError}</p>
          </div>
        )}

        {/* Email Input Section */}
        <div className="space-y-4">
          <div className="flex items-start gap-3">
            <Textarea
              rows={1}
              placeholder="Email addresses, separated by commas"
              className="flex-1"
              value={emailsInput}
              onChange={(e) => setEmailsInput(e.target.value)}
              onKeyDown={handleKeyDown}
            />
            <Button
              color="blue"
              className='flex items-center text-sm gap-2'
              onClick={handleEmailsSubmit}
              disabled={!emailsInput.trim() || isInviting}
            >
              <MaterialSymbol name="add" size="sm" />
              {isInviting ? 'Sending...' : 'Send Invitations'}
            </Button>
          </div>
        </div>
      </div>

      {/* Members List */}
      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4">
          <div className="flex items-center justify-between mb-4">
            <div className="flex items-center gap-4">
              {/* Tab Navigation */}
              <div className="flex border border-zinc-200 dark:border-zinc-700 rounded-lg p-1">
                <button
                  onClick={() => setActiveTab('all')}
                  className={`px-3 py-1 text-sm rounded-md transition-colors ${activeTab === 'all'
                    ? 'bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-white'
                    : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white'
                    }`}
                >
                  All ({totalCount})
                </button>
                <button
                  onClick={() => setActiveTab('active')}
                  className={`px-3 py-1 text-sm rounded-md transition-colors ${activeTab === 'active'
                    ? 'bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-white'
                    : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white'
                    }`}
                >
                  Active ({activeCount})
                </button>
                <button
                  onClick={() => setActiveTab('invited')}
                  className={`px-3 py-1 text-sm rounded-md transition-colors ${activeTab === 'invited'
                    ? 'bg-zinc-100 dark:bg-zinc-800 text-zinc-900 dark:text-white'
                    : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white'
                    }`}
                >
                  Invited ({invitedCount})
                </button>
              </div>
            </div>

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
          {loadingMembers || loadingInvitations ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-zinc-500 dark:text-zinc-400">Loading...</p>
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
                          src={member.type === 'member' ? member.avatar : undefined}
                          initials={member.initials}
                          className="size-8"
                        />
                        <div>
                          <div className="text-sm font-medium text-zinc-900 dark:text-white">
                            {member.name}
                          </div>
                          {member.type === 'invitation' && (
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">
                              Invited {formatDate(member.createdAt)}
                            </div>
                          )}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      {member.email}
                    </TableCell>
                    <TableCell>
                      {member.type === 'member' ? (
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
                      ) : (
                        <span className="text-sm text-zinc-500 dark:text-zinc-400">
                          -
                        </span>
                      )}
                    </TableCell>
                    <TableCell>
                      {getStateBadge(member)}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Dropdown>
                          <DropdownButton plain className="flex items-center gap-2 text-sm">
                            <MaterialSymbol name="more_vert" size="sm" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem onClick={() => handleMemberRemove(member)}>
                              <MaterialSymbol name="delete" />
                              {member.type === 'member' ? 'Remove' : 'Cancel invitation'}
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