import { useState, useMemo } from 'react'
import { Heading } from '../../Heading/heading'
import { Text } from '../../Text/text'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Avatar } from '../../Avatar/avatar'
import { Input, InputGroup } from '../../Input/input'
import { Button } from '../../Button/button'
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem,
} from '../../Dropdown/dropdown'
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell
} from '../../Table/table'
import { Textarea } from '../../Textarea/textarea'
import {
  authorizationGetCanvasUsers,
  authorizationAddUserToCanvasGroup,
  authorizationRemoveUserFromCanvasGroup
} from '../../../api-client/sdk.gen'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'

interface CanvasMembersProps {
  canvasId: string
  organizationId: string
}

interface CanvasMember {
  id: string
  name: string
  email: string
  role: string
  status: 'Active' | 'Pending' | 'Inactive'
  lastActive: string
  initials: string
  avatar?: string
}

export function CanvasMembers({ canvasId }: CanvasMembersProps) {
  const [searchTerm, setSearchTerm] = useState('')
  const [inviteEmails, setInviteEmails] = useState('')
  const [sortConfig, setSortConfig] = useState<{
    key: keyof CanvasMember | null
    direction: 'asc' | 'desc'
  }>({ key: null, direction: 'asc' })

  const queryClient = useQueryClient()

  // Fetch canvas users
  const { data: canvasUsers = [], isLoading: loadingMembers, error } = useQuery({
    queryKey: ['canvasUsers', canvasId],
    queryFn: async () => {
      const response = await authorizationGetCanvasUsers({
        path: { canvasIdOrName: canvasId },
      })
      return response.data?.users || []
    },
    staleTime: 5 * 60 * 1000,
    gcTime: 10 * 60 * 1000,
    enabled: !!canvasId
  })

  // Add user mutation
  const addUserMutation = useMutation({
    mutationFn: async ({ userEmail }: { userEmail: string }) => {
      return await authorizationAddUserToCanvasGroup({
        path: { canvasIdOrName: canvasId, groupName: '' },
        body: {
          userEmail,
        }
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['canvasUsers', canvasId] })
    }
  })

  // Remove user mutation
  const removeUserMutation = useMutation({
    mutationFn: async ({ userId }: { userId: string }) => {
      return await authorizationRemoveUserFromCanvasGroup({
        path: { canvasIdOrName: canvasId, groupName: '' },
        body: {
          userId,
        }
      })
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['canvasUsers', canvasId] })
    }
  })

  // Transform users to CanvasMember interface format
  const members = useMemo(() => {
    return canvasUsers.map((user) => {
      const name = user.displayName || user.userId || 'Unknown User'
      const initials = name.split(' ').map(n => n[0]).join('').toUpperCase().slice(0, 2)

      return {
        id: user.userId || '',
        name: name,
        email: user.email || `${user.userId}@email.placeholder`,
        role: user.roleAssignments?.[0]?.roleName || 'Member',
        status: user.isActive ? 'Active' as const : 'Pending' as const,
        lastActive: user.isActive ? 'Recently' : 'Never',
        initials: initials,
        avatar: user.avatarUrl
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

  const isEmailValid = (email: string) => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    return emailRegex.test(email)
  }

  const handleInviteMembers = async () => {
    if (!inviteEmails.trim()) return

    try {
      const emails = inviteEmails.split(',').map(email => email.trim()).filter(email => email.length > 0 && isEmailValid(email))

      for (const email of emails) {
        await addUserMutation.mutateAsync({ userEmail: email })
      }

      setInviteEmails('')
    } catch (err) {
      console.error('Failed to invite members:', err)
    }
  }

  const handleRemoveMember = async (userId: string) => {
    try {
      await removeUserMutation.mutateAsync({ userId })
    } catch (err) {
      console.error('Error removing member:', err)
    }
  }

  return (
    <div className="space-y-6">
      <Heading level={2}>Members</Heading>

      {/* Invite new members section */}
      <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
        <Heading level={3} className="text-base mb-4">Invite new members</Heading>
        <div className="flex items-start gap-4">
          <div className="flex-1">
            <Textarea
              rows={2}
              placeholder="Email addresses, separated by commas"
              value={inviteEmails}
              onChange={(e) => setInviteEmails(e.target.value)}
              className="w-full"
            />
          </div>
          <Button
            color='blue'
            onClick={handleInviteMembers}
            disabled={!inviteEmails.trim() || addUserMutation.isPending}
            className="flex items-center gap-2"
          >
            <MaterialSymbol name="add" size="sm" />
            {addUserMutation.isPending ? 'Inviting...' : 'Invite'}
          </Button>
        </div>
      </div>

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
                      className="pl-10"
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
                      <span className="text-sm text-zinc-600 dark:text-zinc-400">
                        {member.role}
                      </span>
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