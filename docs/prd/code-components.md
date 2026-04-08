# Code Components

> This is an RFD (Request for Discussion), not a spec. It's meant to frame the problem, explore the design space, and start the conversation. Once we agree on the direction and details, it'll be converted into a PRD.

Sometimes a workflow needs a bit of custom logic and the existing components are too rigid. Parse a weird payload. Normalize fields from two systems that don't line up. Apply some business logic that's too awkward to express with conditions and mappings. This document is about a new kind of component for those cases: one that lets a user write a small piece of JavaScript or Python inside the canvas.

The key product question is what kind of thing this should be. A small programmable transform? A full programmable action that can do side effects? Something in between?

## Why

There are a few problems this could solve.

**The last mile is hard.** Most workflows are easy to build with existing components until you hit one awkward bit of logic. The data shape is slightly off. The conditional is too custom. You need to compute something that's easy in code and painful in a visual builder.

**Users leave the product for small things.** When the workflow builder can't express a piece of logic cleanly, people end up reaching for external services, cloud functions, or sidecar scripts. That breaks the flow of building and debugging in one place.

**Some logic is too specific to deserve a full integration.** We don't want every small transformation or scoring rule to become a first-class component in the product.

## What this component is

The cleanest starting point is a **programmable logic component**.

It takes input from upstream, runs user-authored JavaScript or Python, and emits structured output downstream. It should feel like a normal canvas component, not like leaving the product to go run code somewhere else.

That means the product needs to make six things obvious:

- what data goes in
- what code runs
- what data comes out
- how you test it
- how you debug it
- what the code is allowed to access

## Product directions

There are three possible directions here.

### 1. Transform / logic component

This is the narrowest version. The component takes structured input, runs code, and returns structured output. Its job is shaping data and expressing logic.

This is the best starting point. It's easy to explain, useful immediately, and doesn't turn into a mini platform.

### 2. Full programmable action

This version can do everything the transform version does, but it can also make network calls and do side effects.

This is powerful, but it turns the component into a tiny runtime product. Now the discussion is about network access, secrets, retries, idempotency, side effects, and where it overlaps with HTTP and integration components.

### 3. Two-step path

Start with the transform / logic component, then later expand into a more powerful programmable action if the product needs it.

This is probably the right long-term shape. The important part is not to pretend those two things are the same feature from day one.

## Recommendation

The first version should be a **transform / logic component**.

It should be positioned as the component you reach for when the visual workflow builder gets you 90 percent of the way there and you need code for the last 10 percent. That framing keeps it small, understandable, and complementary to the rest of the product.

If we start with a full programmable action, the feature gets harder to explain and much harder to reason about. It also starts competing with the rest of the component model instead of extending it.

## How others do it

Temporal and Windmill are the more relevant references here, but they represent two very different product philosophies.

| | **Temporal** | **Windmill** |
|---|---|---|
| **Product model** | The workflow system is code | Visual workflows built from scripts |
| **Closest concept** | Workflows, activities, child workflows | Script step inside a flow |
| **Who it is for** | Developers building durable systems | Users building visual workflows who also want real code |
| **What to borrow** | Execution semantics, retries, failure behavior, clear boundaries | Product shape, inputs/outputs, logs, testing, editor UX |

The useful lesson from Temporal is not "make the workflow code-first." It's that code execution units need clear execution semantics. Inputs, outputs, retries, timeouts, and failure behavior should not feel hand-wavy.

The useful lesson from Windmill is that code can be a first-class building block inside a visual workflow product. If we do this, the component should not feel like an awkward escape hatch. It should have proper inputs, outputs, logs, testing, and visibility.

## What it should look like

On the canvas this should look like any other component. The difference is the config panel: instead of a handful of fixed fields, the user picks a language, maps inputs, writes code, and sees the output shape.

The strongest mental model is:

- upstream components provide data
- the code component reads that data through explicit inputs
- the code runs
- it emits one or more structured outputs downstream

The product should bias toward explicitness here. It is tempting to dump the whole upstream payload into the code environment and call it a day, but explicit inputs are easier to understand and debug.

## How it should work

**Inputs.** The component should let the user map named inputs from upstream data into the code environment. This is easier to reason about than making the user fish around in a giant implicit payload. The full upstream context can still be available, but the primary UI should be named inputs.

**Language.** The component should support JavaScript and Python. Supporting both is part of the product value. Different users naturally reach for different languages, and this component will mostly be used by people who already know one of them.

**Output.** The code returns structured output. The simplest model is: return one object and one item goes downstream, return a list and downstream fans out. That is easy to explain and matches how people think about workflow data.

**Editor.** The editor should be good enough that this does not feel like a punishment. Syntax highlighting, a real code editor, clear errors, and a tight test loop matter more here than fancy AI features.

**Debugging.** The user needs to see the exact input passed into the code, the logs from execution, the output that came back, and the error if it failed. If we skip this, the feature will be powerful but miserable.

**Testing.** This should work cleanly with the Test Run concept. The user should be able to run the component in edit mode with test data, inspect the input and output, and iterate without publishing.

## What this is not

- Not a replacement for normal components
- Not a generic compute platform
- Not an excuse to skip building real integrations
- Not the place for long-running jobs or heavy runtime orchestration

If a workflow is mostly code, that's a signal that the user may actually want a different product shape. This feature is for the custom logic in the middle of an otherwise visual workflow.

## MLP

The minimum lovable version is small:

- one code component
- JavaScript and Python
- explicit named inputs mapped from upstream data
- structured output downstream
- logs, output, and errors visible in run history
- works in Test Run

That is enough to solve the core product problem: "the workflow builder can't express this one piece of logic cleanly."

What it does not need on day one:

- external network access
- custom package management
- file system access
- long-running jobs
- code reuse across canvases
- AI generation built into the editor

## Open questions

**Transform only or full action?** The recommendation here is transform only, but this is the biggest product fork.

**How explicit should inputs be?** Should the component primarily use named mapped inputs, or should it expose the whole upstream payload by default?

**What should the output model be?** One object vs many objects is straightforward. Named output channels would be more powerful but also more complex.

**How much environment access should the code get?** Secrets, network, memory, libraries, external requests. These are not implementation details, they define what product this is.

**Where should this sit in the component taxonomy?** Is this a core component, an advanced component, or something closer to a developer tool?

**How do we keep it from becoming a crutch?** If users start solving everything with code, the workflow builder gets weaker instead of stronger.

## Later

- Reusable code components
- Shared snippets / libraries
- More languages
- AI help for scaffolding code
- A more powerful programmable action step if the product needs it
