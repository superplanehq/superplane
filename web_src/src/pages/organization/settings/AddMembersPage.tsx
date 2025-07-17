import { useParams, Link } from 'react-router-dom'
import { Button } from '../../../components/Button/button'
import { MaterialSymbol } from '../../../components/MaterialSymbol/material-symbol'
import { Text } from '../../../components/Text/text'
import { AddMembersSection } from './AddMembersSection'

export function AddMembersPage() {
  const { orgId } = useParams<{ orgId: string }>()

  if (!orgId) {
    return <div>Error: Organization ID not found</div>
  }

  return (
    <div className="min-h-screen bg-zinc-50 dark:bg-zinc-900 pt-20">
      <div className="max-w-6xl mx-auto px-4 py-8">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-2 mb-4">
            <Link
              to={`/organization/${orgId}/settings/members`}
              className="flex items-center gap-2 text-zinc-600 dark:text-zinc-400 hover:text-zinc-900 dark:hover:text-white transition-colors"
            >
              <MaterialSymbol name="arrow_back" size="sm" />
              <span className="text-sm">Back to Members</span>
            </Link>
          </div>

          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-2xl font-semibold text-zinc-900 dark:text-white mb-2">
                Add Members
              </h1>
              <Text className="text-zinc-600 dark:text-zinc-400">
                Invite new members to join your organization
              </Text>
            </div>
          </div>
        </div>

        {/* Add Members Section */}
        <div className="space-y-6">
          <AddMembersSection
            organizationId={orgId}
            onMemberAdded={() => {
              // Optionally navigate back or show success message
              console.log('Member added successfully')
            }}
          />

          {/* Help Section */}
          <div className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6">
            <div className="flex items-start gap-4">
              <div className="bg-blue-100 dark:bg-blue-900/20 rounded-lg p-2">
                <MaterialSymbol name="help" className="h-5 w-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div className="flex-1">
                <Text className="font-medium text-zinc-900 dark:text-white mb-2">
                  How to add members
                </Text>
                <div className="space-y-2 text-sm text-zinc-600 dark:text-zinc-400">
                  <p>1. Enter the email address of the person you want to invite</p>
                  <p>2. Select the group they should be added to</p>
                  <p>3. Choose their role (if role selection is enabled)</p>
                  <p>4. Click "Add to Group" to send the invitation</p>
                </div>
                <div className="mt-4 p-3 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
                  <p className="text-sm text-yellow-800 dark:text-yellow-200">
                    <strong>Note:</strong> Members must be assigned to at least one group to access the organization.
                    Make sure you have created groups before adding members.
                  </p>
                </div>
              </div>
            </div>
          </div>

          {/* Action Buttons */}
          <div className="flex items-center gap-3">
            <Link to={`/organization/${orgId}/settings/members`}>
              <Button outline>
                <MaterialSymbol name="arrow_back" size="sm" />
                Back to Members
              </Button>
            </Link>
            <Link to={`/organization/${orgId}/settings/groups`}>
              <Button outline>
                <MaterialSymbol name="group" size="sm" />
                Manage Groups
              </Button>
            </Link>
          </div>
        </div>
      </div>
    </div>
  )
}