import { useState } from 'react'
import { Button } from '../../../components/Button/button'
import { Input } from '../../../components/Input/input'
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

interface AddMembersSectionProps {
  showRoleSelection?: boolean
}

export function AddMembersSection({ showRoleSelection = true }: AddMembersSectionProps) {
  const [newMemberEmail, setNewMemberEmail] = useState('')
  const [selectedRole, setSelectedRole] = useState('Member')
  const [isInviting, setIsInviting] = useState(false)

  const handleSendInvitation = async () => {
    if (!newMemberEmail.trim()) return
    
    setIsInviting(true)
    // Simulate API call
    setTimeout(() => {
      console.log('Inviting:', newMemberEmail, 'as', selectedRole)
      setNewMemberEmail('')
      setIsInviting(false)
    }, 1000)
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      handleSendInvitation()
    }
  }

  return (
    <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
      <div className="flex items-center justify-between mb-4">
        <div>
          <Text className="font-medium text-zinc-900 dark:text-white">
            Invite new members
          </Text>
          <Text className="text-sm text-zinc-600 dark:text-zinc-400">
            Add people to your organization by sending them an invitation
          </Text>
        </div>
      </div>
      
      <div className="flex gap-3">
        <div className="flex-1">
          <Input
            type="email"
            placeholder="Enter email address"
            value={newMemberEmail}
            onChange={(e) => setNewMemberEmail(e.target.value)}
            onKeyPress={handleKeyPress}
            className="w-full"
          />
        </div>
        
        {showRoleSelection && (
          <div className="min-w-[140px]">
            <Dropdown>
              <DropdownButton outline className="flex items-center gap-2 text-sm w-full justify-between">
                {selectedRole}
                <MaterialSymbol name="keyboard_arrow_down" />
              </DropdownButton>
              <DropdownMenu>
                <DropdownItem onClick={() => setSelectedRole('Owner')}>
                  <DropdownLabel>Owner</DropdownLabel>
                  <DropdownDescription>Full access to organization settings</DropdownDescription>
                </DropdownItem>
                <DropdownItem onClick={() => setSelectedRole('Admin')}>
                  <DropdownLabel>Admin</DropdownLabel>
                  <DropdownDescription>Can manage members and organization settings</DropdownDescription>
                </DropdownItem>
                <DropdownItem onClick={() => setSelectedRole('Member')}>
                  <DropdownLabel>Member</DropdownLabel>
                  <DropdownDescription>Standard member access</DropdownDescription>
                </DropdownItem>
              </DropdownMenu>
            </Dropdown>
          </div>
        )}
        
        <Button 
          color="blue" 
          onClick={handleSendInvitation}
          disabled={!newMemberEmail.trim() || isInviting}
          className="flex items-center gap-2"
        >
          <MaterialSymbol name="send" size="sm" />
          {isInviting ? 'Sending...' : 'Send Invitation'}
        </Button>
      </div>
    </div>
  )
}