import '../src/css/tachyons/tachyons.css';
import '../src/css/tailwind.css';
import '../src/css/app-semaphore.css';

/** @type { import('@storybook/react-webpack5').Preview } */
const preview = {
  parameters: {
    controls: {
      matchers: {
       color: /(background|color)$/i,
       date: /Date$/i,
      },
    },
  },
};

export default preview;