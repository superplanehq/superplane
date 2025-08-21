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

interface User {
  id: string
  username: string
  name: string
  avatar?: string | null
  email: string
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
    if (selectedUsers.length === 0 || !onAddMembers) return
    
    const roleToAssign = showRoleSelection ? userRole : 'Member'
    onAddMembers(selectedUsers, roleToAssign)
    
    // Reset form
    setSelectedUsers([])
    if (showRoleSelection) {
      setUserRole('Member')
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
      
      <div className="space-y-4">
        <div className="flex items-start gap-3 w-full">
          <MultiCombobox
            options={mockUsers}
            displayValue={(user) => user.name}
            placeholder="Search users"
            value={selectedUsers}
            onChange={setSelectedUsers}
            className="flex-grow-1 w-full"
            filter={(user, query) => 
              user?.username?.toLowerCase().includes(query.toLowerCase()) ||
              user?.name?.toLowerCase().includes(query.toLowerCase()) ||
              user?.email?.toLowerCase().includes(query.toLowerCase()) || false
            }
          >
            {(user, isSelected) => (
              <div className='group w-full flex items-center gap-2'>
                <Avatar 
                  src={user.avatar} 
                  initials={user.name ? user.name.split(' ').map(n => n[0]).join('') : ''}
                  className="size-6"
                />
                <MultiComboboxLabel className="flex flex-col">
                  {isSelected ? (
                    // For tags, show just the name
                    <span className="font-medium">{user.name || 'Unknown'}</span>
                  ) : (
                    // For dropdown options, show full info
                    <>
                      <span className="font-medium">{user.name || 'Unknown'}</span>
                      <span className="text-sm text-zinc-600 dark:text-zinc-400 group-hover:text-white">@{user.username || 'unknown'}</span>
                    </>
                  )}
                </MultiComboboxLabel>
              </div>
            )}
          </MultiCombobox>
          
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
            className='flex items-center text-sm gap-2' 
            onClick={handleAddUsers}
            disabled={selectedUsers.length === 0}
          >
            <MaterialSymbol name="add" size="sm" />
            Invite {selectedUsers.length > 0 && `(${selectedUsers.length})`}
          </Button>
        </div>
      </div>
    </div>
  )
}