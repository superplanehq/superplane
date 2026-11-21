import { describe, expect, it } from "bun:test";

import { getLastMentionMapper } from "./get_last_mention";
import type { ExecutionDetailsContext, ExecutionInfo, NodeInfo, OutputPayload } from "../types";

function buildOutput(data: unknown): OutputPayload {
  return { type: "json", timestamp: new Date().toISOString(), data };
}

function buildContext(outputs?: ExecutionInfo["outputs"]): ExecutionDetailsContext {
  const node: NodeInfo = {
    id: "node-1",
    name: "Get Last Mention",
    componentName: "discord.getLastMention",
    isCollapsed: false,
  };
  const timestamp = new Date().toISOString();
  return {
    nodes: [node],
    node,
    execution: {
      id: "exec-1",
      createdAt: timestamp,
      updatedAt: timestamp,
      state: "STATE_FINISHED",
      result: "RESULT_PASSED",
      resultReason: "RESULT_REASON_OK",
      resultMessage: "",
      metadata: {},
      configuration: {},
      rootEvent: undefined,
      outputs,
    },
  };
}

describe("getLastMentionMapper.getExecutionDetails", () => {
  const timestamp = new Date().toISOString();
  const payload = {
    channel_id: "channel-1",
    mention: {
      id: "mention-1",
      content: "Hello <@user-1>",
      timestamp,
      author: { id: "author-1", username: "alice" },
    },
  };

  it.each(["found", "default"])("displays a populated %s output", (channel) => {
    const details = getLastMentionMapper.getExecutionDetails(buildContext({ [channel]: [buildOutput(payload)] }));
    expect(Object.entries(details)).toEqual([
      ["Channel ID", "channel-1"],
      ["Result", "Found"],
      ["Mention ID", "mention-1"],
      ["Author", "@alice"],
      ["Content", "Hello <@user-1>"],
      ["Timestamp", timestamp],
    ]);
  });

  it("displays the notFound channel and placeholders for absent mention fields", () => {
    expect(
      getLastMentionMapper.getExecutionDetails(buildContext({ notFound: [buildOutput({ channel_id: "channel-2" })] })),
    ).toEqual({
      "Channel ID": "channel-2",
      Result: "Not Found",
      "Mention ID": "-",
      Author: "-",
      Content: "-",
      Timestamp: "-",
    });
  });

  it.each([undefined, {}, { found: [], notFound: [], default: [] }])(
    "displays Not Found and placeholders for missing outputs: %j",
    (outputs) => {
      expect(getLastMentionMapper.getExecutionDetails(buildContext(outputs))).toEqual({
        "Channel ID": "-",
        Result: "Not Found",
        "Mention ID": "-",
        Author: "-",
        Content: "-",
        Timestamp: "-",
      });
    },
  );

  it.each([
    {
      name: "found before other channels",
      found: payload,
      notFound: { channel_id: "not-found" },
      expected: "channel-1",
    },
    {
      name: "notFound after null found data",
      found: null,
      notFound: { channel_id: "not-found" },
      expected: "not-found",
    },
    {
      name: "notFound after undefined found data",
      found: undefined,
      notFound: { channel_id: "not-found" },
      expected: "not-found",
    },
    { name: "default after nullish channel data", found: null, notFound: undefined, expected: "legacy" },
    { name: "empty found object stops fallback", found: {}, notFound: { channel_id: "not-found" }, expected: "-" },
    { name: "empty notFound object stops fallback", found: undefined, notFound: {}, expected: "-" },
  ])("preserves payload precedence: $name", ({ found, notFound, expected }) => {
    const details = getLastMentionMapper.getExecutionDetails(
      buildContext({
        found: [buildOutput(found), buildOutput({ channel_id: "later-found" })],
        notFound: [buildOutput(notFound)],
        default: [buildOutput({ channel_id: "legacy" })],
      }),
    );
    expect(details["Channel ID"]).toBe(expected);
  });

  it.each([
    { name: "nonempty found output without an ID", outputs: { found: [buildOutput({})] }, expected: "Found" },
    {
      name: "legacy mention ID",
      outputs: { default: [buildOutput({ mention: { id: "legacy-mention" } })] },
      expected: "Found",
    },
    {
      name: "mention without an ID or found output",
      outputs: { default: [buildOutput({ mention: { content: "Hello" } })] },
      expected: "Not Found",
    },
  ])("detects Result from $name", ({ outputs, expected }) => {
    expect(getLastMentionMapper.getExecutionDetails(buildContext(outputs)).Result).toBe(expected);
  });

  it.each([
    { name: "username only", author: { username: "alice" }, expected: "@alice" },
    { name: "author ID only", author: { id: "author-1" }, expected: "author-1" },
    { name: "username before ID", author: { username: "alice", id: "author-1" }, expected: "@alice" },
    { name: "empty username falls back to ID", author: { username: "", id: "author-1" }, expected: "author-1" },
    { name: "missing author", author: undefined, expected: "-" },
    { name: "empty author fields", author: { username: "", id: "" }, expected: "-" },
  ])("formats the author: $name", ({ author, expected }) => {
    const details = getLastMentionMapper.getExecutionDetails(
      buildContext({ found: [buildOutput({ mention: { author } })] }),
    );
    expect(details.Author).toBe(expected);
  });
});
