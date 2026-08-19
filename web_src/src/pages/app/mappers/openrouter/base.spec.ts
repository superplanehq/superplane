import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import { buildContext, buildDetailsCtx, buildNode, buildOutput } from "./test_helpers";

const PAYLOAD_TYPE = "openrouter.chatCompletion.result";

describe("openrouter baseMapper node metadata", () => {
  it("shows the configured model", () => {
    const props = baseMapper.props(buildContext(buildNode({ configuration: { model: "openai/gpt-4o-mini" } })));
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "openai/gpt-4o-mini" }]);
  });

  it("adds a routing badge when provider routing is configured", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "openai/gpt-4o-mini", provider: { sort: "price" } } })),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "openai/gpt-4o-mini" },
      { icon: "route", label: "Provider routing" },
    ]);
  });

  it("falls back to node metadata when the node has no configuration yet", () => {
    const props = baseMapper.props(
      buildContext(
        buildNode({ configuration: undefined, metadata: { model: "openai/gpt-4o-mini", providerRouting: true } }),
      ),
    );
    expect(props.metadata).toContainEqual({ icon: "route", label: "Provider routing" });
  });

  it("badges structured output and web search", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "openai/gpt-4o-mini", outputSchema: "{}", webSearch: true } })),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "openai/gpt-4o-mini" },
      { icon: "braces", label: "Structured output" },
      { icon: "globe", label: "Web search" },
    ]);
  });

  it("ignores a blank output schema", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "openai/gpt-4o-mini", outputSchema: "   " } })),
    );
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "openai/gpt-4o-mini" }]);
  });

  it("prefers the live configuration over stale metadata", () => {
    // Autosave updates configuration only, so metadata can lag behind.
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "openai/gpt-4o-mini" }, metadata: { providerRouting: true } })),
    );
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "openai/gpt-4o-mini" }]);
  });
});

describe("openrouter baseMapper execution details", () => {
  it("surfaces the model, serving provider, usage and a model link", () => {
    const details = baseMapper.getExecutionDetails!(
      buildDetailsCtx(
        buildOutput(PAYLOAD_TYPE, {
          model: "openai/gpt-4o-mini",
          provider: "OpenAI",
          usage: { prompt_tokens: 14, completion_tokens: 9, total_tokens: 23, cost: 0.0000075 },
        }),
      ),
    );

    expect(Object.keys(details)).toEqual(["Completed At", "Model", "Provider", "Tokens", "Cost", "View Model"]);
    expect(details["Model"]).toBe("openai/gpt-4o-mini");
    expect(details["Provider"]).toBe("OpenAI");
    expect(details["Tokens"]).toBe("23 (14 in / 9 out)");
    expect(details["Cost"]).toBe("$0.000008");
    expect(details["View Model"]).toBe("https://openrouter.ai/openai/gpt-4o-mini");
  });

  it("keeps the timestamp first and stays within six details", () => {
    const details = baseMapper.getExecutionDetails!(
      buildDetailsCtx(
        buildOutput(PAYLOAD_TYPE, {
          model: "openai/gpt-4o-mini",
          provider: "OpenAI",
          usage: { prompt_tokens: 1, completion_tokens: 1, total_tokens: 2, cost: 0 },
        }),
      ),
    );

    expect(Object.keys(details).length).toBeLessThanOrEqual(6);
    expect(Object.keys(details)[0]).toBe("Completed At");
  });

  it("omits usage details when the response carries none", () => {
    const details = baseMapper.getExecutionDetails!(
      buildDetailsCtx(buildOutput(PAYLOAD_TYPE, { model: "openai/gpt-4o-mini" })),
    );

    expect(details["Tokens"]).toBeUndefined();
    expect(details["Cost"]).toBeUndefined();
    expect(details["Provider"]).toBeUndefined();
  });

  it("returns nothing when the execution has no output", () => {
    expect(baseMapper.getExecutionDetails!(buildDetailsCtx())).toEqual({});
  });
});
