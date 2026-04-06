# SuperPlane frontend — Bugbot rules

Use **blocking** findings for clear policy violations, security risks, or patterns we explicitly ban.
Use **non-blocking** for maintainability, consistency, and design expectations that deserve discussion but may be context-dependent.

---

## TypeScript safety

If a changed line adds `@ts-ignore`, then:

- Add a **blocking** Bug titled `Avoid @ts-ignore`
- Body: Prefer proper typing, narrowing with type guards, or rare `@ts-expect-error` with a one-line reason.

If a changed line introduces `as any` in application source (exclude
`**/*.test.*`, `**/__tests__/**`, `**/*.stories.*`, `**/*.spec.*`), then:

- Add a **non-blocking** Bug titled `Avoid unsafe any casts`
- Body: Prefer precise types, React helpers (`Children.toArray`, `isValidElement`), or narrow types instead of `as any`.

If new or changed function/component props or inline handlers use implicit `any`
on parameters (clearly untyped event args or callbacks in TSX/TS), then:

- Add a **non-blocking** Bug titled `Type handler and callback parameters`
- Body: Avoid implicit `any`; type event handlers and callbacks explicitly.

If a changed line introduces **explicit `any`** (`: any`, `Array<any>`, `Promise<any>`, `Record<string, any>`, etc.)
in application source (same exclusions as `as any` above), then:

- Add a **non-blocking** Bug titled `Avoid explicit any`
- Body: Prefer concrete types, generics, `unknown` with narrowing, or shared model types instead of `any`.

If the diff adds **long or multi-branch inline union types** (roughly **four or more** alternatives,
or a union/intersection that spans multiple lines) on props, variables, or return types
**without** a **named** `type` / `interface` at module scope, then:

- Add a **non-blocking** Bug titled `Name non-trivial unions and shapes`
- Body: Extract `type Foo = 'a' | 'b' | …` (or a small interface) so call sites and reviews stay readable; avoid anonymous mega-unions in signatures.

---

## Readability: nesting and conditionals

If the diff adds **deeply nested** non-JSX logic (many nested `if`/`else`, nested callbacks, or compound conditions
several levels deep) where **early returns**, **guard clauses**, or **small named helpers** would flatten the flow,
then:

- Add a **non-blocking** Bug titled `Reduce nesting — prefer early returns or helpers`
- Body: Flatten control flow with early exits and extracted functions; deep nesting is harder to review and test.

If the diff introduces **chained or nested ternary expressions** (`a ? b : c ? d : e`, or a single ternary whose
branches contain more ternaries / large JSX) **without** breaking into variables, `if` statements, or a small render
helper, then:

- Add a **non-blocking** Bug titled `Avoid heavy inline ternaries`
- Body: Prefer `if` + return, named booleans, or a tiny function over long `? :` chains in JSX or expressions.

---

## React components: size and complexity

If a single `*.tsx` file in the diff is **already large** (roughly **500+ lines** of implementation in that file
after the change) **or** the change **adds a large amount** of JSX, state, and handlers in one file without
extracting pieces, then:

- Add a **non-blocking** Bug titled `Large or dense component — consider splitting`
- Body: Prefer self-contained subcomponents, hooks, and `lib/` helpers. Very large files are harder to review, test, and reuse.

If the diff shows **deeply nested JSX** (many levels in one return), **very long** `useEffect` chains,
or **many unrelated concerns** in one component (e.g. data fetching + routing + large presentational tree + form logic)
without separation, then:

- Add a **non-blocking** Bug titled `High complexity — simplify structure`
- Body: Consider extracting hooks, subcomponents, or small pure helpers. Aim for local state with clear callbacks to parents.

---

## Forms and UI primitives

If the diff adds or changes raw HTML form elements (`<input>`, `<select>`, `<textarea>`, `<button>` used as a form control)
where `web_src/src/components/ui/` typically provides a shadcn equivalent, then:

- Add a **non-blocking** Bug titled `Prefer shadcn/ui primitives`
- Body: Use `@/components/ui` (Input, Label, Select, Textarea, Checkbox, Switch, Button, Dialog, etc.) instead of raw HTML when an equivalent exists.

---

## Exports and patterns

If the diff adds a **new** `export default` for a component or page module under `web_src/src` (not Storybook meta defaults, not Vite entry), then:

- Add a **non-blocking** Bug titled `Prefer named exports`
- Body: Named exports improve Vite compile behavior and IDE stability. See `web_src/AGENTS.md` (TypeScript).

---

## Data, UX, and states

If the diff adds **new** user-visible lists, tables, grids, or collections of data **without** any handling for
**empty**, **loading**, or **error** outcomes in the same change (or an obvious follow-up in the same feature area), then:

- Add a **non-blocking** Bug titled `Empty, loading, and error states`
- Body: Design for empty states, async feedback, and recoverable errors. Stories should cover key states when components are story-driven.

If the diff adds **mock or fixture data** inside a production component file
(not `*.stories.*` or test helpers), then:

- Add a **non-blocking** Bug titled `Keep mock data out of component modules`
- Body: Mock data belongs in Storybook stories or tests, not in shipped component files.

---

## Security

If the diff introduces `dangerouslySetInnerHTML`, `eval(`, `new Function(`, or string-constructed code execution patterns, then:

- Add a **blocking** Bug titled `Review injection / dynamic execution risk`
- Body: Ensure HTML is trusted or sanitized; avoid dynamic execution of untrusted strings. Flag for security review.
