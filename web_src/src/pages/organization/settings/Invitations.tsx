import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { Heading } from '../../../components/Heading/heading'
import { Button } from '../../../components/Button/button'
import { Textarea } from '../../../components/Textarea/textarea'
import { Input, InputGroup } from '../../../components/Input/input'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Text } from '../../../components/Text/text'
import { Badge } from '../../../components/Badge/badge'
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell
} from '../../../components/Table/table'
import { organizationsListInvitations, organizationsCreateInvitation } from '../../../api-client/sdk.gen'
import type { OrganizationsInvitation } from '../../../api-client/types.gen'
import { withOrganizationHeader } from '../../../utils/withOrganizationHeader'
import { organizationKeys } from '../../../hooks/useOrganizationData'


interface InvitationsProps {
  organizationId: string
}

export function Invitations({ organizationId }: InvitationsProps) {
  const [emailsInput, setEmailsInput] = useState('')
  const [searchTerm, setSearchTerm] = useState('')
  const [invitationError, setInvitationError] = useState<string | null>(null)
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  // Fetch pending invitations
  const { data: invitations = [], isLoading: loadingInvitations } = useQuery<OrganizationsInvitation[]>({
    queryKey: ['invitations', organizationId],
    queryFn: async () => {
      const response = await organizationsListInvitations(withOrganizationHeader({
        path: { id: organizationId }
      }))
      return response.data.invitations || []
    },
  })

  // Create invitation mutation
  const createInvitationMutation = useMutation({
    mutationFn: async (email: string) => {
      const response = await organizationsCreateInvitation(withOrganizationHeader({
        path: { id: organizationId },
        body: {
          email: email,
        }
      }))
      return response.data
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['invitations', organizationId] })
      setInvitationError(null)
      
      // If the invitation was already accepted, which means that the email being invited already has an account,
      // and was already added to the organization, we redirect to members page.
      if (data.invitation?.status === 'accepted') {
        // Invalidate members list to ensure it gets reloaded when navigating to members page
        queryClient.invalidateQueries({ queryKey: organizationKeys.users(organizationId) })
        navigate(`/${organizationId}/settings/members`)
      }
    },
    onError: (error: Error) => {
      setInvitationError(error.message)
    },
  })

  const isInviting = createInvitationMutation.isPending

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

  const getStatusBadge = (status: string) => {
    switch (status) {
      case 'pending':
        return <Badge color="yellow">Pending</Badge>
      case 'accepted':
        return <Badge color="green">Accepted</Badge>
      default:
        return <Badge color="gray">{status}</Badge>
    }
  }

  const formatDate = (dateString: string) => {
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
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  const getFilteredInvitations = () => {
    if (!searchTerm) return invitations
    
    return invitations.filter(invitation =>
      invitation.email?.toLowerCase().includes(searchTerm.toLowerCase()) ||
      invitation.status?.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }

  return (
    <div className="space-y-6 pt-6">
      <div className="flex items-center justify-between">
        <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
          Invitations
        </Heading>
      </div>

      {/* Send Invitations Section */}
      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <Text className="font-semibold text-zinc-900 dark:text-white mb-1">
              Send invitations
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

      {/* Invitations List */}
      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4">
          <div className="flex items-center justify-between">
            <Text className="font-semibold text-zinc-900 dark:text-white">
              All Invitations
            </Text>
            <InputGroup>
              <Input
                name="search"
                placeholder="Search invitations…"
                aria-label="Search"
                className="w-xs"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </InputGroup>
          </div>
        </div>
        
        <div className="px-6 pb-6">
          {loadingInvitations ? (
            <div className="text-center py-8">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600 mx-auto"></div>
              <Text className="mt-2 text-zinc-500 dark:text-zinc-400">Loading invitations...</Text>
            </div>
          ) : getFilteredInvitations().length > 0 ? (
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader>Email</TableHeader>
                  <TableHeader>Status</TableHeader>
                  <TableHeader>Sent</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {getFilteredInvitations().map((invitation) => (
                  <TableRow key={invitation.id}>
                    <TableCell>
                      <Text className="font-medium">{invitation.email}</Text>
                    </TableCell>
                    <TableCell>
                      {getStatusBadge(invitation.status || 'unknown')}
                    </TableCell>
                    <TableCell>
                      <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                        {formatDate(invitation.createdAt || '')}
                      </Text>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <div className="text-center py-8 bg-zinc-50 dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
              <MaterialSymbol name="mail_outline" className="h-12 w-12 mx-auto mb-4 text-zinc-300" />
              <Text className="text-lg font-medium text-zinc-900 dark:text-white mb-2">
                {searchTerm ? 'No invitations found' : 'No invitations sent yet'}
              </Text>
              <Text className="text-sm text-zinc-500 dark:text-zinc-400">
                {searchTerm ? 'Try adjusting your search criteria' : 'Send invitations to add members to your organization'}
              </Text>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}