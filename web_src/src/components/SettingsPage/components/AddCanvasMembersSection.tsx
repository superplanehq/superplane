import { useState, useEffect, forwardRef, useImperativeHandle, useMemo } from 'react'
import { Button } from '../../Button/button'
import { Textarea } from '../../Textarea/textarea'
import { Input, InputGroup } from '../../Input/input'
import { Avatar } from '../../Avatar/avatar'
import { Checkbox } from '../../Checkbox/checkbox'
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from '../../Dropdown/dropdown'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Text } from '../../Text/text'
import { Field, Label } from '../../Fieldset/fieldset'
import { Tabs, type Tab } from '../../Tabs/tabs'
import {
  useCanvasRoles,
  useCanvasUsers,
  useOrganizationUsersForCanvas,
  useAssignCanvasRole
} from '../../../hooks/useCanvasData'
import Papa from 'papaparse'

interface AddCanvasMembersSectionProps {
  canvasId: string
  organizationId: string
  onMemberAdded?: () => void
  className?: string
}

export interface AddCanvasMembersSectionRef {
  refreshExistingMembers: () => void
}

const AddCanvasMembersSectionComponent = forwardRef<AddCanvasMembersSectionRef, AddCanvasMembersSectionProps>(
  ({ canvasId, organizationId, onMemberAdded, className }, ref) => {
    const [addMembersTab, setAddMembersTab] = useState<'emails' | 'upload' | 'existing'>('emails')
    const [emailsInput, setEmailsInput] = useState('')
    const [uploadFile, setUploadFile] = useState<File | null>(null)
    const [bulkUserRole, setBulkUserRole] = useState('')
    const [emailRole, setEmailRole] = useState('')
    const [selectedMembers, setSelectedMembers] = useState<Set<string>>(new Set())
    const [memberSearchTerm, setMemberSearchTerm] = useState('')


    const { data: canvasRoles = [], isLoading: loadingRoles, error: rolesError } = useCanvasRoles(canvasId)
    const { data: canvasUsers = [] } = useCanvasUsers(canvasId)
    const { data: orgUsers = [], isLoading: loadingOrgUsers, error: orgUsersError } = useOrganizationUsersForCanvas(organizationId)


    const assignRoleMutation = useAssignCanvasRole(canvasId)

    const isInviting = assignRoleMutation.isPending
    const error = rolesError || orgUsersError

    const addMembersTabs: Tab[] = [
      { id: 'emails', label: 'By emails' },
      { id: 'existing', label: 'From organization' },
      { id: 'upload', label: 'By file' }
    ]


    const existingMembers = useMemo(() => {
      const canvasMemberIds = new Set(canvasUsers.map(user => user.metadata?.id))
      return orgUsers.filter(user => !canvasMemberIds.has(user.metadata?.id))
    }, [orgUsers, canvasUsers])

    const loadingMembers = loadingOrgUsers


    useEffect(() => {
      if (canvasRoles.length > 0) {
        const canvasMemberRole = canvasRoles.find(role => role.metadata?.name?.includes('member'))
        const defaultRole = canvasMemberRole?.metadata?.name || canvasRoles[0]?.metadata?.name || ''
        setBulkUserRole(defaultRole)
        setEmailRole(defaultRole)
      }
    }, [canvasRoles])


    useImperativeHandle(ref, () => ({
      refreshExistingMembers: () => {

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

      if (!bulkUserRole) {
        console.error('No role selected')
        return
      }

      try {

        const fileContent = await new Promise<string>((resolve, reject) => {
          const reader = new FileReader()
          reader.onload = (e) => resolve(e.target?.result as string)
          reader.onerror = reject
          reader.readAsText(uploadFile)
        })



        let parseResult = Papa.parse(fileContent, {
          header: true,
          skipEmptyLines: true,
          delimiter: ',',
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


        const criticalErrors = parseResult?.errors?.filter(error =>
          error.type !== 'Delimiter' || error.code !== 'UndetectableDelimiter'
        ) || []

        if (criticalErrors.length > 0) {
          throw new Error(`CSV parsing errors: ${criticalErrors.map(e => e.message).join(', ')}`)
        }


        const csvData = parseResult.data as Array<{ email?: string;[key: string]: string | undefined }>

        const emailsToAdd = csvData
          .map(row => row.email || row['email address'] || '')
          .filter(email => email && isEmailValid(email))


        if (emailsToAdd.length === 0) {
          throw new Error('No valid email addresses found in the CSV file. Please ensure the CSV has an "email" column.')
        }

        const roleToAssign = bulkUserRole


        for (const email of emailsToAdd) {
          await assignRoleMutation.mutateAsync({
            userEmail: email,
            role: roleToAssign
          })
        }

        setUploadFile(null)
        const defaultRole = canvasRoles.find(r => r.metadata?.name?.includes('member'))?.metadata?.name || canvasRoles[0]?.metadata?.name || ''
        setBulkUserRole(defaultRole)
        setAddMembersTab('emails')

        onMemberAdded?.()
      } catch (error) {
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
        const roleToAssign = emailRole


        for (const email of emails) {
          await assignRoleMutation.mutateAsync({
            userEmail: email,
            role: roleToAssign
          })
        }

        setEmailsInput('')
        const defaultRole = canvasRoles.find(r => r.metadata?.name?.includes('member'))?.metadata?.name || canvasRoles[0]?.metadata?.name || ''
        setEmailRole(defaultRole)

        onMemberAdded?.()
      } catch {
        alert('Failed to add members by email')
      }
    }

    const handleExistingMembersSubmit = async () => {
      if (selectedMembers.size === 0) return

      try {
        const selectedUsers = existingMembers.filter(member => selectedMembers.has(member.metadata!.id!))
        const roleToAssign = bulkUserRole


        for (const member of selectedUsers) {

          try {
            await assignRoleMutation.mutateAsync({
              userId: member.metadata?.id,
              role: roleToAssign
            })
          } catch (err) {

            if (member.metadata?.email) {
              await assignRoleMutation.mutateAsync({
                userEmail: member.metadata?.email,
                role: roleToAssign
              })
            } else {
              throw err
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

    return (
      <div className={`bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 ${className}`}>
        <div className="flex items-center justify-between mb-4 text-left">
          <div>
            <Text className="font-semibold text-zinc-900 dark:text-white mb-1">
              Add members
            </Text>
            <Text className="text-sm text-zinc-600 dark:text-zinc-400">
              Add members from your organization to this canvas and assign them a role.
            </Text>
          </div>
        </div>

        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            <p className="text-sm">{error instanceof Error ? error.message : 'Failed to fetch data'}</p>
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

              <Dropdown>
                <DropdownButton outline className="flex items-center gap-2 text-sm">
                  {emailRole ? canvasRoles.find(r => r.metadata?.name === emailRole)?.spec?.displayName || emailRole : 'Select Role'}
                  <MaterialSymbol name="keyboard_arrow_down" />
                </DropdownButton>
                <DropdownMenu>
                  {canvasRoles.map((role) => (
                    <DropdownItem key={role.metadata?.name} onClick={() => setEmailRole(role.metadata?.name || '')}>
                      <DropdownLabel>{role.spec?.displayName}</DropdownLabel>
                      <DropdownDescription>{role.spec?.description || 'No description available'}</DropdownDescription>
                    </DropdownItem>
                  ))}
                </DropdownMenu>
              </Dropdown>

              <Button
                color="blue"
                className='flex items-center text-sm gap-2'
                onClick={handleEmailsSubmit}
                disabled={!emailsInput.trim() || isInviting || !emailRole}
              >
                <MaterialSymbol name="add" size="sm" />
                {isInviting ? 'Adding...' : 'Add to Canvas'}
              </Button>
            </div>
          </div>
        ) : !loadingRoles && addMembersTab === 'existing' ? (

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

            {/* Role Selection for Existing Members */}
            <div className="flex items-center gap-2">
              <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                Role for selected members:
              </Text>
              <Dropdown>
                <DropdownButton outline className="flex items-center gap-2 text-sm">
                  {bulkUserRole ? canvasRoles.find(r => r.metadata?.name === bulkUserRole)?.spec?.displayName || bulkUserRole : 'Select Role'}
                  <MaterialSymbol name="keyboard_arrow_down" />
                </DropdownButton>
                <DropdownMenu>
                  {canvasRoles.map((role) => (
                    <DropdownItem key={role.metadata?.name} onClick={() => setBulkUserRole(role.metadata?.name || '')}>
                      <DropdownLabel>{role.spec?.displayName}</DropdownLabel>
                      <DropdownDescription>{role.spec?.description || 'No description available'}</DropdownDescription>
                    </DropdownItem>
                  ))}
                </DropdownMenu>
              </Dropdown>
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
                      {memberSearchTerm ? 'No members found matching your search' : 'No organization members available to add'}
                    </p>
                  </div>
                ) : (
                  <div className="divide-y divide-zinc-200 dark:divide-zinc-700">
                    {getFilteredExistingMembers().map((member) => (
                      <div key={member.metadata?.id} className="p-3 flex items-center gap-3 hover:bg-zinc-50 dark:hover:bg-zinc-800">
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
                            {member.spec?.displayName || member.metadata?.id}
                          </div>
                          <div className="text-xs text-zinc-500 dark:text-zinc-400 truncate">
                            {member.metadata?.email || "Invalid email"}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
          </div>
        ) : !loadingRoles ? (

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
                    id="add-canvas-members-upload"
                  />

                  <label
                    htmlFor="add-canvas-members-upload"
                    className="inline-flex items-center gap-2 px-3 py-2 text-sm font-medium text-zinc-700 dark:text-zinc-300 bg-white dark:bg-zinc-800 border border-zinc-300 dark:border-zinc-600 rounded-md hover:bg-zinc-50 dark:hover:bg-zinc-700 cursor-pointer"
                  >
                    <MaterialSymbol name="folder_open" size="sm" />
                    Browse
                  </label>
                </div>
              </div>
              <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
                The CSV file should have an "email" column with email addresses.
              </Text>
            </Field>

            {/* Role Selection */}
            <Field>
              <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Role
              </Label>
              <Dropdown>
                <DropdownButton outline className="flex items-center gap-2 text-sm justify-between">
                  {bulkUserRole ? canvasRoles.find(r => r.metadata?.name === bulkUserRole)?.spec?.displayName || bulkUserRole : 'Select Role'}
                  <MaterialSymbol name="keyboard_arrow_down" />
                </DropdownButton>
                <DropdownMenu>
                  {canvasRoles.map((role) => (
                    <DropdownItem key={role.metadata?.name} onClick={() => setBulkUserRole(role.metadata?.name || '')}>
                      <DropdownLabel>{role.spec?.displayName}</DropdownLabel>
                      <DropdownDescription>{role.spec?.description || 'No description available'}</DropdownDescription>
                    </DropdownItem>
                  ))}
                </DropdownMenu>
              </Dropdown>
            </Field>

            {/* Upload Button */}
            <div className="flex justify-end">
              <Button
                color="blue"
                onClick={handleBulkAddSubmit}
                disabled={!uploadFile || isInviting || !bulkUserRole}
                className="flex items-center gap-2"
              >
                <MaterialSymbol name="add" size="sm" />
                {isInviting ? 'Processing...' : 'Add to Canvas'}
              </Button>
            </div>
          </div>
        ) : null}
      </div>
    )
  })

AddCanvasMembersSectionComponent.displayName = 'AddCanvasMembersSection'

export const AddCanvasMembersSection = AddCanvasMembersSectionComponent