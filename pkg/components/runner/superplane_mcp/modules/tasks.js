"use strict";

const TASK_STATES = new Set(["draft", "open", "closed"]);
const CLOSE_RESULTS = new Set(["completed", "rejected", "failed"]);
const MAX_LIST_LIMIT = 100;
const MAX_STATE_VALUES = 3;
const MAX_TITLE_LENGTH = 256;
const MAX_DESCRIPTION_LENGTH = 20000;
const MAX_OWNER_PATHS = 10;

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

async function moveTask(input, client) {
  const column = input && input.column;
  const next = input && input.next;
  const task = input && input.task;
  if (
    (column !== undefined && (typeof column !== "string" || !column.trim())) ||
    (next !== undefined && next !== true) ||
    (column !== undefined) === (next === true)
  ) {
    throw new Error("Pass exactly one of column or next: true");
  }
  if (task !== undefined && (typeof task !== "string" || !task.trim())) throw new Error("task must be a task id or key");
  const body = column !== undefined ? { column: column.trim() } : { next: true };
  if (task !== undefined) body.task = task.trim();
  return client.request("POST", "/api/v1/runner/task/move", body);
}

const MOVE_TASK = {
  definition: {
    name: "move_task",
    description:
      "Move an open task immediately to a stage on its recorded line. Pass exactly one of column or next: true. column is a stable stage key, including custom stages, Backlog, and Done. next follows the recorded stage order. A task run moves its own task: omit task. Unfinished source work is replaced. A run with no task, such as a webhook run, must pass task. It can move only a task that waits in the stage of this automation after the stage work passed. Destination work starts or queues under existing limits. Returns task identity, from_column, column, and state. Current-stage movement is unchanged. Planning runs, tasks without a line, stale runs, and closed or archived tasks cannot move. A missing successor fails.",
    inputSchema: {
      type: "object",
      properties: {
        task: {
          type: "string",
          minLength: 1,
          description: "Task id, number, or key, for example SP-42. Required for a run with no task. Use get_task to find it.",
        },
        column: { type: "string", minLength: 1, description: "Stable stage key on this task's recorded line." },
        next: { type: "boolean", const: true, description: "Move to the next stage in recorded line order." },
      },
      oneOf: [
        { required: ["column"], not: { required: ["next"] } },
        { required: ["next"], not: { required: ["column"] } },
      ],
      additionalProperties: false,
    },
  },
  handler: moveTask,
};

async function attachTask(input, client) {
  const task = input && input.task;
  if (typeof task !== "string" || !task.trim()) throw new Error("task must be a task id or key");
  return client.request("POST", "/api/v1/runner/task/attach", { task: task.trim() });
}

const ATTACH_TASK = {
  definition: {
    name: "attach_task",
    description:
      "Attach this run to a task so the task timeline shows the run in the stage of this automation. Only a run with no task, such as a webhook run, can attach. The task must wait in the stage of this automation after the stage work passed. Attach before move_task. Attaching the same task again has no effect. Returns task identity and column.",
    inputSchema: {
      type: "object",
      properties: {
        task: { type: "string", minLength: 1, description: "Task id, number, or key, for example SP-42. Use get_task to find it." },
      },
      required: ["task"],
      additionalProperties: false,
    },
  },
  handler: attachTask,
};

function ownerPaths(raw) {
  const values = Array.isArray(raw) ? raw : [];
  return values
    .map((value) => String(value || "").trim())
    .filter(Boolean)
    .slice(0, MAX_OWNER_PATHS);
}

async function suggestTaskOwner(input, client) {
  return client.request("POST", "/api/v1/runner/task/suggest-owners", {
    paths: ownerPaths(input && input.paths),
  });
}

async function assignTaskOwner(input, client) {
  const user = String((input && input.user) || "").trim();
  if (!user) throw new Error("user is required");
  const reason = String((input && input.reason) || "").trim();
  const body = { user };
  if (reason) body.reason = reason;
  if (input && input.only_if_unassigned === true) body.only_if_unassigned = true;
  return client.request("PUT", "/api/v1/runner/task/owner", body);
}

const ASSIGN_TASK_OWNER = {
  definition: {
    name: "assign_task_owner",
    description:
      "Set one person as the owner of this run's task. That person replaces all current owners, including manually assigned owners. Pass user as a user_id or github_login from list_members. Returns status assigned with the new owner, or kept with the current owners. A closed or archived task cannot change owners.",
    inputSchema: {
      type: "object",
      properties: {
        user: {
          type: "string",
          description: "user_id or github_login from list_members.",
        },
        reason: {
          type: "string",
          description:
            "Optional. One sentence that SuperPlane shows on the task, for example: You changed pkg/payments/charge.go 3 times this month. SuperPlane keeps the first 280 characters.",
        },
        only_if_unassigned: {
          type: "boolean",
          description:
            "Optional. When true, keep the current owners if the task already has one. The result status is kept. Use this for the default planning assignment.",
        },
      },
      required: ["user"],
    },
  },
  handler: assignTaskOwner,
};

const SUGGEST_TASK_OWNER = {
  definition: {
    name: "suggest_task_owner",
    description:
      "Rank the members who changed the repository paths of this task. Returns members with name, roles, GitHub login, and open_tasks, the current owners of the task, and code_owners: members ranked by their recent commits on the paths. Only members with a linked GitHub account can appear in code_owners.",
    inputSchema: {
      type: "object",
      properties: {
        paths: {
          type: "array",
          maxItems: MAX_OWNER_PATHS,
          items: { type: "string" },
          description:
            "Optional. Repository-relative files or directories, most specific first, for example pkg/payments or web_src/src/pages/checkout.tsx.",
        },
      },
    },
  },
  handler: suggestTaskOwner,
};

const LIST_TASKS = {
  definition: {
    name: "list_tasks",
    description:
      "List tasks in this workspace. Returns at most 100 tasks, newest update first. Pass state to keep draft, open, or closed tasks. Repeat state to combine them. Archived tasks stay hidden. There is no offset. Each task includes id, key, title, state, and url. url is the SuperPlane task link. If a task has a saved external issue link, origin contains its url and label. Use origin.url to identify the issue before commenting with your existing GitHub access. Tasks without a saved origin omit origin. Do not infer issue numbers from task numbers.",
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

function selectorText(value) {
  return value == null ? "" : String(value).trim();
}

async function getTask(input, client) {
  const task = selectorText(input && input.task);
  const pullRequest = selectorText(input && input.pull_request);
  const repository = selectorText(input && input.repository);
  const number = selectorText(input && input.number);
  const commit = selectorText(input && input.commit);
  const selectors = [task, pullRequest, number, commit].filter(Boolean);
  if (selectors.length !== 1 || (repository && !number)) {
    throw new Error("Pass exactly one of task, pull_request, number with an optional repository, or commit");
  }
  const params = new URLSearchParams();
  if (task) params.set("task", task);
  if (pullRequest) params.set("pull_request", pullRequest);
  if (number) {
    if (!/^[1-9][0-9]*$/.test(number)) throw new Error("number must be a positive integer");
    params.set("number", number);
    if (repository) params.set("repository", repository);
  }
  if (commit) params.set("commit", commit);
  return client.request("GET", `/api/v1/runner/task/get?${params.toString()}`);
}

const GET_TASK = {
  definition: {
    name: "get_task",
    description:
      "Find one task in this workspace. Pass exactly one selector: task (id, number, or key such as SP-42), pull_request (the pull request URL), number with an optional repository (owner/name), or commit (a commit SHA of a pull request of the task). Returns id, key, title, description, state, url, origin, column (the board column of the task), and pull_requests with repository, number, url, state, and head_sha. A selector that matches more than one task fails and lists the task keys. Pass repository, or use a more exact selector.",
    inputSchema: {
      type: "object",
      properties: {
        task: { type: "string", minLength: 1, description: "Task id, number, or key." },
        pull_request: { type: "string", minLength: 1, description: "Pull request URL." },
        repository: { type: "string", minLength: 1, description: "Repository as owner/name. Use it with number." },
        number: { type: "integer", minimum: 1, description: "Pull request number." },
        commit: { type: "string", minLength: 7, description: "Commit SHA." },
      },
      additionalProperties: false,
    },
  },
  handler: getTask,
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
  tools: [ASSIGN_TASK_OWNER, SUGGEST_TASK_OWNER, LIST_TASKS, GET_TASK, CREATE_TASK, CLOSE_TASK, MOVE_TASK, ATTACH_TASK],
};
