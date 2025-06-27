# SaaS Design System

A modern, professional design system and component library built with React, TypeScript, and Tailwind CSS. Features comprehensive Storybook documentation, React Flow integration, and enterprise-ready components for SaaS applications.

## 🚀 Features

- **Modern Tech Stack**: React 19, TypeScript, Vite, Tailwind CSS
- **Component Library**: Reusable UI components with consistent design
- **Interactive Documentation**: Comprehensive Storybook with accessibility testing
- **Flow Diagrams**: React Flow integration for interactive diagrams
- **Type Safety**: Full TypeScript support with strict configuration
- **Professional Styling**: Tailwind CSS with custom design tokens
- **Code Quality**: ESLint, Prettier, and automated formatting
- **Accessibility**: Built-in a11y compliance and testing

## 🛠️ Tech Stack

- **Framework**: React 19 with TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS v4
- **Documentation**: Storybook
- **Diagrams**: React Flow
- **Code Quality**: ESLint + Prettier
- **Type Checking**: TypeScript 5.8+

## 📦 Getting Started

### Prerequisites

- Node.js 18+ 
- npm, yarn, or pnpm

### Installation

```bash
# Clone the repository
git clone <repository-url>
cd design2

# Install dependencies
npm install

# Start development server
npm run dev
```

### Development Commands

```bash
# Development
npm run dev              # Start dev server at http://localhost:5173
npm run build           # Build for production
npm run preview         # Preview production build

# Storybook
npm run storybook       # Start Storybook at http://localhost:6006
npm run build-storybook # Build Storybook for production

# Code Quality
npm run lint            # Run ESLint
npm run lint:fix        # Fix ESLint issues
npm run format          # Format code with Prettier
npm run format:check    # Check code formatting
npm run type-check      # Run TypeScript type checking
```

## 📁 Project Structure

```
src/
├── components/          # Reusable UI components
│   ├── ui/             # Basic UI components (buttons, inputs, etc.)
│   ├── layout/         # Layout components (header, sidebar, etc.)
│   ├── feature/        # Feature-specific components
│   └── flow/           # React Flow components
├── pages/              # Page components
├── stories/            # Storybook stories
├── styles/             # Global styles and design tokens
│   └── tokens.ts       # Design system tokens
├── types/              # TypeScript type definitions
├── utils/              # Utility functions
└── lib/                # External library configurations
```

## 🎨 Design System

The design system follows modern SaaS application patterns with:

- **Consistent Spacing**: Tailwind's spacing scale
- **Typography**: Professional font hierarchy
- **Color Palette**: Accessible color system
- **Components**: Reusable, composable UI elements
- **Responsive Design**: Mobile-first approach
- **Accessibility**: WCAG 2.1 compliance

### Design Tokens

Design tokens are centralized in `src/styles/tokens.ts` and integrated with Tailwind CSS configuration.

## 📚 Documentation

### Storybook

Access comprehensive component documentation at `http://localhost:6006` when running Storybook:

- Component API documentation
- Interactive examples
- Accessibility testing
- Design system guidelines

### Component Guidelines

- Use TypeScript interfaces for all props
- Include proper JSDoc comments
- Implement responsive design by default
- Add accessibility features (ARIA labels, keyboard navigation)
- Create corresponding Storybook stories
- Handle loading and error states

## 🔧 Development Guidelines

### Code Standards

- **TypeScript**: Strict configuration with proper typing
- **React**: Functional components with hooks
- **Styling**: Tailwind utility classes
- **Accessibility**: ARIA compliance and keyboard navigation
- **Testing**: Component behavior testing

### Component Development

1. Create component with TypeScript interface
2. Add responsive design and accessibility features
3. Create Storybook story
4. Test across different screen sizes
5. Validate accessibility compliance

### File Naming Conventions

- Components: `PascalCase.tsx`
- Hooks: `useCamelCase.ts`
- Utils: `camelCase.ts`
- Types: `PascalCase` interfaces
- Stories: `ComponentName.stories.tsx`

## 🧪 Testing

```bash
# Type checking
npm run type-check

# Linting
npm run lint

# Formatting check
npm run format:check
```

## 🚀 Deployment

```bash
# Build for production
npm run build

# Build Storybook
npm run build-storybook
```

The `dist/` folder contains the production build, and `storybook-static/` contains the built Storybook documentation.

## 🤝 Contributing

1. Follow the established code conventions
2. Update Storybook stories for component changes
3. Ensure accessibility compliance
4. Test responsive behavior
5. Run linting and type checking before commits

## 📄 License

This project is private and proprietary.

---

For detailed development guidelines and architecture decisions, see [CLAUDE.md](./CLAUDE.md).
