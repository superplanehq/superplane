# TypeScript Runtime (Deno) - Early Preview

This is an initial implementation slice for executing component logic in Deno with TypeScript.

Current scope:

- TypeScript components discovered from filesystem and registered in `pkg/registry`.
- Component execution path (`Execute`) through Deno.
- Worker flow stays unchanged: it calls `component.Execute()` for all components.

## Environment Variables

- `TYPESCRIPT_COMPONENTS_DIR`
  - Base directory that contains one folder per component.
  - Component directory must contain:
    - `index.ts`
    - `manifest.json`
- `DENO_BINARY` (optional, default `deno`)
- `DENO_EXECUTION_TIMEOUT` (optional, default `30s`)

Example:

```text
${TYPESCRIPT_COMPONENTS_DIR}/noop2/index.ts
${TYPESCRIPT_COMPONENTS_DIR}/noop2/manifest.json
```

Discovered components are registered in the same `registry.ListComponents()` flow used by Go components.

## Runtime Protocol

The Go worker sends JSON to `stdin` and expects JSON in `stdout`.

Input operation:

- `component.setup`
- `component.execute`

Output outcomes:

- `pass`
- `fail`
- `noop`

Protocol types are defined in:

- `pkg/runtime/typescript/contract.go`
- `sdk/typescript/types.ts`

## Minimal TypeScript Component Example

```ts
import { runComponentCLI } from "../../typescript/mod.ts";

await runComponentCLI({
  execute(ctx) {
    ctx.logger.info("running component", { nodeId: ctx.nodeId });

    return {
      outcome: "pass",
      outputs: [
        {
          channel: "default",
          payloadType: "custom.finished",
          payload: { ok: true, input: ctx.data },
        },
      ],
    };
  },
});
```

Run directly with Deno:

```bash
cat input.json | deno run --quiet --no-prompt /path/to/component.ts
```

## Notes

- Triggers and actions are not yet implemented for TypeScript runtime components.
