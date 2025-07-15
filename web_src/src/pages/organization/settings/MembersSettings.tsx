import { useState, useEffect } from 'react'
import { Heading } from '../../../components/Heading/heading'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Avatar } from '../../../components/Avatar/avatar'
import { Input, InputGroup } from '../../../components/Input/input'
import {
  Dropdown,
  DropdownButton,
  DropdownMenu,
  DropdownItem,
  DropdownLabel,
  DropdownDescription
} from '../../../components/Dropdown/dropdown'
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell
} from '../../../components/Table/table'
import { AddMembersSection } from './AddMembersSection'
import {
  authorizationListOrganizationGroups
} from '../../../api-client/sdk.gen'

interface Member {
  id: string
  name: string
  email: string
  role: string
  status: 'Active' | 'Pending' | 'Inactive'
  lastActive: string
  initials: string
  avatar?: string
}

interface MembersSettingsProps {
  organizationId: string
}

export function MembersSettings({ organizationId }: MembersSettingsProps) {
  const [members, setMembers] = useState<Member[]>([])
  const [loadingMembers, setLoadingMembers] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [searchTerm, setSearchTerm] = useState('')
  const [sortConfig, setSortConfig] = useState<{
    key: keyof Member | null
    direction: 'asc' | 'desc'
  }>({ key: null, direction: 'asc' })


  useEffect(() => {
    const fetchData = async () => {
      try {
        setLoadingMembers(true)
        setError(null)

        // Fetch groups (future implementation)
        await authorizationListOrganizationGroups({
          query: { organizationId }
        })

        // Mock data for demonstration - in real app, this would come from API
        const mockMembers: Member[] = [
          {
            id: '1',
            name: 'John Doe',
            email: 'john@company.com',
            role: 'Owner',
            status: 'Active',
            lastActive: '2 hours ago',
            initials: 'JD',
            avatar: 'https://images.unsplash.com/photo-1472099645785-5658abf4ff4e?w=64&h=64&fit=crop&crop=face'
          },
          {
            id: '2',
            name: 'Jane Smith',
            email: 'jane@company.com',
            role: 'Admin',
            status: 'Active',
            lastActive: '1 day ago',
            initials: 'JS'
          },
          {
            id: '3',
            name: 'Bob Wilson',
            email: 'bob@company.com',
            role: 'Member',
            status: 'Pending',
            lastActive: 'Never',
            initials: 'BW'
          },
          {
            id: '4',
            name: 'Alice Johnson',
            email: 'alice@company.com',
            role: 'Member',
            status: 'Active',
            lastActive: '3 days ago',
            initials: 'AJ'
          }
        ]

        // For now, use mock data for members
        // In a real implementation, you would fetch members from the API
        setMembers(mockMembers)

      } catch (err) {
        console.error('Error fetching data:', err)
        setError('Failed to fetch members')
      } finally {
        setLoadingMembers(false)
      }
    }

    fetchData()
  }, [organizationId])

  const handleSort = (key: keyof Member) => {
    setSortConfig(prevConfig => ({
      key,
      direction: prevConfig.key === key && prevConfig.direction === 'asc' ? 'desc' : 'asc'
    }))
  }

  const getSortIcon = (columnKey: keyof Member) => {
    if (sortConfig.key !== columnKey) {
      return 'unfold_more'
    }
    return sortConfig.direction === 'asc' ? 'keyboard_arrow_up' : 'keyboard_arrow_down'
  }

  const getSortedMembers = () => {
    if (!sortConfig.key) return members

    return [...members].sort((a, b) => {
      const aValue = a[sortConfig.key!]
      const bValue = b[sortConfig.key!]

      if (aValue == null && bValue == null) return 0
      if (aValue == null) return sortConfig.direction === 'asc' ? -1 : 1
      if (bValue == null) return sortConfig.direction === 'asc' ? 1 : -1

      if (aValue < bValue) {
        return sortConfig.direction === 'asc' ? -1 : 1
      }
      if (aValue > bValue) {
        return sortConfig.direction === 'asc' ? 1 : -1
      }
      return 0
    })
  }

  const getFilteredMembers = () => {
    const sorted = getSortedMembers()
    if (!searchTerm) return sorted

    return sorted.filter(member =>
      member.name.toLowerCase().includes(searchTerm.toLowerCase()) ||
      member.email.toLowerCase().includes(searchTerm.toLowerCase()) ||
      member.role.toLowerCase().includes(searchTerm.toLowerCase())
    )
  }

  const handleRoleChange = async (memberId: string, newRole: string) => {
    try {
      // In a real implementation, you would call the API to update the role
      // const response = await authorizationAssignRole({
      //   body: { userId: memberId, roleAssignment: { role: newRole, domainType: 'DOMAIN_TYPE_ORGANIZATION', domainId: organizationId } }
      // })

      setMembers(prev => prev.map(member =>
        member.id === memberId ? { ...member, role: newRole } : member
      ))
    } catch (err) {
      console.error('Error updating role:', err)
      setError('Failed to update member role')
    }
  }

  const handleMemberAction = async (memberId: string, action: 'edit' | 'suspend' | 'remove') => {
    try {
      switch (action) {
        case 'edit':
          console.log('Edit member:', memberId)
          break
        case 'suspend':
          setMembers(prev => prev.map(member =>
            member.id === memberId ? { ...member, status: 'Inactive' as const } : member
          ))
          break
        case 'remove':
          setMembers(prev => prev.filter(member => member.id !== memberId))
          break
      }
    } catch (err) {
      console.error('Error performing member action:', err)
      setError('Failed to perform action')
    }
  }

  const handleMemberAdded = () => {
    // Refresh members data when a new member is added
    console.log('Member added, refreshing data...')
    // In a real app, you would refetch the members list
  }

  return (
    <div className="space-y-6 pt-6">
      <div className="flex items-center justify-between">
        <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
          Members
        </Heading>
      </div>

      <AddMembersSection
        organizationId={organizationId}
        onMemberAdded={handleMemberAdded}
      />

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error}</p>
        </div>
      )}

      {/* Members List */}
      <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 overflow-hidden">
        <div className="px-6 pt-6 pb-4">
          <div className="flex items-center justify-between">
            <InputGroup>
              <Input
                name="search"
                placeholder="Search members…"
                aria-label="Search"
                className="w-xs"
                value={searchTerm}
                onChange={(e) => setSearchTerm(e.target.value)}
              />
            </InputGroup>
          </div>
        </div>

        <div className="px-6 pb-6">
          {loadingMembers ? (
            <div className="flex justify-center items-center h-32">
              <p className="text-zinc-500 dark:text-zinc-400">Loading members...</p>
            </div>
          ) : (
            <Table dense>
              <TableHead>
                <TableRow>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('name')}
                  >
                    <div className="flex items-center gap-2">
                      Name
                      <MaterialSymbol name={getSortIcon('name')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('email')}
                  >
                    <div className="flex items-center gap-2">
                      Email
                      <MaterialSymbol name={getSortIcon('email')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('role')}
                  >
                    <div className="flex items-center gap-2">
                      Role
                      <MaterialSymbol name={getSortIcon('role')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader
                    className="cursor-pointer hover:bg-zinc-50 dark:hover:bg-zinc-700/50"
                    onClick={() => handleSort('status')}
                  >
                    <div className="flex items-center gap-2">
                      Status
                      <MaterialSymbol name={getSortIcon('status')} size="sm" className="text-zinc-400" />
                    </div>
                  </TableHeader>
                  <TableHeader></TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {getFilteredMembers().map((member) => (
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
                            Last active: {member.lastActive}
                          </div>
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      {member.email}
                    </TableCell>
                    <TableCell>
                      <Dropdown>
                        <DropdownButton outline className="flex items-center gap-2 text-sm">
                          {member.role}
                          <MaterialSymbol name="keyboard_arrow_down" />
                        </DropdownButton>
                        <DropdownMenu>
                          <DropdownItem onClick={() => handleRoleChange(member.id, 'Owner')}>
                            <DropdownLabel>Owner</DropdownLabel>
                            <DropdownDescription>Full access to organization settings</DropdownDescription>
                          </DropdownItem>
                          <DropdownItem onClick={() => handleRoleChange(member.id, 'Admin')}>
                            <DropdownLabel>Admin</DropdownLabel>
                            <DropdownDescription>Can manage members and organization settings</DropdownDescription>
                          </DropdownItem>
                          <DropdownItem onClick={() => handleRoleChange(member.id, 'Member')}>
                            <DropdownLabel>Member</DropdownLabel>
                            <DropdownDescription>Standard member access</DropdownDescription>
                          </DropdownItem>
                        </DropdownMenu>
                      </Dropdown>
                    </TableCell>
                    <TableCell>
                      <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${member.status === 'Active'
                          ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                          : member.status === 'Pending'
                            ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-400'
                            : 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-400'
                        }`}>
                        {member.status}
                      </span>
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end">
                        <Dropdown>
                          <DropdownButton plain className="flex items-center gap-2 text-sm">
                            <MaterialSymbol name="more_vert" size="sm" />
                          </DropdownButton>
                          <DropdownMenu>
                            <DropdownItem onClick={() => handleMemberAction(member.id, 'edit')}>
                              <MaterialSymbol name="edit" />
                              Edit
                            </DropdownItem>
                            <DropdownItem onClick={() => handleMemberAction(member.id, 'suspend')}>
                              <MaterialSymbol name="block" />
                              Suspend
                            </DropdownItem>
                            <DropdownItem onClick={() => handleMemberAction(member.id, 'remove')}>
                              <MaterialSymbol name="delete" />
                              Remove
                            </DropdownItem>
                          </DropdownMenu>
                        </Dropdown>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
                {getFilteredMembers().length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center py-8">
                      <div className="text-zinc-500 dark:text-zinc-400">
                        <MaterialSymbol name="search" className="h-12 w-12 mx-auto mb-4 text-zinc-300" />
                        <p className="text-lg font-medium text-zinc-900 dark:text-white mb-2">
                          {searchTerm ? 'No members found' : 'No members yet'}
                        </p>
                        <p className="text-sm">
                          {searchTerm ? 'Try adjusting your search criteria' : 'Add members to get started'}
                        </p>
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </div>
      </div>
    </div>
  )
}