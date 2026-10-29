"use strict";

const TASK_STATES = new Set(["draft", "open", "closed"]);
const CLOSE_RESULTS = new Set(["completed", "rejected", "failed"]);

function stateValues(raw) {
  if (raw == null || raw === "") return [];
  const values = Array.isArray(raw) ? raw : [raw];
  return values.map((value) => String(value || "").trim().toLowerCase()).filter(Boolean);
}

async function listTasks(input, client) {
  const params = new URLSearchParams();
  for (const state of stateValues(input && input.state)) {
    if (!TASK_STATES.has(state)) throw new Error("state must be draft, open, or closed");
    params.append("state", state);
  }
  if (input && input.limit != null && input.limit !== "") {
    const limit = Number(input.limit);
    if (!Number.isInteger(limit) || limit < 1) throw new Error("limit must be a positive integer");
    params.set("limit", String(limit));
  }
  const query = params.toString();
  const path = query ? `/api/v1/runner/tasks?${query}` : "/api/v1/runner/tasks";
  return client.request("GET", path);
}

async function createTask(input, client) {
  const title = String((input && input.title) || "").trim();
  if (!title) throw new Error("title is required");
  const description = String((input && input.description) || "").trim();
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
      "List tasks in this workspace. Pass state to keep draft, open, or closed tasks. Repeat state to combine them. Archived tasks stay hidden. Each task includes id, key, title, state, and url.",
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
          description: "Maximum number of tasks to return. The default is 100.",
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
        title: { type: "string", description: "Task title." },
        description: { type: "string", description: "Task description. Optional." },
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
