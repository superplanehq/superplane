import { useState, forwardRef, useImperativeHandle, useMemo } from 'react'
import { Button } from '../../Button/button'
import { Input, InputGroup } from '../../Input/input'
import { Avatar } from '../../Avatar/avatar'
import { Checkbox } from '../../Checkbox/checkbox'
import { MaterialSymbol } from '../../MaterialSymbol/material-symbol'
import { Text } from '../../Text/text'
import {
  useCanvasUsers,
  useOrganizationUsersForCanvas,
  useAddCanvasUser
} from '../../../hooks/useCanvasData'

interface AddCanvasMembersSectionProps {
  canvasId: string
  organizationId: string
  onMemberAdded?: () => void
  className?: string
}

export interface AddCanvasMembersSectionRef {
  refreshExistingMembers: () => void
}

const AddCanvasMembersSectionComponent = forwardRef<AddCanvasMembersSectionRef, AddCanvasMembersSectionProps>(
  ({ canvasId, organizationId, onMemberAdded, className }, ref) => {
    const [selectedMembers, setSelectedMembers] = useState<Set<string>>(new Set())
    const [memberSearchTerm, setMemberSearchTerm] = useState('')

    const { data: canvasUsers = [] } = useCanvasUsers(canvasId)
    const { data: orgUsers = [], isLoading: loadingOrgUsers, error: orgUsersError } = useOrganizationUsersForCanvas(organizationId)

    const addUserMutation = useAddCanvasUser(canvasId)

    const isInviting = addUserMutation.isPending
    const error = orgUsersError



    const existingMembers = useMemo(() => {
      const canvasMemberIds = new Set(canvasUsers.map(user => user.metadata?.id))
      return orgUsers.filter(user => !canvasMemberIds.has(user.metadata?.id))
    }, [orgUsers, canvasUsers])

    const loadingMembers = loadingOrgUsers


    useImperativeHandle(ref, () => ({
      refreshExistingMembers: () => {

      }
    }), [])


    const handleExistingMembersSubmit = async () => {
      if (selectedMembers.size === 0) return

      try {
        const selectedUsers = existingMembers.filter(member => selectedMembers.has(member.metadata!.id!))

        for (const member of selectedUsers) {
          await addUserMutation.mutateAsync({
            userId: member.metadata?.id!,
          })
        }

        setSelectedMembers(new Set())
        setMemberSearchTerm('')

        onMemberAdded?.()
      } catch {
        console.error('Failed to add existing members')
      }
    }

    const handleSelectAll = () => {
      const filteredMembers = getFilteredExistingMembers()
      if (selectedMembers.size === filteredMembers.length) {
        setSelectedMembers(new Set())
      } else {
        setSelectedMembers(new Set(filteredMembers.map(m => m.metadata!.id!)))
      }
    }

    const getFilteredExistingMembers = () => {
      if (!memberSearchTerm) return existingMembers

      return existingMembers.filter(member =>
        member.spec?.displayName?.toLowerCase().includes(memberSearchTerm.toLowerCase()) ||
        member.metadata?.email?.toLowerCase().includes(memberSearchTerm.toLowerCase())
      )
    }


    return (
      <div className={`bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 ${className}`}>
        <div className="flex items-center justify-between mb-4 text-left">
          <div>
            <Text className="font-semibold text-zinc-900 dark:text-white mb-1">
              Add members
            </Text>
            <Text className="text-sm text-zinc-600 dark:text-zinc-400">
              Add members from your organization to this canvas and assign them a role.
            </Text>
          </div>
        </div>

        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
            <p className="text-sm">{error instanceof Error ? error.message : 'Failed to fetch data'}</p>
          </div>
        )}

        <div className="space-y-4">
            <div className="flex items-center justify-between">
              <InputGroup>
                <Input
                  name="member-search"
                  placeholder="Search members..."
                  aria-label="Search members"
                  className="w-xs"
                  value={memberSearchTerm}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => setMemberSearchTerm(e.target.value)}
                />
              </InputGroup>
              <div className="flex items-center gap-2">
                <Button
                  outline
                  className="flex items-center gap-2 text-sm"
                  onClick={handleSelectAll}
                  disabled={loadingMembers || getFilteredExistingMembers().length === 0}
                >
                  <MaterialSymbol name="select_all" size="sm" />
                  {selectedMembers.size === getFilteredExistingMembers().length ? 'Deselect All' : 'Select All'}
                </Button>
                <Button
                  color="blue"
                  className="flex items-center gap-2 text-sm"
                  onClick={handleExistingMembersSubmit}
                  disabled={selectedMembers.size === 0 || isInviting}
                >
                  <MaterialSymbol name="add" size="sm" />
                  {isInviting ? 'Adding...' : `Add ${selectedMembers.size} member${selectedMembers.size === 1 ? '' : 's'}`}
                </Button>
              </div>
            </div>


            {loadingMembers ? (
              <div className="flex justify-center items-center h-32">
                <p className="text-zinc-500 dark:text-zinc-400">Loading members...</p>
              </div>
            ) : (
              <div className="max-h-96 overflow-y-auto border border-zinc-200 dark:border-zinc-700 rounded-lg">
                {getFilteredExistingMembers().length === 0 ? (
                  <div className="text-center py-8">
                    <p className="text-zinc-500 dark:text-zinc-400">
                      {memberSearchTerm ? 'No members found matching your search' : 'No organization members available to add'}
                    </p>
                  </div>
                ) : (
                  <div className="divide-y divide-zinc-200 dark:divide-zinc-700">
                    {getFilteredExistingMembers().map((member) => (
                      <div key={member.metadata?.id} className="p-3 flex items-center gap-3 hover:bg-zinc-50 dark:hover:bg-zinc-800">
                        <Checkbox
                          checked={selectedMembers.has(member.metadata!.id!)}
                          onChange={(checked) => {
                            setSelectedMembers(prev => {
                              const newSet = new Set(prev)
                              if (checked) {
                                newSet.add(member.metadata!.id!)
                              } else {
                                newSet.delete(member.metadata!.id!)
                              }
                              return newSet
                            })
                          }}
                        />
                        <Avatar
                          src={member.spec?.accountProviders?.[0]?.avatarUrl}
                          initials={member.spec?.displayName?.charAt(0) || 'U'}
                          className="size-8"
                        />
                        <div className="flex-1 min-w-0">
                          <div className="text-sm font-medium text-zinc-900 dark:text-white truncate">
                            {member.spec?.displayName || member.metadata?.id}
                          </div>
                          <div className="text-xs text-zinc-500 dark:text-zinc-400 truncate">
                            {member.metadata?.email || "Invalid email"}
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
        </div>
      </div>
    )
  })

AddCanvasMembersSectionComponent.displayName = 'AddCanvasMembersSection'

export const AddCanvasMembersSection = AddCanvasMembersSectionComponent