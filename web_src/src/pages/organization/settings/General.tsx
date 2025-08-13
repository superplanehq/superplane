import { Heading } from '../../../components/Heading/heading'
import { Input } from '../../../components/Input/input'
import { Field, Fieldset, Label } from '../../../components/Fieldset/fieldset'
import { Button } from '../../../components/Button/button'
import { useState } from 'react'
import { useUpdateOrganization } from '../../../hooks/useOrganizationData'
import type { OrganizationsOrganization } from '../../../api-client/types.gen'
import { useParams } from 'react-router-dom'
import { Textarea } from '@/components/Textarea/textarea'

interface GeneralProps {
  organization: OrganizationsOrganization
}

export function General({ organization }: GeneralProps) {
  const { organizationId } = useParams<{ organizationId: string }>()
  const [displayName, setDisplayName] = useState(organization.metadata?.displayName || '')
  const [saveMessage, setSaveMessage] = useState<string | null>(null)
  const [organizationDescription, setOrganizationDescription] = useState(organization.metadata?.description || '')

  // Use React Query mutation hook
  const updateOrganizationMutation = useUpdateOrganization(organizationId || '')

  const handleSave = async () => {
    if (!organizationId) {
      console.error('Organization ID is missing')
      return
    }

    try {
      setSaveMessage(null)

      await updateOrganizationMutation.mutateAsync({
        displayName: displayName,
        description: organizationDescription,
      })

      setSaveMessage('Organization updated successfully')
      setTimeout(() => setSaveMessage(null), 3000)
    } catch (err) {
      setSaveMessage('Failed to update organization')
      console.error('Error updating organization:', err)
      setTimeout(() => setSaveMessage(null), 3000)
    }
  }
  return (
    <div className="space-y-6 pt-6 text-left">
      <Heading level={2} className="text-2xl font-semibold text-zinc-900 dark:text-white">
        General
      </Heading>
      <Fieldset className="bg-white dark:bg-zinc-950 rounded-lg border border-zinc-200 dark:border-zinc-800 p-6 space-y-6 max-w-xl">
        <Field>
          <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
            Organization Name
          </Label>
          <Input
            type="text"
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
            className="max-w-lg"
          />
        </Field>
        <Field>
          <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
            Description
          </Label>
          <Textarea
            className='bg-white dark:bg-zinc-950 rounded-lg dark:border-zinc-800 max-w-xl'
            placeholder='Enter organization description'
            value={organizationDescription}
            onChange={(e) => setOrganizationDescription(e.target.value)}
          />
        </Field>

        <div className="flex items-center gap-4">
          <Button
            type="button"
            onClick={handleSave}
            disabled={updateOrganizationMutation.isPending}
            className="bg-blue-600 hover:bg-blue-700 text-white"
          >
            {updateOrganizationMutation.isPending ? 'Saving...' : 'Save Changes'}
          </Button>
          {saveMessage && (
            <span className={`text-sm ${saveMessage.includes('successfully') ? 'text-green-600' : 'text-red-600'}`}>
              {saveMessage}
            </span>
          )}
        </div>
      </Fieldset>
    </div>
  )
}