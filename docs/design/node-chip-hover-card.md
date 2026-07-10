# Node chip hover card redesign

Date: 2026-07-10  
Status: Approved for implementation

## Problem

`node:` chips in Files, Console, and agent markdown show a hover card that is weak on both usefulness and polish:

- Tinted header feels noisy next to markdown/console content
- Config is a raw mono dump with no label
- Connections are only counts (`N incoming · N outgoing`), not neighbor names
- No cue that clicking the chip opens the node on the canvas

## Goals

- **Primary:** Useful preview — enough identity + config to decide whether to open the node
- **Secondary:** Quick identity — name, component, trigger/action at a glance
- **Affordance:** Subtle footer hint only (“Click to open on canvas”); chip click remains the action

Non-goals: changing chip click/navigation behavior, live run status, or making the card itself a second click target.

## Design

Labeled preview card (Option A).

### Structure (top → bottom)

1. **Header** — component icon, node name, subtitle `Component · Trigger|Action`. No tinted header background.
2. **Details** — same secondary metadata the canvas node card shows (repo, channel, URL, …), via the component/trigger mapper registry. Cap at 3 rows. If the mapper returns nothing, fall back to a built-in one-line summarizer (`http`, `ssh`, `if`, …). Omit when empty.
3. **Connected to** — neighbor name chips (`← Upstream`, `Downstream →`) resolved from canvas edges. Omit when there are no edges. Cap at 4 chips, then a `+N more` chip.
4. **Issue** — soft error row only when `node.errorMessage` is set.
5. **Footer** — muted “Click to open on canvas”.

Empty optional sections are omitted so typical cards stay short.

### Unresolved nodes

If the chip cannot resolve a canvas node, keep current behavior: no hover card (chip still renders and navigates with the raw id).

### Content rules

- Prefer mapper `metadata` (configuration + resolved `node.metadata`, e.g. GitHub repository name) so hover cards stay aligned with canvas node cards.
- Fall back to built-in per-component summarizers only when mapper metadata is empty.
- Neighbor labels use node `name` when present, otherwise node id.
- Prefer upstream then downstream ordering within the capped list.
- Dark mode uses the same structure and existing slate/gray tokens (no new color system).

### Interaction

- Hover: open delay ~200ms / close ~100ms (unchanged).
- Click: unchanged — navigate to live canvas with sidebar + node focus (`agent:focus-node` / `focusRequest` path).
- Card content is not interactive beyond hover dismiss.

## Implementation notes

- Primary surface: [`web_src/src/components/AgentSidebar/widgets/NodeChip.tsx`](../../web_src/src/components/AgentSidebar/widgets/NodeChip.tsx)
- Metadata resolution: [`nodeChipHover.ts`](../../web_src/src/components/AgentSidebar/widgets/nodeChipHover.ts) via `getComponentBaseMapper` / `getTriggerRenderer`
- Storybook: [`NodeChip.stories.tsx`](../../web_src/src/components/AgentSidebar/widgets/NodeChip.stories.tsx) covers HTTP details, GitHub repo metadata, neighbors, overflow, error, unknown node

## Success criteria

- Hovering a resolved chip shows identity + canvas-aligned details/neighbors/error without empty chrome
- Integration nodes (e.g. GitHub) show resolved metadata such as repository name
- Neighbor names appear instead of connection counts
- Footer always shows the open-on-canvas hint for resolved nodes
- Visual weight is quieter than the current tinted header card in Files and Console markdown
