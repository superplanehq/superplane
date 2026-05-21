---
name: superplane-component-mappers
description: >-
  Adds or reviews workflow UI mappers for SuperPlane core and integration
  components. Use when implementing a new backend component or trigger, when a
  component has no mapper in web_src, when canvas nodes show generic/noop UI, or
  when the user mentions frontend mappers, workflowv2/mappers, or component
  customization.
---

# SuperPlane Component Frontend Mappers

Backend registration alone is not enough. Every user-facing **component** (and **trigger**) needs a frontend mapper so the canvas shows the right icon, configuration specs, execution states, subtitles, and execution details.

**There are no mappers associated with this component in the frontend** means the component `Name()` from Go is missing from `web_src/src/pages/workflowv2/mappers/index.ts`. The UI then falls back to `noopMapper` — generic node chrome with no expression specs, channel-aware states, or tailored execution details.

## When a mapper is required

| Change | Mapper work |
|--------|-------------|
| New core component (`pkg/components/<name>/`) | `web_src/src/pages/workflowv2/mappers/<name>.ts` + register in `index.ts` |
| New integration component | `web_src/src/pages/workflowv2/mappers/<integration>/<component>.ts` + register in that integration's `index.ts` |
| New output channels or custom pass/fail semantics | Often also `eventStateRegistries` (see `filter`, `if`, `merge`) |
| Custom config field UI | `customFieldRenderers` (see `wait`, `schedule`) |

## Checklist (core component)

Copy and complete:

```
- [ ] Registry key matches Go `Name()` exactly (e.g. `fanOut`, not `fan-out`)
- [ ] `componentBaseMappers` entry in index.ts
- [ ] `iconSlug` matches backend `Icon()` when possible
- [ ] Configuration fields surfaced in `specs` (expressions, labels, etc.)
- [ ] `getExecutionDetails` includes key config + metadata from runs
- [ ] Custom `eventStateRegistries` if outputs use non-default channels (passed/rejected/true/false/item/…)
- [ ] Optional `*.spec.ts` for mapper props / execution details
- [ ] `make format.js` and `make check.build.ui` after UI edits
```

## Minimal mapper pattern

Use `filter.ts` or `if.ts` for expression-based core components. Use `noop.ts` only when the component truly needs no customization.

1. **Create** `web_src/src/pages/workflowv2/mappers/<component>.ts`:

```typescript
import type { ComponentBaseContext, ComponentBaseMapper, ExecutionDetailsContext, SubtitleContext } from "./types";
import type { ComponentBaseProps } from "@/ui/componentBase";
import { renderTimeAgo } from "@/components/TimeAgo";

type FanOutConfiguration = { arrayExpression: string };

export const fanOutMapper: ComponentBaseMapper = {
  props(context: ComponentBaseContext): ComponentBaseProps {
    const configuration = context.node.configuration as FanOutConfiguration;
    return {
      iconSlug: "split", // match backend Icon()
      collapsed: context.node.isCollapsed,
      title: context.node.name || context.componentDefinition.label || "Fan Out",
      specs: configuration.arrayExpression
        ? [{ title: "Array", tooltipTitle: "Array expression", value: configuration.arrayExpression }]
        : undefined,
      includeEmptyState: context.lastExecutions.length === 0,
    };
  },
  subtitle(context: SubtitleContext) {
    return context.execution.createdAt ? renderTimeAgo(new Date(context.execution.createdAt)) : "";
  },
  getExecutionDetails(context: ExecutionDetailsContext) {
    const configuration = context.execution.configuration as FanOutConfiguration;
    return { "Array expression": configuration.arrayExpression ?? "-" };
  },
};
```

2. **Register** in `web_src/src/pages/workflowv2/mappers/index.ts`:

```typescript
import { fanOutMapper } from "./fanOut";

const componentBaseMappers: Record<string, ComponentBaseMapper> = {
  // ...
  fanOut: fanOutMapper,
};
```

3. **Custom states** (if the component emits named channels): define `*_STATE_MAP`, `*StateFunction`, `*_STATE_REGISTRY`, and add to `eventStateRegistries`. Mirror channel names from Go `OutputChannels()` (e.g. Fan Out → `item`).

## Verify registration

```bash
# Registry key must exist (replace fanOut with component Name())
rg 'fanOut:' web_src/src/pages/workflowv2/mappers/index.ts
```

Integration components: key is often `integration.component` via `appMappers`; see `docs/contributing/integrations.md` and `docs/contributing/component-customization.md`.

## Reference docs

- [docs/contributing/component-customization.md](docs/contributing/component-customization.md) — registry types, tutorials
- [docs/contributing/component-design.md](docs/contributing/component-design.md) — mapper responsibilities on the canvas
- [web_src/AGENTS.md](web_src/AGENTS.md) — UI conventions and test commands

## PR review

If a PR adds `pkg/components/*` or `pkg/integrations/*` actions/triggers but no `web_src/src/pages/workflowv2/mappers/` changes, flag it unless the author explicitly documents intentional noop fallback.
