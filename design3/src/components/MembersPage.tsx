import { useState } from 'react'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Avatar } from './lib/Avatar/avatar'
import { Input, InputGroup } from './lib/Input/input'
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem, DropdownLabel, DropdownDescription } from './lib/Dropdown/dropdown'
import { Dialog, DialogTitle, DialogDescription, DialogBody, DialogActions } from './lib/Dialog/dialog'
import { Checkbox } from './lib/Checkbox/checkbox'
import clsx from 'clsx'

interface Member {
  id: string
  name: string
  email: string
  role: string
  status: 'active' | 'pending' | 'inactive'
  lastActive: string
  avatar?: string
  initials: string
}

interface Group {
  id: string
  name: string
  description: string
  memberCount: number
  permissions: string[]
}

interface Role {
  id: string
  name: string
  description: string
  permissions: string[]
  memberCount: number
}

export function MembersPage() {
  const [activeTab, setActiveTab] = useState<'members' | 'groups' | 'roles'>('members')
  const [searchQuery, setSearchQuery] = useState('')
  const [isInviteModalOpen, setIsInviteModalOpen] = useState(false)
  const [isGroupModalOpen, setIsGroupModalOpen] = useState(false)
  const [isRoleModalOpen, setIsRoleModalOpen] = useState(false)
  const [newGroupName, setNewGroupName] = useState('')
  const [newGroupDescription, setNewGroupDescription] = useState('')
  const [selectedGroupMembers, setSelectedGroupMembers] = useState<string[]>([])

  // Mock data
  const members: Member[] = [
    {
      id: '1',
      name: 'John Doe',
      email: 'john@acme.com',
      role: 'Owner',
      status: 'active',
      lastActive: '2 hours ago',
      initials: 'JD',
      avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face'
    },
    {
      id: '2',
      name: 'Jane Smith',
      email: 'jane@acme.com',
      role: 'Admin',
      status: 'active',
      lastActive: '1 day ago',
      initials: 'JS'
    },
    {
      id: '3',
      name: 'Bob Wilson',
      email: 'bob@acme.com',
      role: 'Member',
      status: 'pending',
      lastActive: 'Never',
      initials: 'BW'
    }
  ]

  const groups: Group[] = [
    {
      id: '1',
      name: 'Administrators',
      description: 'Full access to all organization features',
      memberCount: 2,
      permissions: ['org_manage', 'workflow_create', 'workflow_edit', 'member_invite']
    },
    {
      id: '2',
      name: 'Workflow Editors',
      description: 'Can create and edit workflows',
      memberCount: 5,
      permissions: ['workflow_create', 'workflow_edit', 'workflow_view']
    },
    {
      id: '3',
      name: 'Viewers',
      description: 'Read-only access to workflows',
      memberCount: 12,
      permissions: ['workflow_view']
    }
  ]

  const roles: Role[] = [
    {
      id: '1',
      name: 'Owner',
      description: 'Full access to all organization features and settings',
      permissions: ['*'],
      memberCount: 1
    },
    {
      id: '2',
      name: 'Admin',
      description: 'Can manage workflows, members, and most settings',
      permissions: ['org_manage', 'workflow_manage', 'member_manage'],
      memberCount: 3
    },
    {
      id: '3',
      name: 'Member',
      description: 'Basic access to assigned workflows',
      permissions: ['workflow_view', 'workflow_execute'],
      memberCount: 8
    }
  ]

  const filteredMembers = members.filter(member =>
    member.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    member.email.toLowerCase().includes(searchQuery.toLowerCase()) ||
    member.role.toLowerCase().includes(searchQuery.toLowerCase())
  )

  const handleInviteMember = () => {
    setIsInviteModalOpen(true)
  }

  const handleCreateGroup = () => {
    setIsGroupModalOpen(true)
  }

  const handleCreateRole = () => {
    setIsRoleModalOpen(true)
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'active':
        return 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
      case 'pending':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
      case 'inactive':
        return 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400'
      default:
        return 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/20 dark:text-zinc-400'
    }
  }

  const tabs = [
    { id: 'members', label: 'Members', count: members.length },
    { id: 'groups', label: 'Groups', count: groups.length },
    { id: 'roles', label: 'Roles', count: roles.length }
  ]

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <Subheading level={1} className="mb-2">Team Management</Subheading>
          <Text className="text-zinc-600 dark:text-zinc-400">
            Manage members, groups, and roles for your organization
          </Text>
        </div>
        <div className="flex items-center gap-3">
          {activeTab === 'members' && (
            <Button color="blue" onClick={handleInviteMember}>
              <MaterialSymbol name="person_add" />
              Invite Member
            </Button>
          )}
          {activeTab === 'groups' && (
            <Button color="blue" onClick={handleCreateGroup}>
              <MaterialSymbol name="add" />
              Create Group
            </Button>
          )}
          {activeTab === 'roles' && (
            <Button color="blue" onClick={handleCreateRole}>
              <MaterialSymbol name="add" />
              Create Role
            </Button>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-zinc-200 dark:border-zinc-700">
        <nav className="flex space-x-8">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id as 'members' | 'groups' | 'roles')}
              className={clsx(
                'flex items-center gap-2 py-2 px-1 border-b-2 font-medium text-sm transition-colors',
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                  : 'border-transparent text-zinc-500 hover:text-zinc-700 dark:text-zinc-400 dark:hover:text-zinc-300'
              )}
            >
              {tab.label}
              <span className={clsx(
                'px-2 py-1 rounded-full text-xs',
                activeTab === tab.id
                  ? 'bg-blue-100 text-blue-600 dark:bg-blue-900/20 dark:text-blue-400'
                  : 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-400'
              )}>
                {tab.count}
              </span>
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      {activeTab === 'members' && (
        <div className="space-y-6">
          {/* Search */}
          <div className="max-w-md">
            <InputGroup>
              <MaterialSymbol name="search" data-slot="icon" className="absolute left-3 top-1/2 transform -translate-y-1/2 text-zinc-400" size="sm" />
              <Input
                type="text"
                placeholder="Search members..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10"
              />
            </InputGroup>
          </div>

          {/* Members List */}
          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
            <div className="p-6">
              <div className="space-y-4">
                {filteredMembers.map((member) => (
                  <div key={member.id} className="flex items-center justify-between p-4 border border-zinc-200 dark:border-zinc-700 rounded-lg">
                    <div className="flex items-center gap-4">
                      <Avatar
                        src={member.avatar}
                        initials={member.initials}
                        className="size-10"
                      />
                      <div>
                        <div className="font-medium text-zinc-900 dark:text-white">
                          {member.name}
                        </div>
                        <div className="text-sm text-zinc-600 dark:text-zinc-400">
                          {member.email}
                        </div>
                        <div className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                          Last active: {member.lastActive}
                        </div>
                      </div>
                    </div>
                    <div className="flex items-center gap-4">
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(member.status)}`}>
                        {member.status}
                      </span>
                      <Dropdown>
                        <DropdownButton plain>
                          <span className="px-3 py-1 border border-zinc-300 dark:border-zinc-600 rounded-md text-sm">
                            {member.role}
                          </span>
                          <MaterialSymbol name="expand_more" size="sm" />
                        </DropdownButton>
                        <DropdownMenu>
                          <DropdownItem>Owner</DropdownItem>
                          <DropdownItem>Admin</DropdownItem>
                          <DropdownItem>Member</DropdownItem>
                        </DropdownMenu>
                      </Dropdown>
                      <Dropdown>
                        <DropdownButton plain>
                          <MaterialSymbol name="more_vert" size="sm" />
                        </DropdownButton>
                        <DropdownMenu>
                          <DropdownItem>
                            <MaterialSymbol name="edit" />
                            Edit
                          </DropdownItem>
                          <DropdownItem>
                            <MaterialSymbol name="block" />
                            Suspend
                          </DropdownItem>
                          <DropdownItem>
                            <MaterialSymbol name="delete" />
                            Remove
                          </DropdownItem>
                        </DropdownMenu>
                      </Dropdown>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </div>
      )}

      {activeTab === 'groups' && (
        <div className="space-y-6">
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            {groups.map((group) => (
              <div key={group.id} className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div className="flex items-start justify-between mb-4">
                  <div className="p-2 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                    <MaterialSymbol name="group" className="text-blue-600 dark:text-blue-400" size="lg" />
                  </div>
                  <Dropdown>
                    <DropdownButton plain>
                      <MaterialSymbol name="more_vert" size="sm" />
                    </DropdownButton>
                    <DropdownMenu>
                      <DropdownItem>
                        <MaterialSymbol name="edit" />
                        Edit
                      </DropdownItem>
                      <DropdownItem>
                        <MaterialSymbol name="delete" />
                        Delete
                      </DropdownItem>
                    </DropdownMenu>
                  </Dropdown>
                </div>

                <Subheading level={3} className="mb-2">{group.name}</Subheading>
                <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
                  {group.description}
                </Text>

                <div className="flex items-center justify-between">
                  <Text className="text-sm text-zinc-500 dark:text-zinc-400">
                    {group.memberCount} members
                  </Text>
                  <span className="px-2 py-1 bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400 rounded-full text-xs font-medium">
                    {group.permissions.length} permissions
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {activeTab === 'roles' && (
        <div className="space-y-6">
          <div className="space-y-4">
            {roles.map((role) => (
              <div key={role.id} className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                <div className="flex items-center justify-between mb-4">
                  <div className="flex items-center gap-4">
                    <div className="p-2 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                      <MaterialSymbol name="admin_panel_settings" className="text-purple-600 dark:text-purple-400" size="lg" />
                    </div>
                    <div>
                      <Subheading level={3} className="mb-1">{role.name}</Subheading>
                      <Text className="text-zinc-600 dark:text-zinc-400">
                        {role.description}
                      </Text>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="text-right">
                      <div className="text-sm font-medium text-zinc-900 dark:text-white">
                        {role.memberCount} members
                      </div>
                      <div className="text-xs text-zinc-500 dark:text-zinc-400">
                        {role.permissions.length === 1 && role.permissions[0] === '*' 
                          ? 'All permissions' 
                          : `${role.permissions.length} permissions`
                        }
                      </div>
                    </div>
                    <Dropdown>
                      <DropdownButton plain>
                        <MaterialSymbol name="more_vert" size="sm" />
                      </DropdownButton>
                      <DropdownMenu>
                        <DropdownItem>
                          <MaterialSymbol name="edit" />
                          Edit
                        </DropdownItem>
                        <DropdownItem>
                          <MaterialSymbol name="copy" />
                          Duplicate
                        </DropdownItem>
                        <DropdownItem>
                          <MaterialSymbol name="delete" />
                          Delete
                        </DropdownItem>
                      </DropdownMenu>
                    </Dropdown>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}

      {/* Invite Member Modal */}
      <Dialog open={isInviteModalOpen} onClose={() => setIsInviteModalOpen(false)} size="md">
        <DialogTitle>Invite Team Member</DialogTitle>
        <DialogDescription>
          Send an invitation to add a new member to your organization.
        </DialogDescription>
        <DialogBody>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Email Address *
              </label>
              <Input
                type="email"
                placeholder="Enter email address"
                className="w-full"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Role
              </label>
              <Dropdown>
                <DropdownButton className="w-full justify-between">
                  <span>Select role</span>
                  <MaterialSymbol name="expand_more" size="sm" />
                </DropdownButton>
                <DropdownMenu>
                  <DropdownItem>
                    <div>
                      <DropdownLabel>Admin</DropdownLabel>
                      <DropdownDescription>Can manage workflows and members</DropdownDescription>
                    </div>
                  </DropdownItem>
                  <DropdownItem>
                    <div>
                      <DropdownLabel>Member</DropdownLabel>
                      <DropdownDescription>Basic access to assigned workflows</DropdownDescription>
                    </div>
                  </DropdownItem>
                </DropdownMenu>
              </Dropdown>
            </div>
          </div>
        </DialogBody>
        <DialogActions>
          <Button plain onClick={() => setIsInviteModalOpen(false)}>
            Cancel
          </Button>
          <Button color="blue" onClick={() => setIsInviteModalOpen(false)}>
            Send Invitation
          </Button>
        </DialogActions>
      </Dialog>

      {/* Create Group Modal */}
      <Dialog open={isGroupModalOpen} onClose={() => setIsGroupModalOpen(false)} size="md">
        <DialogTitle>Create New Group</DialogTitle>
        <DialogDescription>
          Create a group to organize team members with similar permissions.
        </DialogDescription>
        <DialogBody>
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Group Name *
              </label>
              <Input
                type="text"
                placeholder="Enter group name"
                value={newGroupName}
                onChange={(e) => setNewGroupName(e.target.value)}
                className="w-full"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Description
              </label>
              <Input
                type="text"
                placeholder="Enter group description"
                value={newGroupDescription}
                onChange={(e) => setNewGroupDescription(e.target.value)}
                className="w-full"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-3">
                Add Members
              </label>
              <div className="space-y-2">
                {members.map((member) => (
                  <div key={member.id} className="flex items-center gap-3">
                    <Checkbox
                      checked={selectedGroupMembers.includes(member.id)}
                      onChange={(checked) => {
                        if (checked) {
                          setSelectedGroupMembers([...selectedGroupMembers, member.id])
                        } else {
                          setSelectedGroupMembers(selectedGroupMembers.filter(id => id !== member.id))
                        }
                      }}
                    />
                    <Avatar src={member.avatar} initials={member.initials} className="size-6" />
                    <div className="flex-1">
                      <div className="text-sm font-medium text-zinc-900 dark:text-white">
                        {member.name}
                      </div>
                      <div className="text-xs text-zinc-500 dark:text-zinc-400">
                        {member.email}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </DialogBody>
        <DialogActions>
          <Button plain onClick={() => setIsGroupModalOpen(false)}>
            Cancel
          </Button>
          <Button color="blue" disabled={!newGroupName.trim()}>
            Create Group
          </Button>
        </DialogActions>
      </Dialog>
    </div>
  )
}