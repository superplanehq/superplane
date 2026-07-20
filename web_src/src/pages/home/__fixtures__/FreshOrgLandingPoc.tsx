import { Button } from "@/components/ui/button";
import { RequirePermission } from "@/components/PermissionGate";
import { useAvailableIntegrations, useCreateIntegration } from "@/hooks/useIntegrations";
import { useOrganizationId } from "@/hooks/useOrganizationId";
import { appPath } from "@/lib/appPaths";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { getNextIntegrationName } from "@/pages/organization/settings/components/IntegrationSetup/lib";
import { cn } from "@/lib/utils";
import { canvasAppIds } from "@/pages/app/__fixtures__/handlers";
import { AutoCompleteSelect, type AutoCompleteOption } from "@/components/AutoCompleteSelect";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { IntegrationCreateDialog } from "@/ui/IntegrationCreateDialog";
import {
  ArrowLeft,
  ArrowRight,
  Eye,
  GitPullRequest,
  ListTodo,
  MessageSquare,
  Plus,
  RefreshCw,
  Terminal,
  Trash2,
  UserRound,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";

import { AppDetailModal, LeadIcon, type AppEntry } from "../AppDetailModal";
import { APP_CATALOG } from "../appCatalog";
import { HomePageShell } from "../HomePageShell";
import {
  homeCardTitleClassName,
  homeListCardClassName,
  homePageSubtitleClassName,
  homePageTitleClassName,
} from "../homePageStyles";
import { InstallProgressPanel } from "../InstallProgressPanel";
import { useCreateApp } from "../useCreateApp";

const SETUP_STEPS = [
  {
    title: "Triggers",
    detail: "Select the triggers you want to use to kick off Software Factory work.",
  },
  {
    title: "Version control",
    detail: "Where the factory checks out code and opens pull or merge requests.",
  },
  {
    title: "Coding agent",
    detail: "Pick a harness. Agents run in the SuperPlane sandbox; you provide the API key.",
  },
  {
    title: "Agent settings",
    detail: "Tune the planning, implementation, and PR review components the factory will run.",
  },
  {
    title: "Preview",
    detail: "Confirm your setup before creating the Software Factory.",
  },
] as const;

type AgentStepKind = "prompt" | "bash";

type AgentPipelineStep = {
  id: string;
  kind: AgentStepKind;
  title: string;
  body: string;
};

type AgentComponentId = "planning" | "implementation" | "pr-loop";

type AgentComponentConfig = {
  id: AgentComponentId;
  title: string;
  purpose: string;
  modelId: string;
  machineType: string;
  /** PR review loop only. */
  maxRetries?: number;
  steps: AgentPipelineStep[];
};

function newAgentStepId(): string {
  return `step-${Math.random().toString(36).slice(2, 10)}`;
}

function createAgentStep(kind: AgentStepKind, title: string, body: string): AgentPipelineStep {
  return { id: newAgentStepId(), kind, title, body };
}

/** Storybook fixture machine sizes (mirrors SuperPlane runner fleets). */
const AGENT_MACHINE_OPTIONS: AutoCompleteOption[] = [
  { value: "e1-tiny-amd64", label: "Tiny · AMD64 · 2 vCPU / 4 GB" },
  { value: "e1-tiny-arm64", label: "Tiny · ARM64 · 2 vCPU / 4 GB" },
  { value: "e1-large-amd64", label: "Large · AMD64 · 8 vCPU / 16 GB" },
  { value: "e1-large-arm64", label: "Large · ARM64 · 8 vCPU / 16 GB" },
];

const DEFAULT_AGENT_MACHINE = "e1-large-amd64";
const DEFAULT_PR_LOOP_MAX_RETRIES = 5;

const FIXTURE_MODELS_BY_PROVIDER: Record<string, AutoCompleteOption[]> = {
  claude: [
    { value: "claude-sonnet-4-6", label: "Claude Sonnet 4.6" },
    { value: "claude-opus-4-6", label: "Claude Opus 4.6" },
    { value: "claude-haiku-4-5", label: "Claude Haiku 4.5" },
  ],
  openai: [
    { value: "gpt-5.2", label: "GPT-5.2" },
    { value: "gpt-5.2-codex", label: "GPT-5.2 Codex" },
    { value: "o3-mini", label: "o3-mini" },
  ],
  cursor: [
    { value: "auto", label: "Auto (recommended)" },
    { value: "claude-sonnet-4-6", label: "Claude Sonnet 4.6" },
    { value: "gpt-5.2", label: "GPT-5.2" },
  ],
  gemini: [
    { value: "gemini-2.5-pro", label: "Gemini 2.5 Pro" },
    { value: "gemini-2.5-flash", label: "Gemini 2.5 Flash" },
  ],
  openrouter: [
    { value: "openrouter/auto", label: "OpenRouter Auto" },
    { value: "anthropic/claude-sonnet-4", label: "Claude Sonnet 4" },
    { value: "openai/gpt-5.2", label: "GPT-5.2" },
  ],
  deepseek: [
    { value: "deepseek-chat", label: "DeepSeek Chat" },
    { value: "deepseek-reasoner", label: "DeepSeek Reasoner" },
  ],
  groq: [
    { value: "llama-3.3-70b", label: "Llama 3.3 70B" },
    { value: "qwen-qwq-32b", label: "Qwen QwQ 32B" },
  ],
  mistral: [
    { value: "mistral-large-latest", label: "Mistral Large" },
    { value: "codestral-latest", label: "Codestral" },
  ],
  zen: [
    { value: "big-pickle", label: "Big Pickle (free)" },
    { value: "deepseek-v4-flash-free", label: "DeepSeek V4 Flash Free" },
    { value: "mimo-v2.5-free", label: "MiMo-V2.5 Free" },
  ],
  ollama: [
    { value: "qwen2.5-coder", label: "Qwen 2.5 Coder" },
    { value: "llama3.1", label: "Llama 3.1" },
    { value: "deepseek-coder-v2", label: "DeepSeek Coder V2" },
  ],
};

function modelProviderKeyForHarness(
  harness: AgentHarnessId | null,
  openCodeProvider: OpenCodeProviderId | null,
): string {
  if (harness === "claude-code") return "claude";
  if (harness === "codex") return "openai";
  if (harness === "cursor") return "cursor";
  if (harness === "open-code") {
    if (openCodeProvider === "anthropic") return "claude";
    if (openCodeProvider === "openai") return "openai";
    return openCodeProvider ?? "zen";
  }
  return "claude";
}

function fixtureModelsForHarness(
  harness: AgentHarnessId | null,
  openCodeProvider: OpenCodeProviderId | null,
): AutoCompleteOption[] {
  const key = modelProviderKeyForHarness(harness, openCodeProvider);
  return FIXTURE_MODELS_BY_PROVIDER[key] ?? FIXTURE_MODELS_BY_PROVIDER.claude;
}

function defaultModelForHarness(harness: AgentHarnessId | null, openCodeProvider: OpenCodeProviderId | null): string {
  return fixtureModelsForHarness(harness, openCodeProvider)[0]?.value ?? "claude-sonnet-4-6";
}

function cloneDefaultAgentComponents(
  harness: AgentHarnessId | null = "claude-code",
  openCodeProvider: OpenCodeProviderId | null = null,
): AgentComponentConfig[] {
  const modelId = defaultModelForHarness(harness, openCodeProvider);
  return [
    {
      id: "planning",
      title: "Planning",
      purpose: "Turns the trigger into a short implementation plan before coding starts.",
      modelId,
      machineType: DEFAULT_AGENT_MACHINE,
      steps: [
        createAgentStep(
          "prompt",
          "Create plan",
          "Analyze the issue or prompt and produce a short implementation plan. List files to touch, risks, and a clear definition of done.",
        ),
      ],
    },
    {
      id: "implementation",
      title: "Implementation",
      purpose: "Applies the plan in the repo, runs checks, and opens a pull or merge request.",
      modelId,
      machineType: DEFAULT_AGENT_MACHINE,
      steps: [
        createAgentStep(
          "prompt",
          "Implement",
          "Implement the plan with minimal changes. Prefer small, reviewable diffs and update tests when needed.",
        ),
        createAgentStep("bash", "Format and test", "npm test || go test ./... || yarn test || echo 'Tests completed'"),
        createAgentStep(
          "prompt",
          "Open pull request",
          "Open a pull or merge request that summarizes the change, root cause, and test results.",
        ),
      ],
    },
    {
      id: "pr-loop",
      title: "PR review loop",
      purpose: "Watches checks and review comments, then addresses feedback until the PR is mergeable.",
      modelId,
      machineType: DEFAULT_AGENT_MACHINE,
      maxRetries: DEFAULT_PR_LOOP_MAX_RETRIES,
      steps: [
        createAgentStep(
          "prompt",
          "Respond to feedback",
          "Watch CI checks and review comments. Fix failures and address feedback with minimal follow-up commits.",
        ),
        createAgentStep(
          "prompt",
          "Re-verify",
          "Confirm checks are green and the latest feedback is addressed. Summarize what changed since the last update.",
        ),
      ],
    },
  ];
}

const OUTCOME_STEPS: {
  title: string;
  detail: string;
  phase: "ready" | "running" | "review" | "done";
}[] = [
  {
    title: "Work is triggered",
    detail: "Manual prompt, issue, or PR/MR tag starts a run.",
    phase: "ready",
  },
  {
    title: "Agent plans and codes",
    detail: "Harness runs in the SuperPlane sandbox.",
    phase: "running",
  },
  {
    title: "Opens a pull request",
    detail: "Branch + PR or merge request.",
    phase: "running",
  },
  {
    title: "Keeps checks passing",
    detail: "Watches PR checks and loops on failures until they pass.",
    phase: "running",
  },
  {
    title: "Waits for your review",
    detail: "You stay on the loop.",
    phase: "review",
  },
  {
    title: "Addresses review comments",
    detail: "Agent updates the PR from feedback.",
    phase: "running",
  },
  {
    title: "Gets to a mergeable state",
    detail: "PR ready for you to merge.",
    phase: "done",
  },
];

type TriggerSourceId = "manual" | "issue" | "prOrMrTag";

const TRIGGER_SOURCES: { id: TriggerSourceId; title: string }[] = [
  { id: "manual", title: "Manual prompt" },
  { id: "issue", title: "Assign SuperPlane bot to your issue" },
  { id: "prOrMrTag", title: "Mention SuperPlane in your pull or merge request" },
];

type IntegrationChoice = {
  id: string;
  label: string;
  integrationName: string;
};

const ISSUE_TRACKERS: IntegrationChoice[] = [
  { id: "github", label: "GitHub Issues", integrationName: "github" },
  { id: "gitlab", label: "GitLab Issues", integrationName: "gitlab" },
  { id: "linear", label: "Linear", integrationName: "linear" },
  { id: "jira", label: "Jira", integrationName: "jira" },
];

const PR_MR_PROVIDERS: IntegrationChoice[] = [
  { id: "github", label: "GitHub pull request", integrationName: "github" },
  { id: "gitlab", label: "GitLab merge request", integrationName: "gitlab" },
];

type VcsHostId = "github" | "gitlab";

const VCS_HOSTS: IntegrationChoice[] = [
  { id: "github", label: "GitHub", integrationName: "github" },
  { id: "gitlab", label: "GitLab", integrationName: "gitlab" },
];

/** Storybook-only simulated repos — not fetched from a connected integration. */
const FIXTURE_REPOS: Record<VcsHostId, string[]> = {
  github: ["acme/api", "acme/web", "acme/workers"],
  gitlab: ["acme/backend", "acme/frontend", "acme/platform"],
};

type AgentHarnessId = "claude-code" | "codex" | "cursor" | "open-code";

const AGENT_HARNESSES: IntegrationChoice[] = [
  { id: "claude-code", label: "Claude Code", integrationName: "claude" },
  { id: "codex", label: "Codex", integrationName: "openai" },
  { id: "cursor", label: "Cursor", integrationName: "cursor" },
  { id: "open-code", label: "Open Code", integrationName: "opencode" },
];

type OpenCodeProviderId =
  | "zen"
  | "ollama"
  | "anthropic"
  | "openai"
  | "gemini"
  | "openrouter"
  | "deepseek"
  | "groq"
  | "mistral";

type OpenCodeProviderMode = "integration" | "apiKey" | "none";

type OpenCodeProvider = {
  id: OpenCodeProviderId;
  label: string;
  /** Logo key for IntegrationIcon. */
  iconName: string;
  mode: OpenCodeProviderMode;
  group: "free" | "providers";
  /** SuperPlane integration name when mode is "integration". */
  integrationKey?: "claude" | "openai";
};

/** Providers Open Code can use. Anthropic/OpenAI use native SuperPlane integrations. */
const OPEN_CODE_PROVIDERS: OpenCodeProvider[] = [
  { id: "zen", label: "OpenCode Zen", iconName: "opencode", mode: "apiKey", group: "free" },
  { id: "ollama", label: "Ollama", iconName: "ollama", mode: "none", group: "free" },
  {
    id: "anthropic",
    label: "Anthropic",
    iconName: "anthropic",
    mode: "integration",
    group: "providers",
    integrationKey: "claude",
  },
  {
    id: "openai",
    label: "OpenAI",
    iconName: "openai",
    mode: "integration",
    group: "providers",
    integrationKey: "openai",
  },
  { id: "gemini", label: "Google Gemini", iconName: "gemini", mode: "apiKey", group: "providers" },
  { id: "openrouter", label: "OpenRouter", iconName: "openrouter", mode: "apiKey", group: "providers" },
  { id: "deepseek", label: "DeepSeek", iconName: "deepseek", mode: "apiKey", group: "providers" },
  { id: "groq", label: "Groq", iconName: "groq", mode: "apiKey", group: "providers" },
  { id: "mistral", label: "Mistral", iconName: "mistral", mode: "apiKey", group: "providers" },
];

const OPEN_CODE_FREE_PROVIDERS = OPEN_CODE_PROVIDERS.filter((provider) => provider.group === "free");
const OPEN_CODE_CLOUD_PROVIDERS = OPEN_CODE_PROVIDERS.filter((provider) => provider.group === "providers");

function openCodeProviderById(id: OpenCodeProviderId | null): OpenCodeProvider | undefined {
  return id ? OPEN_CODE_PROVIDERS.find((provider) => provider.id === id) : undefined;
}

function openCodeProviderModeLabel(mode: OpenCodeProviderMode): string {
  if (mode === "integration") return "Connect";
  if (mode === "apiKey") return "API key";
  return "Local";
}

function agentIntegrationForSetup(
  harness: AgentHarnessId | null,
  openCodeProvider: OpenCodeProviderId | null,
): string | null {
  if (harness === "claude-code") return "claude";
  if (harness === "codex") return "openai";
  if (harness === "cursor") return "cursor";
  if (harness === "open-code") {
    const provider = openCodeProviderById(openCodeProvider);
    if (provider?.mode === "integration") {
      return provider.integrationKey ?? null;
    }
  }
  return null;
}

function openCodeNeedsApiKey(harness: AgentHarnessId | null, openCodeProvider: OpenCodeProviderId | null): boolean {
  if (harness !== "open-code") return false;
  return openCodeProviderById(openCodeProvider)?.mode === "apiKey";
}

/**
 * Storybook-only fresh-org landing POC: densified split (decision + outcome)
 * with quiet escape hatches for blank apps and the starter catalog.
 * Not mounted in production routes.
 */
export function FreshOrgLandingPage() {
  return (
    <RequirePermission resource="canvases" action="create">
      <HomePageShell>
        <FreshOrgLandingPoc />
      </HomePageShell>
    </RequirePermission>
  );
}

export function FreshOrgLandingPoc() {
  const { createApp, isSaving } = useCreateApp();
  const [showCatalog, setShowCatalog] = useState(false);
  const [showFactorySetup, setShowFactorySetup] = useState(false);
  const [visibleCount, setVisibleCount] = useState(7);
  const [selectedApp, setSelectedApp] = useState<AppEntry | null>(null);
  const [installingApp, setInstallingApp] = useState<AppEntry | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const busy = isSaving || installingApp !== null;
  const visible = APP_CATALOG.slice(0, visibleCount);

  useEffect(() => {
    if (!showCatalog) return;
    const el = sentinelRef.current;
    if (!el) return;
    if (typeof IntersectionObserver === "undefined") {
      setVisibleCount(APP_CATALOG.length);
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisibleCount((prev) => Math.min(prev + 7, APP_CATALOG.length));
        }
      },
      { rootMargin: "100px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [showCatalog]);

  if (showFactorySetup) {
    return <FactorySetupWizard onExit={() => setShowFactorySetup(false)} />;
  }

  return (
    <>
      {selectedApp && (
        <AppDetailModal
          app={selectedApp}
          busy={busy}
          onBack={() => setSelectedApp(null)}
          onInstall={(e) => {
            e.stopPropagation();
            setInstallingApp(selectedApp);
            setSelectedApp(null);
          }}
          onClose={() => setSelectedApp(null)}
        />
      )}

      <div className="mx-auto w-full max-w-6xl px-8 py-14 lg:py-20">
        <div className="grid items-start gap-12 lg:grid-cols-[0.9fr_1.1fr] lg:gap-14">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-gray-400 dark:text-gray-500">
              Recommended
            </p>
            <h1 className="mt-3 max-w-[16ch] text-2xl font-semibold tracking-tight text-slate-900 sm:text-3xl dark:text-gray-100">
              Ship PRs to a mergeable state
            </h1>
            <p className={cn(homePageSubtitleClassName, "mt-3 max-w-md")}>
              An automated app that orchestrates cloud agents to solve issues end to end. Agents run in the SuperPlane
              sandbox. You review; the factory keeps going until the PR is mergeable.
            </p>
            <div className="mt-7">
              <Button type="button" size="lg" onClick={() => setShowFactorySetup(true)}>
                Start setup
                <ArrowRight />
              </Button>
            </div>

            <ol className="mt-8 space-y-4" aria-label="Setup steps">
              {SETUP_STEPS.map((step, index) => (
                <li key={step.title} className="flex gap-3">
                  <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-slate-900 text-[11px] font-semibold text-white dark:bg-gray-100 dark:text-slate-900">
                    {index + 1}
                  </span>
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">{step.title}</p>
                    <p className="mt-0.5 text-sm leading-snug text-gray-500 dark:text-gray-400">{step.detail}</p>
                  </div>
                </li>
              ))}
            </ol>

            <div className="mt-8 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-gray-400 dark:text-gray-500">
              <span>Or</span>
              <button
                type="button"
                disabled={busy}
                onClick={() => {
                  if (busy) return;
                  void createApp(generateCanvasName());
                }}
                className="inline-flex items-center gap-1.5 font-medium text-gray-500 underline-offset-4 hover:text-slate-900 hover:underline disabled:opacity-50 dark:text-gray-400 dark:hover:text-gray-100"
              >
                <Plus className="h-3.5 w-3.5" aria-hidden />
                Create a blank app
              </button>
              <span className="text-slate-300 dark:text-gray-600" aria-hidden>
                ·
              </span>
              <button
                type="button"
                onClick={() => setShowCatalog((open) => !open)}
                className="font-medium text-gray-500 underline-offset-4 hover:text-slate-900 hover:underline dark:text-gray-400 dark:hover:text-gray-100"
                aria-expanded={showCatalog}
              >
                {showCatalog ? "Hide starter apps" : "Browse other starter apps"}
              </button>
            </div>
          </div>

          <aside
            className={cn(
              "rounded-xl bg-white px-6 py-6 outline outline-slate-950/10",
              "dark:bg-gray-900 dark:outline-gray-700/60",
            )}
          >
            <h2 className="text-sm font-semibold text-slate-900 dark:text-gray-100">What you get</h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              End-to-end orchestration from trigger to a mergeable PR.
            </p>
            <ol className="relative mt-6 space-y-0">
              {OUTCOME_STEPS.map((step, index) => {
                const isLast = index === OUTCOME_STEPS.length - 1;
                return (
                  <li key={step.title} className="relative flex gap-4 pb-6 last:pb-0">
                    {!isLast && (
                      <span
                        className="absolute top-9 bottom-0 left-[15px] w-px bg-slate-200 dark:bg-gray-700"
                        aria-hidden
                      />
                    )}
                    <span
                      className={cn(
                        "relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-bold",
                        step.phase === "ready" && "bg-sky-100 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300",
                        step.phase === "running" &&
                          "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300",
                        step.phase === "review" &&
                          "bg-orange-100 text-orange-700 dark:bg-orange-950/40 dark:text-orange-300",
                        step.phase === "done" && "bg-sky-100 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300",
                      )}
                    >
                      {index + 1}
                    </span>
                    <div className="min-w-0 pt-0.5">
                      <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">{step.title}</p>
                      <p className="mt-0.5 text-sm leading-snug text-gray-500 dark:text-gray-400">{step.detail}</p>
                    </div>
                  </li>
                );
              })}
            </ol>
          </aside>
        </div>

        {showCatalog && (
          <div className="mt-10 flex flex-col gap-3">
            <p className="text-center text-xs font-medium text-gray-400 dark:text-gray-500">
              Automation starters (not Software Factory setup)
            </p>
            {visible.map((app) => (
              <StarterAppListItem
                key={app.repo}
                app={app}
                busy={busy}
                isInstalling={installingApp?.repo === app.repo}
                onSelect={setSelectedApp}
                onInstall={(entry) => {
                  if (busy) return;
                  setInstallingApp(entry);
                  setSelectedApp(null);
                }}
                onCloseInstall={() => setInstallingApp(null)}
              />
            ))}
            {visibleCount < APP_CATALOG.length && <div ref={sentinelRef} className="h-1" />}
          </div>
        )}
      </div>
    </>
  );
}

type RequiredIntegration = { key: string; label: string; integrationName: string };

const INTEGRATION_LABELS: Record<string, string> = {
  github: "GitHub",
  gitlab: "GitLab",
  linear: "Linear",
  jira: "Jira",
  claude: "Claude",
  openai: "OpenAI",
  cursor: "Cursor",
  gemini: "Gemini",
  opencode: "Open Code",
};

function vcsHostsFromTriggers(issueTracker: string | null, prMrProvider: string | null): VcsHostId[] {
  const hosts = new Set<VcsHostId>();
  if (issueTracker === "github" || issueTracker === "gitlab") {
    hosts.add(issueTracker);
  }
  if (prMrProvider === "github" || prMrProvider === "gitlab") {
    hosts.add(prMrProvider);
  }
  return [...hosts];
}

function requiredIntegrationsForSetup(
  triggerSources: Set<TriggerSourceId>,
  issueTracker: string | null,
  prMrProvider: string | null,
  vcsHost: VcsHostId | null,
  agentHarness: AgentHarnessId | null,
  openCodeProvider: OpenCodeProviderId | null,
): RequiredIntegration[] {
  const integrationNames: string[] = [];

  if (triggerSources.has("issue") && issueTracker) {
    const tracker = ISSUE_TRACKERS.find((choice) => choice.id === issueTracker);
    if (tracker) {
      integrationNames.push(tracker.integrationName);
    }
  }

  if (triggerSources.has("prOrMrTag") && prMrProvider) {
    const provider = PR_MR_PROVIDERS.find((choice) => choice.id === prMrProvider);
    if (provider) {
      integrationNames.push(provider.integrationName);
    }
  }

  if (vcsHost) {
    integrationNames.push(vcsHost);
  }

  const agentIntegration = agentIntegrationForSetup(agentHarness, openCodeProvider);
  if (agentIntegration) {
    integrationNames.push(agentIntegration);
  }

  return [...new Set(integrationNames)].map((integrationName) => ({
    key: integrationName,
    label: INTEGRATION_LABELS[integrationName] ?? integrationName,
    integrationName,
  }));
}

const LIVE_CANVAS_STORYBOOK_PATH = "/?path=/story/pages-apppage--live-canvas";

/** Prefer switching the Storybook story; fall back to in-harness AppPage navigation. */
function goToLiveCanvas(navigate: ReturnType<typeof useNavigate>, organizationId: string) {
  try {
    if (window.top && window.top !== window) {
      const origin = window.top.location.origin;
      window.top.location.assign(`${origin}${LIVE_CANVAS_STORYBOOK_PATH}`);
      return;
    }
  } catch {
    // Cross-origin parent (or restricted top access) — use in-app navigation.
  }

  if (
    typeof window !== "undefined" &&
    (window.location.port === "6006" || window.location.search.includes("path=/story"))
  ) {
    window.location.assign(LIVE_CANVAS_STORYBOOK_PATH);
    return;
  }

  navigate(appPath(organizationId || canvasAppIds.organizationId, canvasAppIds.canvasId));
}

function FactorySetupWizard({ onExit }: { onExit: () => void }) {
  const navigate = useNavigate();
  const organizationId = useOrganizationId() ?? "";
  const { data: availableIntegrations = [] } = useAvailableIntegrations({ enabled: !!organizationId });
  const createIntegrationMutation = useCreateIntegration(organizationId, "install_wizard");

  const [stepIndex, setStepIndex] = useState(0);
  const [triggerSources, setTriggerSources] = useState<Set<TriggerSourceId>>(new Set());
  const [issueTracker, setIssueTracker] = useState<string | null>(null);
  const [prMrProvider, setPrMrProvider] = useState<string | null>(null);
  const [vcsHost, setVcsHost] = useState<VcsHostId | null>(null);
  const [defaultRepoId, setDefaultRepoId] = useState<string | null>(null);
  const [agentHarness, setAgentHarness] = useState<AgentHarnessId | null>(null);
  const [openCodeProvider, setOpenCodeProvider] = useState<OpenCodeProviderId | null>(null);
  const [agentApiKey, setAgentApiKey] = useState("");
  const [agentComponents, setAgentComponents] = useState<AgentComponentConfig[]>(() => cloneDefaultAgentComponents());
  const [connectedTools, setConnectedTools] = useState<Set<string>>(new Set());
  const [dialogIntegrationName, setDialogIntegrationName] = useState<string | null>(null);
  const pendingConnectKeyRef = useRef<string | null>(null);

  const step = SETUP_STEPS[stepIndex];
  const isTriggerStep = stepIndex === 0;
  const isVcsStep = stepIndex === 1;
  const isAgentStep = stepIndex === 2;
  const isSettingsStep = stepIndex === 3;
  const isFinalStep = stepIndex >= SETUP_STEPS.length - 1;
  const issueSelected = triggerSources.has("issue");
  const prMrSelected = triggerSources.has("prOrMrTag");
  const triggerChoicesReady =
    triggerSources.size > 0 && (!issueSelected || issueTracker !== null) && (!prMrSelected || prMrProvider !== null);
  const triggerVcsHosts = useMemo(() => vcsHostsFromTriggers(issueTracker, prMrProvider), [issueTracker, prMrProvider]);
  const vcsHostLocked = triggerVcsHosts.length === 1;
  const availableVcsHosts = useMemo(
    () => (triggerVcsHosts.length > 0 ? triggerVcsHosts : (["github", "gitlab"] as VcsHostId[])),
    [triggerVcsHosts],
  );
  const vcsChoicesReady = vcsHost !== null;
  const agentChoicesReady = agentHarness !== null && (agentHarness !== "open-code" || openCodeProvider !== null);
  const needsOpenCodeApiKey = openCodeNeedsApiKey(agentHarness, openCodeProvider);
  const selectedOpenCodeProvider = openCodeProviderById(openCodeProvider);
  const requiredIntegrations = requiredIntegrationsForSetup(
    triggerSources,
    issueTracker,
    prMrProvider,
    vcsHost,
    agentHarness,
    openCodeProvider,
  );
  const openCodeKeyReady = !needsOpenCodeApiKey || agentApiKey.trim().length > 0;
  const defaultRepoReady = !vcsHost || (defaultRepoId !== null && defaultRepoId.length > 0);
  const setupReady =
    requiredIntegrations.every((item) => connectedTools.has(item.key)) && openCodeKeyReady && defaultRepoReady;
  const emphasizeRequiredIntegrations = isFinalStep && !setupReady;
  const canContinue = isTriggerStep
    ? triggerChoicesReady
    : isVcsStep
      ? vcsChoicesReady
      : isAgentStep
        ? agentChoicesReady
        : isSettingsStep
          ? true
          : setupReady;

  useEffect(() => {
    if (!isVcsStep) {
      return;
    }
    if (vcsHostLocked) {
      const lockedHost = triggerVcsHosts[0];
      if (lockedHost && vcsHost !== lockedHost) {
        setVcsHost(lockedHost);
        setDefaultRepoId(null);
      }
      return;
    }
    if (vcsHost && !availableVcsHosts.includes(vcsHost)) {
      setVcsHost(null);
      setDefaultRepoId(null);
    }
  }, [isVcsStep, vcsHostLocked, triggerVcsHosts, availableVcsHosts, vcsHost]);

  const dialogDefinition = useMemo(
    () => (dialogIntegrationName ? availableIntegrations.find((d) => d.name === dialogIntegrationName) : undefined),
    [availableIntegrations, dialogIntegrationName],
  );
  const defaultDialogName = useMemo(
    () => (dialogIntegrationName ? getNextIntegrationName(dialogIntegrationName, new Set()) : ""),
    [dialogIntegrationName],
  );

  const toggleTriggerSource = (id: TriggerSourceId) => {
    const wasSelected = triggerSources.has(id);

    setTriggerSources((prev) => {
      const next = new Set(prev);
      if (wasSelected) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });

    if (id === "issue" && wasSelected) {
      setIssueTracker(null);
    }

    if (id === "prOrMrTag") {
      if (wasSelected) {
        setPrMrProvider(null);
      } else if (issueTracker === "github" || issueTracker === "gitlab") {
        setPrMrProvider(issueTracker);
      }
    }
  };

  const openConnectDialog = (item: RequiredIntegration) => {
    pendingConnectKeyRef.current = item.key;
    setDialogIntegrationName(item.integrationName);
  };

  const selectVcsHost = (id: VcsHostId) => {
    setVcsHost(id);
    setDefaultRepoId(null);
  };

  const rematchComponentModels = (harness: AgentHarnessId | null, provider: OpenCodeProviderId | null) => {
    const models = fixtureModelsForHarness(harness, provider);
    const fallback = models[0]?.value ?? defaultModelForHarness(harness, provider);
    setAgentComponents((prev) =>
      prev.map((component) => ({
        ...component,
        modelId: models.some((model) => model.value === component.modelId) ? component.modelId : fallback,
      })),
    );
  };

  const selectAgentHarness = (id: AgentHarnessId) => {
    setAgentHarness(id);
    setOpenCodeProvider(null);
    setAgentApiKey("");
    rematchComponentModels(id, null);
  };

  const selectOpenCodeProvider = (id: OpenCodeProviderId) => {
    setOpenCodeProvider(id);
    setAgentApiKey("");
    rematchComponentModels(agentHarness, id);
  };

  const handleBack = () => {
    if (stepIndex === 0) {
      onExit();
      return;
    }
    setStepIndex((index) => index - 1);
  };

  const handleContinue = () => {
    if (!canContinue) return;
    if (isFinalStep) {
      goToLiveCanvas(navigate, organizationId);
      return;
    }
    setStepIndex((index) => index + 1);
  };

  return (
    <div className="mx-auto w-full max-w-5xl px-8 py-14">
      <div className="grid items-start gap-8 lg:grid-cols-[1fr_0.95fr]">
        <div className="min-w-0">
          <p className="text-xs font-medium text-gray-400 dark:text-gray-500">
            Step {stepIndex + 1} of {SETUP_STEPS.length}
          </p>
          <h1 className={cn(homePageTitleClassName, "mt-2")}>Software Factory setup</h1>
          <h2 className="mt-6 text-lg font-semibold text-slate-900 dark:text-gray-100">{step.title}</h2>
          <p className={cn(homePageSubtitleClassName, "mt-1")}>{step.detail}</p>

          {isTriggerStep ? (
            <TriggerStepContent
              selectedSources={triggerSources}
              issueTracker={issueTracker}
              prMrProvider={prMrProvider}
              onToggleSource={toggleTriggerSource}
              onSelectIssueTracker={(id) => {
                setIssueTracker(id);
                if (triggerSources.has("prOrMrTag") && prMrProvider === null && (id === "github" || id === "gitlab")) {
                  setPrMrProvider(id);
                }
              }}
              onSelectPrMrProvider={setPrMrProvider}
            />
          ) : isVcsStep ? (
            <VersionControlStepContent
              vcsHost={vcsHost}
              availableHosts={availableVcsHosts}
              hostLocked={vcsHostLocked}
              onSelectHost={selectVcsHost}
            />
          ) : isAgentStep ? (
            <CodingAgentStepContent
              agentHarness={agentHarness}
              openCodeProvider={openCodeProvider}
              onSelectHarness={selectAgentHarness}
              onSelectOpenCodeProvider={selectOpenCodeProvider}
            />
          ) : isSettingsStep ? (
            <AgentSettingsStepContent
              components={agentComponents}
              onChange={setAgentComponents}
              agentHarness={agentHarness}
              openCodeProvider={openCodeProvider}
            />
          ) : (
            <PreviewStepContent
              triggerSources={triggerSources}
              issueTracker={issueTracker}
              prMrProvider={prMrProvider}
              vcsHost={vcsHost}
              defaultRepoId={defaultRepoId}
              agentHarness={agentHarness}
              openCodeProvider={openCodeProvider}
              agentComponents={agentComponents}
            />
          )}

          <div className="mt-10 flex items-center gap-3">
            <Button type="button" variant="outline" onClick={handleBack}>
              <ArrowLeft />
              Back
            </Button>
            <Button type="button" onClick={handleContinue} disabled={!canContinue}>
              {isFinalStep ? "Done" : "Continue"}
              <ArrowRight />
            </Button>
          </div>
        </div>

        <div className="sticky top-8 space-y-3">
          <RequiredIntegrationsPanel
            requiredIntegrations={requiredIntegrations}
            connectedTools={connectedTools}
            onConnect={openConnectDialog}
            emphasize={emphasizeRequiredIntegrations}
            sticky={false}
            emptyHint={
              isTriggerStep
                ? "Select triggers on the left. Matching integrations show up here."
                : "Integrations from earlier steps stay listed here as you continue setup."
            }
            defaultRepoPicker={
              vcsHost
                ? {
                    host: vcsHost,
                    value: defaultRepoId ?? "",
                    onChange: setDefaultRepoId,
                  }
                : undefined
            }
            agentKeyPicker={
              needsOpenCodeApiKey && selectedOpenCodeProvider
                ? {
                    providerLabel: selectedOpenCodeProvider.label,
                    iconName: selectedOpenCodeProvider.iconName,
                    value: agentApiKey,
                    onChange: setAgentApiKey,
                    emphasize: isFinalStep && !openCodeKeyReady,
                  }
                : undefined
            }
            defaultRepoEmphasize={isFinalStep && !defaultRepoReady}
          />
          {emphasizeRequiredIntegrations ? (
            <p
              role="status"
              className="rounded-xl bg-amber-50 px-4 py-3 text-sm font-medium text-amber-950 outline outline-amber-500/40 dark:bg-amber-950/40 dark:text-amber-50 dark:outline-amber-400/40"
            >
              Hey, make sure you connect all the required tools.
            </p>
          ) : null}
        </div>
      </div>

      <IntegrationCreateDialog
        open={!!dialogIntegrationName}
        onOpenChange={(open) => {
          if (!open) {
            setDialogIntegrationName(null);
            createIntegrationMutation.reset();
          }
        }}
        integrationDefinition={dialogDefinition ?? null}
        organizationId={organizationId}
        onCreateIntegration={async (payload) => {
          const res = await createIntegrationMutation.mutateAsync(payload);
          return res.data;
        }}
        onReset={() => createIntegrationMutation.reset()}
        defaultName={defaultDialogName}
        onCreated={() => {
          // Dialog calls onOpenChange(false) before onCreated; keep the key in a ref across that close.
          const key = pendingConnectKeyRef.current;
          pendingConnectKeyRef.current = null;
          if (key) {
            setConnectedTools((prev) => new Set(prev).add(key));
          }
          setDialogIntegrationName(null);
        }}
      />
    </div>
  );
}

function TriggerStepContent({
  selectedSources,
  issueTracker,
  prMrProvider,
  onToggleSource,
  onSelectIssueTracker,
  onSelectPrMrProvider,
}: {
  selectedSources: Set<TriggerSourceId>;
  issueTracker: string | null;
  prMrProvider: string | null;
  onToggleSource: (id: TriggerSourceId) => void;
  onSelectIssueTracker: (id: string) => void;
  onSelectPrMrProvider: (id: string) => void;
}) {
  return (
    <div className="mt-8 space-y-3" role="group" aria-label="Select triggers">
      <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">Select triggers</p>
      {TRIGGER_SOURCES.map((source) => {
        const selected = selectedSources.has(source.id);
        const subsection =
          source.id === "issue" ? (
            <ChoiceGroup
              title="Issue tracker"
              choices={ISSUE_TRACKERS}
              selectedId={issueTracker}
              onSelect={onSelectIssueTracker}
            />
          ) : source.id === "prOrMrTag" ? (
            <ChoiceGroup
              title="Pull request or merge request"
              choices={PR_MR_PROVIDERS}
              selectedId={prMrProvider}
              onSelect={onSelectPrMrProvider}
            />
          ) : null;

        return (
          <div key={source.id} className="space-y-2">
            <button
              type="button"
              aria-pressed={selected}
              aria-expanded={subsection ? selected : undefined}
              onClick={() => onToggleSource(source.id)}
              className={cn(
                "flex w-full flex-col items-start rounded-xl px-4 py-3.5 text-left outline transition-colors",
                selected
                  ? "bg-white outline-slate-900 dark:bg-gray-900 dark:outline-gray-100"
                  : "bg-white outline-slate-950/10 hover:outline-slate-950/20 dark:bg-gray-900 dark:outline-gray-700/70 dark:hover:outline-gray-500",
              )}
            >
              <span className="text-sm font-semibold text-slate-900 dark:text-gray-100">{source.title}</span>
            </button>
            {selected && subsection ? (
              <div className="ml-3 border-l-2 border-slate-200 pl-4 dark:border-gray-700">{subsection}</div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

function VersionControlStepContent({
  vcsHost,
  availableHosts,
  hostLocked,
  onSelectHost,
}: {
  vcsHost: VcsHostId | null;
  availableHosts: VcsHostId[];
  hostLocked: boolean;
  onSelectHost: (id: VcsHostId) => void;
}) {
  if (hostLocked && vcsHost) {
    const host = VCS_HOSTS.find((choice) => choice.id === vcsHost);
    return (
      <div className="mt-8 rounded-xl bg-white px-4 py-4 outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/60">
        <div className="flex items-center gap-2">
          <IntegrationIcon integrationName={vcsHost} className="h-4 w-4" size={16} />
          <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">
            Using {host?.label ?? vcsHost} for version control
          </p>
        </div>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Based on the triggers you selected.</p>
      </div>
    );
  }

  const choices = VCS_HOSTS.filter((choice) => availableHosts.includes(choice.id as VcsHostId));
  return (
    <div className="mt-8">
      <ChoiceGroup
        title="Host"
        choices={choices}
        selectedId={vcsHost}
        onSelect={(id) => onSelectHost(id as VcsHostId)}
      />
    </div>
  );
}

function fixtureRepoOptions(host: VcsHostId): AutoCompleteOption[] {
  return FIXTURE_REPOS[host].map((repo) => ({ value: repo, label: repo }));
}

function OpenCodeProviderGroup({
  title,
  ariaLabel,
  providers,
  openCodeProvider,
  onSelectOpenCodeProvider,
}: {
  title: string;
  ariaLabel: string;
  providers: OpenCodeProvider[];
  openCodeProvider: OpenCodeProviderId | null;
  onSelectOpenCodeProvider: (id: OpenCodeProviderId) => void;
}) {
  return (
    <div>
      <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">{title}</p>
      <div className="mt-3 grid gap-2" role="group" aria-label={ariaLabel}>
        {providers.map((provider) => {
          const active = openCodeProvider === provider.id;
          return (
            <button
              key={provider.id}
              type="button"
              aria-pressed={active}
              onClick={() => onSelectOpenCodeProvider(provider.id)}
              className={cn(
                "flex items-center gap-2 rounded-xl bg-white px-3 py-3 text-left text-sm font-medium outline transition-colors dark:bg-gray-900",
                active
                  ? "text-slate-900 outline-slate-900 dark:text-gray-100 dark:outline-gray-100"
                  : "text-slate-700 outline-slate-950/10 hover:outline-slate-950/20 dark:text-gray-200 dark:outline-gray-700/70",
              )}
            >
              <IntegrationIcon integrationName={provider.iconName} className="h-4 w-4" size={16} />
              <span className="min-w-0 flex-1 truncate">{provider.label}</span>
              <span className="shrink-0 text-xs font-medium text-gray-400 dark:text-gray-500">
                {openCodeProviderModeLabel(provider.mode)}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}

function CodingAgentStepContent({
  agentHarness,
  openCodeProvider,
  onSelectHarness,
  onSelectOpenCodeProvider,
}: {
  agentHarness: AgentHarnessId | null;
  openCodeProvider: OpenCodeProviderId | null;
  onSelectHarness: (id: AgentHarnessId) => void;
  onSelectOpenCodeProvider: (id: OpenCodeProviderId) => void;
}) {
  return (
    <div className="mt-8 space-y-3" role="group" aria-label="Select coding agent harness">
      <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">Harness</p>
      {AGENT_HARNESSES.map((harness) => {
        const selected = agentHarness === harness.id;
        return (
          <div key={harness.id} className="space-y-2">
            <button
              type="button"
              aria-pressed={selected}
              aria-expanded={harness.id === "open-code" ? selected : undefined}
              onClick={() => onSelectHarness(harness.id as AgentHarnessId)}
              className={cn(
                "flex w-full items-center gap-2 rounded-xl bg-white px-4 py-3.5 text-left text-sm font-semibold outline transition-colors",
                "dark:bg-gray-900",
                selected
                  ? "text-slate-900 outline-slate-900 dark:text-gray-100 dark:outline-gray-100"
                  : "text-slate-700 outline-slate-950/10 hover:outline-slate-950/20 dark:text-gray-200 dark:outline-gray-700/70",
              )}
            >
              <IntegrationIcon integrationName={harness.integrationName} className="h-4 w-4" size={16} />
              {harness.label}
            </button>
            {selected && harness.id === "open-code" ? (
              <div className="ml-3 space-y-5 border-l-2 border-slate-200 pl-4 dark:border-gray-700">
                <OpenCodeProviderGroup
                  title="Free / local"
                  ariaLabel="Free or local model provider"
                  providers={OPEN_CODE_FREE_PROVIDERS}
                  openCodeProvider={openCodeProvider}
                  onSelectOpenCodeProvider={onSelectOpenCodeProvider}
                />
                <OpenCodeProviderGroup
                  title="Model provider"
                  ariaLabel="Model provider"
                  providers={OPEN_CODE_CLOUD_PROVIDERS}
                  openCodeProvider={openCodeProvider}
                  onSelectOpenCodeProvider={onSelectOpenCodeProvider}
                />
              </div>
            ) : null}
          </div>
        );
      })}
    </div>
  );
}

function labelForChoice(choices: IntegrationChoice[], id: string | null): string | null {
  if (!id) return null;
  return choices.find((choice) => choice.id === id)?.label ?? id;
}

function labelForOption(options: AutoCompleteOption[], value: string): string {
  return options.find((option) => option.value === value)?.label ?? value;
}

type PreviewTriggerNode = {
  id: TriggerSourceId;
  title: string;
  detail: string;
  integrationName?: string;
};

function previewTriggerNodes(
  triggerSources: Set<TriggerSourceId>,
  issueTracker: string | null,
  prMrProvider: string | null,
): PreviewTriggerNode[] {
  const nodes: PreviewTriggerNode[] = [];
  if (triggerSources.has("manual")) {
    nodes.push({ id: "manual", title: "Manual", detail: "Prompt in SuperPlane" });
  }
  if (triggerSources.has("issue")) {
    const tracker = ISSUE_TRACKERS.find((choice) => choice.id === issueTracker);
    nodes.push({
      id: "issue",
      title: tracker?.label ?? "Issue",
      detail: issueTriggerAssignHint(issueTracker),
      integrationName: tracker?.integrationName,
    });
  }
  if (triggerSources.has("prOrMrTag")) {
    const provider = PR_MR_PROVIDERS.find((choice) => choice.id === prMrProvider);
    nodes.push({
      id: "prOrMrTag",
      title: vcsHostLabelForPr(prMrProvider),
      detail: prMrTriggerMentionHint(prMrProvider),
      integrationName: provider?.integrationName,
    });
  }
  return nodes;
}

function vcsHostLabelForPr(prMrProvider: string | null): string {
  if (prMrProvider === "gitlab") return "MR";
  return "PR";
}

function issueTriggerAssignHint(issueTracker: string | null): string {
  if (issueTracker === "linear") return "Assign @superplane on the Linear issue";
  if (issueTracker === "jira") return "Assign @superplane on the Jira issue";
  if (issueTracker === "gitlab") return "Assign @superplane on the GitLab issue";
  if (issueTracker === "github") return "Assign @superplane on the GitHub issue";
  return "Assign @superplane on the issue";
}

function prMrTriggerMentionHint(prMrProvider: string | null): string {
  if (prMrProvider === "gitlab") return "Mention @superplane on the merge request";
  if (prMrProvider === "github") return "Mention @superplane on the pull request";
  return "Mention @superplane on the PR or MR";
}

function shortMachineLabel(machineType: string): string {
  const label = labelForOption(AGENT_MACHINE_OPTIONS, machineType);
  const head = label.split("·")[0]?.trim();
  return head || label;
}

function WorkflowConnector({ fanIn = false }: { fanIn?: boolean }) {
  if (fanIn) {
    return (
      <div className="relative mx-auto h-7 w-full max-w-sm" aria-hidden>
        <div className="absolute inset-x-[16%] top-0 h-3 rounded-b-xl border-x border-b border-slate-300 dark:border-gray-600" />
        <div className="absolute left-1/2 top-3 h-4 w-px -translate-x-1/2 bg-slate-300 dark:bg-gray-600" />
      </div>
    );
  }
  return (
    <div className="flex justify-center py-1" aria-hidden>
      <div className="h-5 w-px bg-gradient-to-b from-slate-300 to-slate-400 dark:from-gray-600 dark:to-gray-500" />
    </div>
  );
}

function PreviewStepContent({
  triggerSources,
  issueTracker,
  prMrProvider,
  vcsHost,
  defaultRepoId,
  agentHarness,
  openCodeProvider,
  agentComponents,
}: {
  triggerSources: Set<TriggerSourceId>;
  issueTracker: string | null;
  prMrProvider: string | null;
  vcsHost: VcsHostId | null;
  defaultRepoId: string | null;
  agentHarness: AgentHarnessId | null;
  openCodeProvider: OpenCodeProviderId | null;
  agentComponents: AgentComponentConfig[];
}) {
  const harness = AGENT_HARNESSES.find((choice) => choice.id === agentHarness);
  const harnessLabel = harness?.label ?? "Not selected";
  const openCodeLabel = openCodeProviderById(openCodeProvider)?.label;
  const codingAgentDetail =
    agentHarness === "open-code" && openCodeLabel ? `${harnessLabel} · ${openCodeLabel}` : harnessLabel;
  const modelOptions = fixtureModelsForHarness(agentHarness, openCodeProvider);
  const planning = agentComponents.find((component) => component.id === "planning");
  const implementation = agentComponents.find((component) => component.id === "implementation");
  const prLoop = agentComponents.find((component) => component.id === "pr-loop");
  const prNoun = vcsHost === "gitlab" ? "merge request" : "pull request";
  const triggers = previewTriggerNodes(triggerSources, issueTracker, prMrProvider);
  const triggerColumns = triggers.length <= 1 ? "grid-cols-1" : triggers.length === 2 ? "grid-cols-2" : "grid-cols-3";

  const stages: {
    id: string;
    title: string;
    detail: string;
    accent: string;
    icon: ReactNode;
  }[] = [
    {
      id: "planning",
      title: "Planning",
      detail: planning
        ? `${labelForOption(modelOptions, planning.modelId)} · ${shortMachineLabel(planning.machineType)}`
        : "Create an implementation plan",
      accent: "from-sky-500/15 to-transparent",
      icon: <ListTodo className="h-4 w-4" aria-hidden />,
    },
    {
      id: "implementation",
      title: "Implementation",
      detail: implementation
        ? `${labelForOption(modelOptions, implementation.modelId)} · ${implementation.steps.length} steps · opens ${prNoun}`
        : `Codes and opens a ${prNoun}`,
      accent: "from-slate-500/20 to-transparent",
      icon: <Terminal className="h-4 w-4" aria-hidden />,
    },
    {
      id: "pr-loop",
      title: "Review loop",
      detail: prLoop
        ? `${labelForOption(modelOptions, prLoop.modelId)} · max ${prLoop.maxRetries ?? DEFAULT_PR_LOOP_MAX_RETRIES} retries`
        : "Addresses checks and comments",
      accent: "from-orange-500/20 to-transparent",
      icon: <RefreshCw className="h-4 w-4" aria-hidden />,
    },
  ];

  return (
    <div className="mt-8 space-y-4" aria-label="Factory preview">
      <div className="flex flex-wrap gap-2">
        {defaultRepoId ? (
          <span className="inline-flex items-center gap-1.5 rounded-full bg-white px-3 py-1 text-xs font-medium text-slate-700 outline outline-slate-950/10 dark:bg-gray-900 dark:text-gray-200 dark:outline-gray-700/70">
            {vcsHost ? <IntegrationIcon integrationName={vcsHost} className="h-3.5 w-3.5" size={14} /> : null}
            {defaultRepoId}
          </span>
        ) : null}
        <span className="inline-flex items-center gap-1.5 rounded-full bg-white px-3 py-1 text-xs font-medium text-slate-700 outline outline-slate-950/10 dark:bg-gray-900 dark:text-gray-200 dark:outline-gray-700/70">
          {harness ? (
            <IntegrationIcon integrationName={harness.integrationName} className="h-3.5 w-3.5" size={14} />
          ) : null}
          {codingAgentDetail}
        </span>
      </div>

      <div
        className={cn(
          "relative overflow-hidden rounded-2xl px-4 py-5 outline outline-slate-950/10",
          "bg-[radial-gradient(circle_at_top,_rgba(148,163,184,0.18),_transparent_55%),linear-gradient(180deg,#f8fafc_0%,#f1f5f9_100%)]",
          "dark:bg-[radial-gradient(circle_at_top,_rgba(71,85,105,0.35),_transparent_55%),linear-gradient(180deg,#0f172a_0%,#020617_100%)]",
          "dark:outline-gray-700/60",
        )}
      >
        <p className="text-center text-[11px] font-semibold tracking-wide text-slate-500 uppercase dark:text-gray-400">
          Factory workflow
        </p>

        <div className="mt-4">
          {triggers.length === 0 ? (
            <div className="rounded-xl bg-white/80 px-4 py-3 text-center text-sm text-gray-500 outline outline-slate-950/10 dark:bg-gray-900/80 dark:text-gray-400 dark:outline-gray-700/60">
              No triggers selected
            </div>
          ) : (
            <div className={cn("mx-auto grid max-w-md gap-2", triggerColumns)}>
              {triggers.map((trigger) => (
                <div
                  key={trigger.id}
                  className="rounded-xl bg-white/90 px-3 py-3 shadow-sm outline outline-slate-950/10 backdrop-blur-sm dark:bg-gray-900/90 dark:outline-gray-700/70"
                >
                  <div className="flex items-center gap-2">
                    {trigger.integrationName ? (
                      <IntegrationIcon integrationName={trigger.integrationName} className="h-4 w-4" size={16} />
                    ) : (
                      <MessageSquare className="h-4 w-4 text-slate-500" aria-hidden />
                    )}
                    <p className="truncate text-sm font-semibold text-slate-900 dark:text-gray-100">{trigger.title}</p>
                  </div>
                  <p className="mt-1 truncate text-xs text-gray-500 dark:text-gray-400">{trigger.detail}</p>
                </div>
              ))}
            </div>
          )}
        </div>

        {triggers.length > 1 ? <WorkflowConnector fanIn /> : <WorkflowConnector />}

        <div className="mx-auto flex max-w-sm flex-col">
          {stages.map((stage, index) => (
            <div key={stage.id}>
              {index > 0 ? <WorkflowConnector /> : null}
              <div
                className={cn(
                  "relative overflow-hidden rounded-xl bg-white shadow-sm outline outline-slate-950/10",
                  "dark:bg-gray-900 dark:outline-gray-700/70",
                )}
              >
                <div
                  className={cn("pointer-events-none absolute inset-y-0 left-0 w-1.5 bg-gradient-to-b", stage.accent)}
                />
                <div className="flex items-start gap-3 px-4 py-3.5 pl-4">
                  <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-900 text-white dark:bg-gray-100 dark:text-slate-900">
                    {stage.icon}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">{stage.title}</p>
                    <p className="mt-0.5 text-xs leading-snug text-gray-500 dark:text-gray-400">{stage.detail}</p>
                  </div>
                </div>
              </div>
            </div>
          ))}

          <WorkflowConnector />

          <div className="relative overflow-hidden rounded-xl bg-emerald-50 shadow-sm outline outline-emerald-600/20 dark:bg-emerald-950/40 dark:outline-emerald-400/30">
            <div className="flex items-start gap-3 px-4 py-3.5">
              <span className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-emerald-700 text-white dark:bg-emerald-400 dark:text-emerald-950">
                {vcsHost === "gitlab" ? (
                  <GitPullRequest className="h-4 w-4" aria-hidden />
                ) : (
                  <UserRound className="h-4 w-4" aria-hidden />
                )}
              </span>
              <div className="min-w-0 flex-1">
                <p className="text-sm font-semibold text-emerald-950 dark:text-emerald-50">You review and merge</p>
                <p className="mt-0.5 text-xs leading-snug text-emerald-900/70 dark:text-emerald-100/70">
                  Stay in the loop until the {prNoun} is mergeable.
                </p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

function AgentSettingsStepContent({
  components,
  onChange,
  agentHarness,
  openCodeProvider,
}: {
  components: AgentComponentConfig[];
  onChange: (next: AgentComponentConfig[]) => void;
  agentHarness: AgentHarnessId | null;
  openCodeProvider: OpenCodeProviderId | null;
}) {
  const modelOptions = useMemo(
    () => fixtureModelsForHarness(agentHarness, openCodeProvider),
    [agentHarness, openCodeProvider],
  );

  const updateComponent = (
    componentId: AgentComponentId,
    updater: (component: AgentComponentConfig) => AgentComponentConfig,
  ) => {
    onChange(components.map((component) => (component.id === componentId ? updater(component) : component)));
  };

  const updateStep = (
    componentId: AgentComponentId,
    stepId: string,
    patch: Partial<Pick<AgentPipelineStep, "title" | "body" | "kind">>,
  ) => {
    updateComponent(componentId, (component) => ({
      ...component,
      steps: component.steps.map((step) => (step.id === stepId ? { ...step, ...patch } : step)),
    }));
  };

  const removeStep = (componentId: AgentComponentId, stepId: string) => {
    updateComponent(componentId, (component) => ({
      ...component,
      steps: component.steps.filter((step) => step.id !== stepId),
    }));
  };

  const addStep = (componentId: AgentComponentId, kind: AgentStepKind) => {
    updateComponent(componentId, (component) => ({
      ...component,
      steps: [
        ...component.steps,
        createAgentStep(
          kind,
          kind === "prompt" ? "New prompt" : "New bash step",
          kind === "prompt" ? "Describe what the agent should do in this step." : "echo 'Add your command here'",
        ),
      ],
    }));
  };

  return (
    <div className="mt-8 space-y-8" aria-label="Agent component settings">
      {components.map((component) => (
        <section key={component.id} aria-labelledby={`agent-component-${component.id}`}>
          <h3
            id={`agent-component-${component.id}`}
            className="text-sm font-semibold text-slate-900 dark:text-gray-100"
          >
            {component.title}
          </h3>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{component.purpose}</p>

          <div className={cn("mt-3 grid gap-3", component.id === "pr-loop" ? "sm:grid-cols-3" : "sm:grid-cols-2")}>
            <div>
              <Label
                htmlFor={`agent-model-${component.id}`}
                className="text-xs font-medium text-gray-500 dark:text-gray-400"
              >
                Model
              </Label>
              <div className="mt-1.5" id={`agent-model-${component.id}`}>
                <AutoCompleteSelect
                  options={modelOptions}
                  value={component.modelId}
                  onChange={(value) => updateComponent(component.id, (current) => ({ ...current, modelId: value }))}
                  placeholder="Select model"
                />
              </div>
            </div>
            <div>
              <Label
                htmlFor={`agent-machine-${component.id}`}
                className="text-xs font-medium text-gray-500 dark:text-gray-400"
              >
                Machine
              </Label>
              <div className="mt-1.5" id={`agent-machine-${component.id}`}>
                <AutoCompleteSelect
                  options={AGENT_MACHINE_OPTIONS}
                  value={component.machineType}
                  onChange={(value) => updateComponent(component.id, (current) => ({ ...current, machineType: value }))}
                  placeholder="Select machine"
                />
              </div>
            </div>
            {component.id === "pr-loop" ? (
              <div>
                <Label
                  htmlFor="agent-pr-loop-max-retries"
                  className="text-xs font-medium text-gray-500 dark:text-gray-400"
                >
                  Max retries
                </Label>
                <Input
                  id="agent-pr-loop-max-retries"
                  type="number"
                  min={1}
                  max={20}
                  value={component.maxRetries ?? DEFAULT_PR_LOOP_MAX_RETRIES}
                  onChange={(event) => {
                    const next = Number.parseInt(event.target.value, 10);
                    updateComponent(component.id, (current) => ({
                      ...current,
                      maxRetries: Number.isFinite(next) ? Math.min(20, Math.max(1, next)) : DEFAULT_PR_LOOP_MAX_RETRIES,
                    }));
                  }}
                  className="mt-1.5"
                  aria-label="PR review loop max retries"
                />
              </div>
            ) : null}
          </div>

          <div className="mt-4 space-y-3">
            {component.steps.map((step, index) => (
              <div
                key={step.id}
                className="rounded-xl bg-white px-4 py-3.5 outline outline-slate-950/10 dark:bg-gray-900 dark:outline-gray-700/70"
              >
                <div className="flex items-center gap-2">
                  <span
                    className={cn(
                      "inline-flex items-center gap-1 rounded-md px-2 py-0.5 text-[11px] font-semibold",
                      step.kind === "prompt"
                        ? "bg-sky-50 text-sky-800 dark:bg-sky-950/50 dark:text-sky-200"
                        : "bg-orange-50 text-orange-800 dark:bg-orange-950/40 dark:text-orange-200",
                    )}
                  >
                    {step.kind === "prompt" ? (
                      <MessageSquare className="h-3 w-3" aria-hidden />
                    ) : (
                      <Terminal className="h-3 w-3" aria-hidden />
                    )}
                    {step.kind === "prompt" ? "Prompt" : "Bash"}
                  </span>
                  <Input
                    aria-label={`${component.title} step ${index + 1} title`}
                    value={step.title}
                    onChange={(event) => updateStep(component.id, step.id, { title: event.target.value })}
                    className="h-8 flex-1 border-0 bg-transparent px-1 shadow-none focus-visible:ring-0"
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    aria-label={`Remove ${step.title || "step"}`}
                    onClick={() => removeStep(component.id, step.id)}
                    disabled={component.steps.length <= 1}
                    className="shrink-0 text-gray-400 hover:text-slate-900 dark:hover:text-gray-100"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </div>
                <Textarea
                  aria-label={`${component.title} step ${index + 1} body`}
                  value={step.body}
                  onChange={(event) => updateStep(component.id, step.id, { body: event.target.value })}
                  rows={step.kind === "bash" ? 2 : 3}
                  className="mt-2 resize-y text-sm"
                />
              </div>
            ))}
          </div>

          <div className="mt-3 flex flex-wrap gap-2">
            <Button type="button" variant="outline" size="sm" onClick={() => addStep(component.id, "prompt")}>
              <Plus className="h-3.5 w-3.5" />
              Add prompt
            </Button>
            <Button type="button" variant="outline" size="sm" onClick={() => addStep(component.id, "bash")}>
              <Terminal className="h-3.5 w-3.5" />
              Add bash
            </Button>
          </div>
        </section>
      ))}
    </div>
  );
}

function RequiredIntegrationsPanel({
  requiredIntegrations,
  connectedTools,
  onConnect,
  emptyHint,
  emphasize = false,
  sticky = true,
  defaultRepoPicker,
  defaultRepoEmphasize = false,
  agentKeyPicker,
}: {
  requiredIntegrations: RequiredIntegration[];
  connectedTools: Set<string>;
  onConnect: (item: RequiredIntegration) => void;
  emptyHint: string;
  emphasize?: boolean;
  sticky?: boolean;
  defaultRepoPicker?: {
    host: VcsHostId | null;
    value: string;
    onChange: (value: string) => void;
  };
  defaultRepoEmphasize?: boolean;
  agentKeyPicker?: {
    providerLabel: string;
    iconName: string;
    value: string;
    onChange: (value: string) => void;
    emphasize?: boolean;
  };
}) {
  return (
    <aside
      data-emphasize={emphasize ? "true" : undefined}
      className={cn(
        "rounded-xl bg-slate-50 px-5 py-5 outline transition-[outline-color,box-shadow]",
        sticky && "sticky top-8",
        emphasize
          ? "outline-2 outline-amber-500 shadow-[0_0_0_4px_rgba(245,158,11,0.15)] dark:outline-amber-400"
          : "outline-slate-950/10 dark:outline-gray-700/60",
        "dark:bg-gray-950/40",
      )}
    >
      <h3 className="text-sm font-semibold text-slate-900 dark:text-gray-100">Required integrations</h3>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {emphasize
          ? "Connect these before finishing setup."
          : "Helper list of integrations to connect for this factory."}
      </p>

      {requiredIntegrations.length === 0 ? (
        <p className="mt-6 text-sm text-gray-500 dark:text-gray-400">{emptyHint}</p>
      ) : (
        <ul className="mt-5 space-y-3">
          {requiredIntegrations.map((item) => {
            const isConnected = connectedTools.has(item.key);
            const highlightRow = emphasize && !isConnected;
            return (
              <li
                key={item.key}
                className={cn(
                  "flex items-center justify-between gap-3 rounded-lg bg-white px-3 py-2.5 outline dark:bg-gray-900",
                  highlightRow
                    ? "outline-2 outline-amber-500 dark:outline-amber-400"
                    : "outline-slate-950/10 dark:outline-gray-700/70",
                )}
              >
                <div className="flex min-w-0 items-center gap-2">
                  <IntegrationIcon integrationName={item.integrationName} className="h-4 w-4" size={16} />
                  <span className="truncate text-sm font-medium text-slate-900 dark:text-gray-100">{item.label}</span>
                  <span
                    className={cn(
                      "text-xs font-medium",
                      isConnected ? "text-emerald-700 dark:text-emerald-300" : "text-gray-400 dark:text-gray-500",
                    )}
                  >
                    {isConnected ? "Connected" : "Not connected"}
                  </span>
                </div>
                {!isConnected && (
                  <Button type="button" size="sm" onClick={() => onConnect(item)}>
                    Connect
                  </Button>
                )}
              </li>
            );
          })}
        </ul>
      )}

      {defaultRepoPicker ? (
        <div
          className={cn(
            "mt-6 border-t pt-5",
            defaultRepoEmphasize ? "border-amber-400 dark:border-amber-400" : "border-slate-200 dark:border-gray-700",
          )}
        >
          <Label
            htmlFor="factory-default-repository"
            className="text-sm font-semibold text-slate-900 dark:text-gray-100"
          >
            Default repository
          </Label>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {defaultRepoEmphasize
              ? "Select a repository before finishing setup."
              : "Where the factory checks out code and opens pull or merge requests."}
          </p>
          <div className="mt-3" id="factory-default-repository">
            {defaultRepoPicker.host ? (
              <AutoCompleteSelect
                options={fixtureRepoOptions(defaultRepoPicker.host)}
                value={defaultRepoPicker.value}
                onChange={defaultRepoPicker.onChange}
                placeholder="Select repository"
              />
            ) : (
              <AutoCompleteSelect
                options={[]}
                value=""
                onChange={() => undefined}
                placeholder="Select a host to load repositories"
                disabled
              />
            )}
          </div>
        </div>
      ) : null}

      {agentKeyPicker ? (
        <div
          className={cn(
            "mt-6 border-t pt-5",
            agentKeyPicker.emphasize
              ? "border-amber-400 dark:border-amber-400"
              : "border-slate-200 dark:border-gray-700",
          )}
        >
          <Label
            htmlFor="factory-agent-api-key"
            className="flex items-center gap-2 text-sm font-semibold text-slate-900 dark:text-gray-100"
          >
            <IntegrationIcon integrationName={agentKeyPicker.iconName} className="h-4 w-4" size={16} />
            {agentKeyPicker.providerLabel} API key
          </Label>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {agentKeyPicker.emphasize
              ? "Enter an API key before finishing setup."
              : agentKeyPicker.providerLabel === "OpenCode Zen"
                ? "Zen includes free models. Paste your OpenCode Zen API key."
                : "Open Code will use this key to call models from this provider."}
          </p>
          <div className="mt-3">
            <Input
              id="factory-agent-api-key"
              type="password"
              autoComplete="off"
              placeholder={`Enter ${agentKeyPicker.providerLabel} API key`}
              value={agentKeyPicker.value}
              onChange={(event) => agentKeyPicker.onChange(event.target.value)}
            />
          </div>
        </div>
      ) : null}
    </aside>
  );
}

function ChoiceGroup({
  title,
  choices,
  selectedId,
  onSelect,
}: {
  title: string;
  choices: IntegrationChoice[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  return (
    <div>
      <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">{title}</p>
      <div className="mt-3 grid gap-2" role="group" aria-label={title}>
        {choices.map((choice) => {
          const active = selectedId === choice.id;
          return (
            <button
              key={choice.id}
              type="button"
              aria-pressed={active}
              onClick={() => onSelect(choice.id)}
              className={cn(
                "flex items-center gap-2 rounded-xl bg-white px-3 py-3 text-left text-sm font-medium outline transition-colors",
                "dark:bg-gray-900",
                active
                  ? "text-slate-900 outline-slate-900 dark:text-gray-100 dark:outline-gray-100"
                  : "text-slate-700 outline-slate-950/10 hover:outline-slate-950/20 dark:text-gray-200 dark:outline-gray-700/70",
              )}
            >
              <IntegrationIcon integrationName={choice.integrationName} className="h-4 w-4" size={16} />
              {choice.label}
            </button>
          );
        })}
      </div>
    </div>
  );
}

function StarterAppListItem({
  app,
  busy,
  isInstalling,
  onInstall,
  onSelect,
  onCloseInstall,
}: {
  app: AppEntry;
  busy: boolean;
  isInstalling?: boolean;
  onInstall: (app: AppEntry) => void;
  onSelect: (app: AppEntry) => void;
  onCloseInstall: () => void;
}) {
  return (
    <>
      <div onClick={() => onSelect(app)} className={cn("cursor-pointer px-3 py-2.5", homeListCardClassName)}>
        <div className="flex min-h-[30px] items-center justify-between gap-3">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <div className="shrink-0">
              <LeadIcon icon={app.icon} integrations={app.integrations} size="lg" />
            </div>
            <p className={cn(homeCardTitleClassName, "text-sm")}>{app.title}</p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="icon-sm"
              onClick={(e) => {
                e.stopPropagation();
                onSelect(app);
              }}
              aria-label={`Preview ${app.title}`}
            >
              <Eye className="h-4 w-4" />
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onInstall(app);
              }}
              disabled={busy}
            >
              Install
              <ArrowRight />
            </Button>
          </div>
        </div>
      </div>
      {isInstalling && <InstallProgressPanel app={app} onClose={onCloseInstall} />}
    </>
  );
}
