import React from 'react';
import { DevTools } from './devtools';

// Export named FlowDevTools component for imports that expect it
export const FlowDevTools: React.FC = () => {
  // Only render in development environment using Vite's built-in variable
  if (!import.meta.env.DEV) {
    return null;
  }

  return <DevTools />;
};

// Also export as default for imports that use default
export default FlowDevTools;
