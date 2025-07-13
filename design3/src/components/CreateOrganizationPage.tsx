import { useState } from 'react'
import { MaterialSymbol } from './lib/MaterialSymbol/material-symbol'
import { Heading, Subheading } from './lib/Heading/heading'
import { Text } from './lib/Text/text'
import { Button } from './lib/Button/button'
import { Input, InputGroup } from './lib/Input/input'
import { Field, Label } from './lib/Fieldset/fieldset'
import { Link } from './lib/Link/link'
import { NavigationOrg } from './lib/Navigation/navigation-org'

interface CreateOrganizationPageProps {
  onBack?: () => void
  onSuccess?: (organizationData: { name: string; url: string }) => void
}

export function CreateOrganizationPage({ 
  onBack, 
  onSuccess 
}: CreateOrganizationPageProps) {
  const [organizationName, setOrganizationName] = useState('')
  const [organizationUrl, setOrganizationUrl] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [errors, setErrors] = useState<{
    name?: string
    url?: string
  }>({})

  // Validate organization name
  const validateName = (name: string) => {
    if (!name.trim()) {
      return 'Organization name is required'
    }
    if (name.trim().length < 2) {
      return 'Organization name must be at least 2 characters'
    }
    if (name.trim().length > 50) {
      return 'Organization name must be less than 50 characters'
    }
    return null
  }

  // Validate organization URL
  const validateUrl = (url: string) => {
    if (!url.trim()) {
      return 'Organization URL is required'
    }
    if (url.trim().length < 3) {
      return 'URL must be at least 3 characters'
    }
    if (url.trim().length > 30) {
      return 'URL must be less than 30 characters'
    }
    if (!/^[a-z0-9-]+$/.test(url.trim())) {
      return 'URL can only contain lowercase letters, numbers, and hyphens'
    }
    if (url.trim().startsWith('-') || url.trim().endsWith('-')) {
      return 'URL cannot start or end with a hyphen'
    }
    return null
  }

  const handleNameChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    const value = event.target.value
    setOrganizationName(value)
    
    // Clear name error when user starts typing
    if (errors.name) {
      setErrors(prev => ({ ...prev, name: undefined }))
    }
  }

  const handleUrlChange = (event: React.ChangeEvent<HTMLInputElement>) => {
    // Convert to lowercase and remove invalid characters
    const value = event.target.value.toLowerCase().replace(/[^a-z0-9-]/g, '')
    setOrganizationUrl(value)
    
    // Clear URL error when user starts typing
    if (errors.url) {
      setErrors(prev => ({ ...prev, url: undefined }))
    }
  }

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    
    // Validate form
    const nameError = validateName(organizationName)
    const urlError = validateUrl(organizationUrl)
    
    if (nameError || urlError) {
      setErrors({
        name: nameError || undefined,
        url: urlError || undefined
      })
      return
    }

    setIsSubmitting(true)
    
    try {
      // Simulate API call
      await new Promise(resolve => setTimeout(resolve, 1500))
      
      const organizationData = {
        name: organizationName.trim(),
        url: organizationUrl.trim()
      }
      
      console.log('Creating organization:', organizationData)
      
      // Call success callback
      onSuccess?.(organizationData)
      
    } catch (error) {
      console.error('Failed to create organization:', error)
      // Handle error (could set an error state here)
    } finally {
      setIsSubmitting(false)
    }
  }

  const isFormValid = organizationName.trim() && organizationUrl.trim() && !errors.name && !errors.url

  return (
    <div className="min-h-screen bg-white dark:bg-zinc-900 flex flex-col items-start justify-center">
    <NavigationOrg className='w-full'/>

        <div className="w-full max-w-md mt-6 pt-6 flex-1 m-auto">
         {/* Header */}
         <div className="text-center mb-8">
         
         <Heading level={1} className="text-2xl font-bold text-zinc-900 dark:text-white mb-2">
           Create your organization
         </Heading>
         <Text className="text-zinc-600 dark:text-zinc-400">
           Subtitle if needed
         </Text>
         </div>

       {/* Form */}
       <form onSubmit={handleSubmit} className="space-y-6">
         <div className="bg-white dark:bg-zinc-800 rounded-lg space-y-6">
           {/* Organization Name */}
           <Field>
             <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
               Organization name *
             </Label>
             <Input
               type="text"
               value={organizationName}
               onChange={handleNameChange}
               placeholder="Enter your organization name"
               className={errors.name ? 'border-red-500 dark:border-red-400' : ''}
               disabled={isSubmitting}
             />
             {errors.name && (
               <Text className="text-sm text-red-600 dark:text-red-400 mt-1">
                 {errors.name}
               </Text>
             )}
           </Field>

           {/* Organization URL */}
           <Field>
             <Label className="block text-sm font-medium text-zinc-700 dark:text-zinc-300 mb-2">
               Organization URL *
             </Label>
             <div className="flex items-center">
               <Input
                 type="text"
                 value={organizationUrl}
                 onChange={handleUrlChange}
                 placeholder="your-org"
                 className={`!rounded-r-none !border-r-0 ${errors.url ? 'border-red-500 dark:border-red-400' : ''}`}
                 disabled={isSubmitting}
               />
               <div className="px-3 py-2 bg-zinc-50 dark:bg-zinc-700 border border-l-0 border-zinc-300 dark:border-zinc-600 rounded-r-md text-sm text-zinc-500 dark:text-zinc-400">
                 .superplane.com
               </div>
             </div>
             {errors.url && (
               <Text className="text-sm text-red-600 dark:text-red-400 mt-1">
                 {errors.url}
               </Text>
             )}
             {organizationUrl && !errors.url && (
               <Text className="text-sm text-zinc-500 dark:text-zinc-400 mt-1">
                 Your organization will be available at: <span className="font-medium">{organizationUrl}.superplane.com</span>
               </Text>
             )}
           </Field>
         </div>

         {/* Actions */}
         <div className="flex gap-3">
           {onBack && (
             <Button 
               type="button" 
               plain 
               onClick={onBack}
               disabled={isSubmitting}
               className="flex-1"
             >
               Back
             </Button>
           )}
           <Button 
             type="submit" 
             color="blue"
             disabled={!isFormValid || isSubmitting}
             className="flex-1 flex items-center justify-center gap-2"
           >
             {isSubmitting ? (
               <>
                 <MaterialSymbol name="progress_activity" className="animate-spin" />
                 Creating...
               </>
             ) : (
               <>
                 <MaterialSymbol name="add" />
                 Create Organization
               </>
             )}
           </Button>
         </div>
       </form>

        </div>
       
    </div>
  )
}