import type { Preview } from '@storybook/react-vite'
import '../src/App.css'
import '../src/index.css'

// Load Material Symbols font for icons
const link = document.createElement('link')
link.href = 'https://fonts.googleapis.com/css2?family=Material+Symbols+Outlined'
link.rel = 'stylesheet'
document.head.appendChild(link)

const preview: Preview = {
  parameters: {
    options: {
      storySort: {
        method: 'alphabetical',
        locales: "en-US",
      },
    },
    controls: {
      matchers: {
       color: /(background|color)$/i,
       date: /Date$/i,
      },
    },
    backgrounds: {
      default: 'light',
      values: [
        {
          name: 'light',
          value: '#ffffff',
        },
        {
          name: 'dark',
          value: '#1a1a1a',
        },
      ],
    },
  },
};

export default preview;