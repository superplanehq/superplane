import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../select'
import { useOrganizationRoles } from '../../hooks/useOrganizationData'
import { ComponentsConfigurationField } from '../../api-client'

interface RoleFieldRendererProps {
  field: ComponentsConfigurationField
  value: string
  onChange: (value: string | undefined) => void
  domainId: string
}

export const RoleFieldRenderer = ({
  value,
  onChange,
  domainId
}: RoleFieldRendererProps) => {
  // Fetch roles from the organization
  const { data: roles, isLoading, error } = useOrganizationRoles(domainId)

  if (!domainId || domainId.trim() === '') {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Role field requires domainId prop
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load roles: {error instanceof Error ? error.message : 'Unknown error'}
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="text-sm text-gray-500 dark:text-zinc-400">
        Loading roles...
      </div>
    )
  }

  if (!roles || roles.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No roles available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-zinc-400">
          No roles found in this organization.
        </p>
      </div>
    )
  }

  return (
    <Select
      value={value ?? ''}
      onValueChange={(val) => onChange(val || undefined)}
    >
      <SelectTrigger className="w-full">
        <SelectValue placeholder="Select role" />
      </SelectTrigger>
      <SelectContent>
        {roles
          .filter((role) => role.metadata?.name && role.metadata.name.trim() !== '')
          .map((role) => (
            <SelectItem key={role.metadata!.name} value={role.metadata!.name!}>
              {role.spec?.displayName || role.metadata!.name}
            </SelectItem>
          ))}
      </SelectContent>
    </Select>
  )
}
