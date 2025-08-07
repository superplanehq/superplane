import { useState, useEffect, forwardRef, useImperativeHandle, useMemo } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Button } from '../../../components/Button/button'
import { Textarea } from '../../../components/Textarea/textarea'
import { Input, InputGroup } from '../../../components/Input/input'
import { Avatar } from '../../../components/Avatar/avatar'
import { Checkbox } from '../../../components/Checkbox/checkbox'
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from '../../../components/Dropdown/dropdown'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Text } from '../../../components/Text/text'
import { Field, Label } from '../../../components/Fieldset/fieldset'
import { Tabs, type Tab } from '../../../components/Tabs/tabs'
import { Badge } from '../../../components/Badge/badge'
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
  useOrganizationGroupUsers,
  useAssignRole,
  useAddUserToGroup
} from '../../../hooks/useOrganizationData'
import Papa from 'papaparse'

interface Invitation {
  id: string
  organizationId: string
  email: string
  status: 'pending' | 'accepted' | 'expired'
  expiresAt: string
  createdAt: string
}

interface AddMembersSectionProps {
  showRoleSelection?: boolean
  organizationId: string
  groupName?: string
  onMemberAdded?: () => void
  className?: string
}

export interface AddMembersSectionRef {
  refreshExistingMembers: () => void
}

const AddMembersSectionComponent = forwardRef<AddMembersSectionRef, AddMembersSectionProps>(
  ({ showRoleSelection = true, organizationId, groupName, onMemberAdded, className }, ref) => {
    const [addMembersTab, setAddMembersTab] = useState<'emails' | 'upload' | 'existing'>('emails')
    const [emailsInput, setEmailsInput] = useState('')
    const [uploadFile, setUploadFile] = useState<File | null>(null)
    const [bulkUserRole, setBulkUserRole] = useState('')
    const [emailRole, setEmailRole] = useState('')
    const [selectedMembers, setSelectedMembers] = useState<Set<string>>(new Set())
    const [memberSearchTerm, setMemberSearchTerm] = useState('')
    const [invitationError, setInvitationError] = useState<string | null>(null)
    const queryClient = useQueryClient()

    // React Query hooks
    const { data: roles = [], isLoading: loadingRoles, error: rolesError } = useOrganizationRoles(organizationId)
    const { data: orgUsers = [], isLoading: loadingOrgUsers, error: orgUsersError } = useOrganizationUsers(organizationId)
    const { data: groupUsers = [], isLoading: loadingGroupUsers, error: groupUsersError } = useOrganizationGroupUsers(organizationId, groupName || '')
    
    // Fetch pending invitations - only when not in group context
    const { data: invitations = [], isLoading: loadingInvitations } = useQuery<Invitation[]>({
      queryKey: ['invitations', organizationId],
      queryFn: async () => {
        const response = await fetch(`/api/v1/organizations/${organizationId}/invitations`, {
          credentials: 'include',
        })
        if (!response.ok) {
          throw new Error('Failed to fetch invitations')
        }
        const data = await response.json()
        return data.invitations || []
      },
      enabled: !groupName, // Only fetch invitations when not in group context
    })

    // Mutations
    const assignRoleMutation = useAssignRole(organizationId)
    const addUserToGroupMutation = useAddUserToGroup(organizationId)
    
    // Create invitation mutation
    const createInvitationMutation = useMutation({
      mutationFn: async (email: string) => {
        const response = await fetch(`/api/v1/organizations/${organizationId}/invitations`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
          },
          credentials: 'include',
          body: JSON.stringify({
            email: email,
            organization_id: organizationId,
          }),
        })

        if (!response.ok) {
          const errorData = await response.json()
          throw new Error(errorData.message || 'Failed to send invitation')
        }

        return response.json()
      },
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: ['invitations', organizationId] })
        setInvitationError(null)
        onMemberAdded?.()
      },
      onError: (error: Error) => {
        setInvitationError(error.message)
      },
    })

    const isInviting = assignRoleMutation.isPending || addUserToGroupMutation.isPending || createInvitationMutation.isPending
    const error = rolesError || orgUsersError || groupUsersError

    const addMembersTabs: Tab[] = [
      { id: 'emails', label: 'By emails' },
      ...(groupName ? [{ id: 'existing', label: 'From organization' }] : []),
      { id: 'upload', label: 'By file' }
    ]

    // Calculate available members (org users who aren't in the group)
    const existingMembers = useMemo(() => {
      if (!groupName) return []

      const existingMemberIds = new Set(groupUsers.map(user => user.metadata?.id))
      return orgUsers.filter(user => !existingMemberIds.has(user.metadata?.id))
    }, [orgUsers, groupUsers, groupName])

    const loadingMembers = loadingOrgUsers || loadingGroupUsers

    // Set default roles when roles are loaded
    useEffect(() => {
      if (roles.length > 0) {
        const orgMemberRole = roles.find(role => role.metadata?.name?.includes('member'))
        const defaultRole = orgMemberRole?.metadata?.name || roles[0]?.metadata?.name || ''
        setBulkUserRole(defaultRole)
        setEmailRole(defaultRole)
      }
    }, [roles])

    // Expose refresh function to parent
    useImperativeHandle(ref, () => ({
      refreshExistingMembers: () => {
        // No need to manually refresh - React Query will handle it
      }
    }), [])

    const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0]
      if (file) {
        setUploadFile(file)
      }
    }

    const handleBulkAddSubmit = async () => {
      if (!uploadFile) {
        console.error('No file selected')
        return
      }

      const roleToAssign = showRoleSelection ? bulkUserRole : (roles.find(r => r.metadata?.name?.includes('member'))?.metadata?.name || roles[0]?.metadata?.name || '')

      if (!groupName && showRoleSelection && !bulkUserRole) {
        console.error('No role selected')
        return
      }

      try {
        // Parse CSV file
        const fileContent = await new Promise<string>((resolve, reject) => {
          const reader = new FileReader()
          reader.onload = (e) => resolve(e.target?.result as string)
          reader.onerror = reject
          reader.readAsText(uploadFile)
        })


        // Parse CSV content with multiple delimiter attempts
        let parseResult = Papa.parse(fileContent, {
          header: true,
          skipEmptyLines: true,
          delimiter: ',', // Default delimiter
          transformHeader: (header) => header.toLowerCase().trim()
        })

        const delimiters = [',', ';', '\t', '|']

        for (const delimiter of delimiters) {
          const tempResult = Papa.parse(fileContent, {
            header: true,
            skipEmptyLines: true,
            delimiter: delimiter,
            transformHeader: (header) => header.toLowerCase().trim()
          })

          // If we have data and no critical errors, use this result
          if (tempResult.data && tempResult.data.length > 0) {
            const criticalErrors = tempResult.errors.filter(error =>
              error.type === 'Delimiter' && error.code === 'UndetectableDelimiter'
            )
            if (criticalErrors.length === 0) {
              parseResult = tempResult
              break
            }
          }
        }

        // Only throw error for critical parsing issues, not delimiter detection warnings
        const criticalErrors = parseResult?.errors?.filter(error =>
          error.type !== 'Delimiter' || error.code !== 'UndetectableDelimiter'
        ) || []

        if (criticalErrors.length > 0) {
          console.error('Critical CSV parsing errors:', criticalErrors)
          throw new Error(`CSV parsing errors: ${criticalErrors.map(e => e.message).join(', ')}`)
        }

        // Extract emails from CSV data
        const csvData = parseResult.data as Array<{ email?: string;[key: string]: string | undefined }>
        const emailsToAdd = csvData
          .map(row => row.email || row['email address'] || '')
          .filter(email => email && isEmailValid(email))


        if (emailsToAdd.length === 0) {
          throw new Error('No valid email addresses found in the CSV file. Please ensure the CSV has an "email" column.')
        }

        // Process each email
        for (const email of emailsToAdd) {
          if (groupName) {
            // Add user to specific group
            await addUserToGroupMutation.mutateAsync({
              groupName,
              userEmail: email,
              organizationId
            })
          } else {
            // For organization-level, send invitation instead of directly adding user
            await createInvitationMutation.mutateAsync(email)
          }
        }

        setUploadFile(null)
        const defaultRole = roles.find(r => r.metadata?.name?.includes('member'))?.metadata?.name || roles[0]?.metadata?.name || ''
        setBulkUserRole(defaultRole)
        setAddMembersTab('emails')

        onMemberAdded?.()
      } catch (error) {
        console.error('Failed to add members by file:', error)
        alert(`Failed to add members: ${error instanceof Error ? error.message : 'Unknown error'}`)
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
        const roleToAssign = showRoleSelection ? emailRole : (roles.find(r => r.metadata?.name?.includes('member'))?.metadata?.name || roles[0]?.metadata?.name || '')

        // Process each email
        for (const email of emails) {
          if (groupName) {
            // Add user to specific group
            await addUserToGroupMutation.mutateAsync({
              groupName,
              userEmail: email,
              organizationId
            })
          } else {
            // For organization-level, send invitation instead of directly adding user
            await createInvitationMutation.mutateAsync(email)
          }
        }

        setEmailsInput('')
        const defaultRole = roles.find(r => r.metadata?.name?.includes('member'))?.metadata?.name || roles[0]?.metadata?.name || ''
        setEmailRole(defaultRole)

        onMemberAdded?.()
      } catch {
        console.error('Failed to add members by email')
      }
    }

    const handleExistingMembersSubmit = async () => {
      if (selectedMembers.size === 0) return

      try {
        const selectedUsers = existingMembers.filter(member => selectedMembers.has(member.metadata?.id || ''))

        // Process each selected member
        for (const member of selectedUsers) {
          if (groupName) {
            // Add user to specific group - try both userId and email
            try {
              await addUserToGroupMutation.mutateAsync({
                groupName,
                userId: member.metadata?.id || '',
                organizationId
              })
            } catch (err) {
              // If userId fails, try with email
              if (member.metadata?.email) {
                await addUserToGroupMutation.mutateAsync({
                  groupName,
                  userEmail: member.metadata?.email,
                  organizationId
                })
              } else {
                throw err
              }
            }
          }
        }


        setSelectedMembers(new Set())
        setMemberSearchTerm('')

        onMemberAdded?.()
      } catch {
        console.error('Failed to add existing members')
      }
    }


    const handleSelectAll = () => {
      const filteredMembers = getFilteredExistingMembers()
      if (selectedMembers.size === filteredMembers.length) {
        setSelectedMembers(new Set())
      } else {
        setSelectedMembers(new Set(filteredMembers.map(m => m.metadata!.id!)))
      }
    }

    const getFilteredExistingMembers = () => {
      if (!memberSearchTerm) return existingMembers

      return existingMembers.filter(member =>
        member.spec?.displayName?.toLowerCase().includes(memberSearchTerm.toLowerCase()) ||
        member.metadata?.email?.toLowerCase().includes(memberSearchTerm.toLowerCase())
      )
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
        case 'expired':
          return <Badge color="red">Expired</Badge>
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

    return (
      <div className={`bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 ${className}`}>
        <div className="flex items-center justify-between mb-4">
          <div>
            <Text className="font-semibold text-zinc-900 dark:text-white mb-1">
              Add members
            </Text>
          </div>
        </div>

        {(error || invitationError) && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            <p className="text-sm">
              {error instanceof Error ? error.message : error ? 'Failed to fetch data' : invitationError}
            </p>
          </div>
        )}

        {(loadingRoles) && (
          <div className="flex justify-center items-center h-20">
            <p className="text-zinc-500 dark:text-zinc-400">
              {loadingRoles && 'Loading roles...'}
            </p>
          </div>
        )}

        {/* Add Members Tabs */}
        <div className="mb-6">
          <Tabs
            tabs={addMembersTabs}
            defaultTab={addMembersTab}
            variant='underline'
            onTabChange={(tabId) => setAddMembersTab(tabId as 'emails' | 'upload' | 'existing')}
          />
        </div>

        {/* Tab Content */}
        {!loadingRoles && addMembersTab === 'emails' ? (
          // Enter emails tab content
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

              {showRoleSelection && !groupName && (
                <Dropdown>
                  <DropdownButton outline className="flex items-center gap-2 text-sm">
                    {emailRole ? roles.find(r => r.metadata?.name === emailRole)?.spec?.displayName || emailRole : 'Select Role'}
                    <MaterialSymbol name="keyboard_arrow_down" />
                  </DropdownButton>
                  <DropdownMenu>
                    {roles.map((role) => (
                      <DropdownItem key={role.metadata?.name} onClick={() => setEmailRole(role.metadata?.name || '')}>
                        <DropdownLabel>{role.spec?.displayName}</DropdownLabel>
                        <DropdownDescription>{role.spec?.description || 'No description available'}</DropdownDescription>
                      </DropdownItem>
                    ))}
                  </DropdownMenu>
                </Dropdown>
              )}

              <Button
                color="blue"
                className='flex items-center text-sm gap-2'
                onClick={handleEmailsSubmit}
                disabled={!emailsInput.trim() || isInviting || (!groupName && showRoleSelection && !emailRole)}
              >
                <MaterialSymbol name="add" size="sm" />
                {isInviting ? (groupName ? 'Adding...' : 'Inviting...') : (groupName ? 'Add to Group' : 'Send Invitation')}
              </Button>
            </div>
          </div>
        ) : !loadingRoles && addMembersTab === 'existing' && groupName ? (
          // Existing members tab content (only for groups)
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <InputGroup>
                <Input
                  name="member-search"
                  placeholder="Search members..."
                  aria-label="Search members"
                  className="w-xs"
                  value={memberSearchTerm}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMemberSearchTerm(e.target.value)}
                />
              </InputGroup>
              <div className="flex items-center gap-2">
                <Button
                  outline
                  className="flex items-center gap-2 text-sm"
                  onClick={handleSelectAll}
                  disabled={loadingMembers || getFilteredExistingMembers().length === 0}
                >
                  <MaterialSymbol name="select_all" size="sm" />
                  {selectedMembers.size === getFilteredExistingMembers().length ? 'Deselect All' : 'Select All'}
                </Button>
                <Button
                  color="blue"
                  className="flex items-center gap-2 text-sm"
                  onClick={handleExistingMembersSubmit}
                  disabled={selectedMembers.size === 0 || isInviting}
                >
                  <MaterialSymbol name="add" size="sm" />
                  {isInviting ? 'Adding...' : `Add ${selectedMembers.size} member${selectedMembers.size === 1 ? '' : 's'}`}
                </Button>
              </div>
            </div>

            {loadingMembers ? (
              <div className="flex justify-center items-center h-32">
                <p className="text-zinc-500 dark:text-zinc-400">Loading members...</p>
              </div>
            ) : (
              <div className="max-h-96 overflow-y-auto border border-zinc-200 dark:border-zinc-700 rounded-lg">
                {getFilteredExistingMembers().length === 0 ? (
                  <div className="text-center py-8">
                    <p className="text-zinc-500 dark:text-zinc-400">
                      {memberSearchTerm ? 'No members found matching your search' : 'No members available'}
                    </p>
                  </div>
                ) : (
                  <div className="divide-y divide-zinc-200 dark:divide-zinc-700">
                    {getFilteredExistingMembers().map((member) => (
                      <div key={member.metadata!.id!} className="p-3 flex items-center gap-3 hover:bg-zinc-50 dark:hover:bg-zinc-800">
                        <Checkbox
                          checked={selectedMembers.has(member.metadata!.id!)}
                          onChange={(checked) => {
                            setSelectedMembers(prev => {
                              const newSet = new Set(prev)
                              if (checked) {
                                newSet.add(member.metadata!.id!)
                              } else {
                                newSet.delete(member.metadata!.id!)
                              }
                              return newSet
                            })
                          }}
                        />
                        <Avatar
                          src={member.spec?.avatarUrl}
                          initials={member.spec?.displayName?.charAt(0) || 'U'}
                          className="size-8"
                        />
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-zinc-900 dark:text-white truncate">
                            {member.spec?.displayName || member.metadata!.id!}
                          </div>
                          <div className="text-xs text-zinc-500 dark:text-zinc-400 truncate">
                            {member.metadata?.email || `${member.metadata!.id!}@email.placeholder`}
                          </div>
                        </div>
                        <div className="flex items-center">
                          <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${member?.status?.isActive
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                            : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
                            }`}>
                            {member?.status?.isActive ? 'Active' : 'Pending'}
                          </span>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        ) : !loadingRoles ? (
          // Upload file tab content
          <div className="space-y-6 text-left">
            {/* File Upload */}
            <Field>
              <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                CSV/Excel Spreadsheet *
              </Label>
              <div className="border-2 border-dashed border-zinc-300 dark:border-zinc-600 rounded-lg p-6 text-center">
                <MaterialSymbol name="cloud_upload" size="4xl" className="text-zinc-400 mb-2" />
                <div className="space-y-2">
                  <Text className="!text-lg text-zinc-600 dark:text-zinc-400">
                    {uploadFile ? uploadFile.name : 'Drag and drop .csv or'}
                  </Text>
                  <input
                    type="file"
                    accept=".xlsx,.xls,.csv"
                    onChange={handleFileUpload}
                    className="hidden"
                    id="add-members-upload"
                  />

                  <label
                    htmlFor="add-members-upload"
                    className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-zinc-700 dark:text-zinc-300 bg-white dark:bg-zinc-800 border border-zinc-300 dark:border-zinc-600 rounded-md hover:bg-zinc-50 dark:hover:bg-zinc-700 cursor-pointer"
                  >
                    <MaterialSymbol name="folder_open" size="sm" />
                    Browse
                  </label>
                </div>
              </div >
              <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
                The CSV file should have an "email" column with email addresses.
              </Text>
            </Field >

            {/* Role Selection */}
            {
              showRoleSelection && !groupName && (
                <Field>
                  <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                    Role
                  </Label>
                  <Dropdown>
                    <DropdownButton outline className="flex items-center gap-2 text-sm justify-between">
                      {bulkUserRole ? roles.find(r => r.metadata?.name === bulkUserRole)?.spec?.displayName || bulkUserRole : 'Select Role'}
                      <MaterialSymbol name="keyboard_arrow_down" />
                    </DropdownButton>
                    <DropdownMenu>
                      {roles.map((role) => (
                        <DropdownItem key={role.metadata?.name} onClick={() => setBulkUserRole(role.metadata?.name || '')}>
                          <DropdownLabel>{role.spec?.displayName}</DropdownLabel>
                          <DropdownDescription>{role.spec?.description || 'No description available'}</DropdownDescription>
                        </DropdownItem>
                      ))}
                    </DropdownMenu>
                  </Dropdown>
                </Field>
              )
            }

            {/* Upload Button */}
            <div className="flex justify-end">
              <Button
                color="blue"
                onClick={handleBulkAddSubmit}
                disabled={!uploadFile || isInviting || (!groupName && showRoleSelection && !bulkUserRole)}
                className="flex items-center gap-2"
              >
                <MaterialSymbol name="add" size="sm" />
                {isInviting ? (groupName ? 'Processing...' : 'Sending Invitations...') : (groupName ? 'Add to Group' : 'Send Invitations')}
              </Button>
            </div>
          </div >
        ) : null}

        {/* Note for existing members tab when not in group context */}
        {
          addMembersTab === 'existing' && !groupName && (
            <div className="text-center py-8">
              <p className="text-zinc-500 dark:text-zinc-400">
                This option is only available when adding members to a group.
              </p>
            </div>
          )
        }

        {/* Pending Invitations - only show when not in group context */}
        {!groupName && (
          <div className="mt-8 pt-6 border-t border-zinc-200 dark:border-zinc-700">
            <Text className="font-semibold text-zinc-900 dark:text-white mb-4">
              Pending Invitations
            </Text>
            
            {loadingInvitations ? (
              <div className="text-center py-8">
                <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-blue-600 mx-auto"></div>
                <Text className="mt-2 text-zinc-500 dark:text-zinc-400">Loading invitations...</Text>
              </div>
            ) : invitations && invitations.length > 0 ? (
              <div className="bg-white dark:bg-zinc-900 rounded-lg border border-zinc-200 dark:border-zinc-800">
                <Table dense>
                  <TableHead>
                    <TableRow>
                      <TableHeader>Email</TableHeader>
                      <TableHeader>Status</TableHeader>
                      <TableHeader>Sent</TableHeader>
                      <TableHeader>Expires</TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {invitations.map((invitation) => (
                      <TableRow key={invitation.id}>
                        <TableCell>
                          <Text className="font-medium">{invitation.email}</Text>
                        </TableCell>
                        <TableCell>
                          {getStatusBadge(invitation.status)}
                        </TableCell>
                        <TableCell>
                          <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                            {formatDate(invitation.createdAt)}
                          </Text>
                        </TableCell>
                        <TableCell>
                          <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                            {formatDate(invitation.expiresAt)}
                          </Text>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            ) : (
              <div className="text-center py-8 bg-zinc-50 dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <Text className="text-zinc-500 dark:text-zinc-400">
                  No pending invitations
                </Text>
              </div>
            )}
          </div>
        )}
      </div >
    )
  })

AddMembersSectionComponent.displayName = 'AddMembersSection'

export const AddMembersSection = AddMembersSectionComponent