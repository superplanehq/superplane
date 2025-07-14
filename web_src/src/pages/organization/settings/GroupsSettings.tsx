import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Heading } from '../../../components/Heading/heading'
import { Button } from '../../../components/Button/button'
import { Input, InputGroup } from '../../../components/Input/input'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Avatar } from '../../../components/Avatar/avatar'
import { Link } from '../../../components/Link/link'
import { 
  Dropdown, 
  DropdownButton, 
  DropdownMenu, 
  DropdownItem
} from '../../../components/Dropdown/dropdown'
import { 
  Table, 
  TableHead, 
  TableBody, 
  TableRow, 
  TableHeader, 
  TableCell 
} from '../../../components/Table/table'
import { 
  authorizationListOrganizationGroups,
  authorizationCreateOrganizationGroup 
} from '../../../api-client/sdk.gen'
import { AuthorizationGroup } from '../../../api-client/types.gen'

interface GroupsSettingsProps {
  organizationId: string
}

export function GroupsSettings({ organizationId }: GroupsSettingsProps) {
  const navigate = useNavigate()
  const [groups, setGroups] = useState<AuthorizationGroup[]>([])
  const [loadingGroups, setLoadingGroups] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const fetchGroups = async () => {
      try {
        setLoadingGroups(true)
        setError(null)
        const response = await authorizationListOrganizationGroups({
          query: { organizationId }
        })
        if (response.data?.groups) {
          setGroups(response.data.groups)
        }
      } catch (err) {
        console.error('Error fetching groups:', err)
        setError('Failed to fetch groups')
      } finally {
        setLoadingGroups(false)
      }
    }

    fetchGroups()
  }, [organizationId])

  const handleCreateGroup = () => {
    navigate(`/organization/${organizationId}/settings/create-group`)
  }

  return (
    <div className="space-y-6 pt-6">
      <div className="flex items-center justify-between">
        <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
          Groups
        </Heading>
      </div>
      
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error}</p>
        </div>
      )}
      
      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4 flex items-center justify-between">
          <InputGroup>
            <Input name="search" placeholder="Search Groups…" aria-label="Search" className="w-xs" />
          </InputGroup>
          <Button 
            color="blue" 
            className='flex items-center'
            onClick={handleCreateGroup}
          >
            <MaterialSymbol name="add" />
            Create New Group
          </Button>
        </div>
        <div className="px-6 pb-6">
          {loadingGroups ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-zinc-500 dark:text-zinc-400">Loading groups...</p>
            </div>
          ) : (
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader>Group name</TableHeader>
                  <TableHeader>Domain Type</TableHeader>
                  <TableHeader>Role</TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {groups.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center py-8 text-zinc-500 dark:text-zinc-400">
                      No groups found
                    </TableCell>
                  </TableRow>
                ) : (
                  groups.map((group, index) => (
                    <TableRow key={index}>
                      <TableCell>
                        <div className="flex items-center gap-3">
                          <Avatar 
                            className='w-9' 
                            square 
                            initials={group.name?.charAt(0).toUpperCase() || 'G'} 
                          />
                          <div>
                            <Link href="#" className="cursor-pointer text-sm font-medium text-blue-600 dark:text-blue-400">
                              {group.name}
                            </Link>
                          </div>
                        </div>
                      </TableCell>
                      <TableCell>
                        <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400">
                          {group.domainType === 'DOMAIN_TYPE_ORGANIZATION' ? 'Organization' : 
                           group.domainType === 'DOMAIN_TYPE_CANVAS' ? 'Canvas' : 'Unknown'}
                        </span>
                      </TableCell>
                      <TableCell>{group.role || 'No role assigned'}</TableCell>
                      <TableCell>
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
                                Edit Group
                              </DropdownItem>
                              <DropdownItem>
                                <MaterialSymbol name="delete" />
                                Delete Group
                              </DropdownItem>
                            </DropdownMenu>
                          </Dropdown>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </div>
    </div>
  )
}