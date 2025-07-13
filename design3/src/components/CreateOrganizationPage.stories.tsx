import type { Meta, StoryObj } from '@storybook/react'
import { CreateOrganizationPage } from './CreateOrganizationPage'

const meta: Meta<typeof CreateOrganizationPage> = {
  title: 'Pages/CreateOrganizationPage',
  component: CreateOrganizationPage,
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component: 'A page for creating a new organization with name and URL validation.'
      }
    }
  },
  argTypes: {
    onBack: { action: 'back clicked' },
    onSuccess: { action: 'organization created' }
  }
}

export default meta
type Story = StoryObj<typeof CreateOrganizationPage>

export const Default: Story = {
  args: {}
}

export const WithBackButton: Story = {
  args: {
    onBack: () => console.log('Back clicked')
  }
}

export const WithCallbacks: Story = {
  args: {
    onBack: () => console.log('Back clicked'),
    onSuccess: (data) => console.log('Organization created:', data)
  }
}