import React from 'react';
import { designTokens } from '@/styles/tokens';

export const Welcome: React.FC = () => {
  const colorPalettes = [
    { name: 'Primary', colors: designTokens.colors.primary },
    { name: 'Secondary', colors: designTokens.colors.secondary },
    { name: 'Gray', colors: designTokens.colors.gray },
    { name: 'Success', colors: designTokens.colors.success },
    { name: 'Warning', colors: designTokens.colors.warning },
    { name: 'Error', colors: designTokens.colors.error },
  ];

  const typographySizes = [
    { name: 'xs', class: 'text-xs' },
    { name: 'sm', class: 'text-sm' },
    { name: 'base', class: 'text-base' },
    { name: 'lg', class: 'text-lg' },
    { name: 'xl', class: 'text-xl' },
    { name: '2xl', class: 'text-2xl' },
    { name: '3xl', class: 'text-3xl' },
    { name: '4xl', class: 'text-4xl' },
  ];

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900 p-8">
      <div className="max-w-6xl mx-auto space-y-12">
        {/* Header */}
        <div className="text-center space-y-4">
          <h1 className="text-4xl md:text-6xl font-bold text-gray-900 dark:text-white">
            Design System
          </h1>
          <p className="text-xl text-gray-600 dark:text-gray-300 max-w-3xl mx-auto">
            A comprehensive SaaS design system built with React, TypeScript, and Tailwind CSS.
            Ready for Tailwind Plus component integration.
          </p>
          <div className="flex items-center justify-center space-x-4 text-sm text-gray-500 dark:text-gray-400">
            <span className="inline-flex items-center px-3 py-1 rounded-full bg-primary-100 text-primary-800 dark:bg-primary-900 dark:text-primary-200">
              React 18+
            </span>
            <span className="inline-flex items-center px-3 py-1 rounded-full bg-secondary-100 text-secondary-800 dark:bg-secondary-900 dark:text-secondary-200">
              TypeScript
            </span>
            <span className="inline-flex items-center px-3 py-1 rounded-full bg-success-100 text-success-800 dark:bg-success-900 dark:text-success-200">
              Tailwind CSS
            </span>
            <span className="inline-flex items-center px-3 py-1 rounded-full bg-warning-100 text-warning-800 dark:bg-warning-900 dark:text-warning-200">
              Storybook
            </span>
          </div>
        </div>

        {/* Color Palette */}
        <section className="space-y-6">
          <h2 className="text-3xl font-bold text-gray-900 dark:text-white">Color Palette</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            {colorPalettes.map((palette) => (
              <div key={palette.name} className="space-y-3">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  {palette.name}
                </h3>
                <div className="grid grid-cols-5 gap-1 rounded-lg overflow-hidden shadow-sm">
                  {Object.entries(palette.colors).map(([shade, color]) => (
                    <div
                      key={shade}
                      className="aspect-square relative group cursor-pointer"
                      style={{ backgroundColor: color }}
                    >
                      <div className="absolute inset-0 flex items-center justify-center opacity-0 group-hover:opacity-100 transition-opacity">
                        <span className="text-xs font-mono text-white bg-black bg-opacity-50 px-1 rounded">
                          {shade}
                        </span>
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>

        {/* Typography */}
        <section className="space-y-6">
          <h2 className="text-3xl font-bold text-gray-900 dark:text-white">Typography</h2>
          <div className="space-y-4">
            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Font Sizes</h3>
              <div className="space-y-2">
                {typographySizes.map((size) => (
                  <div key={size.name} className="flex items-center space-x-4">
                    <span className="text-sm text-gray-500 dark:text-gray-400 w-12 font-mono">
                      {size.name}
                    </span>
                    <span className={`${size.class} text-gray-900 dark:text-white`}>
                      The quick brown fox jumps over the lazy dog
                    </span>
                  </div>
                ))}
              </div>
            </div>

            <div>
              <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-4">Headings</h3>
              <div className="space-y-3">
                <h1>Heading 1 - The quick brown fox</h1>
                <h2>Heading 2 - The quick brown fox</h2>
                <h3>Heading 3 - The quick brown fox</h3>
                <h4>Heading 4 - The quick brown fox</h4>
                <h5>Heading 5 - The quick brown fox</h5>
                <h6>Heading 6 - The quick brown fox</h6>
                <p>Paragraph - The quick brown fox jumps over the lazy dog. This is a sample paragraph to demonstrate the default text styling and line height.</p>
              </div>
            </div>
          </div>
        </section>

        {/* Components Preview */}
        <section className="space-y-6">
          <h2 className="text-3xl font-bold text-gray-900 dark:text-white">Component Examples</h2>
          
          {/* Buttons */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Buttons</h3>
            <div className="flex flex-wrap gap-4">
              <button className="btn-primary">Primary Button</button>
              <button className="btn-secondary">Secondary Button</button>
              <button className="btn-outline">Outline Button</button>
              <button className="btn-primary" disabled>Disabled Button</button>
            </div>
          </div>

          {/* Form Elements */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Form Elements</h3>
            <div className="max-w-md space-y-4">
              <input
                type="text"
                placeholder="Enter your text here..."
                className="input"
              />
              <textarea
                placeholder="Enter your message here..."
                className="input"
                rows={3}
              />
              <select className="input">
                <option>Choose an option</option>
                <option>Option 1</option>
                <option>Option 2</option>
                <option>Option 3</option>
              </select>
            </div>
          </div>

          {/* Cards */}
          <div className="space-y-4">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white">Cards</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
              <div className="card">
                <h4 className="text-xl font-semibold mb-2">Card Title</h4>
                <p className="text-gray-600 dark:text-gray-300 mb-4">
                  This is a sample card component with some content to demonstrate the styling.
                </p>
                <button className="btn-primary">Action</button>
              </div>
              <div className="card">
                <h4 className="text-xl font-semibold mb-2">Another Card</h4>
                <p className="text-gray-600 dark:text-gray-300 mb-4">
                  Cards can contain various types of content and are perfect for organizing information.
                </p>
                <button className="btn-outline">Learn More</button>
              </div>
              <div className="card">
                <h4 className="text-xl font-semibold mb-2">Third Card</h4>
                <p className="text-gray-600 dark:text-gray-300 mb-4">
                  The design system ensures consistency across all card components.
                </p>
                <button className="btn-secondary">Get Started</button>
              </div>
            </div>
          </div>
        </section>

        {/* Spacing Scale */}
        <section className="space-y-6">
          <h2 className="text-3xl font-bold text-gray-900 dark:text-white">Spacing Scale</h2>
          <div className="space-y-2">
            {[1, 2, 3, 4, 6, 8, 10, 12, 16, 20, 24].map((size) => (
              <div key={size} className="flex items-center space-x-4">
                <span className="text-sm text-gray-500 dark:text-gray-400 w-8 font-mono">
                  {size}
                </span>
                <div
                  className="bg-primary-500 h-4"
                  style={{ width: `${size * 0.25}rem` }}
                />
                <span className="text-sm text-gray-600 dark:text-gray-400 font-mono">
                  {size * 0.25}rem
                </span>
              </div>
            ))}
          </div>
        </section>

        {/* Footer */}
        <footer className="text-center pt-8 border-t border-gray-200 dark:border-gray-700">
          <p className="text-gray-500 dark:text-gray-400">
            Built with React, TypeScript, Tailwind CSS, and Storybook
          </p>
        </footer>
      </div>
    </div>
  );
};