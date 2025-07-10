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
import { Textarea } from './lib/Textarea/textarea'
import { 
  Table, 
  TableHead, 
  TableBody, 
  TableRow, 
  TableHeader, 
  TableCell 
} from './lib/Table/table'

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
  const [activeTab, setActiveTab] = useState<'profile' | 'general' | 'members' | 'groups' | 'roles' | 'tokens' | 'integrations' | 'api_token' | 'security'>('general')
  const [selectedTeam, setSelectedTeam] = useState<{ id: string; name: string; description: string } | null>(null)
  const [isCreatingRole, setIsCreatingRole] = useState(false)
  const [activeRoleTab, setActiveRoleTab] = useState<'organization' | 'canvas'>('organization')
  const [newRoleName, setNewRoleName] = useState('')
  const [newRoleDescription, setNewRoleDescription] = useState('')
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(new Set())
  
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

  // Mock data for members
  const members = [
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

  // Mock data for groups
  const groups = [
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
    setSelectedPermissions(new Set())
  }

  const handleSaveRole = () => {
    // Here you would typically save the role to your backend
    console.log('Creating role:', {
      name: newRoleName,
      description: newRoleDescription,
      permissions: Array.from(selectedPermissions)
    })
    handleBackToRoles()
  }

  const handlePermissionToggle = (permissionId: string) => {
    setSelectedPermissions(prev => {
      const newSet = new Set(prev)
      if (newSet.has(permissionId)) {
        newSet.delete(permissionId)
      } else {
        newSet.add(permissionId)
      }
      return newSet
    })
  }

  const handleCategoryToggle = (categoryPermissions: { id: string; name: string; description: string }[]) => {
    const allSelected = categoryPermissions.every(permission => selectedPermissions.has(permission.id))
    
    setSelectedPermissions(prev => {
      const newSet = new Set(prev)
      if (allSelected) {
        // Uncheck all permissions in this category
        categoryPermissions.forEach(permission => {
          newSet.delete(permission.id)
        })
      } else {
        // Check all permissions in this category
        categoryPermissions.forEach(permission => {
          newSet.add(permission.id)
        })
      }
      return newSet
    })
  }

  const isCategorySelected = (categoryPermissions: { id: string; name: string; description: string }[]) => {
    return categoryPermissions.every(permission => selectedPermissions.has(permission.id))
  }

  const isCategoryIndeterminate = (categoryPermissions: { id: string; name: string; description: string }[]) => {
    const selectedCount = categoryPermissions.filter(permission => selectedPermissions.has(permission.id)).length
    return selectedCount > 0 && selectedCount < categoryPermissions.length
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

  // Organization permissions data categorized
  const organizationPermissions = [
    {
      category: 'General',
      icon: 'business',
      permissions: [
        {
          id: 'organization.view',
          name: 'View Organization',
          description: 'Access to the organization. This permission is needed to access any page within the organization domain.'
        },
        {
          id: 'organization.general_settings.view',
          name: 'View General Settings',
          description: 'View general settings for the organization.'
        },
        {
          id: 'organization.general_settings.manage',
          name: 'Manage General Settings',
          description: 'Manage general settings of the organization.'
        },
        {
          id: 'organization.change_owner',
          name: 'Change Owner',
          description: 'Change the owner of the organization.'
        },
        {
          id: 'organization.delete',
          name: 'Delete Organization',
          description: 'Delete the organization.'
        }
      ]
    },
    {
      category: 'People & Groups',
      icon: 'groups',
      permissions: [
        {
          id: 'organization.people.view',
          name: 'View People',
          description: 'View list of people within the organization, together with the roles they have.'
        },
        {
          id: 'organization.people.invite',
          name: 'Invite People',
          description: 'Invite new people to the organization.'
        },
        {
          id: 'organization.people.manage',
          name: 'Manage People',
          description: 'Remove people from the organization, or change their roles within the organization.'
        },
        {
          id: 'organization.groups.view',
          name: 'View Groups',
          description: 'View user groups within the organization.'
        },
        {
          id: 'organization.groups.manage',
          name: 'Manage Groups',
          description: 'Manage groups within the organization and modify group members.'
        }
      ]
    },
    {
      category: 'Roles & Permissions',
      icon: 'admin_panel_settings',
      permissions: [
        {
          id: 'organization.custom_roles.view',
          name: 'View Custom Roles',
          description: 'View roles within the organization and permissions they carry.'
        },
        {
          id: 'organization.custom_roles.manage',
          name: 'Manage Custom Roles',
          description: 'Modify definition of roles within the organization.'
        }
      ]
    },
    {
      category: 'Projects & Resources',
      icon: 'folder',
      permissions: [
        {
          id: 'organization.projects.create',
          name: 'Create Projects',
          description: 'Create a new project within the organization.'
        },
        {
          id: 'organization.dashboards.view',
          name: 'View Dashboards',
          description: 'View the existing dashboards within the organization.'
        },
        {
          id: 'organization.dashboards.manage',
          name: 'Manage Dashboards',
          description: 'Create new dashboard views.'
        }
      ]
    },
    {
      category: 'Security & Compliance',
      icon: 'security',
      permissions: [
        {
          id: 'organization.audit_logs.view',
          name: 'View Audit Logs',
          description: 'View audit logs.'
        },
        {
          id: 'organization.audit_logs.manage',
          name: 'Manage Audit Logs',
          description: 'Manage audit log settings for the organization (such as log streams).'
        },
        {
          id: 'organization.activity_monitor.view',
          name: 'View Activity Monitor',
          description: 'View organization\'s activity monitor.'
        },
        {
          id: 'organization.secrets.view',
          name: 'View Secrets',
          description: 'View secrets within the organization.'
        },
        {
          id: 'organization.secrets.manage',
          name: 'Manage Secrets',
          description: 'Manage secrets within the organization.'
        },
        {
          id: 'organization.secrets_policy_settings.view',
          name: 'View Secrets Policy',
          description: 'View existing secrets policy settings.'
        },
        {
          id: 'organization.secrets_policy_settings.manage',
          name: 'Manage Secrets Policy',
          description: 'Manage secrets policy settings within the organization.'
        },
        {
          id: 'organization.ip_allow_list.view',
          name: 'View IP Allow List',
          description: 'View the IP allow list.'
        },
        {
          id: 'organization.ip_allow_list.manage',
          name: 'Manage IP Allow List',
          description: 'Modify the IP allow list for the organization.'
        }
      ]
    },
    {
      category: 'Integrations & External Services',
      icon: 'extension',
      permissions: [
        {
          id: 'organization.okta.view',
          name: 'View Okta Integration',
          description: 'View Okta integration settings for the organization.'
        },
        {
          id: 'organization.okta.manage',
          name: 'Manage Okta Integration',
          description: 'Modify existing Okta integrations, or create a new one.'
        },
        {
          id: 'organization.self_hosted_agents.view',
          name: 'View Self-Hosted Agents',
          description: 'View the list of self-hosted agents within the organization.'
        },
        {
          id: 'organization.self_hosted_agents.manage',
          name: 'Manage Self-Hosted Agents',
          description: 'Manage self-hosted agents within the organization.'
        },
        {
          id: 'organization.pre_flight_checks.view',
          name: 'View Pre-Flight Checks',
          description: 'View pre-flight checks within the organization.'
        },
        {
          id: 'organization.pre_flight_checks.manage',
          name: 'Manage Pre-Flight Checks',
          description: 'Modify pre-flight checks within the organization.'
        }
      ]
    },
    {
      category: 'Billing & Support',
      icon: 'payment',
      permissions: [
        {
          id: 'organization.plans_and_billing.view',
          name: 'View Plans & Billing',
          description: 'View the billing page.'
        },
        {
          id: 'organization.plans_and_billing.manage',
          name: 'Manage Plans & Billing',
          description: 'Modify billing information or subscription plan.'
        },
        {
          id: 'organization.contact_support',
          name: 'Contact Support',
          description: 'Contact support on behalf of the organization.'
        }
      ]
    },
    {
      category: 'Notifications',
      icon: 'notifications',
      permissions: [
        {
          id: 'organization.notifications.view',
          name: 'View Notifications',
          description: 'View organization notification settings.'
        },
        {
          id: 'organization.notifications.manage',
          name: 'Manage Notifications',
          description: 'Modify organization notification settings.'
        }
      ]
    }
  ]

  // Canvas permissions data categorized
  const canvasPermissions = [
    {
      category: 'Basic Operations',
      icon: 'palette',
      permissions: [
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
        }
      ]
    },
    {
      category: 'Sharing & Collaboration',
      icon: 'share',
      permissions: [
        {
          id: 'canvas_share',
          name: 'Share Canvases',
          description: 'Share canvases with others and manage access permissions'
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
    },
    {
      category: 'Export & Integration',
      icon: 'download',
      permissions: [
        {
          id: 'canvas_export',
          name: 'Export Canvases',
          description: 'Export canvases to various formats and download data'
        }
      ]
    }
  ]

  const tabs = [
    { id: 'profile', label: 'Profile', icon: 'person' },
    { id: 'general', label: 'General', icon: 'settings' },
    { id: 'members', label: 'Members', icon: 'group' },
    { id: 'groups', label: 'Groups', icon: 'group' },
    { id: 'roles', label: 'Roles', icon: 'admin_panel_settings' }
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
                      className="max-w-lg"
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                      Description
                    </label>
                    <Textarea
                      placeholder="Describe what this role can do"
                      value={newRoleDescription}
                      onChange={(e) => setNewRoleDescription(e.target.value)}
                      className="max-w-lg"
                    />
                  </div>
                

                {/* Permissions */}
               
                  <div className="pt-4 mb-4">
                    <Heading level={2} className="text-xl font-semibold text-zinc-900 dark:text-white mb-2">
                      {activeRoleTab === 'organization' ? 'Organization' : 'Canvas'} Permissions
                    </Heading>
                    <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                      Select the permissions this role should have {activeRoleTab === 'organization' ? 'within the organization' : 'for canvas operations'}.
                    </Text>
                  </div>
                  
                  <div className="space-y-6">
                    {(activeRoleTab === 'organization' ? organizationPermissions : canvasPermissions).map((category) => (
                      <div key={category.category} className="space-y-4">
                        <div className="flex items-center mb-3">
                          <div className="flex items-center">
                            <h3 className="text-md font-semibold text-zinc-900 dark:text-white">{category.category}</h3>
                          </div>
                          <Link 
                            href="#"
                            className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 ml-3"
                            onClick={() => handleCategoryToggle(category.permissions)}
                          >
                            {isCategorySelected(category.permissions) ? 'Deselect all' : 'Select all'}
                          </Link>
                        </div>
                        <div className="space-y-3">
                          {category.permissions.map((permission) => (
                            <CheckboxField key={permission.id}>
                              <Checkbox 
                                name={permission.id} 
                                checked={selectedPermissions.has(permission.id)}
                                onChange={() => handlePermissionToggle(permission.id)}
                              />
                              <Label>{permission.name}</Label>
                              <Description>{permission.description}</Description>
                            </CheckboxField>
                          ))}
                        </div>
                      </div>
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
                  Roles
                </Heading>
              </div>
              
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
              <div className="px-6 py-4 flex items-center justify-between">
                <InputGroup>
                  <Input name="search" placeholder="Search Roles…" aria-label="Search" className="w-xs" />
                </InputGroup>
                <Button color="blue" className='flex items-center' onClick={handleCreateRole}>
                  <MaterialSymbol name="add" />
                  {buttonText}
                </Button>
              </div>
              <div className="px-6">
                <Table dense>
                  <TableHead>
                    <TableRow>
                      <TableHeader>Role name</TableHeader>
                      <TableHeader>Permissions</TableHeader>
                      <TableHeader>Status</TableHeader>
                      <TableHeader></TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {currentRoles.map((role) => (
                      <TableRow key={role.id}>
                        <TableCell className="font-medium">
                          {role.name}
                        </TableCell>
                        <TableCell>
                          {role.permissions}
                        </TableCell>
                        <TableCell>
                          <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400">
                            {role.status}
                          </span>
                        </TableCell>
                        <TableCell>
                          <div className="flex justify-end">
                            <Dropdown>
                              <DropdownButton  plain>
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
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        )

      case 'members':
        return (
          <div className="space-y-6 pt-6">
            <div className="flex items-center justify-between">
              
            <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
              members
            </Heading>
            </div>

            {/* Add Members Section */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
              <div className="flex items-center justify-between mb-4">
                <div>
                  <Subheading level={3} className="text-lg font-semibold text-zinc-900 dark:text-white mb-1">
                    Add members
                  </Subheading>
                  
                </div>
                
              </div>
              
              <div className="flex gap-3">
                <Input
                  type="email"
                  placeholder="Enter email address"
                  className="flex-1"
                />
                <Dropdown>
                  <DropdownButton  outline className="flex items-center text-sm">
                    Member
                    <MaterialSymbol name="keyboard_arrow_down" />
                  </DropdownButton>
                  <DropdownMenu>
                    <DropdownItem>Member</DropdownItem>
                    <DropdownItem>Admin</DropdownItem>
                    <DropdownItem>Owner</DropdownItem>
                  </DropdownMenu>
                </Dropdown>
                <Button color="blue">Send Invite</Button>
              </div>
            </div>

            {/* Members List */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
              <div className="px-6 py-4 ">
                <div className="flex items-center justify-between">
                  <InputGroup>
                    <Input name="search" placeholder="Search members…" aria-label="Search" className="w-xs" />
                  </InputGroup>
                </div>
              </div>
              <div className="px-6">
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
                    {members.map((user) => (
                      <TableRow key={user.id}>
                        <TableCell>
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
                        </TableCell>
                        <TableCell>
                          {user.email}
                        </TableCell>
                        <TableCell>
                          <Dropdown>
                            <DropdownButton  outline className="flex items-center gap-2 text-sm">
                              {user.role}
                              <MaterialSymbol name="keyboard_arrow_down" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem>Owner</DropdownItem>
                              <DropdownItem>Admin</DropdownItem>
                              <DropdownItem>Member</DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </TableCell>
                        <TableCell>
                          <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${
                            user.status === 'Active'
                              ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                              : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
                          }`}>
                            {user.status}
                          </span>
                        </TableCell>
                        <TableCell>
                          <div className="flex justify-end">
                            <Dropdown>
                              <DropdownButton  plain className="flex items-center gap-2 text-sm">
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
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        )

      case 'groups':
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
                    Add members
                  </Subheading>
                  
                </div>
                
              </div>
              
              <div className="flex gap-3">
                <Input
                  type="email"
                  placeholder="Enter email address"
                  className="flex-1"
                />
                <Dropdown>
                  <DropdownButton  outline className="flex items-center gap-2 text-sm">
                    Member
                    <MaterialSymbol name="keyboard_arrow_down" />
                  </DropdownButton>
                  <DropdownMenu>
                    <DropdownItem>Member</DropdownItem>
                    <DropdownItem>Admin</DropdownItem>
                    <DropdownItem>Owner</DropdownItem>
                  </DropdownMenu>
                </Dropdown>
                <Button color="blue">Send Invite</Button>
              </div>
            </div>
              {/* Team members table */}
              <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
                <div className="px-6 py-4 ">
                  <div className="flex items-center justify-between">
                    <InputGroup>
                      <Input name="search" placeholder="Search team members…" aria-label="Search" className="w-xs" />
                    </InputGroup>
                  </div>
                </div>
                <div className="px-6">
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
                      {teamMembers.map((member) => (
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
                                  Joined {member.joinedDate}
                                </div>
                              </div>
                            </div>
                          </TableCell>
                          <TableCell>
                            {member.email}
                          </TableCell>
                          <TableCell>
                            <Dropdown>
                              <DropdownButton  outline className="flex items-center gap-2 text-sm">
                                {member.role}
                                <MaterialSymbol name="keyboard_arrow_down" />
                              </DropdownButton>
                              <DropdownMenu>
                                <DropdownItem>Team Lead</DropdownItem>
                                <DropdownItem>Senior Developer</DropdownItem>
                                <DropdownItem>Developer</DropdownItem>
                                <DropdownItem>Designer</DropdownItem>
                                <DropdownItem>DevOps Engineer</DropdownItem>
                              </DropdownMenu>
                            </Dropdown>
                          </TableCell>
                          <TableCell>
                            <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400">
                              {member.status}
                            </span>
                          </TableCell>
                          <TableCell>
                            <div className="flex justify-end">
                              <Dropdown>
                                <DropdownButton  plain className="flex items-center gap-2 text-sm">
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
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
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
              
            </div>

            {/* Teams Table View */}
            <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 overflow-hidden">
              <div className="px-6 py-4  flex items-center justify-between">
                <InputGroup>
                  <Input name="search" placeholder="Search Groups&hellip;" aria-label="Search" className="w-xs" />
                </InputGroup>
                <Button color="blue" className='flex items-center'>
                  <MaterialSymbol name="add" />
                  Create New Group
                </Button>
              </div>
              <div className="px-6">
                <Table dense>
                  <TableHead>
                    <TableRow>
                      <TableHeader>Team name</TableHeader>
                      <TableHeader>Description</TableHeader>
                      <TableHeader>Members</TableHeader>
                      <TableHeader>Role</TableHeader>
                      <TableHeader></TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {groups.map((team) => (
                      <TableRow 
                        key={team.id} 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleTeamClick(team)}
                      >
                        <TableCell>
                          <div className="flex items-center gap-3">
                            <Avatar className='w-9' square initials={team.name.charAt(0)} />
                            <div>
                              <Link href={`#`} className="cursor-pointer text-sm font-medium text-blue-600 dark:text-blue-400">
                                {team.name}
                              </Link>
                              <div className="text-xs text-zinc-500 dark:text-zinc-400">
                                Created {team.created}
                              </div>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          {team.description}
                        </TableCell>
                        <TableCell>
                          {team.memberCount} members
                        </TableCell>
                        <TableCell>
                          <Dropdown>
                            <DropdownButton  outline 
                              className="flex items-center gap-2 text-sm"
                              onClick={(e: React.MouseEvent) => e.stopPropagation()}
                            >
                              {team.role}
                              <MaterialSymbol name="keyboard_arrow_down" />
                            </DropdownButton>
                            <DropdownMenu>
                              <DropdownItem>Admin</DropdownItem>
                              <DropdownItem>Member</DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </TableCell>
                        <TableCell>
                          <div className="flex justify-end">
                            <Dropdown>
                              <DropdownButton  plain onClick={(e: React.MouseEvent) => e.stopPropagation()}>
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
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
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
                  className="max-w-lg"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                  Description
                </label>
                <Textarea
                  placeholder="Enter organization description"
                  className="max-w-lg"
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                  Company Logo
                </label>
                <div className="flex items-center gap-4">
                  <div className="flex items-center justify-center w-16 h-16 bg-zinc-100 dark:bg-zinc-700 rounded-lg border border-zinc-200 dark:border-zinc-600">
                    <MaterialSymbol name="business" className="text-zinc-400 dark:text-zinc-500" />
                  </div>
                  <div className="flex flex-col gap-2">
                    <Button plain className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                      <MaterialSymbol name="upload" size="sm" className="mr-2" />
                      Upload logo
                    </Button>
                    <p className="text-xs text-zinc-500 dark:text-zinc-400">
                      Recommended: Square image, at least 256x256px
                    </p>
                  </div>
                </div>
              </div>
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
              onClick={() => setActiveTab('api_token')}
              className={`w-full text-left px-3 py-2 text-sm rounded-md transition-colors ${
                activeTab === 'api_token'
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