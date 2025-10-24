import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../ui/select'
import { useOrganizationUsers } from '../../hooks/useOrganizationData'
import { ComponentsConfigurationField } from '../../api-client'

interface UserFieldRendererProps {
  field: ComponentsConfigurationField
  value: string
  onChange: (value: string | undefined) => void
  domainId: string
}

export const UserFieldRenderer = ({
  value,
  onChange,
  domainId
}: UserFieldRendererProps) => {
  // Fetch users from the organization
  const { data: users, isLoading, error } = useOrganizationUsers(domainId)

  if (!domainId || domainId.trim() === '') {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        User field requires domainId prop
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load users: {error instanceof Error ? error.message : 'Unknown error'}
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="text-sm text-gray-500 dark:text-zinc-400">
        Loading users...
      </div>
    )
  }

  if (!users || users.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No users available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-zinc-400">
          No users found in this organization.
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
        <SelectValue placeholder="Select user" />
      </SelectTrigger>
      <SelectContent>
        {users
          .filter((user) => user.metadata?.id && user.metadata.id.trim() !== '')
          .map((user) => (
            <SelectItem key={user.metadata!.id} value={user.metadata!.id!}>
              {user.metadata?.email || user.spec?.displayName || user.metadata!.id}
            </SelectItem>
          ))}
      </SelectContent>
    </Select>
  )
}
