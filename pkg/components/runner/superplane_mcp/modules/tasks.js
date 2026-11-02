"use strict";

const TASK_STATES = new Set(["draft", "open", "closed"]);
const CLOSE_RESULTS = new Set(["completed", "rejected", "failed"]);
const MAX_LIST_LIMIT = 100;
const MAX_STATE_VALUES = 3;
const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 20000;

function exceedsCodePoints(value, max) {
  let count = 0;
  for (let index = 0; index < value.length; ) {
    count += 1;
    if (count > max) return true;
    const code = value.codePointAt(index);
    index += code > 0xffff ? 2 : 1;
  }
  return false;
}

function stateValues(raw) {
  if (raw == null || raw === "") return [];
  const values = Array.isArray(raw) ? raw : [raw];
  return values.map((value) => String(value || "").trim().toLowerCase()).filter(Boolean);
}

async function listTasks(input, client) {
  const states = stateValues(input && input.state);
  if (states.length > MAX_STATE_VALUES) throw new Error("state must be draft, open, or closed");
  const params = new URLSearchParams();
  for (const state of states) {
    if (!TASK_STATES.has(state)) throw new Error("state must be draft, open, or closed");
    params.append("state", state);
  }
  if (input && input.limit != null && input.limit !== "") {
    const limit = Number(input.limit);
    if (!Number.isInteger(limit) || limit < 1 || limit > MAX_LIST_LIMIT) {
      throw new Error("limit must be an integer from 1 to 100");
    }
    params.set("limit", String(limit));
  }
  const query = params.toString();
  const path = query ? `/api/v1/runner/tasks?${query}` : "/api/v1/runner/tasks";
  return client.request("GET", path);
}

async function createTask(input, client) {
  const title = String((input && input.title) || "").trim();
  if (!title) throw new Error("title is required");
  if (exceedsCodePoints(title, MAX_TITLE_LENGTH)) throw new Error("title must be at most 256 characters");
  const description = String((input && input.description) || "").trim();
  if (exceedsCodePoints(description, MAX_DESCRIPTION_LENGTH)) {
    throw new Error("description must be at most 20000 characters");
  }
  return client.request("POST", "/api/v1/runner/tasks", { title, description });
}

async function closeTask(input, client) {
  const result = String((input && input.result) || "").trim().toLowerCase();
  if (result && !CLOSE_RESULTS.has(result)) {
    throw new Error("result must be completed, rejected, or failed");
  }
  const body = result ? { result } : {};
  return client.request("POST", "/api/v1/runner/task/close", body);
}

const LIST_TASKS = {
  definition: {
    name: "list_tasks",
    description:
      "List tasks in this workspace. Returns at most 100 tasks, newest update first. Pass state to keep draft, open, or closed tasks. Repeat state to combine them. Archived tasks stay hidden. There is no offset. Each task includes id, key, title, state, and url.",
    inputSchema: {
      type: "object",
      properties: {
        state: {
          description: "draft, open, or closed. Pass one value or a list.",
          anyOf: [
            { type: "string", enum: ["draft", "open", "closed"] },
            { type: "array", items: { type: "string", enum: ["draft", "open", "closed"] } },
          ],
        },
        limit: {
          type: "integer",
          minimum: 1,
          maximum: 100,
          description: "Maximum number of tasks to return. The default and the maximum are 100.",
        },
      },
    },
  },
  handler: listTasks,
};

const CREATE_TASK = {
  definition: {
    name: "create_task",
    description:
      "Create a draft task in this workspace. The task has no owner. Returns id, key, title, state, and url.",
    inputSchema: {
      type: "object",
      properties: {
        title: { type: "string", maxLength: 256, description: "Task title. At most 256 characters." },
        description: {
          type: "string",
          maxLength: 20000,
          description: "Task description. Optional. At most 20000 characters.",
        },
      },
      required: ["title"],
    },
  },
  handler: createTask,
};

const CLOSE_TASK = {
  definition: {
    name: "close_task",
    description:
      "Close this run's task. On a planning run, this closes the session draft. result is completed, rejected, or failed. The default is completed. A draft task can be rejected. An open task can be completed, rejected, or failed. A closed or archived task cannot be closed again.",
    inputSchema: {
      type: "object",
      properties: {
        result: {
          type: "string",
          enum: ["completed", "rejected", "failed"],
          description: "How the task ended. The default is completed.",
        },
      },
    },
  },
  handler: closeTask,
};

module.exports = {
  tools: [LIST_TASKS, CREATE_TASK, CLOSE_TASK],
};
