import type { Meta, StoryObj } from '@storybook/react'
import { Heading, Subheading } from './heading'

const meta: Meta<typeof Heading> = {
  title: 'Components/Heading',
  component: Heading,
  parameters: {
    layout: 'padded',
  },
  tags: ['autodocs'],
  argTypes: {
    level: {
      control: 'select',
      options: [1, 2, 3, 4, 5, 6],
    },
  },
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    children: 'Page Heading',
    level: 1,
  },
}

export const Levels: Story = {
  render: () => (
    <div className="space-y-4">
      <Heading level={1}>Heading Level 1</Heading>
      <Heading level={2}>Heading Level 2</Heading>
      <Heading level={3}>Heading Level 3</Heading>
      <Heading level={4}>Heading Level 4</Heading>
      <Heading level={5}>Heading Level 5</Heading>
      <Heading level={6}>Heading Level 6</Heading>
    </div>
  ),
}

export const Subheadings: Story = {
  render: () => (
    <div className="space-y-4">
      <Subheading level={1}>Subheading Level 1</Subheading>
      <Subheading level={2}>Subheading Level 2</Subheading>
      <Subheading level={3}>Subheading Level 3</Subheading>
      <Subheading level={4}>Subheading Level 4</Subheading>
      <Subheading level={5}>Subheading Level 5</Subheading>
      <Subheading level={6}>Subheading Level 6</Subheading>
    </div>
  ),
}

export const Hierarchy: Story = {
  render: () => (
    <div className="space-y-6">
      <div>
        <Heading level={1}>Main Page Title</Heading>
        <p className="text-zinc-600 dark:text-zinc-400 mt-2">
          This is the main heading of the page, using level 1.
        </p>
      </div>
      
      <div>
        <Subheading level={2}>Section Title</Subheading>
        <p className="text-zinc-600 dark:text-zinc-400 mt-2">
          This is a section subheading, typically used for major sections.
        </p>
      </div>
      
      <div>
        <Heading level={2}>Secondary Heading</Heading>
        <p className="text-zinc-600 dark:text-zinc-400 mt-2">
          This is a regular heading at level 2, slightly smaller than the main title.
        </p>
      </div>
      
      <div>
        <Subheading level={3}>Subsection Title</Subheading>
        <p className="text-zinc-600 dark:text-zinc-400 mt-2">
          This is a subsection heading, used for smaller content groups.
        </p>
      </div>
      
      <div>
        <Heading level={3}>Tertiary Heading</Heading>
        <p className="text-zinc-600 dark:text-zinc-400 mt-2">
          This is a regular heading at level 3, for detailed sections.
        </p>
      </div>
    </div>
  ),
}

export const InContent: Story = {
  render: () => (
    <article className="max-w-2xl space-y-6">
      <header>
        <Heading level={1}>Building Modern Web Applications</Heading>
        <p className="text-lg text-zinc-600 dark:text-zinc-400 mt-4">
          A comprehensive guide to creating scalable and maintainable web applications using modern technologies.
        </p>
      </header>
      
      <section>
        <Subheading level={2}>Getting Started</Subheading>
        <p className="text-zinc-600 dark:text-zinc-400 mt-3">
          Before diving into the technical details, let's establish the fundamental concepts that will guide our development process.
        </p>
        
        <div className="mt-6">
          <Subheading level={3}>Prerequisites</Subheading>
          <p className="text-zinc-600 dark:text-zinc-400 mt-3">
            To follow along with this guide, you should have a basic understanding of web development concepts.
          </p>
        </div>
        
        <div className="mt-6">
          <Subheading level={3}>Setting Up Your Environment</Subheading>
          <p className="text-zinc-600 dark:text-zinc-400 mt-3">
            We'll start by setting up a development environment that supports all the tools we'll be using.
          </p>
        </div>
      </section>
      
      <section>
        <Subheading level={2}>Core Concepts</Subheading>
        <p className="text-zinc-600 dark:text-zinc-400 mt-3">
          Understanding these core concepts will help you make better architectural decisions throughout your project.
        </p>
      </section>
    </article>
  ),
}

export const CustomStyling: Story = {
  render: () => (
    <div className="space-y-6">
      <Heading level={1} className="text-blue-600 dark:text-blue-400">
        Colored Heading
      </Heading>
      
      <Subheading level={2} className="text-green-600 dark:text-green-400 border-b border-green-200 dark:border-green-800 pb-2">
        Subheading with Border
      </Subheading>
      
      <Heading level={3} className="bg-gradient-to-r from-purple-600 to-blue-600 bg-clip-text text-transparent">
        Gradient Heading
      </Heading>
      
      <Subheading level={2} className="text-center">
        Centered Subheading
      </Subheading>
      
      <Heading level={2} className="font-light text-zinc-500 dark:text-zinc-400">
        Light Weight Heading
      </Heading>
    </div>
  ),
}