import { useState } from 'react'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Avatar } from './lib/Avatar/avatar'
import { Heading, Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { Input, InputGroup } from './lib/Input/input'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem
} from './lib/Dropdown/dropdown'
import { NavigationOrg } from './lib/Navigation/navigation-org'
import { Breadcrumbs } from './lib/Breadcrumbs/breadcrumbs'
import { Link } from './lib/Link/link'
import { Checkbox, CheckboxField } from './lib/Checkbox/checkbox'
import { Description, Label } from './lib/Fieldset/fieldset'
import { ControlledTabs, Tabs, type Tab } from './lib/Tabs/tabs'

interface OrganizationSettingsProps {
  onBack?: () => void
  onSignOut?: () => void
  onSwitchOrganization?: () => void
}

export function OrganizationSettings({ 
  onBack, 
  onSignOut, 
  onSwitchOrganization 
}: OrganizationSettingsProps) {
  const [activeTab, setActiveTab] = useState<'profile' | 'general' | 'users' | 'teams' | 'roles' | 'tokens' | 'integrations' | 'billing' | 'security'>('roles')
  const [selectedTeam, setSelectedTeam] = useState<{ id: string; name: string; description: string } | null>(null)
  const [isCreatingRole, setIsCreatingRole] = useState(false)
  const [activeRoleTab, setActiveRoleTab] = useState<'organization' | 'canvas'>('organization')
  const [newRoleName, setNewRoleName] = useState('')
  const [newRoleDescription, setNewRoleDescription] = useState('')
  
  // Mock data for organization roles
  const organizationRoles = [
    {
      id: '1',
      name: 'Admin',
      permissions: 8,
      status: 'Active'
    },
    {
      id: '2',
      name: 'Member',
      permissions: 4,
      status: 'Active'
    },
    {
      id: '3',
      name: 'Manager',
      permissions: 6,
      status: 'Active'
    }
  ]

  // Mock data for canvas roles
  const canvasRoles = [
    {
      id: '1',
      name: 'Canvas Editor',
      permissions: 5,
      status: 'Active'
    },
    {
      id: '2',
      name: 'Canvas Viewer',
      permissions: 2,
      status: 'Active'
    },
    {
      id: '3',
      name: 'Canvas Admin',
      permissions: 7,
      status: 'Active'
    }
  ]

  // Mock data for users
  const users = [
    {
      id: '1',
      name: 'John Doe',
      email: 'john@acme.com',
      role: 'Owner',
      status: 'Active',
      lastActive: '2 hours ago',
      initials: 'JD',
      avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face'
    },
    {
      id: '2',
      name: 'Jane Smith',
      email: 'jane@acme.com',
      role: 'Admin',
      status: 'Active',
      lastActive: '1 day ago',
      initials: 'JS'
    },
    {
      id: '3',
      name: 'Bob Wilson',
      email: 'bob@acme.com',
      role: 'Member',
      status: 'Pending',
      lastActive: 'Never',
      initials: 'BW'
    },
    {
      id: '4',
      name: 'Alice Johnson',
      email: 'alice@acme.com',
      role: 'Member',
      status: 'Active',
      lastActive: '3 days ago',
      initials: 'AJ'
    }
  ]

  // Mock data for teams
  const teams = [
    {
      id: '1',
      name: 'Engineering',
      description: 'Software development and technical operations',
      memberCount: 8,
      created: '2 months ago',
      role: 'Admin'
    },
    {
      id: '2',
      name: 'Design',
      description: 'UI/UX design and user research',
      memberCount: 3,
      created: '1 month ago',
      role: 'Member'
    },
    {
      id: '3',
      name: 'Marketing',
      description: 'Marketing campaigns and content creation',
      memberCount: 5,
      created: '3 weeks ago',
      role: 'Member'
    },
    {
      id: '4',
      name: 'DevOps',
      description: 'Infrastructure management and deployment',
      memberCount: 4,
      created: '1 week ago',
      role: 'Admin'
    }
  ]

  // Mock data for team members
  const getTeamMembers = (teamId: string) => {
    const allMembers = [
      {
        id: '1',
        name: 'John Doe',
        email: 'john@acme.com',
        role: 'Team Lead',
        status: 'Active',
        joinedDate: '2024-01-15',
        initials: 'JD',
        avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face'
      },
      {
        id: '2',
        name: 'Jane Smith',
        email: 'jane@acme.com',
        role: 'Senior Developer',
        status: 'Active',
        joinedDate: '2024-02-01',
        initials: 'JS'
      },
      {
        id: '3',
        name: 'Mike Johnson',
        email: 'mike@acme.com',
        role: 'Developer',
        status: 'Active',
        joinedDate: '2024-03-10',
        initials: 'MJ'
      },
      {
        id: '4',
        name: 'Sarah Wilson',
        email: 'sarah@acme.com',
        role: 'Designer',
        status: 'Active',
        joinedDate: '2024-02-20',
        initials: 'SW'
      },
      {
        id: '5',
        name: 'Tom Brown',
        email: 'tom@acme.com',
        role: 'DevOps Engineer',
        status: 'Active',
        joinedDate: '2024-03-01',
        initials: 'TB'
      }
    ]

    // Return different members based on team
    switch (teamId) {
      case '1': // Engineering
        return allMembers.slice(0, 3)
      case '2': // Design
        return [allMembers[3]]
      case '3': // Marketing
        return allMembers.slice(1, 3)
      case '4': // DevOps
        return [allMembers[4], allMembers[0]]
      default:
        return []
    }
  }

  const currentUser = {
    id: '1',
    name: 'John Doe',
    email: 'john@acme.com',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
  }

  const currentOrganization = {
    id: '1',
    name: 'Confluent',
    plan: 'Pro Plan',
    initials: 'C',
  }

  // Navigation handlers
  const handleUserMenuAction = (action: 'profile' | 'settings' | 'signout') => {
    switch (action) {
      case 'profile':
        console.log('Navigating to user profile...')
        break
      case 'settings':
        console.log('Opening account settings...')
        break
      case 'signout':
        onSignOut?.()
        break
    }
  }

  const handleOrganizationMenuAction = (action: 'settings' | 'billing' | 'members') => {
    if (action === 'settings') {
      console.log('Already on organization settings page')
    } else {
      console.log(`Organization action: ${action}`)
    }
  }

  const handleTeamClick = (team: { id: string; name: string; description: string }) => {
    setSelectedTeam(team)
  }

  const handleBackToTeams = () => {
    setSelectedTeam(null)
  }

  const handleCreateRole = () => {
    setIsCreatingRole(true)
  }

  const handleBackToRoles = () => {
    setIsCreatingRole(false)
    setNewRoleName('')
    setNewRoleDescription('')
  }

  const handleSaveRole = () => {
    // Here you would typically save the role to your backend
    console.log('Creating role:', {
      name: newRoleName,
      description: newRoleDescription
    })
    handleBackToRoles()
  }

  // Role tabs configuration
  const roleTabs: Tab[] = [
    {
      id: 'organization',
      label: 'Organization Roles',
      count: organizationRoles.length
    },
    {
      id: 'canvas',
      label: 'Canvas Roles',
      count: canvasRoles.length
    }
  ]

  // Organization permissions data
  const organizationPermissions = [
    {
      id: 'org_view',
      name: 'View Organization',
      description: 'View organization details and basic information'
    },
    {
      id: 'org_manage_settings',
      name: 'Manage Organization Settings',
      description: 'Modify organization settings and configuration'
    },
    {
      id: 'org_manage_members',
      name: 'Manage Members',
      description: 'Invite, remove, and manage organization members'
    },
    {
      id: 'org_manage_teams',
      name: 'Manage Teams',
      description: 'Create, edit, and delete teams within the organization'
    },
    {
      id: 'org_manage_roles',
      name: 'Manage Roles',
      description: 'Create, edit, and assign custom roles'
    },
    {
      id: 'org_manage_billing',
      name: 'Manage Billing',
      description: 'Access and manage billing information and subscriptions'
    },
    {
      id: 'org_view_security',
      name: 'View Security Settings',
      description: 'View security settings and audit logs'
    },
    {
      id: 'org_manage_integrations',
      name: 'Manage Integrations',
      description: 'Configure and manage third-party integrations'
    }
  ]

  // Canvas permissions data
  const canvasPermissions = [
    {
      id: 'canvas_view',
      name: 'View Canvases',
      description: 'View existing canvases and their content'
    },
    {
      id: 'canvas_create',
      name: 'Create Canvases',
      description: 'Create new canvases and projects'
    },
    {
      id: 'canvas_edit',
      name: 'Edit Canvases',
      description: 'Modify canvas content, structure, and properties'
    },
    {
      id: 'canvas_delete',
      name: 'Delete Canvases',
      description: 'Remove canvases and associated data permanently'
    },
    {
      id: 'canvas_share',
      name: 'Share Canvases',
      description: 'Share canvases with others and manage access permissions'
    },
    {
      id: 'canvas_export',
      name: 'Export Canvases',
      description: 'Export canvases to various formats and download data'
    },
    {
      id: 'canvas_comment',
      name: 'Comment on Canvases',
      description: 'Add comments and feedback on canvas elements'
    },
    {
      id: 'canvas_collaborate',
      name: 'Real-time Collaboration',
      description: 'Participate in real-time collaborative editing sessions'
    }
  ]

  const tabs = [
    { id: 'profile', label: 'Profile', icon: 'person' },
    { id: 'general', label: 'General', icon: 'settings' },
    { id: 'users', label: 'Users', icon: 'group' },
    { id: 'teams', label: 'Teams', icon: 'group' },
    { id: 'roles', label: 'Roles', icon: 'admin_panel_settings' },
    { id: 'tokens', label: 'Tokens', icon: 'key' },
    { id: 'integrations', label: 'Integrations', icon: 'extension' },
    { id: 'billing', label: 'Billing', icon: 'credit_card' },
    { id: 'security', label: 'Security', icon: 'security' },
  ]

  const renderTabContent = () => {
    switch (activeTab) {
      case 'roles':
        if (isCreatingRole) {
          // Create role view
          return (
            <div className="space-y-6 pt-6">
              {/* Breadcrumbs navigation */}
              <Breadcrumbs
                items={[
                  { label: 'Roles', icon: 'admin_panel_settings', onClick: handleBackToRoles },
                  { label: activeRoleTab === 'organization' ? 'Organization roles' : 'Canvas roles', onClick: handleBackToRoles },
                  { label: activeRoleTab === 'organization' ? 'New organization role' : 'New canvas role', current: true }
                ]}
                showDivider={false}
              />

              {/* Role creation form */}
              <div className="space-y-6">
                <div>
                  <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-1">
                    Create New {activeRoleTab === 'organization' ? 'Organization' : 'Canvas'} Role
                  </Heading>
                  <Text className="text-zinc-600 dark:text-zinc-400">
                    Define a custom role with specific {activeRoleTab === 'organization' ? 'organization' : 'canvas'} permissions.
                  </Text>
                </div>

                {/* Basic Information */}
                <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6 space-y-4">
                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                      Role Name *
                    </label>
                    <Input
                      type="text"
                      placeholder="Enter role name"
                      value={newRoleName}
                      onChange={(e) => setNewRoleName(e.target.value)}
                      className="max-w-md"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                      Description
                    </label>
                    <Input
                      type="text"
                      placeholder="Describe what this role can do"
                      value={newRoleDescription}
                      onChange={(e) => setNewRoleDescription(e.target.value)}
                      className="max-w-lg"
                    />
                  </div>
                </div>

                {/* Permissions */}
                <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
                  <div className="mb-4">
                    <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white mb-2">
                      {activeRoleTab === 'organization' ? 'Organization' : 'Canvas'} Permissions
                    </Subheading>
                    <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                      Select the permissions this role should have {activeRoleTab === 'organization' ? 'within the organization' : 'for canvas operations'}.
                    </Text>
                  </div>
                  
                  <div className="space-y-4">
                    {(activeRoleTab === 'organization' ? organizationPermissions : canvasPermissions).map((permission) => (
                        <CheckboxField key={permission.id}>
                          <Checkbox name={permission.id} value=" " />
                          <Label>{permission.name}</Label>
                          <Description>{permission.description}</Description>
                        </CheckboxField>
                    ))}
                  </div>
                </div>

                {/* Actions */}
                <div className="flex justify-end gap-3">
                  <Button plain onClick={handleBackToRoles}>
                    Cancel
                  </Button>
                  <Button 
                    color="blue" 
                    onClick={handleSaveRole}
                    disabled={!newRoleName.trim()}
                  >
                    Create Role
                  </Button>
                </div>
              </div>
            </div>
          )
        }

        // Roles list view
        const currentRoles = activeRoleTab === 'organization' ? organizationRoles : canvasRoles
        const buttonText = activeRoleTab === 'organization' ? 'New organization role' : 'New canvas role'
        
        return (
          <div className="space-y-6 pt-6">
            <div className="flex items-center justify-between">
              <div>
                <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-1">
                  Role Management
                </Heading>
              </div>
              <Button color="blue" className='flex items-center' onClick={handleCreateRole}>
                <MaterialSymbol name="add" />
                {buttonText}
              </Button>
            </div>

            {/* Role Tabs */}
            <Tabs
              tabs={roleTabs}
              defaultTab={activeRoleTab}
              onTabChange={(tabId) => setActiveRoleTab(tabId as 'organization' | 'canvas')}
              variant="underline"
            />

            {/* Roles Table */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
              {/* Table Header */}
              <div className="border-b border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800/50">
                <div className="grid grid-cols-4 gap-4 px-6 py-3">
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Role name</div>
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Permissions</div>
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Status</div>
                  <div></div>
                </div>
              </div>

              {/* Table Body */}
              <div className="divide-y divide-zinc-200 dark:divide-zinc-700">
                {currentRoles.map((role) => (
                  <div key={role.id} className="grid grid-cols-4 gap-4 px-6 py-4 items-center">
                    <div className="text-sm font-medium text-zinc-900 dark:text-white">
                      {role.name}
                    </div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      {role.permissions}
                    </div>
                    <div>
                      <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400">
                        {role.status}
                      </span>
                    </div>
                    <div className="flex justify-end">
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
                ))}
              </div>
            </div>
          </div>
        )

      case 'users':
        return (
          <div className="space-y-6 pt-6">
            <div className="flex items-center justify-between">
              
              <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
              Users
            </Heading>
            </div>

            {/* Add Members Section */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">
                    Add Users
                  </Subheading>
                  
                </div>
                
              </div>
              
              <div className="flex gap-3">
                <Input
                  type="email"
                  placeholder="Enter email address"
                  className="flex-1"
                />
                <select className="border border-zinc-300 dark:border-zinc-600 rounded-md px-3 py-2 bg-white dark:bg-zinc-800 text-sm">
                  <option>Member</option>
                  <option>Admin</option>
                  <option>Owner</option>
                </select>
                <Button color="blue">Send Invite</Button>
              </div>
            </div>

            {/* Users List */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
              <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                <div className="flex items-center justify-between">
                <InputGroup>
                    <Input name="search" placeholder="Search&hellip;" aria-label="Search" className="w-xs" />
                  </InputGroup>
                  
                  
                </div>
              </div>

              {/* Table Header */}
              <div className="border-b border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800/50">
                <div className="grid grid-cols-5 gap-4 px-6 py-3">
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Name</div>
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Email</div>
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Role</div>
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Status</div>
                  <div></div>
                </div>
              </div>

              {/* Table Body */}
              <div className="divide-y divide-zinc-200 dark:divide-zinc-700">
                {users.map((user) => (
                  <div key={user.id} className="grid grid-cols-5 gap-4 px-6 py-4 items-center">
                    <div className="flex items-center gap-3">
                      <Avatar
                        src={user.avatar}
                        initials={user.initials}
                        className="size-8"
                      />
                      <div>
                        <div className="text-sm font-medium text-zinc-900 dark:text-white">
                          {user.name}
                        </div>
                        <div className="text-xs text-zinc-500 dark:text-zinc-400">
                          Last active: {user.lastActive}
                        </div>
                      </div>
                    </div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      {user.email}
                    </div>
                    <div>
                      <select 
                        defaultValue={user.role}
                        className="text-sm border border-zinc-300 dark:border-zinc-600 rounded-md px-2 py-1 bg-white dark:bg-zinc-800"
                      >
                        <option>Owner</option>
                        <option>Admin</option>
                        <option>Member</option>
                      </select>
                    </div>
                    <div>
                      <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${
                        user.status === 'Active'
                          ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                          : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
                      }`}>
                        {user.status}
                      </span>
                    </div>
                    <div className="flex justify-end">
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
        )

      case 'teams':
        if (selectedTeam) {
          // Team detail view
          const teamMembers = getTeamMembers(selectedTeam.id)
          return (
            <div className="space-y-6 pt-6">
              {/* Breadcrumbs navigation */}
              <Breadcrumbs
                items={[
                  { label: 'Teams', icon: 'group', onClick: handleBackToTeams },
                  { label: selectedTeam.name, current: true }
                ]}
                showDivider={false}
              />

              {/* Team header */}
              <div className='flex items-center gap-3'>
                <Avatar className='w-9' square initials={selectedTeam.name.charAt(0)} />
                <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                  {selectedTeam.name}
                </Heading>
              </div>
              {/* Add Members Section */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">
                    Add Users
                  </Subheading>
                  
                </div>
                
              </div>
              
              <div className="flex gap-3">
                <Input
                  type="email"
                  placeholder="Enter email address"
                  className="flex-1"
                />
                <select className="border border-zinc-300 dark:border-zinc-600 rounded-md px-3 py-2 bg-white dark:bg-zinc-800 text-sm">
                  <option>Member</option>
                  <option>Admin</option>
                  <option>Owner</option>
                </select>
                <Button color="blue">Send Invite</Button>
              </div>
            </div>
              {/* Team members table */}
              <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
                <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                  <div className="flex items-center justify-between">
                    <InputGroup>
                      <Input name="search" placeholder="Search&hellip;" aria-label="Search" className="w-xs" />
                    </InputGroup>
                    
                  </div>
                </div>

                {/* Table Header */}
                <div className="border-b border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800/50">
                  <div className="grid grid-cols-5 gap-4 px-6 py-3">
                    <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Name</div>
                    <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Email</div>
                    <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Role</div>
                    <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Status</div>
                    <div></div>
                  </div>
                </div>

                {/* Table Body */}
                <div className="divide-y divide-zinc-200 dark:divide-zinc-700">
                  {teamMembers.map((member) => (
                    <div key={member.id} className="grid grid-cols-5 gap-4 px-6 py-4 items-center">
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
                            Joined {member.joinedDate}
                          </div>
                        </div>
                      </div>
                      <div className="text-sm text-zinc-600 dark:text-zinc-400">
                        {member.email}
                      </div>
                      <div>
                        <select 
                          defaultValue={member.role}
                          className="text-sm border border-zinc-300 dark:border-zinc-600 rounded-md px-2 py-1 bg-white dark:bg-zinc-800"
                        >
                          <option>Team Lead</option>
                          <option>Senior Developer</option>
                          <option>Developer</option>
                          <option>Designer</option>
                          <option>DevOps Engineer</option>
                        </select>
                      </div>
                      <div>
                        <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400">
                          {member.status}
                        </span>
                      </div>
                      <div className="flex justify-end">
                        <Dropdown>
                          <DropdownButton plain>
                            <MaterialSymbol name="more_vert" size="sm" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem>
                              <MaterialSymbol name="edit" />
                              Edit Member
                            </DropdownItem>
                            <DropdownItem>
                              <MaterialSymbol name="security" />
                              Change Role
                            </DropdownItem>
                            <DropdownItem>
                              <MaterialSymbol name="person_remove" />
                              Remove from Team
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )
        }

        // Teams list view
        return (
          <div className="space-y-6 pt-6">
            <div className="flex items-center justify-between">
              <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                Teams
              </Heading>
              <Button color="blue" className='flex items-center'>
                <MaterialSymbol name="add" />
                Create New Team
              </Button>
            </div>

            {/* Teams Table View */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
              <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                <InputGroup>
                  <Input name="search" placeholder="Search Teams…" aria-label="Search" className="w-xs" />
                </InputGroup>
              </div>

              {/* Table Header */}
              <div className="border-b border-zinc-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800/50">
                <div className="grid grid-cols-5 gap-4 px-6 py-3">
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Team name</div>
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Description</div>
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Members</div>
                  <div className="text-sm font-medium text-zinc-600 dark:text-zinc-400">Role</div>
                  <div></div>
                </div>
              </div>

              {/* Table Body */}
              <div className="divide-y divide-zinc-200 dark:divide-zinc-700">
                {teams.map((team) => (
                  <div 
                    key={team.id} 
                    className="grid grid-cols-5 gap-4 px-6 py-4 items-center hover:bg-zinc-50 dark:hover:bg-zinc-700/50 cursor-pointer transition-colors"
                    onClick={() => handleTeamClick(team)}
                  >
                    <div className="flex items-center gap-3">
                      <Avatar className='w-9' square initials={team.name.charAt(0)} />
                      <div>
                        <Link href={`#`} className="text-sm font-medium text-blue-600 dark:text-blue-400">
                          {team.name}
                        </Link>
                        <div className="text-xs text-zinc-500 dark:text-zinc-400">
                          Created {team.created}
                        </div>
                      </div>
                    </div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      {team.description}
                    </div>
                    <div className="text-sm text-zinc-600 dark:text-zinc-400">
                      {team.memberCount} members
                    </div>
                    <div>
                      <select 
                        defaultValue={team.role}
                        className="text-sm border border-zinc-300 dark:border-zinc-600 rounded-md px-2 py-1 bg-white dark:bg-zinc-800"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <option>Admin</option>
                        <option>Member</option>
                      </select>
                    </div>
                    <div className="flex justify-end">
                      <Dropdown>
                        <DropdownButton plain onClick={(e) => e.stopPropagation()}>
                          <MaterialSymbol name="more_vert" size="sm" />
                        </DropdownButton>
                        <DropdownMenu>
                          <DropdownItem onClick={() => handleTeamClick(team)}>
                            <MaterialSymbol name="group" />
                            View Members
                          </DropdownItem>
                          <DropdownItem>
                            <MaterialSymbol name="edit" />
                            Edit Team
                          </DropdownItem>
                          <DropdownItem>
                            <MaterialSymbol name="person_add" />
                            Add Members
                          </DropdownItem>
                          <DropdownItem>
                            <MaterialSymbol name="security" />
                            Change Role
                          </DropdownItem>
                          <DropdownItem>
                            <MaterialSymbol name="delete" />
                            Delete Team
                          </DropdownItem>
                        </DropdownMenu>
                      </Dropdown>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>
        )

      case 'general':
        return (
          <div className="space-y-6 pt-6">
            <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
              General Settings
            </Heading>
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6 space-y-6 max-w-xl">
              <div>
                <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                  Organization Name
                </label>
                <Input
                  type="text"
                  defaultValue={currentOrganization.name}
                  className="max-w-md"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                  Description
                </label>
                <Input
                  type="text"
                  placeholder="Enter organization description"
                  className="max-w-lg"
                />
              </div>
            </div>
          </div>
        )

      case 'billing':
        return (
          <div className="space-y-6">
            <Subheading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
              Billing & Subscription
            </Subheading>
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <Text className="text-center text-zinc-500 dark:text-zinc-400">
                Billing settings would go here...
              </Text>
            </div>
          </div>
        )

      case 'security':
        return (
          <div className="space-y-6">
            <Subheading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
              Security Settings
            </Subheading>
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <Text className="text-center text-zinc-500 dark:text-zinc-400">
                Security settings would go here...
              </Text>
            </div>
          </div>
        )

      case 'tokens':
        return (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <Subheading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                API Tokens
              </Subheading>
              <Button color="blue">
                <MaterialSymbol name="add" />
                Create Token
              </Button>
            </div>
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <Text className="text-center text-zinc-500 dark:text-zinc-400">
                No API tokens created yet.
              </Text>
            </div>
          </div>
        )

      case 'integrations':
        return (
          <div className="space-y-6">
            <Subheading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
              Integrations
            </Subheading>
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <Text className="text-center text-zinc-500 dark:text-zinc-400">
                Integration settings would go here...
              </Text>
            </div>
          </div>
        )

      default:
        return null
    }
  }

  return (
    <div className="flex flex-col h-screen bg-zinc-50 dark:bg-zinc-900">
      {/* Navigation */}
      <NavigationOrg
        user={currentUser}
        organization={currentOrganization}
        onUserMenuAction={handleUserMenuAction}
        onOrganizationMenuAction={handleOrganizationMenuAction}
      />
      
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        
        <div className="w-80 bg-white dark:bg-zinc-800 border-r border-zinc-200 dark:border-zinc-700">
          {/* User Account Section */}
        
          <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
          
            <div className="flex items-center gap-3 mb-4">
              <Avatar 
                className='w-8 h-8'
                src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&amp;h=64&amp;fit=crop&amp;crop=face"
                alt="My Account"
              />
              <div>
              <div className="text-sm font-medium text-zinc-900 dark:text-white">My Account</div>
            </div>
          </div>
          
          <nav className="space-y-1">
            <button
              onClick={() => setActiveTab('profile')}
              className={`w-full text-left px-3 py-2 text-sm rounded-md transition-colors ${
                activeTab === 'profile'
                  ? 'bg-zinc-100 dark:bg-zinc-700 text-zinc-900 dark:text-white'
                  : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200'
              }`}
            >
              Profile settings
            </button>
            <button
              onClick={() => setActiveTab('profile')}
              className={`w-full text-left px-3 py-2 text-sm rounded-md transition-colors ${
                activeTab === 'profile'
                  ? 'bg-zinc-100 dark:bg-zinc-700 text-zinc-900 dark:text-white'
                  : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200'
              }`}
            >
              API Token
            </button>
          </nav>
        </div>

        {/* Organization Section */}
        <div className="px-6 py-4">
          <div className="flex items-center gap-3 mb-4">
          <Avatar 
                className='w-8 h-8'
                src="https://upload.wikimedia.org/wikipedia/commons/a/ab/Confluent%2C_Inc._logo.svg"
                alt="My Account"
              />
            
            <div>
              <div className="text-sm font-medium text-zinc-900 dark:text-white">{currentOrganization.name}</div>
            </div>
            <MaterialSymbol name="expand_more" className="text-zinc-400 ml-auto" size="sm" />
          </div>
          
          <nav className="space-y-1">
            {tabs.filter(tab => tab.id !== 'profile').map((tab) => (
              <button
                key={tab.id}
                onClick={() => setActiveTab(tab.id as any)}
                className={`w-full text-left px-3 py-2 text-sm rounded-md transition-colors ${
                  activeTab === tab.id
                    ? 'bg-zinc-100 dark:bg-zinc-700 text-zinc-900 dark:text-white'
                    : 'text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </nav>
        </div>
      </div>

        {/* Main Content */}
        <div className="flex-1 overflow-auto">
          <div className="px-8 pb-8">
            {renderTabContent()}
          </div>
        </div>
      </div>
    </div>
  )
}