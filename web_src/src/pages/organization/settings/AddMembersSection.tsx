import { useState, useEffect, useCallback } from 'react'
import { Button } from '../../../components/Button/button'
import { Textarea } from '../../../components/Textarea/textarea'
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
import { Link } from '../../../components/Link/link'
import {
  authorizationAssignRole,
  authorizationListRoles,
  authorizationAddUserToOrganizationGroup,
} from '../../../api-client/sdk.gen'
import { AuthorizationRole } from '../../../api-client/types.gen'
import { capitalizeFirstLetter } from '../../../utils/text'
import Papa from 'papaparse'

interface AddMembersSectionProps {
  showRoleSelection?: boolean
  organizationId: string
  groupName?: string
  onMemberAdded?: () => void
  className?: string
}

export function AddMembersSection({ showRoleSelection = true, organizationId, groupName, onMemberAdded, className }: AddMembersSectionProps) {
  const [addMembersTab, setAddMembersTab] = useState<'emails' | 'upload'>('emails')
  const [emailsInput, setEmailsInput] = useState('')
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [bulkUserRole, setBulkUserRole] = useState('')
  const [emailRole, setEmailRole] = useState('')
  const [isInviting, setIsInviting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [roles, setRoles] = useState<AuthorizationRole[]>([])
  const [loadingRoles, setLoadingRoles] = useState(true)

  const addMembersTabs: Tab[] = [
    { id: 'emails', label: 'By emails' },
    { id: 'upload', label: 'By file' }
  ]

  const fetchRoles = useCallback(async () => {
    try {
      setLoadingRoles(true)
      setError(null)
      const response = await authorizationListRoles({
        query: {
          domainType: 'DOMAIN_TYPE_ORGANIZATION',
          domainId: organizationId
        }
      })

      if (response.data?.roles) {
        setRoles(response.data.roles)
        // Set default role to the first org member role found, or first role if none
        const orgMemberRole = response.data.roles.find(role => role.name?.includes('member'))
        const defaultRole = orgMemberRole?.name || response.data.roles[0]?.name || ''
        setBulkUserRole(defaultRole)
        setEmailRole(defaultRole)
      }
    } catch (err) {
      console.error('Error fetching roles:', err)
      setError('Failed to fetch roles')
    } finally {
      setLoadingRoles(false)
    }
  }, [organizationId])

  useEffect(() => {
    fetchRoles()
  }, [fetchRoles])

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (file) {
      setUploadFile(file)
    }
  }

  const handleBulkAddSubmit = async () => {
    if (!uploadFile) return

    setIsInviting(true)
    setError(null)

    try {
      // Parse CSV file
      const fileContent = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader()
        reader.onload = (e) => resolve(e.target?.result as string)
        reader.onerror = reject
        reader.readAsText(uploadFile)
      })

      // Parse CSV content
      const parseResult = Papa.parse(fileContent, {
        header: true,
        skipEmptyLines: true,
        transformHeader: (header) => header.toLowerCase().trim()
      })

      if (parseResult.errors.length > 0) {
        throw new Error(`CSV parsing errors: ${parseResult.errors.map(e => e.message).join(', ')}`)
      }

      // Extract emails from CSV data
      const csvData = parseResult.data as Array<{ email?: string;[key: string]: string | undefined }>
      const emailsToAdd = csvData
        .map(row => row.email || row['email address'] || '')
        .filter(email => email && isEmailValid(email))

      if (emailsToAdd.length === 0) {
        throw new Error('No valid email addresses found in the CSV file. Please ensure the CSV has an "email" column.')
      }

      const roleToAssign = showRoleSelection ? bulkUserRole : (roles.find(r => r.name?.includes('member'))?.name || roles[0]?.name || '')

      // Process each email
      for (const email of emailsToAdd) {
        if (groupName) {
          // Add user to specific group
          await authorizationAddUserToOrganizationGroup({
            path: { groupName },
            body: { userEmail: email, organizationId: organizationId }
          })
        } else {
          // Add user to organization with role
          await authorizationAssignRole({
            body: {
              userEmail: email,
              roleAssignment: {
                domainType: 'DOMAIN_TYPE_ORGANIZATION',
                domainId: organizationId,
                role: roleToAssign
              }
            }
          })
        }
      }

      console.log('Successfully processed bulk upload:', {
        fileName: uploadFile.name,
        emails: emailsToAdd,
        role: roleToAssign,
        count: emailsToAdd.length
      })

      setUploadFile(null)
      const defaultRole = roles.find(r => r.name?.includes('member'))?.name || roles[0]?.name || ''
      setBulkUserRole(defaultRole)
      setAddMembersTab('emails')

      onMemberAdded?.()
    } catch (err) {
      console.error('Error processing bulk upload:', err)
      setError(err instanceof Error ? err.message : 'Failed to process bulk upload. Please try again.')
    } finally {
      setIsInviting(false)
    }
  }

  const isEmailValid = (email: string) => {
    const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    return emailRegex.test(email)
  }

  const handleEmailsSubmit = async () => {
    if (!emailsInput.trim()) return

    setIsInviting(true)
    setError(null)

    try {
      const emails = emailsInput.split(',').map(email => email.trim()).filter(email => email.length > 0 && isEmailValid(email))
      const roleToAssign = showRoleSelection ? emailRole : (roles.find(r => r.name?.includes('member'))?.name || roles[0]?.name || '')

      // Process each email
      for (const email of emails) {
        if (groupName) {
          // Add user to specific group
          await authorizationAddUserToOrganizationGroup({
            path: { groupName },
            body: { userEmail: email, organizationId: organizationId }
          })
        } else {
          // Add user to organization with role
          await authorizationAssignRole({
            body: {
              userEmail: email,
              roleAssignment: {
                domainType: 'DOMAIN_TYPE_ORGANIZATION',
                domainId: organizationId,
                role: roleToAssign
              }
            }
          })
        }
      }

      console.log('Successfully added members by email:', {
        emails,
        role: roleToAssign,
        count: emails.length
      })

      setEmailsInput('')
      const defaultRole = roles.find(r => r.name?.includes('member'))?.name || roles[0]?.name || ''
      setEmailRole(defaultRole)

      onMemberAdded?.()
    } catch (err) {
      console.error('Error adding members by email:', err)
      setError('Failed to add members. Please try again.')
    } finally {
      setIsInviting(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleEmailsSubmit()
    }
  }

  return (
    <div className={`bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 ${className}`}>
      <div className="flex items-center justify-between mb-4">
        <div>
          <Text className="font-semibold text-zinc-900 dark:text-white mb-1">
            Add members
          </Text>

        </div>
        {groupName && (
          <Text className="text-xs text-zinc-500 dark:text-zinc-400">
            Adding members to group: <strong>{groupName}</strong>
          </Text>
        )}
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          <p className="text-sm">{error}</p>
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
          onTabChange={(tabId) => setAddMembersTab(tabId as 'emails' | 'upload')}
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
                  {emailRole ? capitalizeFirstLetter(emailRole.split('_').at(-1) || '') : 'Select Role'}
                  <MaterialSymbol name="keyboard_arrow_down" />
                </DropdownButton>
                <DropdownMenu>
                  {roles.map((role) => (
                    <DropdownItem key={role.name} onClick={() => setEmailRole(role.name || '')}>
                      <DropdownLabel>{capitalizeFirstLetter(role.name?.split('_').at(-1) || '')}</DropdownLabel>
                      <DropdownDescription>{role.permissions?.length || 0} permissions</DropdownDescription>
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
              {isInviting ? 'Adding...' : (groupName ? 'Add to Group' : 'Invite')}
            </Button>
          </div>
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

                <label htmlFor="add-members-upload">
                  <Button outline className='flex items-center text-sm gap-2' type="button">
                    <MaterialSymbol name="folder_open" size="sm" />
                    Browse
                  </Button>
                </label>
              </div>
            </div>
            <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
              The CSV file should have an "email" column with email addresses.
              <Link href="#" className="text-blue-600 hover:text-blue-700 ml-1">Check .csv file format requirements</Link>
            </Text>
          </Field>

          {/* Role Selection */}
          {showRoleSelection && !groupName && (
            <Field>
              <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Role
              </Label>
              <Dropdown>
                <DropdownButton outline className="flex items-center gap-2 text-sm justify-between">
                  {bulkUserRole ? capitalizeFirstLetter(bulkUserRole.split('_').at(-1) || '') : 'Select Role'}
                  <MaterialSymbol name="keyboard_arrow_down" />
                </DropdownButton>
                <DropdownMenu>
                  {roles.map((role) => (
                    <DropdownItem key={role.name} onClick={() => setBulkUserRole(role.name || '')}>
                      <DropdownLabel>{capitalizeFirstLetter(role.name?.split('_').at(-1) || '')}</DropdownLabel>
                      <DropdownDescription>{role.permissions?.length || 0} permissions</DropdownDescription>
                    </DropdownItem>
                  ))}
                </DropdownMenu>
              </Dropdown>
            </Field>
          )}

          {/* Upload Button */}
          <div className="flex justify-end">
            <Button
              color="blue"
              onClick={handleBulkAddSubmit}
              disabled={!uploadFile || isInviting || (!groupName && showRoleSelection && !bulkUserRole)}
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="add" size="sm" />
              {isInviting ? 'Processing...' : (groupName ? 'Add to Group' : 'Invite')}
            </Button>
          </div>
        </div>
      ) : null}
    </div>
  )
}