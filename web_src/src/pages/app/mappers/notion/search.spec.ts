import { describe, expect, it } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import { getComponentBaseMapper, getState, getStateMap } from "../index";
import { notionContext, notionDetails, notionExecutedContext, notionExecution } from "./test-fixtures";

const componentName = "notion.search";
const mapper = getComponentBaseMapper(componentName);

describe("Notion Search", () => {
  it("shows the result count and whether another page is available", () => {
    const details = mapper.getExecutionDetails(
      notionDetails(componentName, {
        results: [
          { object: "page", id: "page-1" },
          { object: "data_source", id: "source-1" },
        ],
        has_more: true,
        next_cursor: "opaque-next-cursor",
      }),
    );

    expect(details["Started At"]).toBeDefined();
    expect(details["Results"]).toBe("2");
    expect(details["More results"]).toBe("Yes");
  });

  it("distinguishes an empty search from an execution with no output", () => {
    const empty = mapper.getExecutionDetails(notionDetails(componentName, { results: [], has_more: false }));
    expect(empty["Results"]).toBe("0");
    expect(empty["More results"]).toBe("No");

    const pending = mapper.getExecutionDetails(notionDetails(componentName));
    expect(pending["Started At"]).toBeDefined();
    expect(pending["Results"]).toBeUndefined();
    expect(pending["More results"]).toBeUndefined();
  });

  it("keeps partial output useful without inventing a result count", () => {
    const details = mapper.getExecutionDetails(notionDetails(componentName, { has_more: true }));
    expect(details["Started At"]).toBeDefined();
    expect(details["Results"]).toBeUndefined();
    expect(details["More results"]).toBe("Yes");
  });

  it("shows the same result count and timestamp on the canvas and in run history", () => {
    const context = notionExecutedContext(componentName, { results: [{ object: "page", id: "page-1" }] });
    const section = mapper.props(context).eventSections?.[0];
    const history = mapper.subtitle({ node: context.node, execution: context.lastExecutions[0] });
    const canvasText = renderToStaticMarkup(section?.eventSubtitle).replace(/<[^>]*>/g, "");
    const historyText = renderToStaticMarkup(history).replace(/<[^>]*>/g, "");

    expect(canvasText).toBe("1 result · 2m ago");
    expect(historyText).toBe(canvasText);
  });

  it("identifies the query and selected object type on the canvas", () => {
    const props = mapper.props(notionContext(componentName, { query: "Runbook", objectType: "data_source" }));
    expect(props.metadata?.map((item) => item.label)).toEqual(["Runbook", "Data sources"]);
    expect(props.includeEmptyState).toBe(true);
  });

  it("uses all titles when the query is absent and preserves an expression", () => {
    expect(mapper.props(notionContext(componentName)).metadata?.map((item) => item.label)).toEqual(["All titles"]);
    const expression = "{{ $.trigger.data.query }}";
    const props = mapper.props(notionContext(componentName, { query: expression }));
    expect(props.metadata?.[0].label).toBe(expression);
  });

  it("keeps the registered error state visible when Notion rejects a search", () => {
    const execution = notionExecution(undefined, {
      result: "RESULT_FAILED",
      resultReason: "RESULT_REASON_ERROR",
      resultMessage: "Notion rejected the integration token. Update the token in integration settings.",
    });
    const state = getState(componentName)(execution);
    expect(state).toBe("error");
    expect(getStateMap(componentName)[state]).toBeDefined();
    expect(mapper.getExecutionDetails({ ...notionDetails(componentName), execution })["Started At"]).toBeDefined();
  });
});
