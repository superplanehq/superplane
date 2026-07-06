import { Bot, CheckSquare, Database, GitPullRequest, Terminal } from "lucide-react";
import { formatEventTimestamp } from "../runSummary";
import type { TimelineEvent } from "./timelineGroupsModel";

/**
 * Mocked data for the flat timeline-events wireframe. Values are inspired by the real
 * execution/approval shapes but are entirely fake and never hit the backend.
 * Timestamps are derived from `Date.now()` so the wireframe always looks fresh.
 */

const ago = (secondsAgo: number): string => new Date(Date.now() - secondsAgo * 1000).toISOString();
const ts = (secondsAgo: number): string => formatEventTimestamp(ago(secondsAgo)) ?? "";

/** Serialized JSON size of a payload, formatted as KB (shown on Output cards instead of the node name). */
const sizeKb = (value: unknown): string => {
  const bytes = new TextEncoder().encode(JSON.stringify(value ?? {})).length;
  return `${(bytes / 1024).toFixed(2)} KB`;
};

const approvalOutputPayload = {
  result: "rejected",
  records: [
    { index: 0, state: "approved", user: "AleksandarCole" },
    { index: 1, state: "rejected", user: "darkofabijan", reason: "Needs another round of load testing." },
  ],
};
const runBashOutputPayload = { exit_code: 0, tests_passed: 42, duration_seconds: 52 };
const cursorAgentOutputPayload = {
  status: "completed",
  pr_url: "https://github.com/acme/store/pull/482",
  files_changed: 1,
};
const githubOutputPayload = {
  number: 482,
  state: "open",
  html_url: "https://github.com/acme/store/pull/482",
  head: "feature/checkout-fix",
  base: "main",
};
const memoryOutputPayload = {
  namespace: "deploys",
  key: "last-production-release",
  written: true,
  value: { version: "v2.14.0", released_at: "2026-07-06T08:12:00Z" },
};

const approvalIcon = <CheckSquare className="h-3.5 w-3.5" />;
const bashIcon = <Terminal className="h-3.5 w-3.5" />;
const agentIcon = <Bot className="h-3.5 w-3.5" />;
const githubIcon = <GitPullRequest className="h-3.5 w-3.5" />;
const memoryIcon = <Database className="h-3.5 w-3.5" />;

function moreChip(count: number) {
  return (
    <button
      type="button"
      title="Open input chain"
      className="flex shrink-0 items-center rounded bg-slate-100 px-1.5 py-0.5 text-[10px] font-medium text-slate-600 transition-colors hover:bg-slate-200 hover:text-slate-700"
    >
      +{count} more
    </button>
  );
}

export const approvalEvents: TimelineEvent[] = [
  {
    type: "card",
    id: "input",
    card: {
      kind: "payload",
      kicker: "Input",
      status: { dotClassName: "bg-violet-400", label: "Triggered" },
      sourceName: "build-and-test",
      sourceTrailing: moreChip(4),
      meta: ts(600),
      nodeName: "approve-deploy",
      nodeIcon: approvalIcon,
      payload: {
        environment: "production",
        version: "v2.14.0",
        requested_by: "ci-bot",
        changes: ["api", "web"],
      },
    },
  },
  {
    type: "line",
    id: "q-enter",
    line: { id: "q-enter", dotClassName: "bg-orange-400", label: "Entered queue", timestamp: ts(600) },
  },
  {
    type: "line",
    id: "q-exit",
    line: { id: "q-exit", dotClassName: "bg-orange-400", label: "Exited queue", timestamp: ts(598) },
  },
  {
    type: "card",
    id: "config",
    card: {
      kind: "payload",
      kicker: "Runtime Config",
      status: { dotClassName: "bg-blue-500", label: "Running" },
      sourceName: "approve-deploy",
      meta: `6m 38s · ${ts(598)}`,
      nodeName: "approve-deploy",
      nodeIcon: approvalIcon,
      payload: {
        items: [
          { type: "user", user: "AleksandarCole" },
          { type: "user", user: "darkofabijan" },
          { type: "group", group: "platform" },
        ],
      },
    },
  },
  {
    type: "line",
    id: "waiting",
    line: { id: "waiting", dotClassName: "bg-amber-500", label: "Waiting for approval", timestamp: ts(597) },
  },
  {
    type: "line",
    id: "appr-ac",
    line: {
      id: "appr-ac",
      dotClassName: "bg-emerald-500",
      label: (
        <span>
          Approved by <span className="font-medium text-slate-800">AleksandarCole</span>
        </span>
      ),
      actor: { name: "AleksandarCole", initials: "AC" },
      timestamp: ts(420),
      detail: "Looks good on staging, approving from my side.",
    },
  },
  {
    type: "line",
    id: "rej-df",
    line: {
      id: "rej-df",
      dotClassName: "bg-red-500",
      label: (
        <span>
          Rejected by <span className="font-medium text-slate-800">darkofabijan</span>
        </span>
      ),
      actor: { name: "darkofabijan", initials: "DF" },
      timestamp: ts(300),
      detail: "Needs another round of load testing before we ship to production.",
    },
  },
  {
    type: "line",
    id: "rej-ma",
    line: {
      id: "rej-ma",
      dotClassName: "bg-red-500",
      label: (
        <span>
          Rejected by <span className="font-medium text-slate-800">markoa</span>
        </span>
      ),
      actor: { name: "markoa", initials: "MA" },
      timestamp: ts(200),
      detail: "Blocking until the security review is completed.",
    },
  },
  {
    type: "card",
    id: "output",
    card: {
      kind: "payload",
      kicker: "Output",
      status: { dotClassName: "bg-red-500", label: "Rejected" },
      sourceName: sizeKb(approvalOutputPayload),
      meta: ts(200),
      nodeName: "approve-deploy",
      nodeIcon: approvalIcon,
      payload: approvalOutputPayload,
    },
  },
  {
    type: "card",
    id: "summary",
    card: {
      kind: "summary",
      status: { badgeColor: "bg-red-500", label: "Rejected" },
      relativeTime: ago(200),
      details: {
        Result: "rejected",
        "Rejected by": "darkofabijan",
        Reason: "Needs another round of load testing before we ship to production.",
      },
    },
  },
];

export const approvalCanceledEvents: TimelineEvent[] = [
  approvalEvents[0],
  {
    type: "line",
    id: "q-enter",
    line: { id: "q-enter", dotClassName: "bg-orange-400", label: "Entered queue", timestamp: ts(600) },
  },
  {
    type: "line",
    id: "q-cancel",
    line: {
      id: "q-cancel",
      dotClassName: "bg-slate-400",
      label: (
        <span>
          Canceled by <span className="font-medium text-slate-800">markoa</span>
        </span>
      ),
      actor: { name: "markoa", initials: "MA" },
      timestamp: ts(560),
      detail: "Superseded by a newer deploy, canceling this one.",
    },
  },
  {
    type: "card",
    id: "summary",
    card: {
      kind: "summary",
      status: { badgeColor: "bg-slate-400", label: "Cancelled" },
      relativeTime: ago(560),
      details: {
        Result: "cancelled",
        "Cancelled by": "markoa",
      },
    },
  },
];

export const runBashEvents: TimelineEvent[] = [
  {
    type: "card",
    id: "input",
    card: {
      kind: "payload",
      kicker: "Input",
      status: { dotClassName: "bg-violet-400", label: "Triggered" },
      sourceName: "checkout-code",
      sourceTrailing: moreChip(2),
      meta: ts(300),
      nodeName: "run-bash",
      nodeIcon: bashIcon,
      payload: { ref: "main", sha: "9f3c1a2", trigger: "push" },
    },
  },
  {
    type: "line",
    id: "q-enter",
    line: { id: "q-enter", dotClassName: "bg-orange-400", label: "Entered queue", timestamp: ts(300) },
  },
  {
    type: "line",
    id: "q-exit",
    line: { id: "q-exit", dotClassName: "bg-orange-400", label: "Exited queue", timestamp: ts(299) },
  },
  {
    type: "card",
    id: "config",
    card: {
      kind: "payload",
      kicker: "Runtime Config",
      status: { dotClassName: "bg-blue-500", label: "Running" },
      sourceName: "run-bash",
      meta: `52s · ${ts(299)}`,
      nodeName: "run-bash",
      nodeIcon: bashIcon,
      payload: { shell: "bash", image: "node:20", timeout: "600s" },
    },
  },
  {
    type: "line",
    id: "cmd-start",
    line: { id: "cmd-start", dotClassName: "bg-blue-500", label: "Command started", timestamp: ts(299) },
  },
  {
    type: "card",
    id: "logs",
    card: {
      kind: "logs",
      status: { dotClassName: "bg-blue-500", label: "Running" },
      sourceName: "run-bash",
      meta: "52s",
      lines: [
        "$ npm run build",
        "> build",
        "> vite build",
        "vite v5.4.2 building for production...",
        "\u2713 312 modules transformed.",
        "dist/index.html   0.75 kB",
        "\u2713 built in 4.21s",
        "$ npm test",
        "PASS  src/lib/utils.test.ts",
        "PASS  src/ui/Runs/runSummary.test.ts",
        "Tests: 42 passed, 42 total",
        "Done in 52.3s.",
      ],
    },
  },
  {
    type: "line",
    id: "cmd-end",
    line: { id: "cmd-end", dotClassName: "bg-emerald-500", label: "Command finished (exit 0)", timestamp: ts(247) },
  },
  {
    type: "card",
    id: "output",
    card: {
      kind: "payload",
      kicker: "Output",
      status: { dotClassName: "bg-emerald-500", label: "Passed" },
      sourceName: sizeKb(runBashOutputPayload),
      meta: ts(247),
      nodeName: "run-bash",
      nodeIcon: bashIcon,
      payload: runBashOutputPayload,
    },
  },
  {
    type: "card",
    id: "summary",
    card: {
      kind: "summary",
      status: { badgeColor: "bg-emerald-500", label: "Passed" },
      relativeTime: ago(247),
      details: { Result: "passed", "Exit code": "0", Duration: "52s" },
    },
  },
];

export const cursorAgentEvents: TimelineEvent[] = [
  {
    type: "card",
    id: "input",
    card: {
      kind: "payload",
      kicker: "Input",
      status: { dotClassName: "bg-violet-400", label: "Triggered" },
      sourceName: "detect-flaky-test",
      sourceTrailing: moreChip(3),
      meta: ts(320),
      nodeName: "cursor-agent",
      nodeIcon: agentIcon,
      payload: { prompt: "Fix the failing e2e test in the checkout flow", repo: "acme/store" },
    },
  },
  {
    type: "line",
    id: "q-enter",
    line: { id: "q-enter", dotClassName: "bg-orange-400", label: "Entered queue", timestamp: ts(320) },
  },
  {
    type: "line",
    id: "q-exit",
    line: { id: "q-exit", dotClassName: "bg-orange-400", label: "Exited queue", timestamp: ts(319) },
  },
  {
    type: "card",
    id: "config",
    card: {
      kind: "payload",
      kicker: "Runtime Config",
      status: { dotClassName: "bg-blue-500", label: "Running" },
      sourceName: "cursor-agent",
      meta: `3m 5s · ${ts(319)}`,
      nodeName: "cursor-agent",
      nodeIcon: agentIcon,
      payload: { model: "composer-1", max_polls: 10, poll_interval_seconds: 60 },
    },
  },
  {
    type: "line",
    id: "sent",
    line: {
      id: "sent",
      dotClassName: "bg-blue-500",
      label: "Request",
      timestamp: ts(318),
      request: {
        method: "POST",
        url: "https://api.cursor.com/v1/agents",
        status: 201,
        statusText: "Created",
        duration: "312ms",
        requestHeaders: {
          Authorization: "Bearer sk-***",
          "Content-Type": "application/json",
          "Idempotency-Key": "run_2c8f1a4e",
        },
        requestBody: {
          prompt: "Fix the failing e2e test in the checkout flow",
          repo: "acme/store",
          model: "composer-1",
        },
        responseHeaders: {
          "Content-Type": "application/json",
          "X-Request-Id": "req_9a1b2c3d",
        },
        responseBody: { id: "agent_7f3c", status: "queued" },
      },
    },
  },
  {
    type: "line",
    id: "poll-1",
    line: {
      id: "poll-1",
      dotClassName: "bg-blue-500",
      label: "Poll #1",
      timestamp: ts(258),
      request: {
        method: "GET",
        url: "https://api.cursor.com/v1/agents/agent_7f3c",
        status: 200,
        statusText: "OK",
        duration: "88ms",
        requestHeaders: { Authorization: "Bearer sk-***" },
        responseHeaders: { "Content-Type": "application/json", "X-Request-Id": "req_1f2e3d4c" },
        responseBody: { status: "running", step: "inspecting the failing test", progress: 0.2 },
      },
    },
  },
  {
    type: "line",
    id: "poll-2",
    line: {
      id: "poll-2",
      dotClassName: "bg-blue-500",
      label: "Poll #2",
      timestamp: ts(198),
      request: {
        method: "GET",
        url: "https://api.cursor.com/v1/agents/agent_7f3c",
        status: 200,
        statusText: "OK",
        duration: "91ms",
        requestHeaders: { Authorization: "Bearer sk-***" },
        responseHeaders: { "Content-Type": "application/json", "X-Request-Id": "req_5b6a7988" },
        responseBody: { status: "running", step: "editing checkout.spec.ts", progress: 0.6 },
      },
    },
  },
  {
    type: "line",
    id: "poll-3",
    line: {
      id: "poll-3",
      dotClassName: "bg-blue-500",
      label: "Poll #3",
      timestamp: ts(138),
      request: {
        method: "GET",
        url: "https://api.cursor.com/v1/agents/agent_7f3c",
        status: 200,
        statusText: "OK",
        duration: "84ms",
        requestHeaders: { Authorization: "Bearer sk-***" },
        responseHeaders: { "Content-Type": "application/json", "X-Request-Id": "req_c4d3e2f1" },
        responseBody: { status: "running", step: "re-running the test suite", progress: 0.9 },
      },
    },
  },
  {
    type: "line",
    id: "response",
    line: {
      id: "response",
      dotClassName: "bg-blue-500",
      label: "Response",
      timestamp: ts(134),
      request: {
        method: "GET",
        url: "https://api.cursor.com/v1/agents/agent_7f3c",
        status: 200,
        statusText: "OK",
        duration: "79ms",
        requestHeaders: { Authorization: "Bearer sk-***" },
        responseHeaders: { "Content-Type": "application/json", "X-Request-Id": "req_a0b1c2d3" },
        responseBody: {
          status: "completed",
          summary: "Fixed a selector race condition in checkout.spec.ts and stabilized the wait.",
          files_changed: ["e2e/checkout.spec.ts"],
          pr_url: "https://github.com/acme/store/pull/482",
        },
      },
    },
  },
  {
    type: "card",
    id: "output",
    card: {
      kind: "payload",
      kicker: "Output",
      status: { dotClassName: "bg-emerald-500", label: "Passed" },
      sourceName: sizeKb(cursorAgentOutputPayload),
      meta: ts(134),
      nodeName: "cursor-agent",
      nodeIcon: agentIcon,
      payload: cursorAgentOutputPayload,
    },
  },
  {
    type: "card",
    id: "summary",
    card: {
      kind: "summary",
      status: { badgeColor: "bg-emerald-500", label: "Passed" },
      relativeTime: ago(134),
      details: { Result: "passed", "Files changed": "1", Polls: "3" },
    },
  },
];

/** A GitHub API component (create pull request) that completes successfully via a single API call. */
export const githubEvents: TimelineEvent[] = [
  {
    type: "card",
    id: "input",
    card: {
      kind: "payload",
      kicker: "Input",
      status: { dotClassName: "bg-violet-400", label: "Triggered" },
      sourceName: "cursor-agent",
      sourceTrailing: moreChip(3),
      meta: ts(90),
      nodeName: "open-pull-request",
      nodeIcon: githubIcon,
      payload: { repo: "acme/store", head: "feature/checkout-fix", base: "main" },
    },
  },
  {
    type: "line",
    id: "q-enter",
    line: { id: "q-enter", dotClassName: "bg-orange-400", label: "Entered queue", timestamp: ts(90) },
  },
  {
    type: "line",
    id: "q-exit",
    line: { id: "q-exit", dotClassName: "bg-orange-400", label: "Exited queue", timestamp: ts(89) },
  },
  {
    type: "card",
    id: "config",
    card: {
      kind: "payload",
      kicker: "Runtime Config",
      status: { dotClassName: "bg-blue-500", label: "Running" },
      sourceName: "open-pull-request",
      meta: `1s · ${ts(89)}`,
      nodeName: "open-pull-request",
      nodeIcon: githubIcon,
      payload: { owner: "acme", repo: "store", draft: false },
    },
  },
  {
    type: "line",
    id: "request",
    line: {
      id: "request",
      dotClassName: "bg-blue-500",
      label: "Request",
      timestamp: ts(89),
      request: {
        method: "POST",
        url: "https://api.github.com/repos/acme/store/pulls",
        status: 201,
        statusText: "Created",
        duration: "486ms",
        requestHeaders: {
          Authorization: "Bearer ghp_***",
          Accept: "application/vnd.github+json",
          "Content-Type": "application/json",
        },
        requestBody: {
          title: "Fix flaky checkout e2e test",
          head: "feature/checkout-fix",
          base: "main",
          body: "Stabilizes the checkout spec by fixing a selector race condition.",
        },
        responseHeaders: {
          "Content-Type": "application/json",
          "X-GitHub-Request-Id": "A1B2:C3D4:E5F6",
        },
        responseBody: githubOutputPayload,
      },
    },
  },
  {
    type: "card",
    id: "output",
    card: {
      kind: "payload",
      kicker: "Output",
      status: { dotClassName: "bg-emerald-500", label: "Passed" },
      sourceName: sizeKb(githubOutputPayload),
      meta: ts(89),
      nodeName: "open-pull-request",
      nodeIcon: githubIcon,
      payload: githubOutputPayload,
    },
  },
  {
    type: "card",
    id: "summary",
    card: {
      kind: "summary",
      status: { badgeColor: "bg-emerald-500", label: "Passed" },
      relativeTime: ago(89),
      details: { Result: "passed", "PR number": "482", State: "open" },
    },
  },
];

/** A GitHub API component whose API call fails validation (422), so the step errors and emits no output. */
export const githubErrorEvents: TimelineEvent[] = [
  {
    type: "card",
    id: "input",
    card: {
      kind: "payload",
      kicker: "Input",
      status: { dotClassName: "bg-violet-400", label: "Triggered" },
      sourceName: "cursor-agent",
      sourceTrailing: moreChip(3),
      meta: ts(70),
      nodeName: "open-pull-request",
      nodeIcon: githubIcon,
      payload: { repo: "acme/store", head: "feature/checkout-fix", base: "main" },
    },
  },
  {
    type: "line",
    id: "q-enter",
    line: { id: "q-enter", dotClassName: "bg-orange-400", label: "Entered queue", timestamp: ts(70) },
  },
  {
    type: "line",
    id: "q-exit",
    line: { id: "q-exit", dotClassName: "bg-orange-400", label: "Exited queue", timestamp: ts(69) },
  },
  {
    type: "card",
    id: "config",
    card: {
      kind: "payload",
      kicker: "Runtime Config",
      status: { dotClassName: "bg-blue-500", label: "Running" },
      sourceName: "open-pull-request",
      meta: `1s · ${ts(69)}`,
      nodeName: "open-pull-request",
      nodeIcon: githubIcon,
      payload: { owner: "acme", repo: "store", draft: false },
    },
  },
  {
    type: "line",
    id: "request",
    line: {
      id: "request",
      dotClassName: "bg-red-500",
      label: "Request",
      timestamp: ts(69),
      request: {
        method: "POST",
        url: "https://api.github.com/repos/acme/store/pulls",
        status: 422,
        statusText: "Unprocessable Entity",
        duration: "203ms",
        requestHeaders: {
          Authorization: "Bearer ghp_***",
          Accept: "application/vnd.github+json",
          "Content-Type": "application/json",
        },
        requestBody: {
          title: "Fix flaky checkout e2e test",
          head: "feature/checkout-fix",
          base: "main",
        },
        responseHeaders: {
          "Content-Type": "application/json",
          "X-GitHub-Request-Id": "F6E5:D4C3:B2A1",
        },
        responseBody: {
          message: "Validation Failed",
          errors: [
            {
              resource: "PullRequest",
              code: "custom",
              message: "A pull request already exists for acme:feature/checkout-fix.",
            },
          ],
          documentation_url: "https://docs.github.com/rest/pulls/pulls#create-a-pull-request",
        },
      },
    },
  },
  {
    type: "card",
    id: "error",
    card: {
      kind: "error",
      message: "A pull request already exists for acme:feature/checkout-fix.",
      reason: "RESULT_REASON_ERROR",
      metadata: {
        "Status code": 422,
        Endpoint: "POST /repos/acme/store/pulls",
        "Request ID": "F6E5:D4C3:B2A1",
      },
    },
  },
];

/** A memory component (upsert) that stores a value and completes successfully. */
export const memoryEvents: TimelineEvent[] = [
  {
    type: "card",
    id: "input",
    card: {
      kind: "payload",
      kicker: "Input",
      status: { dotClassName: "bg-violet-400", label: "Triggered" },
      sourceName: "open-pull-request",
      sourceTrailing: moreChip(5),
      meta: ts(40),
      nodeName: "remember-release",
      nodeIcon: memoryIcon,
      payload: { namespace: "deploys", key: "last-production-release", version: "v2.14.0" },
    },
  },
  {
    type: "line",
    id: "q-enter",
    line: { id: "q-enter", dotClassName: "bg-orange-400", label: "Entered queue", timestamp: ts(40) },
  },
  {
    type: "line",
    id: "q-exit",
    line: { id: "q-exit", dotClassName: "bg-orange-400", label: "Exited queue", timestamp: ts(40) },
  },
  {
    type: "card",
    id: "config",
    card: {
      kind: "payload",
      kicker: "Runtime Config",
      status: { dotClassName: "bg-blue-500", label: "Running" },
      sourceName: "remember-release",
      meta: `0.2s · ${ts(40)}`,
      nodeName: "remember-release",
      nodeIcon: memoryIcon,
      payload: { namespace: "deploys", ttl_seconds: null, overwrite: true },
    },
  },
  {
    type: "line",
    id: "stored",
    line: {
      id: "stored",
      dotClassName: "bg-emerald-500",
      label: "Stored key last-production-release",
      timestamp: ts(40),
    },
  },
  {
    type: "card",
    id: "output",
    card: {
      kind: "payload",
      kicker: "Output",
      status: { dotClassName: "bg-emerald-500", label: "Passed" },
      sourceName: sizeKb(memoryOutputPayload),
      meta: ts(40),
      nodeName: "remember-release",
      nodeIcon: memoryIcon,
      payload: memoryOutputPayload,
    },
  },
  {
    type: "card",
    id: "summary",
    card: {
      kind: "summary",
      status: { badgeColor: "bg-emerald-500", label: "Passed" },
      relativeTime: ago(40),
      details: { Result: "passed", Namespace: "deploys", Key: "last-production-release" },
    },
  },
];
