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
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from './lib/Dropdown/dropdown'
import { NavigationOrg } from './lib/Navigation/navigation-org'
import { Breadcrumbs } from './lib/Breadcrumbs/breadcrumbs'
import { Link } from './lib/Link/link'
import { Checkbox, CheckboxField } from './lib/Checkbox/checkbox'
import { Description, Field, Fieldset, Label } from './lib/Fieldset/fieldset'
import { ControlledTabs, Tabs, type Tab } from './lib/Tabs/tabs'
import { Textarea } from './lib/Textarea/textarea'
import { AddMembersSection } from './AddMembersSection'
import { 
  Table, 
  TableHead, 
  TableBody, 
  TableRow, 
  TableHeader, 
  TableCell 
} from './lib/Table/table'
import { 
  Dialog, 
  DialogTitle, 
  DialogDescription, 
  DialogBody, 
  DialogActions 
} from './lib/Dialog/dialog'
import { Sidebar, SidebarBody, SidebarDivider, SidebarHeader, SidebarItem, SidebarLabel, SidebarSection, SidebarSpacer } from './lib/Sidebar/sidebar'
import { useToast, ToastProvider } from './lib/Toasts/toast'
import { AddMembersSectionSimple } from './AddMembersSectionSimple'

interface OrganizationSettingsProps {
  onBack?: () => void
  onSignOut?: () => void
  onSwitchOrganization?: () => void
}

function OrganizationSettingsInner({ 
  onBack, 
  onSignOut, 
  onSwitchOrganization 
}: OrganizationSettingsProps) {
  const { addToast } = useToast();
  const [activeTab, setActiveTab] = useState<'profile' | 'general' | 'members' | 'groups' | 'roles' | 'tokens' | 'integrations' | 'api_token' | 'security'>('general')
  const [selectedTeam, setSelectedTeam] = useState<{ id: string; name: string; description: string } | null>(null)
  const [isCreatingRole, setIsCreatingRole] = useState(false)
  const [activeRoleTab, setActiveRoleTab] = useState<'organization' | 'canvas'>('organization')
  const [newRoleName, setNewRoleName] = useState('')
  const [newRoleDescription, setNewRoleDescription] = useState('')
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(new Set())
  const [sortConfig, setSortConfig] = useState<{
    key: string | null
    direction: 'asc' | 'desc'
    table: 'roles' | 'members' | 'teams' | 'teamMembers'
  }>({
    key: null,
    direction: 'asc',
    table: 'roles'
  })
  
  // Inline editing state
  const [isEditingGroupName, setIsEditingGroupName] = useState(false)
  const [isEditingGroupDescription, setIsEditingGroupDescription] = useState(false)
  const [editedGroupName, setEditedGroupName] = useState('')
  const [editedGroupDescription, setEditedGroupDescription] = useState('')
  
  // Create group modal state
  const [showCreateGroupModal, setShowCreateGroupModal] = useState(false)
  const [isCreatingGroup, setIsCreatingGroup] = useState(false)
  const [newGroupName, setNewGroupName] = useState('')
  const [newGroupDescription, setNewGroupDescription] = useState('')
  const [newGroupRole, setNewGroupRole] = useState('Member')
  
  
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

  // Mock data for members - now stateful
  const [members, setMembers] = useState([
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
      status: 'Invited',
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
  ])

  // Function to add new members
  const handleAddMembers = (emails: string, role: string) => {
    // Parse emails (split by comma, semicolon, space, or newline)
    const emailList = emails
      .split(/[,;\s\n]+/)
      .map(email => email.trim())
      .filter(email => email.length > 0 && email.includes('@'))

    if (emailList.length === 0) {
      addToast('error', 'Invalid emails', 'Please enter valid email addresses');
      return 0;
    }

    // Create new member objects
    const newMembers = emailList.map((email, index) => ({
      id: `invited-${Date.now()}-${index}`,
      name: email.split('@')[0], // Use part before @ as temporary name
      email: email,
      role: role,
      status: 'Invited' as const,
      lastActive: 'Never',
      initials: email.charAt(0).toUpperCase() + (email.split('@')[0].charAt(1) || email.charAt(1)).toUpperCase()
    }))

    // Add new members to the existing list
    setMembers(prevMembers => [...prevMembers, ...newMembers])
    
    // Show success toast
    addToast(
      'success', 
      'Members invited successfully', 
      `${newMembers.length} member${newMembers.length > 1 ? 's' : ''} invited with ${role} role`
    );
    
    return newMembers.length // Return count of added members
  }

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
    avatar: 'https://confluent.io/favicon.ico',
    initials: 'C'
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

  // Inline editing handlers
  const handleEditGroupName = () => {
    if (selectedTeam) {
      setEditedGroupName(selectedTeam.name)
      setIsEditingGroupName(true)
    }
  }

  const handleSaveGroupName = () => {
    if (selectedTeam && editedGroupName.trim()) {
      // Update the group name - in a real app this would call an API
      setSelectedTeam({ ...selectedTeam, name: editedGroupName.trim() })
      setIsEditingGroupName(false)
      console.log('Saving group name:', editedGroupName)
    }
  }

  const handleCancelGroupName = () => {
    setIsEditingGroupName(false)
    setEditedGroupName('')
  }

  const handleEditGroupDescription = () => {
    if (selectedTeam) {
      setEditedGroupDescription(selectedTeam.description)
      setIsEditingGroupDescription(true)
    }
  }

  const handleSaveGroupDescription = () => {
    if (selectedTeam && editedGroupDescription.trim()) {
      // Update the group description - in a real app this would call an API
      setSelectedTeam({ ...selectedTeam, description: editedGroupDescription.trim() })
      setIsEditingGroupDescription(false)
      console.log('Saving group description:', editedGroupDescription)
    }
  }

  const handleCancelGroupDescription = () => {
    setIsEditingGroupDescription(false)
    setEditedGroupDescription('')
  }

  // Create group modal handlers
  const handleCreateGroupClick = () => {
    setIsCreatingGroup(true)
  }

  const handleCreateGroupCancel = () => {
    setShowCreateGroupModal(false)
    setIsCreatingGroup(false)
    setNewGroupName('')
    setNewGroupDescription('')
    setNewGroupRole('Member')
  }

  const handleBackToGroups = () => {
    setIsCreatingGroup(false)
    setNewGroupName('')
    setNewGroupDescription('')
    setNewGroupRole('Member')
  }

  const handleCreateGroupSubmit = () => {
    if (newGroupName.trim()) {
      // Create new group object
      const newGroup = {
        id: (groups.length + 1).toString(),
        name: newGroupName.trim(),
        description: newGroupDescription.trim() || 'No description provided',
        memberCount: 0,
        created: 'Just now'
      }
      
      // Set as selected team to show the group page
      setSelectedTeam(newGroup)
      
      // Close modal/form and reset
      setShowCreateGroupModal(false)
      setIsCreatingGroup(false)
      setNewGroupName('')
      setNewGroupDescription('')
      setNewGroupRole('Member')
      
      console.log('Created new group:', newGroup)
    }
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

  const handleSort = (key: string, table: 'roles' | 'members' | 'teams' | 'teamMembers') => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc',
      table
    }))
  }

  const getSortedData = (data: any[], table: 'roles' | 'members' | 'teams' | 'teamMembers') => {
    if (!sortConfig.key || sortConfig.table !== table) return data

    return [...data].sort((a, b) => {
      const aValue = a[sortConfig.key!]
      const bValue = b[sortConfig.key!]
      
      if (aValue < bValue) {
        return sortConfig.direction === 'asc' ? -1 : 1
      }
      if (aValue > bValue) {
        return sortConfig.direction === 'asc' ? 1 : -1
      }
      return 0
    })
  }

  const getSortIcon = (columnKey: string, table: 'roles' | 'members' | 'teams' | 'teamMembers') => {
    if (sortConfig.table !== table || sortConfig.key !== columnKey) {
      return 'unfold_more'
    }
    return sortConfig.direction === 'asc' ? 'keyboard_arrow_up' : 'keyboard_arrow_down'
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
                  { label: 'Roles', onClick: handleBackToRoles },
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
                <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 space-y-4">
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
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
              <div className="px-6 pt-6 pb-4 flex items-center justify-between">
                <InputGroup>
                  <Input name="search" placeholder="Search Roles…" aria-label="Search" className="w-xs" />
                </InputGroup>
                <Button color="blue" className='flex items-center' onClick={handleCreateRole}>
                  <MaterialSymbol name="add" />
                  {buttonText}
                </Button>
              </div>
              <div className="px-6 pb-6">
                <Table dense>
                  <TableHead>
                    <TableRow>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('name', 'roles')}
                      >
                        <div className="flex items-center gap-2">
                          Role name
                          <MaterialSymbol name={getSortIcon('name', 'roles')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('permissions', 'roles')}
                      >
                        <div className="flex items-center gap-2">
                          Permissions
                          <MaterialSymbol name={getSortIcon('permissions', 'roles')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('status', 'roles')}
                      >
                        <div className="flex items-center gap-2">
                          Status
                          <MaterialSymbol name={getSortIcon('status', 'roles')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader></TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {getSortedData(currentRoles, 'roles').map((role) => (
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
              Members
            </Heading>
            </div>
            <AddMembersSection onAddMembers={handleAddMembers}/>

            {/* Members List */}
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
              <div className="px-6 pt-6 pb-4 ">
                <div className="flex items-center">
                  <InputGroup>
                    <Input name="search" placeholder="Search members…" aria-label="Search" className="w-xs" />
                  </InputGroup>
                  <Dropdown>
                    <DropdownButton color='white' className="d-flex items-center ml-2">
                     Status 
                     <MaterialSymbol name="expand_more" size="md" />
                    </DropdownButton>
                    <DropdownMenu anchor="bottom start">
                      <DropdownItem>
                        All
                      </DropdownItem>
                      <DropdownItem>
                        Active
                      </DropdownItem>
                      <DropdownItem>
                        Invited
                      </DropdownItem>
                    </DropdownMenu>
                  </Dropdown>
                </div>
              </div>
              <div className="px-6 pb-6">
                <Table dense>
                  <TableHead>
                    <TableRow>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('name', 'members')}
                      >
                        <div className="flex items-center gap-2">
                          Name
                          <MaterialSymbol name={getSortIcon('name', 'members')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('email', 'members')}
                      >
                        <div className="flex items-center gap-2">
                          Email
                          <MaterialSymbol name={getSortIcon('email', 'members')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('role', 'members')}
                      >
                        <div className="flex items-center gap-2">
                          Role
                          <MaterialSymbol name={getSortIcon('role', 'members')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('status', 'members')}
                      >
                        <div className="flex items-center gap-2">
                          Status
                          <MaterialSymbol name={getSortIcon('status', 'members')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader></TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {getSortedData(members, 'members').map((user) => (
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
                                {user.status === 'Invited' ? (  
                                  "Invited on Tue Aug 20 2025, 10:30 AM"
                                ) : (
                                  "Last active on Tue Aug 20 2025, 10:30 AM"
                                )}
                              </div>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          {user.email}
                        </TableCell>
                        <TableCell>
                     
                            <Dropdown>
                              <DropdownButton  outline className="flex items-center gap-2 text-sm" disabled={user.status === 'Invited'}>
                                {user.role}
                                <MaterialSymbol name="keyboard_arrow_down" />
                              </DropdownButton>
                              <DropdownMenu anchor="bottom start">
                                <DropdownItem>
                                  <DropdownLabel>Owner</DropdownLabel>
                                  <DropdownDescription>Owner role description.</DropdownDescription>
                                </DropdownItem>
                                <DropdownItem>
                                  <DropdownLabel>Admin</DropdownLabel>
                                  <DropdownDescription>Admin role description.</DropdownDescription>
                                </DropdownItem>
                                <DropdownItem>
                                  <DropdownLabel>Member</DropdownLabel>
                                  <DropdownDescription>Member role description.</DropdownDescription>
                                </DropdownItem>
                              </DropdownMenu>
                            </Dropdown>
                          
                        </TableCell>
                        <TableCell>
                          <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${
                            user.status === 'Active'
                              ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                              : user.status === 'Invited'
                              ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
                              : 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400'
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
                              <DropdownMenu anchor="bottom-end">
                                {user.status === 'Invited' ? (
                                  <>
                                  <DropdownItem className='flex items-center gap-2'>
                                    <MaterialSymbol name="refresh " />
                                    Resend invitation
                                  </DropdownItem>
                                  <DropdownItem className='!text-red-600 dark:!text-red-400 hover:!bg-red-50 dark:hover:!bg-red-900/20 flex items-center gap-2'>
                                    <MaterialSymbol name="person_cancel" />
                                    Revoke invitation
                                  </DropdownItem>
                                  </>
                                ) : (
                                <DropdownItem className='!text-red-600 dark:!text-red-400 hover:!bg-red-50 dark:hover:!bg-red-900/20 flex items-center gap-2'>
                                  <MaterialSymbol name="delete" />
                                  Remove
                                </DropdownItem>
                                )}
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
                  { label: 'Groups', onClick: handleBackToTeams },
                  { label: selectedTeam.name, current: true }
                ]}
                showDivider={false}
              />
              <div className="bg-zinc-100 dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 space-y-6">
              {/* Team header */}
              <div className='flex items-start justify-between'>
                <div className='flex items-start gap-3'>
                  <Avatar className='w-12 bg-blue-200 dark:bg-blue-800 border border-blue-300 dark:border-blue-700' square initials={selectedTeam.name.charAt(0)}/>
                  <div className='flex flex-col space-y-2'>
                    {/* Group Name - Inline Edit */}
                    <div className="group">
                      {isEditingGroupName ? (
                        <div className="flex items-center gap-2">
                          <Input
                            type="text"
                            value={editedGroupName}
                            onChange={(e) => setEditedGroupName(e.target.value)}
                            className="text-2xl font-semibold bg-white dark:bg-zinc-800 border-zinc-300 dark:border-zinc-600"
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') handleSaveGroupName()
                              if (e.key === 'Escape') handleCancelGroupName()
                            }}
                            autoFocus
                          />
                          <Button plain onClick={handleSaveGroupName} className="text-green-600 hover:text-green-700">
                            <MaterialSymbol name="check" size="sm" />
                          </Button>
                          <Button plain onClick={handleCancelGroupName} className="text-red-600 hover:text-red-700">
                            <MaterialSymbol name="close" size="sm" />
                          </Button>
                        </div>
                      ) : (
                        <div className="flex items-center gap-2">
                          <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                            {selectedTeam.name}
                          </Heading>
                          <Button 
                            plain 
                            onClick={handleEditGroupName}
                            className="opacity-0 group-hover:opacity-100 transition-opacity text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                          >
                            <MaterialSymbol name="edit" size="sm" />
                          </Button>
                        </div>
                      )}
                    </div>

                    {/* Group Description - Inline Edit */}
                    <div className="group">
                      {isEditingGroupDescription ? (
                        <div className="flex items-center gap-2">
                          <Input
                            type="text"
                            value={editedGroupDescription}
                            onChange={(e) => setEditedGroupDescription(e.target.value)}
                            className="text-lg bg-white dark:bg-zinc-800 border-zinc-300 dark:border-zinc-600"
                            onKeyDown={(e) => {
                              if (e.key === 'Enter') handleSaveGroupDescription()
                              if (e.key === 'Escape') handleCancelGroupDescription()
                            }}
                            autoFocus
                          />
                          <Button plain onClick={handleSaveGroupDescription} className="text-green-600 hover:text-green-700">
                            <MaterialSymbol name="check" size="sm" />
                          </Button>
                          <Button plain onClick={handleCancelGroupDescription} className="text-red-600 hover:text-red-700">
                            <MaterialSymbol name="close" size="sm" />
                          </Button>
                        </div>
                      ) : (
                        <div className="flex items-center gap-2">
                          <Subheading level={3} className="text-lg !font-normal text-zinc-600 dark:text-zinc-400">
                            {selectedTeam.description}
                          </Subheading>
                          <Button 
                            plain 
                            onClick={handleEditGroupDescription}
                            className="opacity-0 group-hover:opacity-100 transition-opacity text-zinc-400 hover:text-zinc-600 dark:hover:text-zinc-300"
                          >
                            <MaterialSymbol name="edit" size="sm" />
                          </Button>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
                <div className='flex items-center gap-2'>
                <Dropdown>
                  <DropdownButton  color='white'
                    className="flex items-center gap-2 text-sm"
                    onClick={(e: React.MouseEvent) => e.stopPropagation()}
                  >
                    Admin
                    <MaterialSymbol name="keyboard_arrow_down" />
                  </DropdownButton>
                  <DropdownMenu anchor="bottom start">
                    <DropdownItem>
                      <DropdownLabel>Admin</DropdownLabel>
                      <DropdownDescription>Admin role description.</DropdownDescription>
                    </DropdownItem>
                    <DropdownItem>
                      <DropdownLabel>Member</DropdownLabel>
                      <DropdownDescription>Member role description.</DropdownDescription>
                    </DropdownItem>
                  </DropdownMenu>
                </Dropdown>
                <Dropdown>
                <DropdownButton plain aria-label="More options">
                  <MaterialSymbol name="more_vert" size="sm" />
                </DropdownButton>
                <DropdownMenu>
                  <DropdownItem href="">Delete group</DropdownItem>
                </DropdownMenu>
                </Dropdown>
                </div>
              </div>
              {/* Add Members Section */}
              <AddMembersSectionSimple  showRoleSelection={false}/>
              {/* Team members table */}
              <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
                <div className="px-6 pt-6 pb-4 ">
                  <div className="flex items-center justify-between">
                    <InputGroup>
                      <Input name="search" placeholder="Search team members…" aria-label="Search" className="w-xs" />
                    </InputGroup>
                  </div>
                </div>
                <div className="px-6 pb-6">
                  <Table dense>
                    <TableHead>
                      <TableRow>
                        <TableHeader 
                          className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                          onClick={() => handleSort('name', 'teamMembers')}
                        >
                          <div className="flex items-center gap-2">
                            Name
                            <MaterialSymbol name={getSortIcon('name', 'teamMembers')} size="sm" className="text-zinc-400" />
                          </div>
                        </TableHeader>
                        <TableHeader 
                          className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                          onClick={() => handleSort('email', 'teamMembers')}
                        >
                          <div className="flex items-center gap-2">
                            Email
                            <MaterialSymbol name={getSortIcon('email', 'teamMembers')} size="sm" className="text-zinc-400" />
                          </div>
                        </TableHeader>
                        
                        <TableHeader 
                          className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                          onClick={() => handleSort('status', 'teamMembers')}
                        >
                          <div className="flex items-center gap-2">
                            Status
                            <MaterialSymbol name={getSortIcon('status', 'teamMembers')} size="sm" className="text-zinc-400" />
                          </div>
                        </TableHeader>
                        <TableHeader></TableHeader>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {getSortedData(teamMembers, 'teamMembers').map((member) => (
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
                      {(teamMembers.length === 0) && (
                        <TableRow>
                          <TableCell colSpan={5} className="text-center h-[200px] py-6">
                            No team members yet
                          </TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </div>
              </div>
            </div>
            </div>
          )
        }

        // Groups list view or create group view
        if (isCreatingGroup) {
          // Create group view
          return (
            <div className="space-y-6 pt-6">
              {/* Breadcrumbs navigation */}
              <Breadcrumbs
                items={[
                  { label: 'Groups', onClick: handleBackToGroups },
                  { label: 'Create new group', current: true }
                ]}
                showDivider={false}
              />
              
              {/* Create Group Form */}
             
                <div>
                  <Heading level={2} className="text-xl font-semibold text-zinc-900 dark:text-white mb-2">
                    Create new group
                  </Heading>
                  <Text className="text-zinc-600 dark:text-zinc-400">
                    Groups help you organize team members and manage permissions effectively.
                  </Text>
                </div>
                <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 space-y-6">
                {/* Group Name */}
                <Field className="max-w-lg">
                  <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                    Group name *
                  </Label>
                  <Input
                    type="text"
                    value={newGroupName}
                    onChange={(e) => setNewGroupName(e.target.value)}
                    placeholder="Enter group name"
                    className="w-full"
                  />
                </Field>

                {/* Group Description */}
                <Field className="max-w-lg">
                  <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                    Group description
                  </Label>
                  <Textarea
                    value={newGroupDescription}
                    onChange={(e) => setNewGroupDescription(e.target.value)}
                    placeholder="Describe the purpose and responsibilities of this group"
                    rows={3}
                    className="w-full"
                  />
                </Field>

                {/* Group Role */}
                <Field className="max-w-lg">
                  <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                    Role
                  </Label>
                  <Dropdown>
                    <DropdownButton outline className="flex items-center gap-2 text-sm justify-between">
                      {newGroupRole}
                      <MaterialSymbol name="keyboard_arrow_down" />
                    </DropdownButton>
                    <DropdownMenu>
                      <DropdownItem onClick={() => setNewGroupRole('Owner')}>
                        <DropdownLabel>Owner</DropdownLabel>
                        <DropdownDescription>Full access to organization settings</DropdownDescription>
                      </DropdownItem>
                      <DropdownItem onClick={() => setNewGroupRole('Admin')}>
                        <DropdownLabel>Admin</DropdownLabel>
                        <DropdownDescription>Can manage members and organization settings</DropdownDescription>
                      </DropdownItem>
                      <DropdownItem onClick={() => setNewGroupRole('Member')}>
                        <DropdownLabel>Member</DropdownLabel>
                        <DropdownDescription>Standard member access</DropdownDescription>
                      </DropdownItem>
                    </DropdownMenu>
                  </Dropdown>
                </Field>
                </div>
                {/* Action Buttons */}
                <div className="flex gap-3 pt-4 justify-end">
                  
                  <Button 
                    plain 
                    onClick={handleBackToGroups}
                    className="flex items-center gap-2"
                  >
                   
                    Cancel
                  </Button>
                  <Button 
                    color="blue" 
                    onClick={handleCreateGroupSubmit}
                    disabled={!newGroupName.trim()}
                    className="flex items-center gap-2"
                  >
                    <MaterialSymbol name="add" size="sm" />
                    Create Group
                  </Button>
                </div>
            </div>
          )
        }

        // Groups list view
        return (
          <div className="space-y-6 pt-6">
            <div className="flex items-center justify-between">
              <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
                Groups
              </Heading>
              
            </div>

            {/* Groups Table View */}
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
              <div className="px-6 pt-6 pb-4  flex items-center justify-between">
                <InputGroup>
                  <Input name="search" placeholder="Search Groups&hellip;" aria-label="Search" className="w-xs" />
                </InputGroup>
                <Button color="blue" className='flex items-center' onClick={handleCreateGroupClick}>
                  <MaterialSymbol name="add" />
                  Create New Group
                </Button>
              </div>
              <div className="px-6 pb-6">
                <Table dense>
                  <TableHead>
                    <TableRow>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('name', 'teams')}
                      >
                        <div className="flex items-center gap-2">
                          Team name
                          <MaterialSymbol name={getSortIcon('name', 'teams')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('description', 'teams')}
                      >
                        <div className="flex items-center gap-2">
                          Created
                          <MaterialSymbol name={getSortIcon('description', 'teams')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('memberCount', 'teams')}
                      >
                        <div className="flex items-center gap-2">
                          Members
                          <MaterialSymbol name={getSortIcon('memberCount', 'teams')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader 
                        className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                        onClick={() => handleSort('role', 'teams')}
                      >
                        <div className="flex items-center gap-2">
                          Role
                          <MaterialSymbol name={getSortIcon('role', 'teams')} size="sm" className="text-zinc-400" />
                        </div>
                      </TableHeader>
                      <TableHeader></TableHeader>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {getSortedData(groups, 'teams').map((team) => (
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
                                {team.description}
                              </div>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                         {team.created}
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
                              <DropdownItem>
                                  <DropdownLabel>Admin</DropdownLabel>
                                <DropdownDescription>Admin role description.</DropdownDescription>
                              </DropdownItem>
                              <DropdownItem>
                                <DropdownLabel>Member</DropdownLabel>
                                <DropdownDescription>Member role description.</DropdownDescription>
                              </DropdownItem>
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
              General
            </Heading>
            <Fieldset className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 space-y-6 max-w-xl">
              <Field>
                <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                  Organization Name
                </Label>
                <Input
                  type="text"
                  defaultValue={currentOrganization.name}
                  className="max-w-lg"
                />
              </Field>
              <Field>
                <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                  Description
                </Label>
                <Textarea
                  placeholder="Enter organization description"
                  className="max-w-lg"
                />
              </Field>
              <Field>
                
                <div className="flex items-start gap-4">
                  
                  <div className='w-1/2 flex-col gap-2'>
                  <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                    Company Logo
                  </Label>
                    <div className="flex-none grow-0">
                      <div className="inline-block h-15 py-4 bg-white dark:bg-zinc-700 rounded-lg border border-zinc-200 dark:border-zinc-600 border-dashed px-4">  
                        <img
                            src="https://upload.wikimedia.org/wikipedia/commons/a/ab/Confluent%2C_Inc._logo.svg"
                            alt="Confluent, Inc."
                            className='h-full'
                          />
                      </div>
                      <div className="flex items-center gap-2">
                        <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                          Upload new 
                        </Link>
                        <span className="text-xs text-zinc-500 dark:text-zinc-400">
                          &bull;
                        </span>
                        <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                          Remove  
                        </Link>
                      </div>
                      <p className="text-xs text-zinc-500 dark:text-zinc-400">
                        Rectangle image 96X20px
                      </p>
                    </div>
                  </div>
                  <div className='w-1/2 flex-col gap-2'>
                  <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                    Company Icon
                  </Label> 
                    <div className="flex-none grow-0 ">
                      <div className="w-15 h-15inline-block py-4 bg-white dark:bg-zinc-700 rounded-lg border border-zinc-200 dark:border-zinc-600 border-dashed px-4">
                        <img
                            src="https://confluent.io/favicon.ico"
                            alt="Confluent, Inc."
                            height={24}
                          />
                      </div>
                    </div>
                    <div className="flex flex-col">
                      <div className="flex items-center gap-2">
                        <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                          Upload new 
                        </Link>
                        <span className="text-xs text-zinc-500 dark:text-zinc-400">
                          &bull;
                        </span>
                        <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                          Remove  
                        </Link>
                      </div>
                      <p className="text-xs text-zinc-500 dark:text-zinc-400">
                        Square image 64X64px
                      </p>
                    </div>
                  </div>
                </div>
              </Field>
              
            </Fieldset>
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
    <div className="flex flex-col h-screen bg-zinc-50 dark:bg-zinc-950">
      {/* Navigation */}
      <NavigationOrg
        user={currentUser}
        organization={currentOrganization}
        onUserMenuAction={handleUserMenuAction}
        onOrganizationMenuAction={handleOrganizationMenuAction}
      />
      
      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <Sidebar className='w-70 bg-white dark:bg-zinc-950 border-r bw-1 border-zinc-200 dark:border-zinc-800'>
          
          <SidebarBody>
            
              <SidebarSection>
                <div className='flex items-center gap-3 text-sm font-bold py-3'>
                    <Avatar 
                        className='w-6 h-6'
                        src="https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&amp;h=64&amp;fit=crop&amp;crop=face"
                        alt="My Account"
                      />
                    <SidebarLabel className='text-zinc-900 dark:text-white'>My Account</SidebarLabel>
                </div>
                <SidebarItem className={`${activeTab === 'profile' ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`} onClick={() => setActiveTab('profile')}>
                <span className='px-7'>
                  <SidebarLabel>My Profile</SidebarLabel>
                </span>
                </SidebarItem>
                <SidebarItem className={`${activeTab === 'api_token' ? 'bg-zinc-100 dark:bg-zinc-800 rounded-md' : ''}`} onClick={() => setActiveTab('api_token')}>
                <span className='px-7'>
                <SidebarLabel>API Token</SidebarLabel>
                </span>
                </SidebarItem>
                
                
              </SidebarSection>
            <SidebarDivider className='dark:border-zinc-800'/>
            <SidebarSection>
              <div className='flex items-center gap-3 text-sm font-bold py-3'>
                  <Avatar 
                      className='w-6 h-6'
                      slot="icon"
                      src="https://www.confluent.io/favicon.ico"
                      alt="Confluent"
                    />
                  <SidebarLabel className='text-zinc-900 dark:text-white'>Confluent</SidebarLabel>
              </div>
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
              
              
            </SidebarSection>
          </SidebarBody>
        </Sidebar>
        

        {/* Main Content */}
        <div className="flex-1 overflow-auto bg-zinc-50 dark:bg-zinc-900">
          <div className="px-8 pb-8">
            {renderTabContent()}
          </div>
        </div>
      </div>

      {/* Create Group Modal */}
      <Dialog open={showCreateGroupModal} onClose={handleCreateGroupCancel} size="lg">
        <DialogTitle>Create New Group</DialogTitle>
        <DialogDescription>
          Create a new group to organize team members and manage permissions.
        </DialogDescription>
        
        <DialogBody>
          <div className="space-y-6">
            {/* Group Name */}
            <Field>
              <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Group Name *
              </Label>
              <Input
                type="text"
                placeholder="Enter group name"
                value={newGroupName}
                onChange={(e) => setNewGroupName(e.target.value)}
                className="w-full"
                autoFocus
              />
            </Field>

            {/* Group Description */}
            <Field>
              <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Description
              </Label>
              <Textarea
                placeholder="Enter group description (optional)"
                value={newGroupDescription}
                onChange={(e) => setNewGroupDescription(e.target.value)}
                className="w-full"
                rows={3}
              />
            </Field>

            {/* Initial Role */}
            <Field>
              <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Your Role in this Group
              </Label>
              <Dropdown>
                <DropdownButton outline className="flex items-center gap-2 text-sm w-full justify-between">
                  {newGroupRole}
                  <MaterialSymbol name="keyboard_arrow_down" />
                </DropdownButton>
                <DropdownMenu>
                  <DropdownItem onClick={() => setNewGroupRole('Admin')}>
                    <DropdownLabel>Admin</DropdownLabel>
                    <DropdownDescription>Full control over group settings and members</DropdownDescription>
                  </DropdownItem>
                  <DropdownItem onClick={() => setNewGroupRole('Member')}>
                    <DropdownLabel>Member</DropdownLabel>
                    <DropdownDescription>Standard group member permissions</DropdownDescription>
                  </DropdownItem>
                </DropdownMenu>
              </Dropdown>
            </Field>
          </div>
        </DialogBody>
        
        <DialogActions>
          <Button color="blue" onClick={handleCreateGroupSubmit} disabled={!newGroupName.trim()}>
            Create Group
          </Button>
          <Button plain onClick={handleCreateGroupCancel}>
            Cancel
          </Button>
        </DialogActions>
      </Dialog>

    </div>
  )
}

// Wrapper component that provides toast context
export function OrganizationSettings(props: OrganizationSettingsProps) {
  return (
    <ToastProvider>
      <OrganizationSettingsInner {...props} />
    </ToastProvider>
  );
}