import { useState, useEffect } from 'react'
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
import { 
  authorizationListOrganizationGroups,
  authorizationAddUserToOrganizationGroup 
} from '../../../api-client/sdk.gen'
import { AuthorizationGroup } from '../../../api-client/types.gen'

interface AddMembersSectionProps {
  showRoleSelection?: boolean
  organizationId: string
  onMemberAdded?: () => void
}

export function AddMembersSection({ showRoleSelection = true, organizationId, onMemberAdded }: AddMembersSectionProps) {
  const [newMemberEmail, setNewMemberEmail] = useState('')
  const [selectedRole, setSelectedRole] = useState('Member')
  const [selectedGroup, setSelectedGroup] = useState('')
  const [groups, setGroups] = useState<AuthorizationGroup[]>([])
  const [isInviting, setIsInviting] = useState(false)
  const [loadingGroups, setLoadingGroups] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchGroups = async () => {
      try {
        setLoadingGroups(true)
        setError(null)
        const response = await authorizationListOrganizationGroups({
          query: { organizationId }
        })
        if (response.data?.groups) {
          setGroups(response.data.groups)
          if (response.data.groups.length > 0) {
            setSelectedGroup(response.data.groups[0].name || '')
          }
        }
      } catch (err) {
        console.error('Error fetching groups:', err)
        setError('Failed to fetch groups')
      } finally {
        setLoadingGroups(false)
      }
    }

    fetchGroups()
  }, [organizationId])

  const handleSendInvitation = async () => {
    if (!newMemberEmail.trim() || !selectedGroup) return
    
    setIsInviting(true)
    setError(null)
    
    try {
      // Note: In a real implementation, you would typically:
      // 1. First create/invite the user to get their user ID
      // 2. Then add them to the group
      // For now, we'll simulate with a placeholder user ID
      const userId = newMemberEmail // This would be replaced with actual user ID from user creation/invitation
      
      await authorizationAddUserToOrganizationGroup({
        path: { groupName: selectedGroup },
        body: {
          organizationId,
          userId
        }
      })
      
      console.log('Successfully added user to group:', newMemberEmail, 'to', selectedGroup, 'with role', selectedRole)
      setNewMemberEmail('')
      onMemberAdded?.()
    } catch (err) {
      console.error('Error adding member:', err)
      setError('Failed to add member. Please try again.')
    } finally {
      setIsInviting(false)
    }
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
            Add people to your organization by assigning them to a group
          </Text>
        </div>
      </div>
      
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          <p className="text-sm">{error}</p>
        </div>
      )}
      
      {loadingGroups ? (
        <div className="flex justify-center items-center h-20">
          <p className="text-zinc-500 dark:text-zinc-400">Loading groups...</p>
        </div>
      ) : groups.length === 0 ? (
        <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
          <div className="flex">
            <MaterialSymbol name="warning" className="h-5 w-5 text-yellow-600 dark:text-yellow-400 mr-3 mt-0.5" />
            <div className="text-sm">
              <p className="text-yellow-800 dark:text-yellow-200 font-medium">
                No groups available
              </p>
              <p className="text-yellow-700 dark:text-yellow-300 mt-1">
                Create a group first to add members to your organization.
              </p>
            </div>
          </div>
        </div>
      ) : (
        <div className="space-y-3">
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
            
            <div className="min-w-[140px]">
              <Dropdown>
                <DropdownButton outline className="flex items-center gap-2 text-sm w-full justify-between">
                  {selectedGroup || 'Select Group'}
                  <MaterialSymbol name="keyboard_arrow_down" />
                </DropdownButton>
                <DropdownMenu>
                  {groups.map((group, index) => (
                    <DropdownItem key={index} onClick={() => setSelectedGroup(group.name || '')}>
                      <DropdownLabel>{group.name}</DropdownLabel>
                      <DropdownDescription>Role: {group.role || 'No role assigned'}</DropdownDescription>
                    </DropdownItem>
                  ))}
                </DropdownMenu>
              </Dropdown>
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
              disabled={!newMemberEmail.trim() || !selectedGroup || isInviting}
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="send" size="sm" />
              {isInviting ? 'Adding...' : 'Add to Group'}
            </Button>
          </div>
          
          <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
            <div className="flex">
              <MaterialSymbol name="info" className="h-4 w-4 text-blue-600 dark:text-blue-400 mr-2 mt-0.5" />
              <p className="text-xs text-blue-800 dark:text-blue-200">
                <strong>Note:</strong> This is a simplified implementation. In a production system, 
                you would first invite the user via email, then add them to the selected group once they accept.
              </p>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}