import { useState, useEffect } from 'react'
import { Heading } from '../../../components/Heading/heading'
import { Button } from '../../../components/Button/button'
import { Input, InputGroup } from '../../../components/Input/input'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
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
  authorizationListRoles,
  authorizationCreateRole 
} from '../../../api-client/sdk.gen'
import { AuthorizationRole } from '../../../api-client/types.gen'

interface RolesSettingsProps {
  organizationId: string
}

export function RolesSettings({ organizationId }: RolesSettingsProps) {
  const [roles, setRoles] = useState<AuthorizationRole[]>([])
  const [loadingRoles, setLoadingRoles] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [isCreatingRole, setIsCreatingRole] = useState(false)

  useEffect(() => {
    const fetchRoles = async () => {
      try {
        setLoadingRoles(true)
        setError(null)
        const response = await authorizationListRoles({
          query: {
            domainType: 'DOMAIN_TYPE_ORGANIZATION',
            domainId: organizationId
          }
        })
        if (response.data?.roles) {
          setRoles(response.data.roles)
        }
      } catch (err) {
        console.error('Error fetching roles:', err)
        setError('Failed to fetch roles')
      } finally {
        setLoadingRoles(false)
      }
    }

    fetchRoles()
  }, [organizationId])

  const handleCreateRole = async () => {
    const roleName = prompt('Enter role name:')
    if (!roleName?.trim()) return

    setIsCreatingRole(true)
    setError(null)

    try {
      await authorizationCreateRole({
        body: {
          name: roleName,
          domainType: 'DOMAIN_TYPE_ORGANIZATION',
          domainId: organizationId,
          permissions: [] // Empty permissions for now
        }
      })

      // Refresh roles list
      const response = await authorizationListRoles({
        query: {
          domainType: 'DOMAIN_TYPE_ORGANIZATION',
          domainId: organizationId
        }
      })
      if (response.data?.roles) {
        setRoles(response.data.roles)
      }
    } catch (err) {
      console.error('Error creating role:', err)
      setError('Failed to create role')
    } finally {
      setIsCreatingRole(false)
    }
  }

  return (
    <div className="space-y-6 pt-6">
      <div className="flex items-center justify-between">
        <div>
          <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-1">
            Roles
          </Heading>
        </div>
      </div>
      
      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error}</p>
        </div>
      )}
      
      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4 flex items-center justify-between">
          <InputGroup>
            <Input name="search" placeholder="Search Roles…" aria-label="Search" className="w-xs" />
          </InputGroup>
          <Button 
            color="blue" 
            className='flex items-center'
            onClick={handleCreateRole}
            disabled={isCreatingRole}
          >
            <MaterialSymbol name="add" />
            {isCreatingRole ? 'Creating...' : 'New role'}
          </Button>
        </div>
        <div className="px-6 pb-6">
          {loadingRoles ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-zinc-500 dark:text-zinc-400">Loading roles...</p>
            </div>
          ) : (
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader>Role name</TableHeader>
                  <TableHeader>Permissions</TableHeader>
                  <TableHeader>Domain Type</TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {roles.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center py-8 text-zinc-500 dark:text-zinc-400">
                      No roles found
                    </TableCell>
                  </TableRow>
                ) : (
                  roles.map((role, index) => (
                    <TableRow key={index}>
                      <TableCell className="font-medium">{role.name}</TableCell>
                      <TableCell>{role.permissions?.length || 0}</TableCell>
                      <TableCell>
                        <span className="inline-flex px-2 py-1 text-xs font-medium rounded-full bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400">
                          {role.domainType === 'DOMAIN_TYPE_ORGANIZATION' ? 'Organization' : 
                           role.domainType === 'DOMAIN_TYPE_CANVAS' ? 'Canvas' : 'Unknown'}
                        </span>
                      </TableCell>
                      <TableCell>
                        <div className="flex justify-end">
                          <Dropdown>
                            <DropdownButton plain>
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