import { useState } from 'react'
import { NavigationVertical, type User, type Organization, type NavigationLink } from './lib/Navigation/navigation-vertical'
import { Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { type Tab } from './lib/Tabs/tabs'
import clsx from 'clsx'
import { Dropdown, DropdownButton, DropdownMenu, DropdownItem, DropdownLabel, DropdownDescription } from './lib/Dropdown/dropdown'
import { Dialog, DialogTitle, DialogDescription, DialogBody, DialogActions } from './lib/Dialog/dialog'
import { Input, InputGroup } from './lib/Input/input'
import { Checkbox } from './lib/Checkbox/checkbox'
import { Sidebar, SidebarBody, SidebarItem, SidebarLabel, SidebarSection } from './lib/Sidebar/sidebar'
import { Textarea } from './lib/Textarea/textarea'

interface SettingsPageProps {
  onSignOut?: () => void
  navigationLinks?: NavigationLink[]
  onLinkClick?: (linkId: string) => void
  onConfigurationClick?: () => void
}

export function SettingsPage({ 
  onSignOut, 
  navigationLinks = [], 
  onLinkClick,
  onConfigurationClick 
}: SettingsPageProps) {
  const [activeTab, setActiveTab] = useState<'users' | 'secrets' | 'integrations' | 'delete'>('users')
  const [secretsSection, setSecretsSection] = useState<'list' | 'new'>('list')
  const [searchUsers, setSearchUsers] = useState('')
  const [inviteRole, setInviteRole] = useState('Member')
  const [userRoles, setUserRoles] = useState<Record<string, string>>({
    '1': 'Owner',
    '2': 'Admin', 
    '3': 'Member'
  })
  
  // Group modal state
  const [isGroupModalOpen, setIsGroupModalOpen] = useState(false)
  const [newGroupName, setNewGroupName] = useState('')
  const [newGroupDescription, setNewGroupDescription] = useState('')
  const [newGroupMembers, setNewGroupMembers] = useState('')

  // Role modal state
  const [isRoleModalOpen, setIsRoleModalOpen] = useState(false)
  const [newRoleName, setNewRoleName] = useState('')
  const [newRoleDescription, setNewRoleDescription] = useState('')
  const [newRolePermissions, setNewRolePermissions] = useState<string[]>([])

  // Secret form state
  const [secretName, setSecretName] = useState('')
  const [secretDescription, setSecretDescription] = useState('')
  const [environmentVariables, setEnvironmentVariables] = useState<Array<{id: string, name: string, value: string}>>([
    { id: '1', name: '', value: '' }
  ])
  const [configurationFiles, setConfigurationFiles] = useState<Array<{id: string, path: string, file?: File}>>([
    { id: '1', path: '', file: undefined }
  ])

  // Available permissions
  const organizationPermissions = [
    { id: 'org_view', name: 'View Organization', description: 'View organization details and members' },
    { id: 'org_invite', name: 'Invite Members', description: 'Send invitations to new organization members' },
    { id: 'org_manage_members', name: 'Manage Members', description: 'Edit member roles and remove members' },
    { id: 'org_billing', name: 'Billing Access', description: 'View and manage billing information' },
    { id: 'org_settings', name: 'Organization Settings', description: 'Modify organization settings and configuration' },
    { id: 'org_delete', name: 'Delete Organization', description: 'Remove the entire organization' },
  ]

  const canvasPermissions = [
    { id: 'canvas_view', name: 'View Canvases', description: 'View existing canvases and their content' },
    { id: 'canvas_create', name: 'Create Canvases', description: 'Create new canvases' },
    { id: 'canvas_edit', name: 'Edit Canvases', description: 'Modify canvas content and structure' },
    { id: 'canvas_share', name: 'Share Canvases', description: 'Share canvases with others and manage access' },
    { id: 'canvas_export', name: 'Export Canvases', description: 'Export canvases to various formats' },
    { id: 'canvas_delete', name: 'Delete Canvases', description: 'Remove canvases permanently' },
  ]

  // Role definitions
  const roles = [
    {
      name: 'Owner',
      description: 'Owners have access to all functionalities within the organization and any of its projects. They cannot be removed from the organization.'
    },
    {
      name: 'Admin', 
      description: 'Admins can modify settings within the organization or any of its projects. However, they cannot change general organization details, such as the organization name and URL, delete organization, change owners or create new roles.'
    },
    {
      name: 'Member',
      description: 'Members can access the organization\'s homepage and the projects they are assigned to. However, they are not able to modify any settings.'
    }
  ]

  // Tab configuration
  const tabs: Tab[] = [
    {
      id: 'users',
      label: 'Members',
      icon: <MaterialSymbol name="person" size="sm" />,
    },
    {
      id: 'secrets',
      label: 'Secrets',
      icon: <MaterialSymbol name="key" size="sm" />,
    },
    {
      id: 'integrations',
      label: 'Integrations',
      icon: <MaterialSymbol name="integration_instructions" size="sm" />,
    },
    {
      id: 'delete',
      label: 'Delete',
      icon: <MaterialSymbol name="delete" size="sm" />,
    },
  ]

  // Mock user and organization data
  const currentUser: User = {
    id: '1',
    name: 'John Doe',
    email: 'john@superplane.com',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face',
  }

  const currentOrganization: Organization = {
    id: '1',
    name: 'Acme Corporation',
    plan: 'Pro Plan',
    initials: 'AC',
  }

  // Navigation handlers
  const handleHelpClick = () => {
    console.log('Opening help documentation...')
  }

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
      // Already on settings page
      console.log('Already on organization settings page')
    } else {
      console.log(`Organization action: ${action}`)
    }
  }

  const handleLinkClick = (linkId: string) => {
    if (onLinkClick) {
      onLinkClick(linkId)
    } else {
      console.log(`Navigation link clicked: ${linkId}`)
    }
  }

  // Group modal handlers
  const handleCreateGroup = () => {
    setIsGroupModalOpen(true)
  }

  const handleCloseGroupModal = () => {
    setIsGroupModalOpen(false)
    setNewGroupName('')
    setNewGroupDescription('')
    setNewGroupMembers('')
  }

  const handleCreateRole = () => {
    setIsRoleModalOpen(true)
  }

  const handleCloseRoleModal = () => {
    setIsRoleModalOpen(false)
    setNewRoleName('')
    setNewRoleDescription('')
    setNewRolePermissions([])
  }

  const handlePermissionChange = (permissionId: string, checked: boolean) => {
    if (checked) {
      setNewRolePermissions(prev => [...prev, permissionId])
    } else {
      setNewRolePermissions(prev => prev.filter(id => id !== permissionId))
    }
  }

  const handleSaveRole = () => {
    // Here you would typically save the role to your backend
    console.log('Creating role:', {
      name: newRoleName,
      description: newRoleDescription,
      permissions: newRolePermissions
    })
    handleCloseRoleModal()
  }

  const handleSaveGroup = () => {
    if (newGroupName.trim()) {
      // Here you would typically save the group to your backend
      console.log('Creating group:', { 
        name: newGroupName, 
        description: newGroupDescription,
        members: newGroupMembers.split(',').map(email => email.trim()).filter(email => email)
      })
      handleCloseGroupModal()
    }
  }

  // Secret section handlers
  const handleCreateSecret = () => {
    setSecretsSection('new')
  }

  const handleBackToSecrets = () => {
    setSecretsSection('list')
    setSecretName('')
    setSecretDescription('')
    setEnvironmentVariables([{ id: '1', name: '', value: '' }])
    setConfigurationFiles([{ id: '1', path: '', file: undefined }])
  }

  const handleSaveSecret = () => {
    if (secretName.trim()) {
      // Here you would typically save the secret to your backend
      console.log('Creating secret:', {
        name: secretName,
        description: secretDescription,
        environmentVariables: environmentVariables.filter(env => env.name.trim() || env.value.trim()),
        configurationFiles: configurationFiles.filter(file => file.path.trim() || file.file)
      })
      handleBackToSecrets()
    }
  }

  const handleAddEnvironmentVariable = () => {
    const newId = Date.now().toString()
    setEnvironmentVariables(prev => [...prev, { id: newId, name: '', value: '' }])
  }

  const handleRemoveEnvironmentVariable = (id: string) => {
    setEnvironmentVariables(prev => prev.filter(env => env.id !== id))
  }

  const handleUpdateEnvironmentVariable = (id: string, field: 'name' | 'value', value: string) => {
    setEnvironmentVariables(prev => 
      prev.map(env => env.id === id ? { ...env, [field]: value } : env)
    )
  }

  const handleAddConfigurationFile = () => {
    const newId = Date.now().toString()
    setConfigurationFiles(prev => [...prev, { id: newId, path: '', file: undefined }])
  }

  const handleRemoveConfigurationFile = (id: string) => {
    setConfigurationFiles(prev => prev.filter(file => file.id !== id))
  }

  const handleUpdateConfigurationFile = (id: string, field: 'path', value: string) => {
    setConfigurationFiles(prev => 
      prev.map(file => file.id === id ? { ...file, [field]: value } : file)
    )
  }

  const handleFileUpload = (id: string, file: File) => {
    setConfigurationFiles(prev => 
      prev.map(configFile => configFile.id === id ? { ...configFile, file } : configFile)
    )
  }

  // Mock data for demonstration
  const mockUsers = [
    { id: '1', name: 'John Doe', email: 'john@superplane.com', role: 'Owner', status: 'Active' },
    { id: '2', name: 'Jane Smith', email: 'jane@superplane.com', role: 'Admin', status: 'Active' },
    { id: '3', name: 'Bob Wilson', email: 'bob@superplane.com', role: 'Member', status: 'Active' },
  ]

  // Filter users based on search query
  const filteredUsers = mockUsers.filter(user =>
    user.name.toLowerCase().includes(searchUsers.toLowerCase()) ||
    user.email.toLowerCase().includes(searchUsers.toLowerCase()) ||
    user.role.toLowerCase().includes(searchUsers.toLowerCase())
  )

  const mockGroups = [
    { id: '1', name: 'Administrators', description: 'Administrative team members', members: 2 },
    { id: '2', name: 'Editors', description: 'Content creation team', members: 5 },
    { id: '3', name: 'Viewers', description: 'General organization members', members: 12 },
  ]

  const mockRoles = [
    { id: '1', name: 'Admin', description: 'Full system access', permissions: ['read', 'write', 'delete', 'manage'] },
    { id: '2', name: 'Editor', description: 'Can create and edit', permissions: ['read', 'write'] },
    { id: '3', name: 'Viewer', description: 'Read-only access', permissions: ['read'] },
  ]

  return (
    <div className="flex min-h-screen h-full bg-gray-50">
      {/* Vertical Navigation */}
      
      {/* Main Content */}
      <div className="flex-1 flex flex-col">
        {/* Header */}
      

        {/* Settings Content */}
        <main className="flex-1 flex">
          {/* Sidebar Navigation */}
          <Sidebar className='w-70 bg-white dark:bg-zinc-950 border-r bw-1 border-zinc-200 dark:border-zinc-800'>
                    
            <SidebarBody>
              
         
                {tabs.filter(tab => tab.id !== 'profile').map((tab) => (
                  <SidebarItem 
                    key={tab.id} onClick={() => setActiveTab(tab.id as any)} 
                    className={`${activeTab === tab.id ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`}
                  >
                    <span className={`px-7 ${activeTab === tab.id ? 'font-semibold' : 'font-normal'}`}>
                      <SidebarLabel>{tab.label}</SidebarLabel>
                    </span>
                  </SidebarItem>
                ))}
                
          
            </SidebarBody>
          </Sidebar>
       

          {/* Main Content Area */}
          <div className="flex-1 p-6">
            <div className="max-w-5xl mx-auto">
              {/* Tab Content */}
            {activeTab === 'users' && (
              <div className="space-y-6">
                {/* Invite new members section */}
                <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
                  <h3 className="text-base font-medium text-zinc-900 dark:text-zinc-100 mb-4">Invite new members</h3>
                  <div className="flex items-start gap-4">
                    <div className="flex-1">
                      <Input
                        type="text"
                        placeholder="Enter an email or username"
                        className="w-full"
                      />
                    </div>
                    <div className="flex items-center gap-2">
                     
                        <Dropdown>
                          <DropdownButton color='white'>
                            <span>{inviteRole}</span>
                            <MaterialSymbol name="expand_more" size="sm" className="text-zinc-400" />
                          </DropdownButton>
                          <DropdownMenu className="!max-w-[400px]">
                            {roles.map((role) => (
                              <DropdownItem key={role.name} onClick={() => setInviteRole(role.name)}>
                                <div className="flex items-start gap-3 w-full">
                                  <div className="flex-shrink-0 w-4 flex justify-center mt-0.5">
                                    {inviteRole === role.name && (
                                      <MaterialSymbol name="check" size="lg" className="text-blue-500" />
                                    )}
                                  </div>
                                  <div className="flex-1">
                                    <DropdownLabel className="font-medium text-zinc-900 dark:text-zinc-100">
                                      {role.name}
                                    </DropdownLabel>
                                    <DropdownDescription className="text-xs text-zinc-600 dark:text-zinc-400 mt-1 leading-relaxed">
                                      {role.description}
                                    </DropdownDescription>
                                  </div>
                                </div>
                              </DropdownItem>
                            ))}
                          </DropdownMenu>
                        </Dropdown>
                     
                      <Button color='blue' disabled={true}>
                          <MaterialSymbol name="add" size="sm" />
                          Invite
                        </Button>
                    </div>
                  </div>
                </div>

                {/* Members list section */}
                <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700">
                  <div className="px-6 py-4 border-b border-zinc-200 dark:border-zinc-700">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-3">
                        {/* Search Users */}
                        <div className="max-w-sm">
                          <div className="relative">
                          <InputGroup>
                            <MaterialSymbol 
                              name="search" 
                              className="absolute left-3 top-1/2 transform -translate-y-1/2 text-zinc-400" 
                              size="sm" 
                            />
                            <Input
                              type="text"
                              placeholder="Search members..."
                              value={searchUsers}
                              onChange={(e) => setSearchUsers(e.target.value)}
                              className="w-full pl-10 pr-4"
                            />
                          </InputGroup>
                            
                          </div>
                        </div>
                      </div>
                      <h3 className="text-base font-medium text-zinc-900 dark:text-zinc-100">
                        {filteredUsers.length} member{filteredUsers.length !== 1 ? 's' : ''}
                      </h3>
                    </div>
                  </div>
                  
                  <div className="p-6">
                    <div className="space-y-4">
                      {filteredUsers.map((user) => (
                        <div key={user.id} className="flex items-center justify-between py-3">
                          <div className="flex items-center space-x-3">
                            <div className="w-10 h-10 bg-zinc-200 dark:bg-zinc-700 rounded-full flex items-center justify-center">
                              <span className="text-sm font-medium text-zinc-600 dark:text-zinc-400">
                                {user.name.split(' ').map(n => n[0]).join('').toUpperCase()}
                              </span>
                            </div>
                            <div>
                              <div className="flex items-center gap-2">
                                <span className="text-sm font-medium text-zinc-900 dark:text-zinc-100">{user.name}</span>
                                {(userRoles[user.id] || user.role) === 'Owner' && (
                                  <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-zinc-100 text-zinc-800 dark:bg-zinc-700 dark:text-zinc-200">
                                    OWNER
                                  </span>
                                )}
                              </div>
                              <p className="text-sm text-zinc-500 dark:text-zinc-400">
                                Joined on {user.id === '1' ? 'January 22, 2021 11:52' : 'Recently'}
                              </p>
                            </div>
                          </div>
                          <div className="flex items-center gap-3">
                            {/* Permissions Dropdown */}
                            <div>
                              {user.role === 'Owner' && user.id === '1' ? (
                                // Disabled state for current owner
                                <div className="px-3 py-1.5 text-sm border border-zinc-300 dark:border-zinc-600 rounded-md bg-zinc-100 dark:bg-zinc-700 text-zinc-500 dark:text-zinc-400 min-w-[120px] opacity-50">
                                  {userRoles[user.id] || user.role}
                                </div>
                              ) : (
                                <Dropdown>
                                  <DropdownButton color='white'>
                                    <span>{userRoles[user.id] || user.role}</span>
                                    <MaterialSymbol name="expand_more" size="lg" className="text-zinc-400" />
                                  </DropdownButton>
                                  <DropdownMenu className="!max-w-[400px]">
                                    {roles.map((role) => (
                                      <DropdownItem 
                                        key={role.name} 
                                        onClick={() => setUserRoles(prev => ({ ...prev, [user.id]: role.name }))}
                                      >
                                        <div className="flex items-start gap-3 w-full">
                                          <div className="flex-shrink-0 w-4 flex justify-center mt-0.5">
                                            {(userRoles[user.id] || user.role) === role.name && (
                                              <MaterialSymbol name="check" size="lg" className="text-blue-500" />
                                            )}
                                          </div>
                                          <div className="flex-1">
                                            <DropdownLabel className="font-medium text-zinc-900 dark:text-zinc-100">
                                              {role.name}
                                            </DropdownLabel>
                                            <DropdownDescription className="text-xs text-zinc-600 dark:text-zinc-400 mt-1 leading-relaxed">
                                              {role.description}
                                            </DropdownDescription>
                                          </div>
                                        </div>
                                      </DropdownItem>
                                    ))}
                                  </DropdownMenu>
                                </Dropdown>
                              )}
                            </div>
                            
                            {/* Remove Button */}
                            {!(user.role === 'Owner' && user.id === '1') && (
                              <button
                                className="p-1.5 text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300 transition-colors"
                                title="Remove user"
                              >
                                <MaterialSymbol name="close" size="lg" />
                              </button>
                            )}
                          </div>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'secrets' && (
              <div className="space-y-6">
                {/* Breadcrumbs */}
                <div className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
                  <button 
                    className={secretsSection === 'list' ? 'font-medium text-zinc-900 dark:text-zinc-100' : 'hover:text-zinc-900 dark:hover:text-zinc-100'}
                    onClick={() => setSecretsSection('list')}
                  >
                    Secrets
                  </button>
                  {secretsSection === 'new' && (
                    <>
                      <MaterialSymbol name="chevron_right" size="sm" />
                      <span className="font-medium text-zinc-900 dark:text-zinc-100">New secret</span>
                    </>
                  )}
                </div>

                {secretsSection === 'list' && (
                  <>
                    <div className="flex items-center justify-between">
                      <Subheading level={2}>Secrets</Subheading>
                      <Button color="blue" onClick={handleCreateSecret}>
                        <MaterialSymbol name="add" />
                        Add Secret
                      </Button>
                    </div>
                    
                    <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
                      <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
                        Manage environment variables and secrets for your workflows. These values are encrypted and can be used in your stage configurations.
                      </Text>
                      
                      <div className="space-y-4">
                        <div className="flex items-center justify-between py-3 border-b border-zinc-200 dark:border-zinc-700">
                          <div>
                            <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100">DATABASE_URL</div>
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">Added 2 days ago</div>
                          </div>
                          <div className="flex space-x-2">
                            <Button plain>
                              <MaterialSymbol name="edit" />
                            </Button>
                            <Button plain>
                              <MaterialSymbol name="delete" />
                            </Button>
                          </div>
                        </div>
                        
                        <div className="flex items-center justify-between py-3 border-b border-zinc-200 dark:border-zinc-700">
                          <div>
                            <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100">API_KEY</div>
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">Added 1 week ago</div>
                          </div>
                          <div className="flex space-x-2">
                            <Button plain>
                              <MaterialSymbol name="edit" />
                            </Button>
                            <Button plain>
                              <MaterialSymbol name="delete" />
                            </Button>
                          </div>
                        </div>
                        
                        <div className="flex items-center justify-between py-3">
                          <div>
                            <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100">WEBHOOK_SECRET</div>
                            <div className="text-xs text-zinc-500 dark:text-zinc-400">Added 2 weeks ago</div>
                          </div>
                          <div className="flex space-x-2">
                            <Button plain>
                              <MaterialSymbol name="edit" />
                            </Button>
                            <Button plain>
                              <MaterialSymbol name="delete" />
                            </Button>
                          </div>
                        </div>
                      </div>
                    </div>
                  </>
                )}

                {secretsSection === 'new' && (
                  <>
                    <div className="flex items-center justify-between">
                      <Subheading level={2}>New secret</Subheading>
                    </div>
                    
                    <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
                      <div className="space-y-6">
                        {/* Name of the Secret */}
                        <div>
                          <label htmlFor="secretName" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                            Name of the Secret
                          </label>
                          <Input
                            id="secretName"
                            type="text"
                            placeholder="Enter name..."
                            value={secretName}
                            onChange={(e) => setSecretName(e.target.value)}
                            className="w-full"
                          />
                        </div>

                        {/* Description of the Secret */}
                        <div>
                          <label htmlFor="secretDescription" className="block text-sm font-medium text-zinc-900 dark:text-zinc-100 mb-2">
                            Description of the Secret
                          </label>
                          <Textarea
                            id="secretDescription"
                            placeholder="Describe secret contents..."
                            value={secretDescription}
                            onChange={(e) => setSecretDescription(e.target.value)}
                            className="w-full"
                            rows={3}
                          />
                        </div>

                        {/* Content */}
                        <div>
                          <Subheading level={3} className="mb-4">Content</Subheading>
                          
                          {/* Environment Variables */}
                          <div className="mb-6">
                            <Subheading level={4} className="mb-3">Environment Variables</Subheading>
                            <div className="space-y-3">
                              {environmentVariables.map((env) => (
                                <div key={env.id} className="flex gap-3 items-center">
                                  <div className="flex-1">
                                    <Input
                                      placeholder="Variable Name"
                                      value={env.name}
                                      onChange={(e) => handleUpdateEnvironmentVariable(env.id, 'name', e.target.value)}
                                      className="w-full"
                                    />
                                  </div>
                                  <div className="flex-1">
                                    <Input
                                      placeholder="Value"
                                      value={env.value}
                                      onChange={(e) => handleUpdateEnvironmentVariable(env.id, 'value', e.target.value)}
                                      className="w-full"
                                    />
                                  </div>
                                  <Button
                                    plain
                                    onClick={() => handleRemoveEnvironmentVariable(env.id)}
                                    className="text-red-600 hover:text-red-700"
                                  >
                                    <MaterialSymbol name="delete" size="sm" />
                                  </Button>
                                </div>
                              ))}
                            </div>
                            <Button
                              plain
                              onClick={handleAddEnvironmentVariable}
                              className="mt-3 text-zinc-900 dark:text-zinc-100 hover:text-zinc-700 dark:hover:text-zinc-300"
                            >
                              + Add Environment Variable
                            </Button>
                          </div>

                          {/* Configuration Files */}
                          <div>
                            <Subheading level={4} className="mb-3">Configuration Files</Subheading>
                            <div className="space-y-3">
                              {configurationFiles.map((file) => (
                                <div key={file.id} className="flex gap-3 items-center">
                                  <div className="flex-1">
                                    <Input
                                      placeholder="/path/to/file"
                                      value={file.path}
                                      onChange={(e) => handleUpdateConfigurationFile(file.id, 'path', e.target.value)}
                                      className="w-full"
                                    />
                                  </div>
                                  <Button
                                    color="zinc"
                                    onClick={() => {
                                      const input = document.createElement('input')
                                      input.type = 'file'
                                      input.onchange = (e) => {
                                        const selectedFile = (e.target as HTMLInputElement).files?.[0]
                                        if (selectedFile) {
                                          handleFileUpload(file.id, selectedFile)
                                        }
                                      }
                                      input.click()
                                    }}
                                  >
                                    Upload File
                                  </Button>
                                  <Button
                                    plain
                                    onClick={() => handleRemoveConfigurationFile(file.id)}
                                    className="text-red-600 hover:text-red-700"
                                  >
                                    <MaterialSymbol name="delete" size="sm" />
                                  </Button>
                                </div>
                              ))}
                            </div>
                            <Button
                              plain
                              onClick={handleAddConfigurationFile}
                              className="mt-3 text-zinc-900 dark:text-zinc-100 hover:text-zinc-700 dark:hover:text-zinc-300"
                            >
                              + Add Configuration File
                            </Button>
                          </div>
                        </div>

                        {/* Actions */}
                        <div className="flex items-center gap-3 pt-4 border-t border-zinc-200 dark:border-zinc-700">
                          <Button
                            color="blue"
                            onClick={handleSaveSecret}
                            disabled={!secretName.trim()}
                          >
                            Save Secret
                          </Button>
                          <Button
                            plain
                            onClick={handleBackToSecrets}
                            className="text-zinc-600 hover:text-zinc-900 dark:text-zinc-400 dark:hover:text-zinc-100"
                          >
                            Cancel
                          </Button>
                        </div>
                      </div>
                    </div>
                  </>
                )}
              </div>
            )}

            {activeTab === 'integrations' && (
              <div className="space-y-6">
                <div className="flex items-center justify-between">
                  <Subheading level={2}>Integrations</Subheading>
                  <Button color="blue">
                    <MaterialSymbol name="add" />
                    Add Integration
                  </Button>
                </div>
                
                <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                  <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-black rounded flex items-center justify-center">
                          <MaterialSymbol name="code" className="text-white" size="sm" />
                        </div>
                        <Subheading level={3}>GitHub</Subheading>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="w-2 h-2 bg-green-500 rounded-full"></div>
                        <span className="text-xs text-green-600 dark:text-green-400">Connected</span>
                      </div>
                    </div>
                    <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
                      Connected to your GitHub account for repository access and webhook triggers.
                    </Text>
                    <div className="flex space-x-2">
                      <Button plain>
                        <MaterialSymbol name="settings" />
                        Configure
                      </Button>
                      <Button plain>
                        <MaterialSymbol name="link_off" />
                        Disconnect
                      </Button>
                    </div>
                  </div>
                  
                  <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-blue-500 rounded flex items-center justify-center">
                          <MaterialSymbol name="cloud" className="text-white" size="sm" />
                        </div>
                        <Subheading level={3}>Slack</Subheading>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="w-2 h-2 bg-zinc-400 rounded-full"></div>
                        <span className="text-xs text-zinc-500 dark:text-zinc-400">Not Connected</span>
                      </div>
                    </div>
                    <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
                      Get notifications about workflow runs and deployments in your Slack channels.
                    </Text>
                    <div className="flex space-x-2">
                      <Button color="blue">
                        <MaterialSymbol name="link" />
                        Connect
                      </Button>
                    </div>
                  </div>
                  
                  <div className="bg-white dark:bg-zinc-800 p-6 rounded-lg border border-zinc-200 dark:border-zinc-700">
                    <div className="flex items-center justify-between mb-4">
                      <div className="flex items-center gap-3">
                        <div className="w-8 h-8 bg-orange-500 rounded flex items-center justify-center">
                          <MaterialSymbol name="webhook" className="text-white" size="sm" />
                        </div>
                        <Subheading level={3}>Custom Webhook</Subheading>
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="w-2 h-2 bg-zinc-400 rounded-full"></div>
                        <span className="text-xs text-zinc-500 dark:text-zinc-400">Not Connected</span>
                      </div>
                    </div>
                    <Text className="text-zinc-600 dark:text-zinc-400 mb-4">
                      Send workflow events to your custom webhook endpoint.
                    </Text>
                    <div className="flex space-x-2">
                      <Button color="blue">
                        <MaterialSymbol name="link" />
                        Connect
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            )}

            {activeTab === 'delete' && (
              <div className="space-y-6">
                <Subheading level={2}>Danger Zone</Subheading>
                
                <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-6">
                  <div className="flex items-start gap-4">
                    <MaterialSymbol name="warning" className="text-red-600 dark:text-red-400 mt-1" />
                    <div className="flex-1">
                      <Subheading level={3} className="text-red-900 dark:text-red-100 mb-2">
                        Delete Canvas
                      </Subheading>
                      <Text className="text-red-700 dark:text-red-300 mb-4">
                        Once you delete this canvas, there is no going back. This action cannot be undone.
                        All workflows, configurations, and associated data will be permanently removed.
                      </Text>
                      
                      <div className="space-y-3">
                        <Text className="text-sm text-red-600 dark:text-red-400">
                          This will permanently delete:
                        </Text>
                        <ul className="text-sm text-red-600 dark:text-red-400 space-y-1 ml-4">
                          <li>• All workflow stages and configurations</li>
                          <li>• All secrets and environment variables</li>
                          <li>• All integration connections</li>
                          <li>• All execution history and logs</li>
                          <li>• All team member access</li>
                        </ul>
                      </div>
                      
                      <div className="mt-6">
                        <Button color="red">
                          <MaterialSymbol name="delete" />
                          Delete Canvas
                        </Button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            )}
            </div>
          </div>
        </main>
      </div>

      {/* Create Group Modal */}
      <Dialog open={isGroupModalOpen} onClose={handleCloseGroupModal} size="md">
        <DialogTitle>Create New Group</DialogTitle>
        <DialogDescription>
          Create a new group to organize team members in your organization.
        </DialogDescription>
        <DialogBody>
          <div className="space-y-4">
            <div>
              <label htmlFor="groupName" className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Group Name *
              </label>
              <Input
                id="groupName"
                type="text"
                placeholder="Enter group name"
                value={newGroupName}
                onChange={(e) => setNewGroupName(e.target.value)}
                className="w-full"
              />
            </div>
            <div>
              <label htmlFor="groupDescription" className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Description
              </label>
              <Input
                id="groupDescription"
                type="text"
                placeholder="Enter group description (optional)"
                value={newGroupDescription}
                onChange={(e) => setNewGroupDescription(e.target.value)}
                className="w-full"
              />
            </div>
            <div>
              <label htmlFor="groupMembers" className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Members
              </label>
              <Input
                id="groupMembers"
                type="text"
                placeholder="Enter email addresses separated by commas (optional)"
                value={newGroupMembers}
                onChange={(e) => setNewGroupMembers(e.target.value)}
                className="w-full"
              />
              <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                Example: john@example.com, jane@example.com
              </p>
            </div>
          </div>
        </DialogBody>
        <DialogActions>
          <Button plain onClick={handleCloseGroupModal}>
            Cancel
          </Button>
          <Button color="blue" onClick={handleSaveGroup} disabled={!newGroupName.trim()}>
            Create Group
          </Button>
        </DialogActions>
      </Dialog>

      {/* Create Role Modal */}
      <Dialog open={isRoleModalOpen} onClose={handleCloseRoleModal} size="2xl">
        <DialogTitle>Create New Role</DialogTitle>
        <DialogDescription>
          Create a new role with specific permissions for your organization members.
        </DialogDescription>
        <DialogBody>
          <div className="space-y-6">
            {/* Role Name */}
            <div>
              <label htmlFor="roleName" className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Role Name *
              </label>
              <Input
                id="roleName"
                type="text"
                placeholder="Enter role name"
                value={newRoleName}
                onChange={(e) => setNewRoleName(e.target.value)}
                className="w-full"
              />
            </div>

            {/* Role Description */}
            <div>
              <label htmlFor="roleDescription" className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Description
              </label>
              <Input
                id="roleDescription"
                type="text"
                placeholder="Enter role description (optional)"
                value={newRoleDescription}
                onChange={(e) => setNewRoleDescription(e.target.value)}
                className="w-full"
              />
            </div>

            {/* Permissions */}
            <div className='flex items-start gap-3'>
              <div>
                <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-3">
                  Organization Permissions
                </label>
                <div className="space-y-3">
                  {organizationPermissions.map((permission) => (
                    <div key={permission.id} className="flex items-start gap-3">
                      <Checkbox
                        checked={newRolePermissions.includes(permission.id)}
                        onChange={(checked) => handlePermissionChange(permission.id, checked)}
                        className="mt-0.5"
                      />
                      <div className="flex-1">
                        <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
                          {permission.name}
                        </div>
                        <div className="text-xs text-zinc-500 dark:text-zinc-400">
                          {permission.description}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-3">
                  Select organization-level permissions for this role.
                </p>
              </div>

              {/* Canvas Permissions */}
              <div>
                <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-3">
                  Canvas Permissions
                </label>
                <div className="space-y-3">
                  {canvasPermissions.map((permission) => (
                    <div key={permission.id} className="flex items-start gap-3">
                      <Checkbox
                        checked={newRolePermissions.includes(permission.id)}
                        onChange={(checked) => handlePermissionChange(permission.id, checked)}
                        className="mt-0.5"
                      />
                      <div className="flex-1">
                        <div className="text-sm font-medium text-zinc-900 dark:text-zinc-100">
                          {permission.name}
                        </div>
                        <div className="text-xs text-zinc-500 dark:text-zinc-400">
                          {permission.description}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400 mt-3">
                  Select canvas-specific permissions for this role.
                </p>
              </div>
            </div>
          </div>
        </DialogBody>
        <DialogActions>
          <Button plain onClick={handleCloseRoleModal}>
            Cancel
          </Button>
          <Button 
            color="blue" 
            onClick={handleSaveRole} 
            disabled={!newRoleName.trim() || newRolePermissions.length === 0}
          >
            Create Role
          </Button>
        </DialogActions>
      </Dialog>

    </div>
  )
}