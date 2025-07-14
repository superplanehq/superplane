import { Heading } from '../../../components/Heading/heading'
import { Input } from '../../../components/Input/input'
import { Field, Fieldset, Label } from '../../../components/Fieldset/fieldset'
import { Textarea } from '../../../components/Textarea/textarea'

interface GeneralSettingsProps {
  organizationName: string
}

export function GeneralSettings({ organizationName }: GeneralSettingsProps) {
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
            defaultValue={organizationName}
            className="max-w-lg"
          />
        </Field>
        <Field>
          <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
            Description
          </Label>
          <Textarea
            placeholder="Enter organization description"
            className="max-w-lg"
          />
        </Field>
      </Fieldset>
    </div>
  )
}