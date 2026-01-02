# Quality Standards

This document outlines the high-level principles and standards we follow to ensure SuperPlane delivers exceptional
value to our users while maintaining a codebase that is maintainable, testable, and AI-friendly.

## User-First Thinking

Every line of code we write should be evaluated through the lens of user value and user experience:

- **Solve real problems** - Before writing code, deeply understand the user problem you're solving. Is this feature something users actually need? Will it make their lives meaningfully better?
- **Prioritize UX** - Code quality isn't just about clean code, it's about creating genuinely delightful experiences. Performance, reliability, and intuitive interfaces aren't nice-to-haves—they're requirements.
- **Think end-to-end** - Consider how your changes affect the entire user journey, not just your immediate feature. Every interaction matters.
- **Measure impact** - Use metrics and user feedback to validate that your changes dramatically improve the product experience. Good isn't good enough.

Remember: Beautiful code that doesn't serve users is technical debt in disguise.

## Maintainable and AI-Drivable Code

SuperPlane is built to be AI-native. This means our code should be:

- **Self-documenting** - Code should explain itself through clear naming, structure, and organization. Comments should explain "why," not "what."
- **Consistent patterns** - Follow established patterns in the codebase. When AI agents read your code, they should be able to infer similar patterns elsewhere.
- **Well-structured** - Organize code logically. Clear separation of concerns makes it easier for both humans and AI to understand and modify.
- **Type-safe** - Leverage TypeScript and Go's type systems. Types serve as documentation and catch errors early.
- **Predictable** - Avoid clever tricks and magic. Prefer explicit, straightforward solutions that are easy to reason about.

The best code is code that both you and an AI agent can understand and modify confidently months later.

## Backward Compatibility

SuperPlane is used in production environments. Breaking changes have real consequences:

- **Preserve APIs** - When modifying APIs (REST, gRPC, or internal), maintain backward compatibility when possible. Use versioning for breaking changes.
- **Gradual migrations** - When introducing breaking changes, provide migration paths and deprecation warnings.
- **Database schema** - Database migrations should be additive when possible. Breaking schema changes require careful planning and communication.
- **Configuration** - Respect existing configuration formats. Introduce new options without breaking existing setups.

When breaking changes are necessary, document them clearly, provide migration guides, and consider the impact on existing users.

## Comprehensive Testing

Quality is built through testing, not verified after the fact:

- **Test coverage** - Aim for comprehensive test coverage, especially for critical paths and business logic. Don't settle for "good enough", cover the edge cases too.
- **Test types** - Use the right test for the job:
  - **Unit tests** - Fast, isolated tests for individual functions and components
  - **Integration tests** - Test interactions between components
  - **E2E tests** - Test complete user workflows (see [E2E Testing](e2e-tests.md))
- **Test quality** - Excellent tests are readable, maintainable, and test behavior, not implementation details. Write tests that tell a story.
- **Test-driven development** - Consider writing tests first to clarify requirements and design. Let tests guide your architecture.

Tests are not just about preventing bugs, they're documentation, design tools, and confidence builders that let us ship fearlessly.

## Industry Best Practices

SuperPlane should be the best product in the industry. This means:

- **Study excellence** - Learn from the best products in the industry. What do they do well? Where can we be dramatically better? Don't just match. Surpass.
- **Follow and exceed standards** - Adhere to language-specific best practices, security guidelines, and architectural patterns proven in production, then push beyond them.
- **Performance is non-negotiable** - Optimize aggressively for speed, efficiency, and resource usage. Users notice when things are slow. Make SuperPlane blazingly fast.
- **Security first, always** - Security is not an afterthought. Follow security best practices, handle secrets properly, and validate inputs. Build security into every layer.
- **Observability by design** - Make the system deeply observable with comprehensive logging, metrics, and error handling. Production issues should be trivial to diagnose.
