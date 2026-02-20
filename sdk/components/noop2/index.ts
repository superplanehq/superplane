import { runComponentCLI } from "../../typescript/mod.ts";

type Noop2Config = {
  message?: string;
  includeInput?: boolean;
};

function parseConfig(value: unknown): Noop2Config {
  if (!value || typeof value !== "object") {
    return {};
  }

  return value as Noop2Config;
}

await runComponentCLI({
  setup(ctx) {
    const config = parseConfig(ctx.configuration);
    const message = (config.message ?? "").trim();

    if (!message) {
      throw new Error("message is required");
    }
  },

  execute(ctx) {
    const config = parseConfig(ctx.configuration);
    const message = (config.message ?? "hello from noop2").trim();
    const includeInput = config.includeInput ?? true;

    ctx.logger.info("noop2 component executed", {
      nodeId: ctx.nodeId,
      workflowId: ctx.workflowId,
      includeInput,
    });

    const payload: Record<string, unknown> = {
      ok: true,
      runtime: "deno-ts",
      component: "noop2",
      message,
    };

    if (includeInput) {
      payload.input = ctx.data;
    }

    return {
      outcome: "pass",
      outputs: [
        {
          channel: "default",
          payloadType: "noop2.finished",
          payload,
        },
      ],
    };
  },
});
