import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '../select'
import { useOrganizationGroups } from '../../hooks/useOrganizationData'
import { ConfigurationField } from '../../api-client'

interface GroupFieldRendererProps {
  field: ConfigurationField
  value: string
  onChange: (value: string | undefined) => void
  domainId: string
}

export const GroupFieldRenderer = ({
  value,
  onChange,
  domainId
}: GroupFieldRendererProps) => {
  // Fetch groups from the organization
  const { data: groups, isLoading, error } = useOrganizationGroups(domainId)

  if (!domainId || domainId.trim() === '') {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Group field requires domainId prop
      </div>
    )
  }

  if (error) {
    return (
      <div className="text-sm text-red-500 dark:text-red-400">
        Failed to load groups: {error instanceof Error ? error.message : 'Unknown error'}
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="text-sm text-gray-500 dark:text-zinc-400">
        Loading groups...
      </div>
    )
  }

  if (!groups || groups.length === 0) {
    return (
      <div className="space-y-2">
        <Select disabled>
          <SelectTrigger className="w-full">
            <SelectValue placeholder="No groups available" />
          </SelectTrigger>
        </Select>
        <p className="text-xs text-gray-500 dark:text-zinc-400">
          No groups found in this organization.
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
        <SelectValue placeholder="Select group" />
      </SelectTrigger>
      <SelectContent>
        {groups
          .filter((group) => group.metadata?.name && group.metadata.name.trim() !== '')
          .map((group) => (
            <SelectItem key={group.metadata!.name} value={group.metadata!.name!}>
              {group.spec?.displayName || group.metadata!.name}
            </SelectItem>
          ))}
      </SelectContent>
    </Select>
  )
}
