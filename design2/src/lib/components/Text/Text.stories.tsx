import type { Meta, StoryObj } from '@storybook/react'
import { Text, TextLink, Strong, Code } from './text'

const meta: Meta<typeof Text> = {
  title: 'Components/Text',
  component: Text,
  parameters: {
    layout: 'padded',
  },
  tags: ['autodocs'],
}

export default meta
type Story = StoryObj<typeof meta>

export const Default: Story = {
  args: {
    children: 'This is a default text component.',
  },
}

export const AllVariants: Story = {
  render: () => (
    <div className="space-y-4 max-w-2xl">
      <Text>
        This is regular text. Lorem ipsum dolor sit amet, consectetur adipiscing elit. 
        Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua.
      </Text>
      
      <Text>
        Text with <Strong>strong emphasis</Strong> and <Code>inline code</Code>. 
        You can also include <TextLink href="#">text links</TextLink> within paragraphs.
      </Text>
      
      <Text>
        Code examples like <Code>npm install</Code> or <Code>const value = 42</Code> 
        are styled with a subtle background and border.
      </Text>
    </div>
  ),
}

export const TextOnly: Story = {
  render: () => (
    <div className="space-y-4 max-w-2xl">
      <Text>
        This is a paragraph of text that demonstrates the default typography styles. 
        The text should be readable and have appropriate spacing.
      </Text>
      
      <Text>
        Another paragraph to show how multiple text blocks look together. 
        Notice the consistent spacing and typography treatment across paragraphs.
      </Text>
      
      <Text>
        A longer paragraph to demonstrate how text flows and wraps. Lorem ipsum dolor sit amet, 
        consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. 
        Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat.
      </Text>
    </div>
  ),
}

export const StrongText: Story = {
  render: () => (
    <div className="space-y-4 max-w-2xl">
      <Text>
        This paragraph contains <Strong>important information</Strong> that should stand out 
        from the rest of the text.
      </Text>
      
      <Text>
        You can use <Strong>multiple strong elements</Strong> in the same paragraph to 
        <Strong>emphasize different parts</Strong> of your content.
      </Text>
      
      <Text>
        <Strong>Strong text</Strong> is perfect for highlighting key terms, 
        <Strong>important warnings</Strong>, or <Strong>call-to-action phrases</Strong>.
      </Text>
    </div>
  ),
}

export const CodeText: Story = {
  render: () => (
    <div className="space-y-4 max-w-2xl">
      <Text>
        To install the package, run <Code>npm install @headlessui/react</Code> in your terminal.
      </Text>
      
      <Text>
        You can access environment variables using <Code>process.env.NODE_ENV</Code> in Node.js 
        or <Code>import.meta.env.VITE_API_URL</Code> in Vite.
      </Text>
      
      <Text>
        Common CSS properties include <Code>display: flex</Code>, <Code>margin: 0 auto</Code>, 
        and <Code>border-radius: 8px</Code>.
      </Text>
      
      <Text>
        File paths like <Code>/src/components/Button.tsx</Code> or commands like 
        <Code>git commit -m "Initial commit"</Code> are clearly distinguished from regular text.
      </Text>
    </div>
  ),
}

export const TextLinks: Story = {
  render: () => (
    <div className="space-y-4 max-w-2xl">
      <Text>
        Visit our <TextLink href="#">documentation</TextLink> to learn more about the components.
      </Text>
      
      <Text>
        Check out the <TextLink href="#">getting started guide</TextLink> and 
        <TextLink href="#">API reference</TextLink> for detailed information.
      </Text>
      
      <Text>
        For support, contact us at <TextLink href="mailto:support@example.com">support@example.com</TextLink> 
        or visit our <TextLink href="#">help center</TextLink>.
      </Text>
    </div>
  ),
}

export const MixedContent: Story = {
  render: () => (
    <div className="space-y-6 max-w-2xl">
      <Text>
        <Strong>Getting Started:</Strong> To begin using this component library, 
        first install it with <Code>npm install @company/ui</Code> and then 
        import the components you need.
      </Text>
      
      <Text>
        The <Code>Button</Code> component accepts several props including <Code>variant</Code>, 
        <Code>size</Code>, and <Code>disabled</Code>. For more details, see the 
        <TextLink href="#">Button documentation</TextLink>.
      </Text>
      
      <Text>
        <Strong>Important:</Strong> Always test your components in both light and dark modes. 
        You can toggle themes using <Code>document.documentElement.classList.toggle('dark')</Code> 
        or check our <TextLink href="#">theming guide</TextLink> for more advanced patterns.
      </Text>
      
      <Text>
        Questions? Visit our <TextLink href="#">GitHub repository</TextLink> to report issues 
        or contribute to the project. We welcome <Strong>all contributions</Strong> and feedback!
      </Text>
    </div>
  ),
}

export const CustomStyling: Story = {
  render: () => (
    <div className="space-y-4 max-w-2xl">
      <Text className="text-blue-600 dark:text-blue-400">
        This text has custom blue coloring applied through className.
      </Text>
      
      <Text className="text-lg">
        This text is larger than the default size.
      </Text>
      
      <Text className="text-center">
        This text is center-aligned.
      </Text>
      
      <Text className="italic">
        This text is italicized for emphasis.
      </Text>
      
      <Text>
        You can also apply custom styles to <Strong className="text-green-600 dark:text-green-400">strong text</Strong>, 
        <Code className="bg-blue-50 border-blue-200 text-blue-800 dark:bg-blue-900/20 dark:border-blue-800 dark:text-blue-200">code snippets</Code>, 
        and <TextLink href="#" className="text-purple-600 dark:text-purple-400">text links</TextLink>.
      </Text>
    </div>
  ),
}

export const Documentation: Story = {
  render: () => (
    <article className="max-w-3xl space-y-6">
      <header>
        <h1 className="text-2xl font-bold text-zinc-900 dark:text-white mb-4">
          Text Component Documentation
        </h1>
      </header>
      
      <Text>
        The Text component is designed for body text and provides consistent typography 
        across your application. It includes several related components for different 
        text treatments.
      </Text>
      
      <section>
        <h2 className="text-xl font-semibold text-zinc-900 dark:text-white mb-3">
          Basic Usage
        </h2>
        
        <Text>
          Import the <Code>Text</Code> component and use it for paragraph content:
        </Text>
        
        <div className="bg-zinc-50 dark:bg-zinc-900 p-4 rounded-lg mt-3">
          <Code className="block">
            import &#123; Text &#125; from './text'
          </Code>
        </div>
      </section>
      
      <section>
        <h2 className="text-xl font-semibold text-zinc-900 dark:text-white mb-3">
          Related Components
        </h2>
        
        <Text>
          The text module exports several components for different use cases:
        </Text>
        
        <ul className="space-y-2 mt-3 ml-6">
          <li>
            <Text className="inline">
              <Strong>Text</Strong> - For regular paragraph content
            </Text>
          </li>
          <li>
            <Text className="inline">
              <Strong>Strong</Strong> - For emphasized text within paragraphs
            </Text>
          </li>
          <li>
            <Text className="inline">
              <Strong>Code</Strong> - For inline code snippets and technical terms
            </Text>
          </li>
          <li>
            <Text className="inline">
              <Strong>TextLink</Strong> - For links within text content
            </Text>
          </li>
        </ul>
      </section>
      
      <Text>
        For more examples and API details, visit our <TextLink href="#">complete documentation</TextLink>.
      </Text>
    </article>
  ),
}