import { useState, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { Heading } from '../../../components/Heading/heading'
import { Button } from '../../../components/Button/button'
import { Input, InputGroup } from '../../../components/Input/input'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import debounce from 'lodash.debounce'
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
} from '../../../api-client/sdk.gen'
import { AuthorizationRole } from '../../../api-client/types.gen'
import { Tabs } from '@/components/Tabs/tabs'

interface RolesSettingsProps {
  organizationId: string
}


export function RolesSettings({ organizationId }: RolesSettingsProps) {
  const navigate = useNavigate()
  const [roles, setRoles] = useState<AuthorizationRole[]>([])
  const [loadingRoles, setLoadingRoles] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [roleTabs, setRoleTabs] = useState([
    {
      id: 'organization',
      label: 'Organization Roles',
      count: 0
    },
    {
      id: 'canvas',
      label: 'Canvas Roles',
      count: 0
    }
  ])

  const [activeRoleTab, setActiveRoleTab] = useState<'organization' | 'canvas'>('organization')

  const fetchDomainRoles = useCallback(async (domainType: 'organization' | 'canvas') => {
    const response = await authorizationListRoles({
      query: {
        domainType: domainType === 'organization' ? 'DOMAIN_TYPE_ORGANIZATION' : 'DOMAIN_TYPE_CANVAS',
        domainId: organizationId
      }
    })

    return response.data?.roles || []
  }, [organizationId])

  const setDebouncedSearch = debounce((search: string) => setSearch(search), 500)

  const setupRoles = useCallback(async () => {
    try {
      setLoadingRoles(true)
      setError(null)
      const organizationRoles = await fetchDomainRoles('organization')

      setRoleTabs(prev => prev.map((tab) => {
        if (tab.id === 'organization') {
          return {
            ...tab,
            count: organizationRoles.length
          }
        }
        return tab
      }))

      setRoles([...organizationRoles])
    } catch (err) {
      console.error('Error fetching roles:', err)
      setError('Failed to fetch roles')
    } finally {
      setLoadingRoles(false)
    }
  }, [fetchDomainRoles, setRoles])

  useEffect(() => {
    setupRoles()
  }, [setupRoles])

  const handleCreateRole = () => {
    navigate(`/organization/${organizationId}/settings/create-role`)
  }

  const capitalizeFirstLetter = (str: string) => {
    return str.charAt(0).toUpperCase() + str.slice(1)
  }

  const filteredRoles = useMemo(() => roles.filter((role) => {
    if (activeRoleTab === 'organization') {
      return role.domainType === 'DOMAIN_TYPE_ORGANIZATION'
    }
    return role.domainType === 'DOMAIN_TYPE_CANVAS'
  }).filter((role) => {
    if (search === '') {
      return true
    }
    return role.name?.toLowerCase().includes(search.toLowerCase())
  }), [roles, search, activeRoleTab])

  return (
    <div className="space-y-6 pt-6">
      <div className="flex items-center justify-between">
        <div>
          <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white mb-1">
            Roles
          </Heading>
        </div>
      </div>
      <Tabs
        tabs={roleTabs}
        defaultTab={activeRoleTab}
        onTabChange={(tabId) => setActiveRoleTab(tabId as 'organization' | 'canvas')}
        variant="underline"
      />

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error}</p>
        </div>
      )}

      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4 flex items-center justify-between">
          <InputGroup>
            <Input name="search" placeholder="Search Roles…" aria-label="Search" className="w-xs" onChange={(e) => setDebouncedSearch(e.target.value)} />
          </InputGroup>
          <Button
            color="blue"
            className='flex items-center'
            onClick={handleCreateRole}
          >
            <MaterialSymbol name="add" />
            New {capitalizeFirstLetter(activeRoleTab)} role
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
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {filteredRoles.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center py-8 text-zinc-500 dark:text-zinc-400">
                      No roles found
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredRoles.map((role, index) => (
                    <TableRow key={index}>
                      <TableCell className="font-medium">{capitalizeFirstLetter(role.name?.split('_').at(-1) || '')}</TableCell>
                      <TableCell>{role.permissions?.length || 0}</TableCell>
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
