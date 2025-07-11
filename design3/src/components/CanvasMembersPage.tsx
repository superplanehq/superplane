import React, { useState } from 'react'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Avatar } from './lib/Avatar/avatar'
import { Heading } from './lib/Heading/heading'
import { Button } from './lib/Button/button'
import { Input, InputGroup } from './lib/Input/input'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem
} from './lib/Dropdown/dropdown'
import { NavigationOrg, type User, type Organization } from './lib/Navigation/navigation-org'
import { 
  Table, 
  TableHead, 
  TableBody, 
  TableRow, 
  TableHeader, 
  TableCell 
} from './lib/Table/table'

interface CanvasMembersPageProps {
  canvasId: string
  onBack?: () => void
}

// Mock data for canvas members
const canvasMembers = [
  {
    id: '1',
    name: 'John Doe',
    email: 'john@acme.com',
    role: 'Editor',
    permission: 'Can edit',
    lastActive: '2 hours ago',
    initials: 'JD',
    avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face'
  },
  {
    id: '2',
    name: 'Jane Smith',
    email: 'jane@acme.com',
    role: 'Viewer',
    permission: 'Can view',
    lastActive: '1 day ago',
    initials: 'JS'
  },
  {
    id: '3',
    name: 'Bob Wilson',
    email: 'bob@acme.com',
    role: 'Editor',
    permission: 'Can edit',
    lastActive: '3 days ago',
    initials: 'BW'
  },
  {
    id: '4',
    name: 'Alice Johnson',
    email: 'alice@acme.com',
    role: 'Owner',
    permission: 'Full access',
    lastActive: '5 minutes ago',
    initials: 'AJ'
  }
]

export function CanvasMembersPage({ 
  canvasId, 
  onBack
}: CanvasMembersPageProps) {
  const [searchQuery, setSearchQuery] = useState('')

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

  // Get canvas name based on ID
  const getCanvasName = (id: string) => {
    const canvasNames: Record<string, string> = {
      '1': 'Production Deployment Pipeline',
      '2': 'Development Workflow',
      '3': 'Testing Environment Setup',
      '4': 'Staging Release Process',
      'new': 'New Canvas'
    }
    return canvasNames[id] || `Canvas ${id}`
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
        console.log('Signing out...')
        break
    }
  }

  const handleOrganizationMenuAction = (action: 'settings' | 'billing' | 'members') => {
    if (action === 'settings') {
      window.history.pushState(null, '', '/settings')
      window.dispatchEvent(new PopStateEvent('popstate'))
    } else {
      console.log(`Organization action: ${action}`)
    }
  }

  const handleBackToCanvas = () => {
    window.history.pushState(null, '', `/canvas/${canvasId}`)
    window.dispatchEvent(new PopStateEvent('popstate'))
  }

  const handleBackToCanvases = () => {
    window.history.pushState(null, '', '/canvases')
    window.dispatchEvent(new PopStateEvent('popstate'))
  }

  const handleInviteMember = () => {
    console.log('Inviting new member to canvas...')
    // TODO: Implement invite functionality
  }

  const handleRemoveMember = (memberId: string) => {
    console.log('Removing member:', memberId)
    // TODO: Implement remove member functionality
  }

  const handleChangePermission = (memberId: string, newRole: string) => {
    console.log('Changing member permission:', { memberId, newRole })
    // TODO: Implement permission change functionality
  }

  // Filter members based on search query
  const filteredMembers = canvasMembers.filter(member =>
    member.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
    member.email.toLowerCase().includes(searchQuery.toLowerCase())
  )

  return (
    <div className="flex flex-col min-h-screen bg-zinc-50 dark:bg-zinc-900">
      {/* Navigation */}
      <NavigationOrg
        user={currentUser}
        organization={currentOrganization}
        breadcrumbs={[
          {
            label: "Canvases",
            icon: "automation",
            onClick: handleBackToCanvases
          },
          {
            label: getCanvasName(canvasId),
            onClick: handleBackToCanvas
          },
          {
            label: "Manage Members",
            current: true
          }
        ]}
        onHelpClick={handleHelpClick}
        onUserMenuAction={handleUserMenuAction}
        onOrganizationMenuAction={handleOrganizationMenuAction}
      />

      {/* Main Content */}
      <div className="flex-1 px-8 py-6">
        <div className="max-w-5xl mx-auto space-y-6">
          {/* Header */}
          <div className="flex items-center justify-between">
            <div>
              <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-2">
                Canvas Members
              </Heading>
              <p className="text-zinc-600 dark:text-zinc-400">
                Manage who has access to "{getCanvasName(canvasId)}" and what they can do.
              </p>
            </div>
            <Button color="blue" onClick={handleInviteMember}>
              <MaterialSymbol name="person_add" size="sm" className="mr-2" />
              Invite Member
            </Button>
          </div>

          {/* Search and Filters */}
          <div className="bg-white dark:bg-zinc-800 rounded-lg border border-zinc-200 dark:border-zinc-700 p-6">
            <div className="flex items-center justify-between mb-4">
              <InputGroup>
                <Input
                  name="search"
                  placeholder="Search members..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-80"
                />
              </InputGroup>
            </div>

            {/* Members Table */}
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader>Member</TableHeader>
                  <TableHeader>Email</TableHeader>
                  <TableHeader>Permission</TableHeader>
                  <TableHeader>Last Active</TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredMembers.map((member) => (
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
                            {member.role}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-zinc-600 dark:text-zinc-400">
                        {member.email}
                      </span>
                    </TableCell>
                    <TableCell>
                      <Dropdown>
                        <DropdownButton outline className="flex items-center gap-2 text-sm">
                          {member.permission}
                          <MaterialSymbol name="keyboard_arrow_down" />
                        </DropdownButton>
                        <DropdownMenu>
                          <DropdownItem onClick={() => handleChangePermission(member.id, 'owner')}>
                            Full access
                          </DropdownItem>
                          <DropdownItem onClick={() => handleChangePermission(member.id, 'editor')}>
                            Can edit
                          </DropdownItem>
                          <DropdownItem onClick={() => handleChangePermission(member.id, 'viewer')}>
                            Can view
                          </DropdownItem>
                        </DropdownMenu>
                      </Dropdown>
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-zinc-500 dark:text-zinc-400">
                        {member.lastActive}
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Dropdown>
                          <DropdownButton plain>
                            <MaterialSymbol name="more_vert" size="sm" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem onClick={() => handleChangePermission(member.id, 'viewer')}>
                              <MaterialSymbol name="edit" />
                              Change Permission
                            </DropdownItem>
                            <DropdownItem onClick={() => handleRemoveMember(member.id)}>
                              <MaterialSymbol name="person_remove" />
                              Remove from Canvas
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>

            {/* Empty State */}
            {filteredMembers.length === 0 && searchQuery && (
              <div className="text-center py-8">
                <MaterialSymbol name="search_off" className="text-zinc-400 text-4xl mb-2" />
                <p className="text-zinc-500 dark:text-zinc-400">
                  No members found matching "{searchQuery}"
                </p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}