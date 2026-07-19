# Software Factory

Storybook fixture for the **Files** tab and Console markdown panel.

Automates issue-to-PR work: plan, implement, open a draft PR, babysit CI, and notify when ready.

## Flow

1. Mentions / labels kick off a run
2. Agent writes a plan and opens a draft PR
3. CI loop retries until green (or escalates)
4. Card moves to human review when ready

## Chips

- Node: [Implementation](node:runner-implement) · [Write Plan](node:implementation-implementation-2-td4042) · [CI Loop](node:ci-loop)
- Integration: [GitHub](integration:github) · [Discord](integration:discord) · [Semaphore](integration:semaphore)
