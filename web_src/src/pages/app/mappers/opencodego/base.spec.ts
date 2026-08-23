import { describe, expect, it } from "vitest";

import { baseMapper } from "./base";
import { buildContext, buildDetailsCtx, buildNode, buildOutput } from "./test_helpers";

const PAYLOAD_TYPE = "opencodego.chatCompletion.result";

describe("opencodego baseMapper node metadata", () => {
  it("shows the model stored in node metadata", () => {
    const props = baseMapper.props(buildContext(buildNode({ metadata: { model: "glm-5.2" }, configuration: {} })));
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "glm-5.2" }]);
  });

  it("falls back to configuration when metadata has no model yet", () => {
    const props = baseMapper.props(buildContext(buildNode({ configuration: { model: "deepseek-v4" } })));
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "deepseek-v4" }]);
  });

  it("prefers metadata over configuration", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ metadata: { model: "glm-5.2" }, configuration: { model: "deepseek-v4" } })),
    );
    expect(props.metadata).toEqual([{ icon: "sparkles", label: "glm-5.2" }]);
  });

  it("returns no items when no model is set", () => {
    const props = baseMapper.props(buildContext(buildNode({ metadata: undefined, configuration: undefined })));
    expect(props.metadata).toEqual([]);
  });

  it("shows the structured output badge", () => {
    const props = baseMapper.props(
      buildContext(buildNode({ configuration: { model: "glm-5.2", outputSchema: "{}" } })),
    );
    expect(props.metadata).toEqual([
      { icon: "sparkles", label: "glm-5.2" },
      { icon: "braces", label: "Structured output" },
    ]);
  });
});

describe("opencodego baseMapper execution details", () => {
  it("surfaces completed time, model and token usage", () => {
    const details = baseMapper.getExecutionDetails!(
      buildDetailsCtx(
        buildOutput(PAYLOAD_TYPE, {
          id: "chatcmpl-20260818-0001",
          model: "glm-5.2",
          text: "Hello, world!",
          finishReason: "stop",
          usage: { prompt_tokens: 14, completion_tokens: 9, total_tokens: 23 },
        }),
      ),
    );

    expect(Object.keys(details)).toEqual(["Completed At", "Model", "Tokens"]);
    expect(details["Completed At"]).toBe(new Date("2026-08-18T12:00:00.000Z").toLocaleString());
    expect(details["Model"]).toBe("glm-5.2");
    expect(details["Tokens"]).toBe("23 (14 in / 9 out)");
  });

  it("omits Tokens when the payload has no usage", () => {
    const details = baseMapper.getExecutionDetails!(
      buildDetailsCtx(
        buildOutput(PAYLOAD_TYPE, {
          id: "chatcmpl-20260818-0002",
          model: "glm-5.2",
          text: "Hello!",
          finishReason: "stop",
        }),
      ),
    );

    expect(details["Tokens"]).toBeUndefined();
    expect(details["Model"]).toBe("glm-5.2");
  });

  it("omits usage details when only total_tokens is zero or missing", () => {
    const details = baseMapper.getExecutionDetails!(
      buildDetailsCtx(buildOutput(PAYLOAD_TYPE, { model: "glm-5.2", usage: { prompt_tokens: 3 } })),
    );

    expect(details["Tokens"]).toBeUndefined();
  });

  it("returns nothing when the execution has no output", () => {
    expect(baseMapper.getExecutionDetails!(buildDetailsCtx())).toEqual({});
  });
});
