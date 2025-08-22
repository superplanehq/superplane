import { useState } from 'react'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Subheading } from './lib/Heading/heading'
import { Button } from './lib/Button/button'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from './lib/Dropdown/dropdown'
import { MultiCombobox, MultiComboboxLabel } from './lib/Combobox/multi-combobox'
import { Avatar } from './lib/Avatar/avatar'
import { Table, TableHead, TableBody, TableRow, TableHeader, TableCell } from './lib/Table/table'
import { Badge } from './lib/Badge/badge'
import { Link } from './lib/Link/link'

interface User {
  id: string
  username: string
  name: string
  avatar?: string | null
  email: string
}

interface PendingInvitation {
  id: string
  name: string
  email: string
  status: 'Pending' | 'Invited'
  invitedDate: string
  initials: string
  avatar?: string | null
}


interface AddMembersSectionSimpleProps {
  className?: string
  showRoleSelection?: boolean
  onAddMembers?: (users: User[], role: string) => void
  users?: User[]
}

export function AddMembersSectionSimple({ className, showRoleSelection = true, onAddMembers, users = [] }: AddMembersSectionSimpleProps) {
  const [selectedUsers, setSelectedUsers] = useState<User[]>([])
  const [userRole, setUserRole] = useState('Member')
  const [showSuccessMessage, setShowSuccessMessage] = useState(false)
  const [invitedEmails, setInvitedEmails] = useState<string[]>([])
  const [pendingInvitations, setPendingInvitations] = useState<PendingInvitation[]>([])

  // Email validation regex - more comprehensive
  const emailRegex = /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/

  // Function to validate email format
  const isValidEmail = (email: string): boolean => {
    const trimmed = email.trim()
    if (trimmed.length === 0) return false
    if (trimmed.length > 254) return false // RFC 5321 limit
    
    // Check for basic structure
    if (!trimmed.includes('@')) return false
    if (trimmed.indexOf('@') !== trimmed.lastIndexOf('@')) return false // Only one @
    
    // Check that domain has at least one dot
    const [localPart, domain] = trimmed.split('@')
    if (!localPart || !domain) return false
    if (!domain.includes('.')) return false
    if (domain.endsWith('.') || domain.startsWith('.')) return false
    
    return emailRegex.test(trimmed)
  }

  // Function to create a user object from email
  const createUserFromEmail = (email: string): User => {
    const trimmedEmail = email.trim()
    return {
      id: `email_${trimmedEmail}`,
      username: trimmedEmail.split('@')[0],
      name: trimmedEmail,
      email: trimmedEmail,
      avatar: null
    }
  }

  // Mock users data - in real app this would come from props or API
  const mockUsers: User[] = users.length > 0 ? users : [
    { id: '1', username: 'john_doe', name: 'John Doe', email: 'john@company.com', avatar: null },
    { id: '2', username: 'jane_smith', name: 'Jane Smith', email: 'jane@company.com', avatar: null },
    { id: '3', username: 'mike_wilson', name: 'Mike Wilson', email: 'mike@company.com', avatar: null },
    { id: '4', username: 'sarah_jones', name: 'Sarah Jones', email: 'sarah@company.com', avatar: null },
    { id: '5', username: 'pedro_leao', name: 'Pedro Foresti Leao', email: 'pforesti@semaphore.io', avatar: null },
    { id: '6', username: 'aleksandar_m', name: 'Aleksandar Mitrovic', email: 'amitrovic@renderedtext.com', avatar: null },
    { id: '7', username: 'svetlana_cs', name: 'Svetlana Cosovic Stajic', email: 'scosovic@renderedtext.com', avatar: null },
    { id: '8', username: 'tijana_b', name: 'Tijana Banovic', email: 'tbanovic@renderedtext.com', avatar: null },
  ]


  const handleAddUsers = () => {
    if (selectedUsers.length === 0) return
    
    const roleToAssign = showRoleSelection ? userRole : 'Member'
    
    // Separate users from list vs custom emails
    const usersFromList = selectedUsers.filter(user => 
      mockUsers.some(mockUser => mockUser.id === user.id)
    )
    
    const customUsers = selectedUsers.filter(user => 
      !mockUsers.some(mockUser => mockUser.id === user.id)
    )

    // Add custom users to pending invitations
    if (customUsers.length > 0) {
      const newPendingInvitations: PendingInvitation[] = customUsers.map((user, index) => ({
        id: user.id.startsWith('email_') || user.id.startsWith('custom_') 
          ? `pending-${Date.now()}-${index}` 
          : user.id,
        name: user.name,
        email: user.email,
        status: 'Pending' as const,
        invitedDate: new Date().toISOString().split('T')[0],
        initials: user.name.split(' ').map((n: string) => n[0]).join('').toUpperCase(),
        avatar: user.avatar || null
      }))

      setPendingInvitations(prev => [...prev, ...newPendingInvitations])
    }

    // Call external handler only for users from the predefined list
    if (usersFromList.length > 0) {
      onAddMembers?.(usersFromList, roleToAssign)
    }
    
    // Extract email addresses from selected users for success message
    const emailAddresses = selectedUsers.map(user => user.email)
    setInvitedEmails(emailAddresses)
    
    // Show success message
    setShowSuccessMessage(true)
    
    // Hide success message after 5 seconds
    setTimeout(() => {
      setShowSuccessMessage(false)
      setInvitedEmails([])
    }, 5000)
    
    // Reset form
    setSelectedUsers([])
    if (showRoleSelection) {
      setUserRole('Member')
    }
  }

  // Handle sending invite to a specific member
  const handleSendInvite = (member: PendingInvitation) => {
    // Update the member status to "Invited"
    setPendingInvitations(prev => 
      prev.map(m => 
        m.id === member.id ? { ...m, status: 'Invited' as const } : m
      )
    )
    
    // In a real app, this would send an API call to invite the member
    console.log('Sending invite to:', member.email)
  }

  // Handle inviting all pending members
  const handleInviteAll = () => {
    const pendingMembers = pendingInvitations.filter(m => m.status === 'Pending')
    
    if (pendingMembers.length === 0) return

    // Update all pending members to "Invited" status
    setPendingInvitations(prev => 
      prev.map(m => 
        m.status === 'Pending' ? { ...m, status: 'Invited' as const } : m
      )
    )
    
    console.log(`Inviting ${pendingMembers.length} members`)
  }

  return (
    <div>
    <div className={`bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 ${className}`}>
      <div className="flex items-center justify-between mb-4">
        <div>
          <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">
            Add members
          </Subheading>
        </div>
      </div>
      
      <div className="space-y-4">
        <div className="flex items-start gap-3 w-full">
          <div className="flex-grow-1 w-full relative">
              <MultiCombobox
                options={mockUsers}
                displayValue={(user) => user.name}
                placeholder="Search users or enter email"
                value={selectedUsers}
                onChange={setSelectedUsers}
                className="w-full"
                allowCustomValues={true}
                createCustomValue={(query) => {
                  const trimmedQuery = query.trim()
                  // Since validateInput ensures only valid emails reach here,
                  // we can safely create a user from the email
                  return createUserFromEmail(trimmedQuery)
                }}
                validateValue={(user) => {
                  // Validate if the user's email is valid
                  return isValidEmail(user.email)
                }}
                validateInput={(input) => {
                  // Only allow valid email addresses to be added as tags
                  const trimmed = input.trim()
                  if (trimmed === '') return false
                  
                  // Must be a complete, valid email address
                  return isValidEmail(trimmed)
                }}
                filter={(user, query) => {
                  return user?.username?.toLowerCase().includes(query.toLowerCase()) ||
                         user?.name?.toLowerCase().includes(query.toLowerCase()) ||
                         user?.email?.toLowerCase().includes(query.toLowerCase()) || false
                }}
              >
            {(user, isSelected) => {
              // Check if this is a custom email suggestion (created from user input)
              const isCustomEmailSuggestion = user.id.startsWith('email_') || user.id.startsWith('custom_')
              
              return (
                <div className='group w-full flex items-center gap-2'>
                  {isCustomEmailSuggestion ? (
                    // For custom emails, always show mail icon (both in dropdown and as tags)
                    <div className="flex items-center justify-center size-6 bg-zinc-100 dark:bg-zinc-800 rounded-full">
                      <MaterialSymbol name="mail" size="sm" className="text-zinc-600 dark:text-zinc-400" />
                    </div>
                  ) : (
                    // For users from predefined list, show avatar
                    <Avatar 
                      src={user.avatar} 
                      initials={user.name ? user.name.split(' ').map(n => n[0]).join('') : ''}
                      className="size-6"
                    />
                  )}
                  <MultiComboboxLabel className="flex flex-col">
                    {isSelected ? (
                      // For tags, show different content based on type
                      <span className="font-medium">
                        {isCustomEmailSuggestion ? user.email : (user.name || 'Unknown')}
                      </span>
                    ) : (
                      // For dropdown options, show full info
                      <>
                        {isCustomEmailSuggestion ? (
                          <>
                            <span className="font-medium">{user.email}</span>
                            <span className="text-sm text-zinc-600 dark:text-zinc-400 group-hover:text-white">
                              Invite to organization
                            </span>
                          </>
                        ) : (
                          <>
                            <span className="font-medium">{user.name || 'Unknown'}</span>
                            <span className="text-sm text-zinc-600 dark:text-zinc-400 group-hover:text-white">@{user.username || 'unknown'}</span>
                          </>
                        )}
                      </>
                    )}
                  </MultiComboboxLabel>
                </div>
              )
            }}
          </MultiCombobox>
          </div>
          
          {showRoleSelection && (
            <Dropdown>
              <DropdownButton outline className="flex items-center gap-2 text-sm">
                {userRole}
                <MaterialSymbol name="expand_more" size="md" />
              </DropdownButton>
              <DropdownMenu anchor="bottom end">
                <DropdownItem onClick={() => setUserRole('Owner')}>
                  <DropdownLabel>Owner</DropdownLabel>
                  <DropdownDescription>Full access to organization settings</DropdownDescription>
                </DropdownItem>
                <DropdownItem onClick={() => setUserRole('Admin')}>
                  <DropdownLabel>Admin</DropdownLabel>
                  <DropdownDescription>Can manage members and organization settings</DropdownDescription>
                </DropdownItem>
                <DropdownItem onClick={() => setUserRole('Member')}>
                  <DropdownLabel>Member</DropdownLabel>
                  <DropdownDescription>Standard member access</DropdownDescription>
                </DropdownItem>
              </DropdownMenu>
            </Dropdown>
          )}
          
          <Button 
            color="blue" 
            className='flex items-center text-sm gap-2 whitespace-nowrap' 
            onClick={handleAddUsers}
            disabled={selectedUsers.length === 0}
          >
            <MaterialSymbol name="add" size="sm" />
            Invite {selectedUsers.length > 0 && `(${selectedUsers.length})`}
          </Button>
        </div>

        

      </div>
      
    </div>
    {/* Pending Invitations Table */}
    {pendingInvitations.length > 0 && (
      <div className="mt-6 bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4">
          <div className="flex items-center justify-between">
            <div>
              <h4 className="text-sm font-medium text-zinc-900 dark:text-white">Pending Invitations</h4>
              <p className="text-sm text-zinc-500 dark:text-zinc-400">We identified users that are not part of the organization and should be added before adding to the group</p>
            </div>
            {pendingInvitations.length > 1 && (
              <Link
                href="#"
                onClick={handleInviteAll}
                className="flex items-center gap-2 text-sm text-blue-600 hover:text-blue-700"
              >
                Invite All
              </Link>
            )}
          </div>
        </div>
        <div className="px-6 pb-6">
          <Table dense>
            <TableHead>
              <TableRow>
                <TableHeader>Name</TableHeader>
                <TableHeader>Email</TableHeader>
                <TableHeader>Role</TableHeader>
                <TableHeader>Status</TableHeader>
                <TableHeader></TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {pendingInvitations.map((member) => (
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
                          {member.email}
                        </div>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    {member.email}
                  </TableCell>
                  <TableCell>
                    <Dropdown>
                      <DropdownButton disabled outline className="flex items-center gap-2 text-sm">
                        Member
                        <MaterialSymbol name="expand_more" size="md" />
                      </DropdownButton>
                      <DropdownMenu anchor="bottom end">
                        <DropdownItem onClick={() => setUserRole('Owner')}>
                          <DropdownLabel>Owner</DropdownLabel>
                          <DropdownDescription>Full access to organization settings</DropdownDescription>
                        </DropdownItem>
                        <DropdownItem onClick={() => setUserRole('Admin')}>
                          <DropdownLabel>Admin</DropdownLabel>
                          <DropdownDescription>Can manage members and organization settings</DropdownDescription>
                        </DropdownItem>
                        <DropdownItem onClick={() => setUserRole('Member')}>
                          <DropdownLabel>Member</DropdownLabel>
                          <DropdownDescription>Standard member access</DropdownDescription>
                        </DropdownItem>
                      </DropdownMenu>
                    </Dropdown>
                  </TableCell>
                  <TableCell>
                    <Badge 
                      color={member.status === 'Invited' ? 'amber' : 'green'} 
                      className="!text-xs"
                    >
                      {member.status}
                    </Badge>
                  </TableCell>
                  
                  <TableCell className='text-right'>
                    <Button
                      color="white"
                      onClick={() => handleSendInvite(member)}
                      disabled={member.status === 'Invited'}
                    >
                      {member.status === 'Invited' ? 'Pending invitation' : 'Send Invite'}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      </div>
    )}
    </div>
  )
}