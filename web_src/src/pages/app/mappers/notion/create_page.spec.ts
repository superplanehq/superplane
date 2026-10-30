import { describe, expect, it } from "bun:test";
import { renderToStaticMarkup } from "react-dom/server";
import { getComponentBaseMapper } from "../index";
import { notionContext, notionDetails, notionExecutedContext } from "./test-fixtures";

const componentName = "notion.createPage";
const mapper = getComponentBaseMapper(componentName);
const page = {
  id: "page-1",
  url: "https://app.notion.com/p/page-1",
  properties: {
    Status: { type: "status", status: { name: "Draft" } },
    Runbook: { type: "title", title: [{ plain_text: "Incident " }, { plain_text: "response" }] },
  },
};

describe("Notion Create Page", () => {
  it("shows the native page link and title from a renamed title property", () => {
    const details = mapper.getExecutionDetails(notionDetails(componentName, page));
    expect(details["Started At"]).toBeDefined();
    expect(details["Page ID"]).toBe("page-1");
    expect(details["Page URL"]).toBe("https://app.notion.com/p/page-1");
    expect(details["Title"]).toBe("Incident response");
  });

  it("preserves available fields in a partial response and handles missing output", () => {
    const partial = mapper.getExecutionDetails(notionDetails(componentName, { id: "page-2", properties: {} }));
    expect(partial["Page ID"]).toBe("page-2");
    expect(partial["Title"]).toBeUndefined();
    expect(partial["Page URL"]).toBeUndefined();
    const pending = mapper.getExecutionDetails(notionDetails(componentName));
    expect(pending["Started At"]).toBeDefined();
    expect(pending["Page ID"]).toBeUndefined();
  });

  it("identifies the configured title and parent without showing JSON content", () => {
    const props = mapper.props(
      notionContext(componentName, {
        parentType: "data_source",
        parentId: "source-1",
        title: "Runbook",
        children: "[]",
      }),
    );
    expect(props.metadata?.map((item) => item.label)).toEqual(["Runbook", "Data source: source-1"]);
  });

  it("uses the page parent label and remains usable before configuration", () => {
    const props = mapper.props(notionContext(componentName, { parentType: "page", parentId: "page-parent" }));
    expect(props.metadata?.map((item) => item.label)).toEqual(["Page: page-parent"]);
    expect(mapper.props(notionContext(componentName)).includeEmptyState).toBe(true);
  });

  it("shows the same created title and timestamp on the canvas and in run history", () => {
    const context = notionExecutedContext(componentName, page);
    const section = mapper.props(context).eventSections?.[0];
    const history = mapper.subtitle({ node: context.node, execution: context.lastExecutions[0] });
    const canvasText = renderToStaticMarkup(section?.eventSubtitle).replace(/<[^>]*>/g, "");
    const historyText = renderToStaticMarkup(history).replace(/<[^>]*>/g, "");

    expect(canvasText).toBe("Incident response · 2m ago");
    expect(historyText).toBe(canvasText);
  });
});
