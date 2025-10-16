import { useState, useEffect } from 'react'
import { useParams, Link, useNavigate, useSearchParams } from 'react-router-dom'
import { Button } from '../../../components/Button/button'
import { Input } from '../../../components/Input/input'
import { Text } from '../../../components/Text/text'
import { Checkbox, CheckboxField } from '../../../components/Checkbox/checkbox'
import { Label, Description } from '../../../components/Fieldset/fieldset'
import { Breadcrumbs } from '../../../components/Breadcrumbs/breadcrumbs'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { useRole, useCreateRole, useUpdateRole, useOrganizationCanvases } from '../../../hooks/useOrganizationData'
import { useCanvasRole, useCreateCanvasRole, useUpdateCanvasRole } from '../../../hooks/useCanvasData'
import { Select, type SelectOption } from '../../../components/Select/index'
import { AuthorizationDomainType } from '../../../api-client/types.gen'
import { Heading } from '@/components/Heading/heading'

interface Permission {
  id: string
  name: string
  description: string
  category: string
  resource: string
  action: string
}

interface PermissionCategory {
  category: string
  icon: string
  permissions: Permission[]
}

// Organization permissions based on RBAC policy
const ORGANIZATION_PERMISSIONS: PermissionCategory[] = [
  {
    category: 'General',
    icon: 'business',
    permissions: [
      { id: 'org.read', name: 'View Organization', description: 'View organization details and settings', category: 'General', resource: 'org', action: 'read' },
      { id: 'org.update', name: 'Manage Organization', description: 'Update organization settings and configuration', category: 'General', resource: 'org', action: 'update' },
      { id: 'org.delete', name: 'Delete Organization', description: 'Delete the organization (dangerous)', category: 'General', resource: 'org', action: 'delete' }
    ]
  },
  {
    category: 'People & Groups',
    icon: 'group',
    permissions: [
      { id: 'member.read', name: 'View Members', description: 'View organization members and their details', category: 'People & Groups', resource: 'member', action: 'read' },
      { id: 'member.create', name: 'Add Members', description: 'Add new members to the organization', category: 'People & Groups', resource: 'member', action: 'create' },
      { id: 'member.update', name: 'Update Members', description: 'Update member information and settings', category: 'People & Groups', resource: 'member', action: 'update' },
      { id: 'member.delete', name: 'Remove Members', description: 'Remove members from the organization', category: 'People & Groups', resource: 'member', action: 'delete' },
      { id: 'group.read', name: 'View Groups', description: 'View organization groups and their members', category: 'People & Groups', resource: 'group', action: 'read' },
      { id: 'group.create', name: 'Create Groups', description: 'Create new groups within the organization', category: 'People & Groups', resource: 'group', action: 'create' },
      { id: 'group.update', name: 'Manage Groups', description: 'Update group settings and membership', category: 'People & Groups', resource: 'group', action: 'update' },
      { id: 'group.delete', name: 'Delete Groups', description: 'Delete groups from the organization', category: 'People & Groups', resource: 'group', action: 'delete' }
    ]
  },
  {
    category: 'Roles & Permissions',
    icon: 'admin_panel_settings',
    permissions: [
      { id: 'role.read', name: 'View Roles', description: 'View organization roles and their permissions', category: 'Roles & Permissions', resource: 'role', action: 'read' },
      { id: 'role.create', name: 'Create Roles', description: 'Create new roles within the organization', category: 'Roles & Permissions', resource: 'role', action: 'create' },
      { id: 'role.update', name: 'Manage Roles', description: 'Update role permissions and settings', category: 'Roles & Permissions', resource: 'role', action: 'update' },
      { id: 'role.delete', name: 'Delete Roles', description: 'Delete roles from the organization', category: 'Roles & Permissions', resource: 'role', action: 'delete' }
    ]
  },
  {
    category: 'Projects & Resources',
    icon: 'folder',
    permissions: [
      { id: 'canvas.read', name: 'View Canvases', description: 'View organization canvases', category: 'Projects & Resources', resource: 'canvas', action: 'read' },
      { id: 'canvas.create', name: 'Create Canvases', description: 'Create new canvases within the organization', category: 'Projects & Resources', resource: 'canvas', action: 'create' },
      { id: 'canvas.update', name: 'Manage Canvases', description: 'Update canvas settings and configuration', category: 'Projects & Resources', resource: 'canvas', action: 'update' },
      { id: 'canvas.delete', name: 'Delete Canvases', description: 'Delete canvases from the organization', category: 'Projects & Resources', resource: 'canvas', action: 'delete' }
    ]
  },
  {
    category: 'Integrations & Secrets',
    icon: 'integration_instructions',
    permissions: [
      { id: 'integration.create', name: 'Create Integrations', description: 'Create new integrations within the organization', category: 'Integrations & Secrets', resource: 'integration', action: 'create' },
      { id: 'integration.read', name: 'View Integrations', description: 'View organization integrations', category: 'Integrations & Secrets', resource: 'integration', action: 'read' },
      { id: 'integration.update', name: 'Manage Integrations', description: 'Update integration settings and configuration', category: 'Integrations & Secrets', resource: 'integration', action: 'update' },
      { id: 'secret.create', name: 'Create Secrets', description: 'Create new secrets within the organization', category: 'Integrations & Secrets', resource: 'secret', action: 'create' },
      { id: 'secret.read', name: 'View Secrets', description: 'View organization secrets', category: 'Integrations & Secrets', resource: 'secret', action: 'read' },
      { id: 'secret.update', name: 'Manage Secrets', description: 'Update secret values and settings', category: 'Integrations & Secrets', resource: 'secret', action: 'update' },
      { id: 'secret.delete', name: 'Delete Secrets', description: 'Delete secrets from the organization', category: 'Integrations & Secrets', resource: 'secret', action: 'delete' }
    ]
  }
]

// Canvas permissions based on RBAC policy
const CANVAS_PERMISSIONS: PermissionCategory[] = [
  {
    category: 'General',
    icon: 'canvas',
    permissions: [
      { id: 'member.read', name: 'View Members', description: 'View canvas members and their details', category: 'General', resource: 'member', action: 'read' },
      { id: 'member.create', name: 'Add Members', description: 'Add new members to the canvas', category: 'General', resource: 'member', action: 'create' },
      { id: 'member.update', name: 'Update Members', description: 'Update member information and settings', category: 'General', resource: 'member', action: 'update' },
      { id: 'member.delete', name: 'Remove Members', description: 'Remove members from the canvas', category: 'General', resource: 'member', action: 'delete' },
      { id: 'group.read', name: 'View Groups', description: 'View canvas groups and their members', category: 'General', resource: 'group', action: 'read' },
      { id: 'group.create', name: 'Create Groups', description: 'Create new groups within the canvas', category: 'General', resource: 'group', action: 'create' },
      { id: 'group.update', name: 'Manage Groups', description: 'Update group settings and membership', category: 'General', resource: 'group', action: 'update' },
      { id: 'group.delete', name: 'Delete Groups', description: 'Delete groups from the canvas', category: 'General', resource: 'group', action: 'delete' }
    ]
  },
  {
    category: 'Roles & Permissions',
    icon: 'admin_panel_settings',
    permissions: [
      { id: 'role.read', name: 'View Roles', description: 'View canvas roles and their permissions', category: 'Roles & Permissions', resource: 'role', action: 'read' },
      { id: 'role.create', name: 'Create Roles', description: 'Create new roles within the canvas', category: 'Roles & Permissions', resource: 'role', action: 'create' },
      { id: 'role.update', name: 'Manage Roles', description: 'Update role permissions and settings', category: 'Roles & Permissions', resource: 'role', action: 'update' },
      { id: 'role.delete', name: 'Delete Roles', description: 'Delete roles from the canvas', category: 'Roles & Permissions', resource: 'role', action: 'delete' }
    ]
  },
  {
    category: 'Data Sources & Events',
    icon: 'data_usage',
    permissions: [
      { id: 'eventsource.read', name: 'View Event Sources', description: 'View canvas event sources and their configuration', category: 'Data Sources & Events', resource: 'eventsource', action: 'read' },
      { id: 'eventsource.create', name: 'Create Event Sources', description: 'Create new event sources within the canvas', category: 'Data Sources & Events', resource: 'eventsource', action: 'create' },
      { id: 'eventsource.update', name: 'Manage Event Sources', description: 'Update event source settings and configuration', category: 'Data Sources & Events', resource: 'eventsource', action: 'update' },
      { id: 'eventsource.delete', name: 'Delete Event Sources', description: 'Delete event sources from the canvas', category: 'Data Sources & Events', resource: 'eventsource', action: 'delete' },
      { id: 'event.read', name: 'View Events', description: 'View events flowing through the canvas', category: 'Data Sources & Events', resource: 'event', action: 'read' },
      { id: 'event.create', name: 'Create Events', description: 'Create new events within the canvas', category: 'Data Sources & Events', resource: 'event', action: 'create' }
    ]
  },
  {
    category: 'Pipeline & Execution',
    icon: 'account_tree',
    permissions: [
      { id: 'stage.read', name: 'View Stages', description: 'View pipeline stages and their configuration', category: 'Pipeline & Execution', resource: 'stage', action: 'read' },
      { id: 'stage.create', name: 'Create Stages', description: 'Create new pipeline stages', category: 'Pipeline & Execution', resource: 'stage', action: 'create' },
      { id: 'stage.update', name: 'Manage Stages', description: 'Update stage settings and configuration', category: 'Pipeline & Execution', resource: 'stage', action: 'update' },
      { id: 'stage.delete', name: 'Delete Stages', description: 'Delete stages from the pipeline', category: 'Pipeline & Execution', resource: 'stage', action: 'delete' },
      { id: 'stageevent.read', name: 'View Stage Events', description: 'View events processed by pipeline stages', category: 'Pipeline & Execution', resource: 'stageevent', action: 'read' },
      { id: 'stageevent.approve', name: 'Approve Stage Events', description: 'Approve events for processing by stages', category: 'Pipeline & Execution', resource: 'stageevent', action: 'approve' },
      { id: 'stageevent.discard', name: 'Discard Stage Events', description: 'Discard events from stage processing', category: 'Pipeline & Execution', resource: 'stageevent', action: 'discard' },
      { id: 'stageexecution.read', name: 'View Stage Executions', description: 'View stage execution history and results', category: 'Pipeline & Execution', resource: 'stageexecution', action: 'read' },
      { id: 'stageexecution.create', name: 'Create Stage Executions', description: 'Trigger new stage executions', category: 'Pipeline & Execution', resource: 'stageexecution', action: 'create' },
      { id: 'stageexecution.update', name: 'Manage Stage Executions', description: 'Update stage execution settings', category: 'Pipeline & Execution', resource: 'stageexecution', action: 'update' },
      { id: 'stageexecution.delete', name: 'Delete Stage Executions', description: 'Delete stage execution records', category: 'Pipeline & Execution', resource: 'stageexecution', action: 'delete' }
    ]
  },
  {
    category: 'Connections & Integrations',
    icon: 'hub',
    permissions: [
      { id: 'connectiongroup.read', name: 'View Connection Groups', description: 'View connection groups and their configuration', category: 'Connections & Integrations', resource: 'connectiongroup', action: 'read' },
      { id: 'connectiongroup.create', name: 'Create Connection Groups', description: 'Create new connection groups', category: 'Connections & Integrations', resource: 'connectiongroup', action: 'create' },
      { id: 'connectiongroup.update', name: 'Manage Connection Groups', description: 'Update connection group settings', category: 'Connections & Integrations', resource: 'connectiongroup', action: 'update' },
      { id: 'connectiongroup.delete', name: 'Delete Connection Groups', description: 'Delete connection groups', category: 'Connections & Integrations', resource: 'connectiongroup', action: 'delete' },
      { id: 'connectiongroupfieldset.read', name: 'View Connection Group Fieldsets', description: 'View connection group fieldset configurations', category: 'Connections & Integrations', resource: 'connectiongroupfieldset', action: 'read' },
      { id: 'integration.read', name: 'View Integrations', description: 'View canvas integrations', category: 'Connections & Integrations', resource: 'integration', action: 'read' },
      { id: 'integration.create', name: 'Create Integrations', description: 'Create new integrations within the canvas', category: 'Connections & Integrations', resource: 'integration', action: 'create' },
      { id: 'integration.update', name: 'Manage Integrations', description: 'Update integration settings and configuration', category: 'Connections & Integrations', resource: 'integration', action: 'update' },
      { id: 'integration.delete', name: 'Delete Integrations', description: 'Delete integrations from the canvas', category: 'Connections & Integrations', resource: 'integration', action: 'delete' }
    ]
  },
  {
    category: 'Security & Monitoring',
    icon: 'security',
    permissions: [
      { id: 'secret.read', name: 'View Secrets', description: 'View canvas secrets', category: 'Security & Monitoring', resource: 'secret', action: 'read' },
      { id: 'secret.create', name: 'Create Secrets', description: 'Create new secrets within the canvas', category: 'Security & Monitoring', resource: 'secret', action: 'create' },
      { id: 'secret.update', name: 'Manage Secrets', description: 'Update secret values and settings', category: 'Security & Monitoring', resource: 'secret', action: 'update' },
      { id: 'secret.delete', name: 'Delete Secrets', description: 'Delete secrets from the canvas', category: 'Security & Monitoring', resource: 'secret', action: 'delete' },
      { id: 'alert.read', name: 'View Alerts', description: 'View canvas alerts and notifications', category: 'Security & Monitoring', resource: 'alert', action: 'read' },
      { id: 'alert.acknowledge', name: 'Acknowledge Alerts', description: 'Acknowledge and dismiss canvas alerts', category: 'Security & Monitoring', resource: 'alert', action: 'acknowledge' }
    ]
  }
]

export function CreateRolePage() {
  const { roleName: roleNameParam } = useParams<{ roleName?: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { organizationId } = useParams<{ organizationId: string }>()
  const orgId = organizationId
  const isEditMode = !!roleNameParam
  const canvasIdFromParams = searchParams.get('canvasId')
  const isCanvasRole = !!canvasIdFromParams

  const [roleName, setRoleName] = useState('')
  const [roleDescription, setRoleDescription] = useState('')
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(new Set())
  const [selectedCanvasId, setSelectedCanvasId] = useState<string>(canvasIdFromParams || '')

  // React Query hooks - use canvas or org hooks based on role type
  const { data: existingOrgRole, isLoading: isLoadingOrgRole, error: orgError } = useRole(orgId || '', roleNameParam || '')
  const { data: existingCanvasRole, isLoading: isLoadingCanvasRole, error: canvasError } = useCanvasRole(selectedCanvasId, roleNameParam || '')
  const { data: canvases = [] } = useOrganizationCanvases(orgId || '')

  // Use the appropriate role data and loading state
  const existingRole = isCanvasRole ? existingCanvasRole : existingOrgRole
  const isLoading = isCanvasRole ? isLoadingCanvasRole : isLoadingOrgRole
  const error = isCanvasRole ? canvasError : orgError

  // Use appropriate mutation hooks
  const createOrgRoleMutation = useCreateRole(orgId || '')
  const updateOrgRoleMutation = useUpdateRole(orgId || '')
  const createCanvasRoleMutation = useCreateCanvasRole(selectedCanvasId)
  const updateCanvasRoleMutation = useUpdateCanvasRole(selectedCanvasId)

  const createRoleMutation = isCanvasRole ? createCanvasRoleMutation : createOrgRoleMutation
  const updateRoleMutation = isCanvasRole ? updateCanvasRoleMutation : updateOrgRoleMutation

  const isSubmitting = createRoleMutation.isPending || updateRoleMutation.isPending

  // Canvas options for the select
  const canvasOptions: SelectOption[] = canvases
    .filter((canvas) => canvas.metadata?.id)
    .map((canvas) => ({
      value: canvas.metadata!.id!,
      label: canvas.metadata?.name || 'Unnamed Canvas',
      description: canvas.metadata?.description,
    }))

  // Check if this is a default role
  const isDefaultRole = (roleName: string | undefined) => {
    if (!roleName) return false
    const orgDefaultRoles = ['org_viewer', 'org_admin', 'org_owner']
    const canvasDefaultRoles = ['canvas_viewer', 'canvas_admin', 'canvas_owner']
    const defaultRoles = isCanvasRole ? canvasDefaultRoles : orgDefaultRoles
    return defaultRoles.includes(roleName)
  }

  const isViewingDefaultRole = Boolean(isEditMode && existingRole && isDefaultRole(existingRole.metadata?.name))

  const handleCategoryToggle = (permissions: Permission[]) => {
    const permissionIds = permissions.map(p => p.id)
    const allSelected = permissionIds.every(id => selectedPermissions.has(id))

    setSelectedPermissions(prev => {
      const newSet = new Set(prev)
      if (allSelected) {
        // Deselect all in category
        permissionIds.forEach(id => newSet.delete(id))
      } else {
        // Select all in category
        permissionIds.forEach(id => newSet.add(id))
      }
      return newSet
    })
  }

  const isCategorySelected = (permissions: Permission[]) => {
    const permissionIds = permissions.map(p => p.id)
    return permissionIds.every(id => selectedPermissions.has(id))
  }

  // Load role data when in edit mode
  useEffect(() => {
    if (isEditMode && existingRole) {
      setRoleName(existingRole.spec?.displayName || existingRole.metadata?.name || '')
      setRoleDescription(existingRole.spec?.description || '')

      // Convert permissions back to selected format
      const permissionIds = new Set<string>()
      existingRole.spec?.permissions?.forEach(perm => {
        const matchingPerm = (isCanvasRole ? CANVAS_PERMISSIONS : ORGANIZATION_PERMISSIONS)
          .flatMap(cat => cat.permissions)
          .find(p => p.resource === perm.resource && p.action === perm.action)

        if (matchingPerm) {
          permissionIds.add(matchingPerm.id)
        }
      })
      setSelectedPermissions(permissionIds)
    }
  }, [isEditMode, existingRole, isCanvasRole])


  const handleSubmitRole = async () => {
    if (!roleName.trim() || selectedPermissions.size === 0 || !orgId) return
    if (isCanvasRole && !isEditMode && !selectedCanvasId) return

    try {
      // Convert selected permissions to the protobuf format
      const permissions = Array.from(selectedPermissions).map(permId => {
        const permission = (isCanvasRole ? CANVAS_PERMISSIONS : ORGANIZATION_PERMISSIONS)
          .flatMap(cat => cat.permissions)
          .find(p => p.id === permId)

        if (!permission) {
          throw new Error(`Permission ${permId} not found`)
        }

        return {
          resource: permission.resource,
          action: permission.action,
          domainType: (isCanvasRole ? 'DOMAIN_TYPE_CANVAS' : 'DOMAIN_TYPE_ORGANIZATION') as AuthorizationDomainType
        }
      })

      if (isEditMode && roleNameParam) {
        // Update existing role
        const domainType = (isCanvasRole ? 'DOMAIN_TYPE_CANVAS' : 'DOMAIN_TYPE_ORGANIZATION') as AuthorizationDomainType
        const domainId = isCanvasRole ? selectedCanvasId : orgId

        await updateRoleMutation.mutateAsync({
          roleName: roleNameParam,
          domainType: domainType,
          domainId: domainId,
          permissions: permissions,
          displayName: roleName.trim(),
          description: roleDescription.trim() || undefined
        })
      } else {
        // Create new role
        const domainType = (isCanvasRole ? 'DOMAIN_TYPE_CANVAS' : 'DOMAIN_TYPE_ORGANIZATION') as AuthorizationDomainType
        const domainId = isCanvasRole ? selectedCanvasId : orgId

        await createRoleMutation.mutateAsync({
          role: {
            metadata: {
              name: roleName,
            },
            spec: {
              permissions: permissions,
              displayName: roleName.trim(),
              description: roleDescription.trim() || undefined
            }
          },
          domainType: domainType,
          domainId: domainId,
        })
      }

      navigate(`/${orgId}/settings/roles`)
    } catch {
      console.error('Failed to create role')
    }
  }



  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900 pt-4 text-left">
      <div className="max-w-8xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="mb-4">
            <Breadcrumbs
              items={[
                {
                  label: 'Roles',
                  onClick: () => navigate(`/${orgId}/settings/roles`)
                },
                {
                  label: isEditMode
                    ? (isViewingDefaultRole
                        ? (isCanvasRole ? 'View canvas role' : 'View organization role')
                        : (isCanvasRole ? 'Edit canvas role' : 'Edit organization role')
                      )
                    : (isCanvasRole ? 'Create new canvas role' : 'Create new organization role'),
                  current: true
                }
              ]}
              showDivider={false}
            />
          </div>

          <div className="flex items-center text-left">
            <div>
              <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-2">
                {isEditMode
                  ? (isViewingDefaultRole
                      ? (isCanvasRole ? 'View Canvas Role' : 'View Organization Role')
                      : (isCanvasRole ? 'Edit Canvas Role' : 'Edit Organization Role')
                    )
                  : (isCanvasRole ? 'Create New Canvas Role' : 'Create New Organization Role')
                }
              </Heading>
              <Text className="text-zinc-600 dark:text-zinc-400">
                {isEditMode
                  ? (isViewingDefaultRole
                      ? 'View the permissions and details of this default role. Default roles cannot be modified.'
                      : (isCanvasRole
                          ? 'Update the role with specific canvas permissions.'
                          : 'Update the role with specific organization permissions.'
                        )
                    )
                  : (isCanvasRole
                      ? 'Define a custom role with specific canvas permissions.'
                      : 'Define a custom role with specific organization permissions.'
                    )
                }
              </Text>
            </div>
          </div>
        </div>

        {/* Role Form */}
        <div className="space-y-6">
          {isLoading ? (
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
              <div className="flex justify-center items-center h-32">
                <p className="text-zinc-500 dark:text-zinc-400">Loading role data...</p>
              </div>
            </div>
          ) : (
            <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
              {error && (
                <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-6">
                  <p className="text-sm">{error instanceof Error ? error.message : 'Failed to load role data'}</p>
                </div>
              )}
              {isViewingDefaultRole && (
                <div className="bg-blue-50 border border-blue-200 text-blue-800 px-4 py-3 rounded mb-6 dark:bg-blue-900/20 dark:border-blue-800 dark:text-blue-200">
                  <div className="flex items-center gap-2">
                    <MaterialSymbol name="info" size="sm" />
                    <p className="text-sm">
                      This is a default role that cannot be modified. You can view its permissions but cannot make changes.
                    </p>
                  </div>
                </div>
              )}

              <div className="space-y-6">
                {/* Role Name */}
                <div>
                  <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                    Role Name *
                  </label>
                  <Input
                    type="text"
                    placeholder="Enter role name"
                    value={roleName}
                    onChange={(e) => setRoleName(e.target.value)}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' && !e.shiftKey) {
                        e.preventDefault()
                        handleSubmitRole()
                      }
                    }}
                    className="max-w-lg"
                    disabled={isEditMode}
                  />
                  {isEditMode && (
                    <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                      {isViewingDefaultRole
                        ? 'Role name cannot be changed for default roles'
                        : 'Role name cannot be changed when editing'
                      }
                    </Text>
                  )}
                </div>

                {/* Role Description */}
                <div>
                  <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                    Description
                  </label>
                  <textarea
                    placeholder="Describe what this role can do"
                    value={roleDescription}
                    onChange={(e) => setRoleDescription(e.target.value)}
                    className="max-w-lg w-full px-3 py-2 border border-zinc-300 dark:border-zinc-700 rounded-md shadow-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 dark:bg-zinc-800 dark:text-white resize-none"
                    rows={3}
                    disabled={isViewingDefaultRole}
                  />
                </div>

                {/* Canvas Selection for Canvas Roles */}
                {isCanvasRole && !isEditMode && (
                  <div>
                    <label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                      Canvas *
                    </label>
                    <div className="max-w-lg">
                      <Select
                        options={canvasOptions}
                        value={selectedCanvasId}
                        onChange={setSelectedCanvasId}
                        placeholder="Select a canvas for this role..."
                      />
                    </div>
                    <Text className="text-xs text-zinc-500 dark:text-zinc-400 mt-1">
                      This role will apply to the selected canvas
                    </Text>
                  </div>
                )}

                {/* Permissions */}
                <div className="pt-4 mb-4">
                  <h2 className="text-xl font-semibold text-zinc-900 dark:text-white mb-2">
                    {isCanvasRole ? 'Canvas Permissions' : 'Organization Permissions'}
                  </h2>
                  <Text className="text-sm text-zinc-600 dark:text-zinc-400">
                    {isCanvasRole
                      ? 'Select the permissions this role should have within the canvas.'
                      : 'Select the permissions this role should have within the organization.'
                    }
                  </Text>
                </div>

                <div className="space-y-6">
                  {(isCanvasRole ? CANVAS_PERMISSIONS : ORGANIZATION_PERMISSIONS).map((category) => (
                    <div key={category.category} className="space-y-4">
                      <div className="flex items-center mb-3">
                        <h3 className="text-md font-semibold text-zinc-900 dark:text-white">{category.category}</h3>
                        {!isViewingDefaultRole && (
                          <button
                            type="button"
                            className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 ml-3 bg-transparent border-none cursor-pointer"
                            onClick={() => handleCategoryToggle(category.permissions)}
                          >
                            {isCategorySelected(category.permissions) ? 'Deselect all' : 'Select all'}
                          </button>
                        )}
                      </div>
                      <div className="space-y-3">
                        {category.permissions.map((permission) => (
                          <CheckboxField
                            key={permission.id}
                            onClick={!isViewingDefaultRole ? () => {
                              setSelectedPermissions(prev => {
                                const newSet = new Set(prev)
                                if (newSet.has(permission.id)) {
                                  newSet.delete(permission.id)
                                } else {
                                  newSet.add(permission.id)
                                }
                                return newSet
                              })
                            } : undefined}
                          >
                            <Checkbox
                              name={permission.id}
                              checked={selectedPermissions.has(permission.id)}
                              onChange={!isViewingDefaultRole ? (checked) => {
                                setSelectedPermissions(prev => {
                                  const newSet = new Set(prev)
                                  if (checked) {
                                    newSet.add(permission.id)
                                  } else {
                                    newSet.delete(permission.id)
                                  }
                                  return newSet
                                })
                              } : undefined}
                              disabled={isViewingDefaultRole}
                            />
                            <Label className='cursor-pointer'>{permission.name}</Label>
                            <Description>{permission.description}</Description>
                          </CheckboxField>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>

                {selectedPermissions.size === 0 && (
                  <Text className="text-sm text-red-600 dark:text-red-400 mt-2">
                    Please select at least one permission for this role
                  </Text>
                )}
                {isCanvasRole && !isEditMode && !selectedCanvasId && (
                  <Text className="text-sm text-red-600 dark:text-red-400 mt-2">
                    Please select a canvas for this role
                  </Text>
                )}
              </div>
            </div>
          )}

          {/* Action Buttons */}
          <div className="flex justify-end gap-3">
            <Link to={`/${orgId}/settings/roles`}>
              <Button outline>
                {isViewingDefaultRole ? 'Back' : 'Cancel'}
              </Button>
            </Link>
            {!isViewingDefaultRole && (
              <Button
                color="blue"
                onClick={handleSubmitRole}
                disabled={!roleName.trim() || selectedPermissions.size === 0 || isSubmitting || isLoading || (isCanvasRole && !isEditMode && !selectedCanvasId)}
              >
                {isSubmitting
                  ? (isEditMode ? 'Updating...' : 'Creating...')
                  : (isEditMode ? 'Update Role' : 'Create Role')
                }
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}