import React, { useState } from 'react'
import { useParams, Link, useNavigate } from 'react-router-dom'
import { Button } from '../../../components/Button/button'
import { Input } from '../../../components/Input/input'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Text } from '../../../components/Text/text'
import { authorizationCreateRole } from '../../../api-client/sdk.gen'

interface Permission {
  id: string
  name: string
  description: string
  category: string
}

const AVAILABLE_PERMISSIONS: Permission[] = [
  // Organization Management
  { id: 'org.read', name: 'View Organization', description: 'View organization details and settings', category: 'Organization' },
  { id: 'org.update', name: 'Manage Organization', description: 'Update organization settings and configuration', category: 'Organization' },
  { id: 'org.delete', name: 'Delete Organization', description: 'Delete the organization (dangerous)', category: 'Organization' },
  
  // Member Management
  { id: 'members.read', name: 'View Members', description: 'View organization members and their details', category: 'Members' },
  { id: 'members.invite', name: 'Invite Members', description: 'Invite new members to the organization', category: 'Members' },
  { id: 'members.remove', name: 'Remove Members', description: 'Remove members from the organization', category: 'Members' },
  { id: 'members.update', name: 'Manage Members', description: 'Update member roles and permissions', category: 'Members' },
  
  // Group Management
  { id: 'groups.read', name: 'View Groups', description: 'View organization groups and their members', category: 'Groups' },
  { id: 'groups.create', name: 'Create Groups', description: 'Create new groups within the organization', category: 'Groups' },
  { id: 'groups.update', name: 'Manage Groups', description: 'Update group settings and membership', category: 'Groups' },
  { id: 'groups.delete', name: 'Delete Groups', description: 'Delete groups from the organization', category: 'Groups' },
  
  // Role Management
  { id: 'roles.read', name: 'View Roles', description: 'View organization roles and their permissions', category: 'Roles' },
  { id: 'roles.create', name: 'Create Roles', description: 'Create new roles within the organization', category: 'Roles' },
  { id: 'roles.update', name: 'Manage Roles', description: 'Update role permissions and settings', category: 'Roles' },
  { id: 'roles.delete', name: 'Delete Roles', description: 'Delete roles from the organization', category: 'Roles' },
  
  // Project Management
  { id: 'projects.read', name: 'View Projects', description: 'View organization projects', category: 'Projects' },
  { id: 'projects.create', name: 'Create Projects', description: 'Create new projects within the organization', category: 'Projects' },
  { id: 'projects.update', name: 'Manage Projects', description: 'Update project settings and configuration', category: 'Projects' },
  { id: 'projects.delete', name: 'Delete Projects', description: 'Delete projects from the organization', category: 'Projects' },
  
  // Billing
  { id: 'billing.read', name: 'View Billing', description: 'View billing information and usage', category: 'Billing' },
  { id: 'billing.update', name: 'Manage Billing', description: 'Update billing information and payment methods', category: 'Billing' },
]

export function CreateRolePage() {
  const { orgId } = useParams<{ orgId: string }>()
  const navigate = useNavigate()
  
  const [roleName, setRoleName] = useState('')
  const [roleDescription, setRoleDescription] = useState('')
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([])
  const [isCreating, setIsCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const handlePermissionToggle = (permissionId: string) => {
    setSelectedPermissions(prev => 
      prev.includes(permissionId) 
        ? prev.filter(id => id !== permissionId)
        : [...prev, permissionId]
    )
  }

  const handleSelectAllInCategory = (category: string) => {
    const categoryPermissions = AVAILABLE_PERMISSIONS
      .filter(p => p.category === category)
      .map(p => p.id)
    
    const allSelected = categoryPermissions.every(id => selectedPermissions.includes(id))
    
    if (allSelected) {
      // Deselect all in category
      setSelectedPermissions(prev => prev.filter(id => !categoryPermissions.includes(id)))
    } else {
      // Select all in category
      setSelectedPermissions(prev => [...new Set([...prev, ...categoryPermissions])])
    }
  }

  const handleCreateRole = async () => {
    if (!roleName.trim() || selectedPermissions.length === 0 || !orgId) return
    
    setIsCreating(true)
    setError(null)
    
    try {
      await authorizationCreateRole({
        body: {
          domainType: 'organization',
          domainId: orgId,
          name: roleName.trim(),
          description: roleDescription.trim(),
          permissions: selectedPermissions
        }
      })
      
      console.log('Successfully created role:', roleName)
      navigate(`/organization/${orgId}/settings/roles`)
    } catch (err) {
      console.error('Error creating role:', err)
      setError('Failed to create role. Please try again.')
    } finally {
      setIsCreating(false)
    }
  }

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleCreateRole()
    }
  }

  if (!orgId) {
    return <div>Error: Organization ID not found</div>
  }

  const groupedPermissions = AVAILABLE_PERMISSIONS.reduce((acc, permission) => {
    if (!acc[permission.category]) {
      acc[permission.category] = []
    }
    acc[permission.category].push(permission)
    return acc
  }, {} as Record<string, Permission[]>)

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900 pt-20">
      <div className="max-w-4xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-2 mb-4">
            <Link 
              to={`/organization/${orgId}/settings/roles`}
              className="flex items-center gap-2 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white transition-colors"
            >
              <MaterialSymbol name="arrow_back" size="sm" />
              <span className="text-sm">Back to Roles</span>
            </Link>
          </div>
          
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white mb-2">
                Create New Role
              </h1>
              <Text className="text-zinc-600 dark:text-zinc-400">
                Define a role with specific permissions for organization members
              </Text>
            </div>
          </div>
        </div>

        {/* Create Role Form */}
        <div className="space-y-6">
          <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
            {error && (
              <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-6">
                <p className="text-sm">{error}</p>
              </div>
            )}

            <div className="space-y-6">
              {/* Role Name */}
              <div>
                <label className="block text-sm font-medium text-zinc-900 dark:text-white mb-2">
                  Role Name *
                </label>
                <Input
                  type="text"
                  placeholder="Enter role name (e.g., Admin, Editor, Viewer)"
                  value={roleName}
                  onChange={(e) => setRoleName(e.target.value)}
                  onKeyPress={handleKeyPress}
                  className="w-full"
                />
                <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                  Choose a descriptive name that clearly identifies the role's purpose
                </Text>
              </div>

              {/* Role Description */}
              <div>
                <label className="block text-sm font-medium text-zinc-900 dark:text-white mb-2">
                  Description
                </label>
                <textarea
                  placeholder="Describe the role's purpose and responsibilities..."
                  value={roleDescription}
                  onChange={(e) => setRoleDescription(e.target.value)}
                  className="w-full px-3 py-2 border border-zinc-300 dark:border-zinc-700 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-zinc-800 dark:text-white resize-none"
                  rows={3}
                />
                <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                  Optional: Provide context about what this role is for
                </Text>
              </div>

              {/* Permissions */}
              <div>
                <label className="block text-sm font-medium text-zinc-900 dark:text-white mb-4">
                  Permissions * ({selectedPermissions.length} selected)
                </label>
                
                <div className="space-y-6">
                  {Object.entries(groupedPermissions).map(([category, permissions]) => {
                    const categoryPermissionIds = permissions.map(p => p.id)
                    const allSelected = categoryPermissionIds.every(id => selectedPermissions.includes(id))
                    const someSelected = categoryPermissionIds.some(id => selectedPermissions.includes(id))
                    
                    return (
                      <div key={category} className="border border-zinc-200 dark:border-zinc-700 rounded-lg">
                        <div className="bg-zinc-50 dark:bg-zinc-800 px-4 py-3 border-b border-zinc-200 dark:border-zinc-700 rounded-t-lg">
                          <div className="flex items-center justify-between">
                            <Text className="font-medium text-zinc-900 dark:text-white">
                              {category}
                            </Text>
                            <Button
                              size="sm"
                              outline
                              onClick={() => handleSelectAllInCategory(category)}
                              className="text-xs"
                            >
                              {allSelected ? 'Deselect All' : 'Select All'}
                            </Button>
                          </div>
                          {someSelected && !allSelected && (
                            <Text className="text-xs text-blue-600 dark:text-blue-400 mt-1">
                              {categoryPermissionIds.filter(id => selectedPermissions.includes(id)).length} of {categoryPermissionIds.length} selected
                            </Text>
                          )}
                        </div>
                        <div className="p-4 space-y-3">
                          {permissions.map((permission) => (
                            <label key={permission.id} className="flex items-start gap-3 cursor-pointer">
                              <input
                                type="checkbox"
                                checked={selectedPermissions.includes(permission.id)}
                                onChange={() => handlePermissionToggle(permission.id)}
                                className="mt-1 h-4 w-4 text-blue-600 border-zinc-300 rounded focus:ring-blue-500"
                              />
                              <div className="flex-1">
                                <div className="flex items-center gap-2">
                                  <Text className="font-medium text-zinc-900 dark:text-white">
                                    {permission.name}
                                  </Text>
                                  <code className="text-xs bg-zinc-100 dark:bg-zinc-800 px-2 py-1 rounded text-zinc-600 dark:text-zinc-400">
                                    {permission.id}
                                  </code>
                                </div>
                                <Text className="text-sm text-zinc-600 dark:text-zinc-400 mt-1">
                                  {permission.description}
                                </Text>
                              </div>
                            </label>
                          ))}
                        </div>
                      </div>
                    )
                  })}
                </div>
                
                {selectedPermissions.length === 0 && (
                  <Text className="text-sm text-red-600 dark:text-red-400 mt-2">
                    Please select at least one permission for this role
                  </Text>
                )}
              </div>
            </div>
          </div>

          {/* Help Section */}
          <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
            <div className="flex items-start gap-4">
              <div className="bg-blue-100 dark:bg-blue-900/20 rounded-lg p-2">
                <MaterialSymbol name="help" className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div className="flex-1">
                <Text className="font-medium text-zinc-900 dark:text-white mb-2">
                  About Roles and Permissions
                </Text>
                <div className="space-y-2 text-sm text-zinc-600 dark:text-zinc-400">
                  <p>• Roles define what actions members can perform within the organization</p>
                  <p>• Each role consists of a collection of permissions that grant specific capabilities</p>
                  <p>• Members inherit permissions from all roles assigned to their groups</p>
                  <p>• Start with essential permissions and add more as needed</p>
                </div>
                <div className="mt-4 p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
                  <p className="text-sm text-yellow-800 dark:text-yellow-200">
                    <strong>Tip:</strong> Consider creating roles like "Admin" (full access), 
                    "Editor" (read/write access), and "Viewer" (read-only access) to cover common use cases.
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-3">
            <Button 
              color="blue"
              onClick={handleCreateRole}
              disabled={!roleName.trim() || selectedPermissions.length === 0 || isCreating}
              className="flex items-center gap-2"
            >
              <MaterialSymbol name="shield" size="sm" />
              {isCreating ? 'Creating...' : 'Create Role'}
            </Button>
            
            <Link to={`/organization/${orgId}/settings/roles`}>
              <Button outline>
                Cancel
              </Button>
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}