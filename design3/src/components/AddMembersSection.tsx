import { useState } from 'react'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { Textarea } from './lib/Textarea/textarea'
import { Field, Label } from './lib/Fieldset/fieldset'
import { Tabs, type Tab } from './lib/Tabs/tabs'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from './lib/Dropdown/dropdown'
import { Link } from './lib/Link/link'

interface AddMembersSectionProps {
  className?: string
  showRoleSelection?: boolean
  onAddMembers?: (emails: string, role: string) => number
}

export function AddMembersSection({ className, showRoleSelection = true, onAddMembers }: AddMembersSectionProps) {
  // Add members tabs state
  const [addMembersTab, setAddMembersTab] = useState<'emails' | 'upload'>('emails')
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [bulkUserRole, setBulkUserRole] = useState('Member')
  const [emailRole, setEmailRole] = useState('Member')
  const [emailInput, setEmailInput] = useState('')
  
  // Add members tabs configuration
  const addMembersTabs: Tab[] = [
    { id: 'emails', label: 'By emails' },
    { id: 'upload', label: 'By file' }
  ]

  const handleFileUpload = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0]
    if (file) {
      setUploadFile(file)
    }
  }

  const handleBulkAddSubmit = () => {
    if (uploadFile) {
      // In a real app, you would parse the Excel file here
      console.log('Processing Excel file:', uploadFile.name)
      
      // For demo purposes, simulate processing
      const emailsToAdd = ['example1@company.com', 'example2@company.com', 'example3@company.com']
      const roleToAssign = showRoleSelection ? bulkUserRole : 'Member'
      
      console.log('Adding users:', {
        fileName: uploadFile.name,
        emails: emailsToAdd,
        role: roleToAssign,
        count: emailsToAdd.length
      })
      
      // Reset form and switch back to emails tab
      setUploadFile(null)
      if (showRoleSelection) {
        setBulkUserRole('Member')
      }
      setAddMembersTab('emails')
      
      // TODO: Add API call to actually process the Excel file and add users
      alert(`Successfully processed ${emailsToAdd.length} email addresses from ${uploadFile.name} for invitation with role: ${roleToAssign}.`)
    }
  }

  const handleEmailsSubmit = () => {
    if (!emailInput.trim() || !onAddMembers) return
    
    const roleToAssign = showRoleSelection ? emailRole : 'Member'
    const addedCount = onAddMembers(emailInput, roleToAssign)
    
    if (addedCount > 0) {
      // Reset form
      setEmailInput('')
      if (showRoleSelection) {
        setEmailRole('Member')
      }
    }
  }

  return (
    <div className={`bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 ${className}`}>
      <div className="flex items-center justify-between mb-4">
        <div>
          <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">
            Add members
          </Subheading>
        </div>
      </div>
      
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
      {addMembersTab === 'emails' ? (
        // Enter emails tab content
        <div className="space-y-4">
          <div className="flex items-start gap-3">
            <Textarea
              rows={1}
              placeholder="Email addresses, separated by commas"
              className="flex-1"
              value={emailInput}
              onChange={(e) => setEmailInput(e.target.value)}
            />
            
            {showRoleSelection && (
              <Dropdown>
                <DropdownButton outline className="flex items-center gap-2 text-sm">
                  {emailRole}
                  <MaterialSymbol name="expand_more" size="md" />
                </DropdownButton>
                <DropdownMenu anchor="bottom end">
                  <DropdownItem onClick={() => setEmailRole('Owner')}>
                    <DropdownLabel>Owner</DropdownLabel>
                    <DropdownDescription>Full access to organization settings</DropdownDescription>
                  </DropdownItem>
                  <DropdownItem onClick={() => setEmailRole('Admin')}>
                    <DropdownLabel>Admin</DropdownLabel>
                    <DropdownDescription>Can manage members and organization settings</DropdownDescription>
                  </DropdownItem>
                  <DropdownItem onClick={() => setEmailRole('Member')}>
                    <DropdownLabel>Member</DropdownLabel>
                    <DropdownDescription>Standard member access</DropdownDescription>
                  </DropdownItem>
                </DropdownMenu>
              </Dropdown>
            )}
            
            <Button color="blue" className='flex items-center text-sm gap-2' onClick={handleEmailsSubmit}>
              <MaterialSymbol name="add" size="sm" />
              Invite
            </Button>
          </div>
        </div>
      ) : (
        // Upload file tab content
        <div className="space-y-6">
          {/* File Upload */}
          <Field>
            <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
              Excel Spreadsheet *
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
                
                <Button outline className='flex items-center text-sm gap-2'>
                <MaterialSymbol name="folder_open" size="sm" />
                Browse
                </Button> 
                
              </div>
              
            </div>
            <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-2">
                  
                  <Link href="#" className="text-blue-600 hover:text-blue-700 ml-1">Check .csv file format requirements</Link>
                </Text>
          </Field>

          {/* Role Selection */}
          {showRoleSelection && (
            <Field>
              <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Role
              </Label>
              <Dropdown>
                <DropdownButton outline className="flex items-center gap-2 text-sm justify-between">
                  {bulkUserRole}
                  <MaterialSymbol name="keyboard_arrow_down" />
                </DropdownButton>
                <DropdownMenu>
                  <DropdownItem onClick={() => setBulkUserRole('Owner')}>
                    <DropdownLabel>Owner</DropdownLabel>
                    <DropdownDescription>Full access to organization settings</DropdownDescription>
                  </DropdownItem>
                  <DropdownItem onClick={() => setBulkUserRole('Admin')}>
                    <DropdownLabel>Admin</DropdownLabel>
                    <DropdownDescription>Can manage members and organization settings</DropdownDescription>
                  </DropdownItem>
                  <DropdownItem onClick={() => setBulkUserRole('Member')}>
                    <DropdownLabel>Member</DropdownLabel>
                    <DropdownDescription>Standard member access</DropdownDescription>
                  </DropdownItem>
                </DropdownMenu>
              </Dropdown>
            </Field>
          )}
          
          {/* Upload Button */}
          <div className="flex justify-end">
            <Button 
              color="blue" 
              onClick={handleBulkAddSubmit} 
              disabled={!uploadFile}
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="add" size="sm" />
              Invite
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}