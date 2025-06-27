# SaaS Application Development Guidelines

## Project Overview
This is a modern SaaS application built with React, TypeScript, Tailwind CSS, Storybook, and React Flow. The project emphasizes clean architecture, reusable components, and professional UI/UX design.

## Tech Stack
- **Frontend**: React 18+ with TypeScript
- **Styling**: Tailwind CSS + Tailwind Plus components
- **Documentation**: Storybook
- **Diagramming**: React Flow
- **Build Tool**: Vite
- **Package Manager**: npm/yarn/pnpm

## Code Standards

### TypeScript
- Use strict TypeScript configuration
- Define proper interfaces for all props and data structures
- Prefer type safety over `any` types
- Use proper generic types for reusable components

### React Patterns
- Use functional components with hooks
- Implement proper error boundaries for production components
- Follow React best practices for state management
- Use proper key props for lists and dynamic content

### Styling Guidelines
- Use Tailwind utility classes for styling
- Follow mobile-first responsive design approach
- Implement consistent spacing using Tailwind's spacing scale
- Use CSS custom properties for theme variables when needed
- Prefer Tailwind Plus components over custom implementations

### Component Architecture
- Create reusable, composable components
- Separate presentation components from business logic
- Use proper prop interfaces with TypeScript
- Include proper accessibility attributes (ARIA labels, roles, etc.)
- Document components with Storybook stories

## File Naming Conventions
- Components: PascalCase (e.g., `UserProfile.tsx`)
- Hooks: camelCase starting with "use" (e.g., `useApiData.ts`)
- Utils: camelCase (e.g., `formatDate.ts`)
- Types: PascalCase with descriptive names (e.g., `UserProfileProps`)
- Stories: ComponentName.stories.tsx

## Component Development Guidelines

### New Components
- Start with TypeScript interface for props
- Include proper JSDoc comments
- Implement responsive design by default
- Add accessibility features (keyboard navigation, screen reader support)
- Create corresponding Storybook story
- Handle loading and error states appropriately

### React Flow Components
- Custom nodes should extend base React Flow node types
- Use consistent styling that matches the main design system
- Implement proper data flow and state management
- Include proper TypeScript types for node data

### SaaS-Specific Features
- Implement proper authentication patterns
- Include proper error handling for API calls
- Use consistent loading states across the application
- Implement proper data validation
- Follow security best practices

## Performance Guidelines
- Use React.memo for expensive components
- Implement proper code splitting for routes
- Optimize images and assets
- Use proper dependency arrays in useEffect
- Avoid unnecessary re-renders

## Testing Approach
- Write unit tests for utility functions
- Test component behavior, not implementation details
- Include accessibility testing
- Test error states and edge cases

## Design System Integration
- Use design tokens for consistent spacing, colors, and typography
- Leverage Tailwind Plus components as the foundation
- Customize theme through Tailwind configuration
- Maintain visual consistency across all components
- Document design decisions in Storybook

## Development Workflow
- Create feature branches for new development
- Write meaningful commit messages
- Update Storybook stories when modifying components
- Test responsive behavior on multiple screen sizes
- Validate accessibility compliance

## Documentation Standards
- Include README files for complex features
- Document API integrations and data flows
- Maintain up-to-date Storybook stories
- Include inline code comments for complex logic
- Document deployment and environment setup

## Security Considerations
- Validate all user inputs
- Implement proper authentication and authorization
- Use environment variables for sensitive configuration
- Follow OWASP security guidelines
- Regularly update dependencies

When creating new features or components, always consider:
1. How does this fit into the overall design system?
2. Is this component reusable across different parts of the application?
3. Does this follow our established patterns and conventions?
4. Is this accessible to users with disabilities?
5. How will this perform at scale?

Focus on building a maintainable, scalable, and professional SaaS application that users will love to use.
