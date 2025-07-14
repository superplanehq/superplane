import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { Heading } from '../../../components/Heading/heading'
import { Button } from '../../../components/Button/button'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Avatar } from '../../../components/Avatar/avatar'
import { AddMembersSection } from './AddMembersSection'
import {
  authorizationListOrganizationGroups,
} from '../../../api-client/sdk.gen'
import { AuthorizationGroup } from '../../../api-client/types.gen'

interface MembersSettingsProps {
  organizationId: string
}

export function MembersSettings({ organizationId }: MembersSettingsProps) {
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

  const handleAddMembers = () => {
    navigate(`/organization/${organizationId}/settings/add-members`)
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
        onMemberAdded={() => {
          // Refresh groups when a member is added
          console.log('Member added, refreshing data...')
        }}
      />

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded">
          <p>{error}</p>
        </div>
      )}

      {loadingGroups ? (
        <div className="flex justify-center items-center h-32">
          <p className="text-zinc-500 dark:text-zinc-400">Loading members...</p>
        </div>
      ) : groups.length === 0 ? (
        <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-8 text-center">
          <div className="text-zinc-500 dark:text-zinc-400">
            <MaterialSymbol name="group" className="h-12 w-12 mx-auto mb-4 text-zinc-300" />
            <h3 className="text-lg font-medium text-zinc-900 dark:text-white mb-2">No groups found</h3>
            <p className="mb-4">Create groups to organize and manage members in your organization.</p>
            <Button
              color="blue"
              onClick={handleCreateGroup}
            >
              Create Your First Group
            </Button>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          <div className="bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg p-4">
            <div className="flex">
              <MaterialSymbol name="info" className="h-5 w-5 text-yellow-600 dark:text-yellow-400 mr-3 mt-0.5" />
              <div className="text-sm">
                <p className="text-yellow-800 dark:text-yellow-200 font-medium">
                  Members are organized by groups
                </p>
                <p className="text-yellow-700 dark:text-yellow-300 mt-1">
                  View and manage members by selecting a group below. To see all organization members, check each group individually.
                </p>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {groups.map((group, index) => (
              <div key={index} className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 hover:shadow-md transition-shadow">
                <div className="flex items-center gap-3 mb-4">
                  <Avatar
                    className='w-10 h-10'
                    square
                    initials={group.name?.charAt(0).toUpperCase() || 'G'}
                  />
                  <div>
                    <h3 className="text-lg font-medium text-zinc-900 dark:text-white">{group.name}</h3>
                    <p className="text-sm text-zinc-500 dark:text-zinc-400">
                      Role: {group.role || 'No role assigned'}
                    </p>
                  </div>
                </div>

                <div className="space-y-3">
                  <Button
                    outline
                    className="w-full text-sm"
                    onClick={() => {
                      // TODO: Implement view members functionality
                      console.log('View members for group:', group.name)
                    }}
                  >
                    <MaterialSymbol name="group" className="mr-2" />
                    View Members
                  </Button>
                  <div className="flex gap-2">
                    <Button
                      outline
                      className="flex-1 text-sm"
                      onClick={handleAddMembers}
                    >
                      <MaterialSymbol name="person_add" className="mr-2" />
                      Add Member
                    </Button>
                    <Button
                      outline
                      className="flex-1 text-sm"
                      onClick={() => {
                        // TODO: Implement edit group functionality
                        console.log('Edit group:', group.name)
                      }}
                    >
                      <MaterialSymbol name="edit" className="mr-2" />
                      Edit
                    </Button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}