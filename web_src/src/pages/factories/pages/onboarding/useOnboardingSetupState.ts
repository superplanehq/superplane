import { useCallback, useEffect, useMemo, useRef, useState } from "react";

import {
  START_WORK_ORDER,
  fixtureIssueCount,
  type AgentHarnessId,
  type IntegrationId,
  type IssuesChoiceId,
  type VcsHostId,
} from "./onboardingFixtures";

export type OnboardingSetupState = {
  workspaceName: string;
  connected: Set<IntegrationId>;
  vcsHost: VcsHostId | null;
  selectedRepo: string | null;
  issuesRepo: string | null;
  issuesDiscovering: boolean;
  issuesDiscovered: boolean;
  issuesChoice: IssuesChoiceId | null;
  agent: AgentHarnessId | null;
  workOrderTitle: string;
  workOrderDescription: string;
  finished: boolean;
};

function isIssuesReady(issuesChoice: IssuesChoiceId | null, connected: Set<IntegrationId>): boolean {
  if (issuesChoice === "skip" || issuesChoice === "vcs") {
    return true;
  }
  if (issuesChoice === "linear") {
    return connected.has("linear");
  }
  if (issuesChoice === "jira") {
    return connected.has("jira");
  }
  return false;
}

function isAgentReady(agent: AgentHarnessId | null, connected: Set<IntegrationId>): boolean {
  if (agent === "claude-code") {
    return connected.has("claude");
  }
  if (agent === "cursor") {
    return connected.has("cursor");
  }
  if (agent === "codex") {
    return connected.has("openai");
  }
  return false;
}

function setupReadiness(input: {
  workspaceName: string;
  vcsHost: VcsHostId | null;
  selectedRepo: string | null;
  connected: Set<IntegrationId>;
  issuesChoice: IssuesChoiceId | null;
  agent: AgentHarnessId | null;
  workOrderTitle: string;
  workOrderDescription: string;
}) {
  const nameReady = input.workspaceName.trim().length > 0;
  const vcsReady = input.vcsHost !== null && input.connected.has(input.vcsHost);
  const repoReady = vcsReady && input.selectedRepo !== null;
  const issuesReady = isIssuesReady(input.issuesChoice, input.connected);
  const agentReady = isAgentReady(input.agent, input.connected);
  const startReady = input.workOrderTitle.trim().length > 0 && input.workOrderDescription.trim().length > 0;
  return {
    nameReady,
    vcsReady,
    repoReady,
    issuesReady,
    agentReady,
    startReady,
    canFinish: nameReady && repoReady && agentReady && startReady,
  };
}

export function useOnboardingSetupState(
  initialName = "",
  options?: {
    connected?: Set<IntegrationId>;
    simulateDiscovery?: boolean;
  },
) {
  const [workspaceName, setWorkspaceName] = useState(() => initialName.trim());
  const [workspaceNameEdited, setWorkspaceNameEdited] = useState(false);
  const [localConnected, setLocalConnected] = useState<Set<IntegrationId>>(() => new Set());
  const [vcsHost, setVcsHost] = useState<VcsHostId | null>(null);
  const [selectedRepo, setSelectedRepo] = useState<string | null>(null);
  /** True after Continue to issues — starts repository analysis. */
  const [repoCommitted, setRepoCommitted] = useState(false);
  /** Backlog repository for GitHub/GitLab Issues (may differ from the app repo). */
  const [issuesRepo, setIssuesRepo] = useState<string | null>(null);
  const [issuesDiscovering, setIssuesDiscovering] = useState(false);
  const [issuesDiscovered, setIssuesDiscovered] = useState(false);
  const [issuesChoice, setIssuesChoice] = useState<IssuesChoiceId | null>(null);
  /** True after Continue to coding agent — starts backlog analysis. */
  const [issuesCommitted, setIssuesCommitted] = useState(false);
  const [agent, setAgent] = useState<AgentHarnessId | null>(null);
  const [workOrderTitle, setWorkOrderTitle] = useState<string>(START_WORK_ORDER.title);
  const [workOrderDescription, setWorkOrderDescription] = useState<string>(START_WORK_ORDER.description);
  const [finished, setFinished] = useState(false);
  const discoveryTimerRef = useRef<number | null>(null);
  const connected = options?.connected ?? localConnected;

  const clearDiscoveryTimer = useCallback(() => {
    if (discoveryTimerRef.current === null) {
      return;
    }
    window.clearTimeout(discoveryTimerRef.current);
    discoveryTimerRef.current = null;
  }, []);

  useEffect(() => clearDiscoveryTimer, [clearDiscoveryTimer]);

  const connectIntegration = useCallback((id: IntegrationId) => {
    setLocalConnected((prev) => new Set(prev).add(id));
  }, []);

  const editWorkspaceName = useCallback((name: string) => {
    setWorkspaceName(name);
    setWorkspaceNameEdited(true);
  }, []);

  /** Derived names never overwrite a name the user typed. */
  const suggestWorkspaceName = useCallback(
    (name: string) => {
      if (workspaceNameEdited) return;
      setWorkspaceName(name);
    },
    [workspaceNameEdited],
  );

  const resetIssuesState = useCallback(() => {
    clearDiscoveryTimer();
    setIssuesRepo(null);
    setIssuesDiscovering(false);
    setIssuesDiscovered(false);
    setIssuesChoice(null);
    setIssuesCommitted(false);
  }, [clearDiscoveryTimer]);

  const selectVcsHost = useCallback(
    (host: VcsHostId) => {
      // Re-clicking the active host must not clear repo / issues selections.
      if (host === vcsHost) return;
      setVcsHost(host);
      setSelectedRepo(null);
      setRepoCommitted(false);
      resetIssuesState();
    },
    [vcsHost, resetIssuesState],
  );

  const selectRepo = useCallback(
    (repo: string) => {
      setSelectedRepo(repo);
      setRepoCommitted(false);
      resetIssuesState();
    },
    [resetIssuesState],
  );

  const commitRepoStep = useCallback(() => {
    setRepoCommitted(true);
  }, []);

  const commitIssuesStep = useCallback(() => {
    setIssuesCommitted(true);
  }, []);

  const runIssuesDiscovery = useCallback(
    (backlogRepo: string) => {
      clearDiscoveryTimer();
      setIssuesRepo(backlogRepo);
      if (options?.simulateDiscovery === false) {
        setIssuesDiscovering(false);
        setIssuesDiscovered(true);
        setIssuesChoice((current) => current ?? "vcs");
        return;
      }

      setIssuesDiscovering(true);
      setIssuesDiscovered(false);
      discoveryTimerRef.current = window.setTimeout(() => {
        discoveryTimerRef.current = null;
        setIssuesDiscovering(false);
        setIssuesDiscovered(true);
        // Default to the connected host's Issues (GitHub or GitLab).
        setIssuesChoice((current) => current ?? "vcs");
      }, 900);
    },
    [clearDiscoveryTimer, options?.simulateDiscovery],
  );

  const startIssuesDiscovery = useCallback(() => {
    const backlogRepo = issuesRepo ?? selectedRepo;
    if (!backlogRepo) return;
    runIssuesDiscovery(backlogRepo);
  }, [issuesRepo, selectedRepo, runIssuesDiscovery]);

  const selectIssuesRepo = useCallback(
    (repo: string) => {
      setIssuesCommitted(false);
      runIssuesDiscovery(repo);
    },
    [runIssuesDiscovery],
  );

  const backlogRepo = issuesRepo ?? selectedRepo;
  const issueCount =
    options?.simulateDiscovery === false ? undefined : backlogRepo ? fixtureIssueCount(backlogRepo) : 0;
  const { nameReady, vcsReady, repoReady, issuesReady, agentReady, startReady, canFinish } = setupReadiness({
    workspaceName,
    vcsHost,
    selectedRepo,
    connected,
    issuesChoice,
    agent,
    workOrderTitle,
    workOrderDescription,
  });

  const summary = useMemo(
    () => ({
      workspaceName: workspaceName.trim(),
      vcsHost,
      selectedRepo,
      issuesRepo: backlogRepo,
      issuesChoice,
      agent,
      issueCount,
      workOrderTitle: workOrderTitle.trim(),
    }),
    [workspaceName, vcsHost, selectedRepo, backlogRepo, issuesChoice, agent, issueCount, workOrderTitle],
  );

  return {
    workspaceName,
    editWorkspaceName,
    suggestWorkspaceName,
    connected,
    connectIntegration,
    vcsHost,
    selectVcsHost,
    selectedRepo,
    selectRepo,
    repoCommitted,
    commitRepoStep,
    issuesRepo: backlogRepo,
    selectIssuesRepo,
    issuesDiscovering,
    issuesDiscovered,
    startIssuesDiscovery,
    issuesChoice,
    setIssuesChoice,
    issuesCommitted,
    commitIssuesStep,
    agent,
    setAgent,
    workOrderTitle,
    setWorkOrderTitle,
    workOrderDescription,
    setWorkOrderDescription,
    finished,
    setFinished,
    issueCount,
    nameReady,
    vcsReady,
    repoReady,
    issuesReady,
    agentReady,
    startReady,
    canFinish,
    summary,
  };
}

export type OnboardingSetupApi = ReturnType<typeof useOnboardingSetupState>;
