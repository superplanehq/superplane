// This file imports all custom execution renderers to ensure they're registered
// To add a new custom renderer:
// 1. Create a new file in this directory (e.g., httpRenderer.tsx)
// 2. Import and register it using registerExecutionRenderer
// 3. Import it here

export { registerExecutionRenderer, getExecutionRenderer, hasCustomRenderer } from './registry'
export type { ExecutionRenderer, ExecutionRendererProps } from './registry'

// Import custom renderers here to auto-register them
import './httpRenderer'
import './ifRenderer'
import './approvalRenderer'
import './filterRenderer'
