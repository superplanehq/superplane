import { useState } from 'react'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Avatar } from './lib/Avatar/avatar'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { Input } from './lib/Input/input'
import { Switch } from './lib/Switch/switch'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem
} from './lib/Dropdown/dropdown'

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
  
  // Mock data for roles
  const roles = [
    {
      id: '1',
      name: 'test',
      permissions: 4,
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

  const currentUser = {
    email: 'john@acme.com'
  }

  const currentOrganization = {
    name: 'Acme corporation'
  }

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
        return (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <div>
                <Subheading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-1">
                  Role Management
                </Subheading>
              </div>
              <Button color="blue">
                Add custom role
              </Button>
            </div>

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
                {roles.map((role) => (
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
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <Subheading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                User Management
              </Subheading>
            </div>

            {/* Add Members Section */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">
                    Add Members
                  </Subheading>
                  <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                    Invite new team members to your organization
                  </Text>
                </div>
                <Button color="blue">
                  <MaterialSymbol name="person_add" />
                  Invite Member
                </Button>
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
                  <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white">
                    Organization Members ({users.length})
                  </Subheading>
                  <div className="flex items-center gap-3">
                    <Input
                      type="text"
                      placeholder="Search members..."
                      className="w-64"
                    />
                  </div>
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
        return (
          <div className="space-y-6">
            <div className="flex items-center justify-between">
              <Subheading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                Teams
              </Subheading>
              <Button color="blue">
                <MaterialSymbol name="add" />
                Add New Team
              </Button>
            </div>

            {/* Teams List */}
            <div className="hidden grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              {teams.map((team) => (
                <div key={team.id} className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
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
                          Edit Team
                        </DropdownItem>
                        <DropdownItem>
                          <MaterialSymbol name="person_add" />
                          Add Members
                        </DropdownItem>
                        <DropdownItem>
                          <MaterialSymbol name="delete" />
                          Delete Team
                        </DropdownItem>
                      </DropdownMenu>
                    </Dropdown>
                  </div>

                  <Subheading level={3} className="mb-2">{team.name}</Subheading>
                  <Text className="text-zinc-600 dark:text-zinc-400 mb-4 text-sm">
                    {team.description}
                  </Text>

                  <div className="space-y-2">
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-zinc-500 dark:text-zinc-400">Members</span>
                      <span className="font-medium text-zinc-900 dark:text-white">{team.memberCount}</span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-zinc-500 dark:text-zinc-400">Role</span>
                      <span className={`px-2 py-1 rounded-full text-xs font-medium ${
                        team.role === 'Admin'
                          ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400'
                          : 'bg-zinc-100 text-zinc-800 dark:bg-zinc-900/20 dark:text-zinc-400'
                      }`}>
                        {team.role}
                      </span>
                    </div>
                    <div className="flex items-center justify-between text-sm">
                      <span className="text-zinc-500 dark:text-zinc-400">Created</span>
                      <span className="text-zinc-600 dark:text-zinc-400">{team.created}</span>
                    </div>
                  </div>

                  <div className="mt-4 pt-4 border-t border-zinc-200 dark:border-zinc-700">
                    <Button plain className="w-full">
                      <MaterialSymbol name="group" />
                      View Members
                    </Button>
                  </div>
                </div>
              ))}

              {/* Add New Team Card */}
              <div className="hidden bg-white dark:bg-zinc-800 rounded-lg border-2 border-dashed border-zinc-300 dark:border-zinc-600 p-6 flex flex-col items-center justify-center text-center min-h-[280px]">
                <div className="p-3 bg-zinc-50 dark:bg-zinc-700 rounded-lg mb-4">
                  <MaterialSymbol name="add" className="text-zinc-400" size="lg" />
                </div>
                <Subheading level={3} className="mb-2 text-zinc-600 dark:text-zinc-400">
                  Create New Team
                </Subheading>
                <Text className="text-sm text-zinc-500 dark:text-zinc-500 mb-4">
                  Organize your members into teams for better collaboration
                </Text>
                <Button color="blue">
                  <MaterialSymbol name="add" />
                  Add Team
                </Button>
              </div>
            </div>

            {/* Teams Table View (Alternative) */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
              <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white">
                  All Teams ({teams.length})
                </Subheading>
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
                  <div key={team.id} className="grid grid-cols-5 gap-4 px-6 py-4 items-center">
                    <div className="flex items-center gap-3">
                      <div className="p-1.5 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                        <MaterialSymbol name="group" className="text-blue-600 dark:text-blue-400" size="sm" />
                      </div>
                      <div>
                        <div className="text-sm font-medium text-zinc-900 dark:text-white">
                          {team.name}
                        </div>
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
                      >
                        <option>Admin</option>
                        <option>Member</option>
                      </select>
                    </div>
                    <div className="flex justify-end">
                      <Dropdown>
                        <DropdownButton plain>
                          <MaterialSymbol name="more_vert" size="sm" />
                        </DropdownButton>
                        <DropdownMenu>
                          <DropdownItem>
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
          <div className="space-y-6">
            <Subheading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
              General Settings
            </Subheading>
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6 space-y-6">
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
    <div className="flex h-screen bg-zinc-50 dark:bg-zinc-900">
      {/* Back to portal button */}
      <div className="absolute top-4 left-4 z-10">
        <button
          onClick={onBack}
          className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-zinc-200 transition-colors"
        >
          <MaterialSymbol name="arrow_back" size="sm" />
          Back to portal
        </button>
      </div>

      {/* Sidebar */}
      <div className="w-80 bg-white dark:bg-zinc-800 border-r border-zinc-200 dark:border-zinc-700 pt-16">
        {/* User Account Section */}
        <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-teal-500 flex items-center justify-center">
              <MaterialSymbol name="person" className="text-white" size="sm" />
            </div>
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
              Profile
            </button>
          </nav>
        </div>

        {/* Organization Section */}
        <div className="px-6 py-4">
          <div className="flex items-center gap-3 mb-4">
            <div className="w-8 h-8 rounded-full bg-zinc-200 dark:bg-zinc-700 flex items-center justify-center">
              <MaterialSymbol name="business" className="text-zinc-600 dark:text-zinc-400" size="sm" />
            </div>
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
        {/* User email header */}
        <div className="px-8 pt-16 pb-2">
          <Text className="text-sm text-purple-600 dark:text-purple-400 font-medium">
            {currentUser.email}
          </Text>
        </div>
        
        <div className="px-8 pb-8">
          {renderTabContent()}
        </div>
      </div>
    </div>
  )
}