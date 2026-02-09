# Opening PRs for Integrations

How to open a pull request for a new or updated integration. For general PR
workflow (fork, branch, push), see [Pull Requests](pull-requests.md).

## Table of contents

- [Title](#title)
- [Description](#description)
  - [Start with link to the issue](#start-with-link-to-the-issue)
  - [Describe the implementation](#describe-the-implementation)
  - [Include a video demo](#include-a-video-demo)
- [Backend Implementation](#backend-implementation)
- [Frontend Implementation](#frontend-implementation)
- [Docs](#docs)
- [Tests](#tests)
- [CI and BugBot](#ci-and-bugbot)
- [BugBot](#bugbot)
- [DCO](#dco)

## Title 

Use the semantic format: `feat: Add <Integration>` or `feat: Add <Integration> <Trigger/Action>`. 

Examples of what to do:

- ✅ `feat: Add Rootly integration`
- ✅ `feat: Add Slack Send Message action`

Examples of what not to do:

- ❌ `Add Rootly integration` (missing `feat:`)
- ❌ `feat(Rootly): Add integration` (wrong format)
- ❌ `[Rootly] Add integration` (wrong format)
- ❌ `Add Rootly` (missing `feat:`)

See [title rules](pull-requests.md#title-format-rules).

## Description  

### Start with link to the issue

Start with a link to the issue. e.g. `Implements #1234`.

### Describe the implementation

Say what was implemented, why, and how (e.g. which API, which endpoints, etc.). 
If there are any limitations or things to note, include those as well.

e.g. 

```
This PR implements the Rootly integration, which allows users to create incidents in Rootly.

Authorization is via API key, which users can generate in their Rootly account by
going to Settings > API Keys.
```

### Include a video demo

Include a link to a short demo video.

What to do:

- ✅ Show how to set up the integration (e.g. where to find the API key in Rootly, how to enter it in SuperPlane, etc.).
- ✅ Show the workflow in action, e.g. creating an incident in Rootly triggers a workflow in SuperPlane.
- ✅ Show how to configure the component.
- ✅ Keep it short (1-2 minutes max).

What not to do:

- ❌ Don't just show the canvas without showing the integration in action.
- ❌ Don't make it too long or include unnecessary details.
- ❌ Don't show the code or implementation details in the video.
- ❌ Don't show unit tests or CI checks in the video.

## Backend Implementation

The backend implementation should include the integration code in `pkg/integrations/<name>/`. 
e.g. for a Rootly integration, the code would be in `pkg/integrations/rootly/`.

What to do:

- ✅ Follow the existing structure and patterns in the codebase for integrations.
- ✅ Write clean, modular, and well-documented code.
- ✅ Add examples output for the components.

What not to do:

- ❌ Don't create a new structure or pattern for your integration, unless there's a good reason to do so.
- ❌ Don't include unrelated code or changes in the PR.
- ❌ Don't make breaking changes to existing code without a good reason and without documenting them.
- ❌ Don't make changes in the core workflow engine or other unrelated parts of the codebase unless necessary for the integration.

## Frontend Implementation

The frontend implementation should include mappers in `web_src/src/pages/workflowv2/mappers/<name>/`.
e.g. for a Rootly integration, the mappers would be in `web_src/src/pages/workflowv2/mappers/rootly/`.

What to do:

- ✅ Follow the existing structure and patterns in the codebase for integrations.
- ✅ Write clean, modular, and well-documented code.

What not to do:

- ❌ Don't create a new structure or pattern for your integration, unless there's a good reason to do so.
- ❌ Don't include unrelated code or changes in the PR.
- ❌ Don't make breaking changes to existing code without a good reason and without documenting them.
- ❌ Don't make changes in UI components or other unrelated parts of the codebase unless necessary for the integration.

## Docs

Documentation is generated based the code from the `pkg/integrations/`.
Run `make gen.components.docs` to generate the docs after implementing the backend code. 
This will create a doc in `docs/components/` (e.g. `Rootly.mdx`).

What to do:

- ✅ Write documentation in `pkg/integrations/<name>/` that is clear and comprehensive. 
- ✅ Include instructions on how to set up the integration, how to use it, and any limitations or things to note.
- ✅ Follow the existing structure and patterns in the codebase for integration docs.

What not to do:

- ❌ Don't write documentation in the `docs/components/` directly. It should be generated with `make gen.components.docs`.

## Tests

Write unit tests for the backend code in `pkg/integrations/<name>/`.

What to do:

- ✅ Write tests that cover the main functionality of the integration, including edge cases and error handling.
- ✅ Make sure the tests are deterministic and can be run in any order.

What not to do:

- ❌ Don't write tests for static content. e.g. Name or the Label of the component.

## CI and BugBot

Every PR must pass all CI checks, including unit tests and linting.

## BugBot

BugBot will automatically comment on the PR with any issues found in the code, such as linting errors, 
test failures, or other issues. Make sure to address any comments from BugBot.

## DCO

Every commit must be signed off (`git commit -s`). See [Commit Sign-off](commit_sign-off.md).
