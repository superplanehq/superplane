import React from 'react'
import type { Meta, StoryObj } from '@storybook/react'
import { Textarea } from './textarea'
import { Label, Field, Description } from '@headlessui/react'

const meta: Meta<typeof Textarea> = {
  title: 'Components/Textarea',
  component: Textarea,
  parameters: {
    layout: 'centered',
  },
  tags: ['autodocs'],
  argTypes: {
    resizable: { control: 'boolean' },
    disabled: { control: 'boolean' },
    invalid: { control: 'boolean' },
    rows: { control: 'number' },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    placeholder: 'Enter your message...',
    rows: 4,
  },
}

export const Sizes: Story = {
  render: () => (
    <div className="space-y-4 w-80">
      <Field>
        <Label>Small (3 rows)</Label>
        <Textarea rows={3} placeholder="Small textarea..." />
      </Field>
      <Field>
        <Label>Medium (4 rows)</Label>
        <Textarea rows={4} placeholder="Medium textarea..." />
      </Field>
      <Field>
        <Label>Large (6 rows)</Label>
        <Textarea rows={6} placeholder="Large textarea..." />
      </Field>
      <Field>
        <Label>Extra Large (8 rows)</Label>
        <Textarea rows={8} placeholder="Extra large textarea..." />
      </Field>
    </div>
  ),
}

export const States: Story = {
  render: () => (
    <div className="space-y-4 w-80">
      <Field>
        <Label>Normal</Label>
        <Textarea placeholder="Normal textarea" rows={3} />
      </Field>
      <Field>
        <Label>Disabled</Label>
        <Textarea placeholder="Disabled textarea" disabled rows={3} />
      </Field>
      <Field>
        <Label>Invalid</Label>
        <Textarea placeholder="Invalid textarea" invalid rows={3} />
        <Description className="text-red-500">This field has an error</Description>
      </Field>
      <Field>
        <Label>With Content</Label>
        <Textarea
          defaultValue="This textarea has some pre-filled content that demonstrates how text looks inside the component."
          rows={3}
        />
      </Field>
    </div>
  ),
}

export const Resizable: Story = {
  render: () => (
    <div className="space-y-4 w-80">
      <Field>
        <Label>Resizable (default)</Label>
        <Textarea 
          placeholder="You can resize this textarea vertically..." 
          rows={4}
          resizable={true}
        />
        <Description>Try dragging the bottom-right corner to resize</Description>
      </Field>
      <Field>
        <Label>Not Resizable</Label>
        <Textarea 
          placeholder="This textarea cannot be resized..." 
          rows={4}
          resizable={false}
        />
        <Description>This textarea has a fixed size</Description>
      </Field>
    </div>
  ),
}

export const WithDescriptions: Story = {
  render: () => (
    <div className="space-y-6 w-96">
      <Field>
        <Label>Feedback</Label>
        <Textarea 
          placeholder="Please share your thoughts..."
          rows={4}
        />
        <Description>Help us improve by sharing your feedback and suggestions.</Description>
      </Field>
      
      <Field>
        <Label>Description *</Label>
        <Textarea 
          placeholder="Describe your project..."
          rows={5}
          required
        />
        <Description>Provide a detailed description of your project (minimum 50 characters).</Description>
      </Field>
      
      <Field>
        <Label>Additional Notes</Label>
        <Textarea 
          placeholder="Any additional information..."
          rows={3}
        />
        <Description>Optional: Add any extra information that might be helpful.</Description>
      </Field>
    </div>
  ),
}

export const FormExample: Story = {
  render: () => (
    <form className="space-y-6 w-full max-w-lg">
      <div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-4">
          Contact Form
        </h3>
        
        <div className="space-y-4">
          <Field>
            <Label>Name *</Label>
            <input 
              type="text" 
              placeholder="Your full name"
              className="w-full px-3 py-2 border border-zinc-300 rounded-md dark:border-zinc-600 dark:bg-zinc-800"
              required 
            />
          </Field>
          
          <Field>
            <Label>Email *</Label>
            <input 
              type="email" 
              placeholder="your@email.com"
              className="w-full px-3 py-2 border border-zinc-300 rounded-md dark:border-zinc-600 dark:bg-zinc-800"
              required 
            />
          </Field>
          
          <Field>
            <Label>Subject *</Label>
            <input 
              type="text" 
              placeholder="What is this about?"
              className="w-full px-3 py-2 border border-zinc-300 rounded-md dark:border-zinc-600 dark:bg-zinc-800"
              required 
            />
          </Field>
          
          <Field>
            <Label>Message *</Label>
            <Textarea 
              placeholder="Please describe your inquiry in detail..."
              rows={6}
              required
            />
            <Description>Provide as much detail as possible to help us assist you better.</Description>
          </Field>
        </div>
      </div>
      
      <div className="flex gap-2">
        <button 
          type="submit"
          className="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"
        >
          Send Message
        </button>
        <button 
          type="button"
          className="px-4 py-2 bg-zinc-200 text-zinc-700 rounded-md hover:bg-zinc-300"
        >
          Cancel
        </button>
      </div>
    </form>
  ),
}

export const BlogPost: Story = {
  render: () => (
    <form className="space-y-6 w-full max-w-2xl">
      <div>
        <h3 className="text-lg font-semibold text-zinc-900 dark:text-white mb-4">
          Create Blog Post
        </h3>
        
        <div className="space-y-4">
          <Field>
            <Label>Title *</Label>
            <input 
              type="text" 
              placeholder="Enter a compelling title..."
              className="w-full px-3 py-2 border border-zinc-300 rounded-md dark:border-zinc-600 dark:bg-zinc-800"
              required 
            />
          </Field>
          
          <Field>
            <Label>Excerpt</Label>
            <Textarea 
              placeholder="Write a brief summary or excerpt..."
              rows={3}
            />
            <Description>A short description that appears in search results and previews.</Description>
          </Field>
          
          <Field>
            <Label>Content *</Label>
            <Textarea 
              placeholder="Write your blog post content here..."
              rows={12}
              required
            />
            <Description>Write your full blog post content. You can use Markdown for formatting.</Description>
          </Field>
          
          <Field>
            <Label>Tags</Label>
            <input 
              type="text" 
              placeholder="react, javascript, web-development"
              className="w-full px-3 py-2 border border-zinc-300 rounded-md dark:border-zinc-600 dark:bg-zinc-800"
            />
            <Description>Comma-separated list of tags to categorize your post.</Description>
          </Field>
        </div>
      </div>
      
      <div className="flex gap-2">
        <button 
          type="submit"
          className="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600"
        >
          Publish Post
        </button>
        <button 
          type="button"
          className="px-4 py-2 bg-zinc-500 text-white rounded-md hover:bg-zinc-600"
        >
          Save Draft
        </button>
        <button 
          type="button"
          className="px-4 py-2 bg-zinc-200 text-zinc-700 rounded-md hover:bg-zinc-300"
        >
          Cancel
        </button>
      </div>
    </form>
  ),
}

export const CodeInput: Story = {
  render: () => (
    <div className="space-y-4 w-full max-w-2xl">
      <Field>
        <Label>CSS Code</Label>
        <Textarea 
          placeholder="Enter your CSS code..."
          rows={8}
          className="font-mono text-sm"
          defaultValue={`.button {
  background-color: #3b82f6;
  color: white;
  padding: 0.5rem 1rem;
  border-radius: 0.375rem;
  border: none;
  cursor: pointer;
  transition: background-color 0.2s;
}

.button:hover {
  background-color: #2563eb;
}`}
        />
        <Description>Enter CSS code with syntax highlighting support.</Description>
      </Field>
      
      <Field>
        <Label>JavaScript Code</Label>
        <Textarea 
          placeholder="Enter your JavaScript code..."
          rows={6}
          className="font-mono text-sm"
          defaultValue={`function greetUser(name) {
  const greeting = \`Hello, \${name}!\`;
  console.log(greeting);
  return greeting;
}

greetUser('World');`}
        />
        <Description>Enter JavaScript code for your application.</Description>
      </Field>
    </div>
  ),
}

export const CharacterCount: Story = {
  render: () => {
    const [content, setContent] = React.useState('')
    const maxLength = 280
    const remaining = maxLength - content.length

    return (
      <div className="w-full max-w-md">
        <Field>
          <Label>Tweet</Label>
          <Textarea 
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder="What's happening?"
            rows={4}
            maxLength={maxLength}
          />
          <Description className={remaining < 20 ? 'text-red-500' : 'text-zinc-500'}>
            {remaining} characters remaining
          </Description>
        </Field>
      </div>
    )
  },
}