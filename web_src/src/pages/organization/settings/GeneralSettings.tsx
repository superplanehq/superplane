import { Heading } from '../../../components/Heading/heading'
import { Input } from '../../../components/Input/input'
import { Field, Fieldset, Label } from '../../../components/Fieldset/fieldset'
import { Textarea } from '../../../components/Textarea/textarea'
import { Link } from '../../../components/Link/link'

interface GeneralSettingsProps {
  organizationName: string
}

export function GeneralSettings({ organizationName }: GeneralSettingsProps) {
  return (
    <div className="space-y-6 pt-6">
      <Heading level={1} className="text-2xl font-semibold text-zinc-900 dark:text-white">
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
        
        <Field>
          <div className="flex items-start gap-4">
            <div className='w-1/2 flex-col gap-2'>
              <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Company Logo
              </Label>
              <div className="flex-none grow-0">
                <div className="inline-block h-15 py-4 bg-white dark:bg-zinc-700 rounded-lg border border-zinc-200 dark:border-zinc-600 border-dashed px-4">  
                  <img
                    src="https://upload.wikimedia.org/wikipedia/commons/a/ab/Confluent%2C_Inc._logo.svg"
                    alt="Confluent, Inc."
                    className='h-full'
                  />
                </div>
                <div className="flex items-center gap-2">
                  <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                    Upload new 
                  </Link>
                  <span className="text-xs text-zinc-500 dark:text-zinc-400">
                    &bull;
                  </span>
                  <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                    Remove  
                  </Link>
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  Rectangle image 96X20px
                </p>
              </div>
            </div>
            <div className='w-1/2 flex-col gap-2'>
              <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
                Company Icon
              </Label> 
              <div className="flex-none grow-0">
                <div className="w-15 h-15 inline-block py-4 bg-white dark:bg-zinc-700 rounded-lg border border-zinc-200 dark:border-zinc-600 border-dashed px-4">
                  <img
                    src="https://confluent.io/favicon.ico"
                    alt="Confluent, Inc."
                    height={24}
                  />
                </div>
              </div>
              <div className="flex flex-col">
                <div className="flex items-center gap-2">
                  <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                    Upload new 
                  </Link>
                  <span className="text-xs text-zinc-500 dark:text-zinc-400">
                    &bull;
                  </span>
                  <Link href="#" className="text-sm text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300">
                    Remove  
                  </Link>
                </div>
                <p className="text-xs text-zinc-500 dark:text-zinc-400">
                  Square image 64X64px
                </p>
              </div>
            </div>
          </div>
        </Field>
      </Fieldset>
    </div>
  )
}