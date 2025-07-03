SuperPlane - Claude Assistant Guide
Project Overview
SuperPlane is a modern SaaS application built with cutting-edge web technologies, emphasizing clean architecture, reusable components, and professional UI/UX design.

Tech Stack
Frontend Framework: React (latest version)
Build Tool: Vite
Language: TypeScript
Styling: Tailwind CSS
Component Documentation: Storybook
Flow/Diagram Library: React Flow
Architecture: Clean architecture principles
Project Structure
superplane/
├── src/
│   ├── components/
│   │   └── lib/            # Reusable UI components library
│   │       └── component/  # Individual component folders
│   │           ├── index.tsx
│   │           ├── component.tsx
│   │           ├── types.ts
│   │           └── story/  # Component-specific stories
│   │               └── component.stories.tsx
│   ├── pages/              # Page components
│   ├── hooks/              # Custom React hooks
│   ├── utils/              # Utility functions
│   ├── types/              # TypeScript type definitions
│   ├── services/           # API services
│   ├── stores/             # State management
│   └── assets/             # Static assets
├── public/                 # Public assets
└── docs/                   # Documentation
Development Guidelines
Component Development
Follow React functional component patterns with hooks
Use TypeScript for all components with proper type definitions
Organize components in src/components/lib/component/ structure
Place Storybook stories in src/components/lib/component/story/ folder
Follow atomic design principles (atoms, molecules, organisms)
Ensure components are reusable and composable
Keep component files, types, and stories co-located
Styling Guidelines
Use Tailwind CSS utility classes for styling
Follow mobile-first responsive design
Maintain consistent design system (colors, spacing, typography)
Use Tailwind's component layer for complex reusable styles
Implement dark mode support where applicable
Code Quality Standards
Use TypeScript strict mode
Implement proper error handling
Follow ESLint and Prettier configurations
Use meaningful variable and function names
Add JSDoc comments for complex functions
Implement proper accessibility (a11y) practices
State Management
Use React hooks (useState, useReducer, useContext) for local state
Implement custom hooks for complex state logic
Consider state management libraries for global state if needed
Follow immutable state update patterns
API Integration
Create typed service functions for API calls
Implement proper error handling for network requests
Use async/await patterns consistently
Add loading states and error boundaries
React Flow Integration
Implement custom nodes and edges for flow diagrams
Use React Flow's built-in controls and minimap
Handle node/edge interactions properly
Implement custom styling for flow elements
Add drag-and-drop functionality for enhanced UX
Testing Strategy
Write unit tests for utility functions
Implement component tests with React Testing Library
Use Storybook for visual testing
Add integration tests for critical user flows
Maintain high test coverage
Performance Optimization
Implement code splitting with React.lazy()
Use React.memo() for expensive components
Optimize bundle size with proper tree shaking
Implement proper loading states and skeleton screens
Use appropriate caching strategies
UI/UX Principles
Follow modern SaaS design patterns
Implement intuitive navigation and user flows
Use consistent iconography and visual elements
Provide clear feedback for user actions
Ensure responsive design across all devices
Implement proper loading and error states
Storybook Integration
Create stories in src/components/lib/component/story/ folder
Keep stories co-located with their respective components
Use Storybook controls for interactive documentation
Implement visual regression testing
Document component APIs and usage examples
Organize stories with proper hierarchy and naming
Development Workflow
Create feature branches from main
Create component folders in src/components/lib/component/
Implement components with TypeScript and proper file structure
Add Tailwind styling following design system
Create Storybook stories in component's story/ folder
Write tests for new functionality
Ensure responsive design and accessibility
Submit pull requests with proper documentation
Common Patterns
Custom hooks for reusable logic
Compound components for complex UI elements
Render props pattern for flexible components
Higher-order components for cross-cutting concerns
Context providers for shared state
Best Practices
Keep components small and focused
Use proper TypeScript types throughout
Implement proper error boundaries
Follow React performance best practices
Use semantic HTML elements
Implement proper ARIA attributes
Maintain consistent naming conventions
Tools and Commands
bash
# Development
npm run dev          # Start development server
npm run build        # Build for production
npm run preview      # Preview production build

# Testing
npm run test         # Run tests
npm run test:watch   # Run tests in watch mode

# Storybook
npm run storybook    # Start Storybook
npm run build-storybook  # Build Storybook

# Linting
npm run lint         # Run ESLint
npm run lint:fix     # Fix ESLint issues
When Working with Claude
Provide specific component requirements and user stories
Share relevant code context when asking for help
Specify if you need Storybook stories or tests
Mention any specific design system constraints
Request TypeScript type definitions when needed
Ask for responsive design considerations
Specify if React Flow integration is required
Architecture Decisions
Prefer composition over inheritance
Use functional programming principles
Implement proper separation of concerns
Follow SOLID principles where applicable
Maintain clean import/export patterns
Use consistent file and folder naming
This guide helps ensure consistent development practices and provides context for effective collaboration with Claude on the SuperPlane project.

