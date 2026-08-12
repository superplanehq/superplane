import { generateWorkspaceName } from "@/lib/workspaceNameGenerator";
import { useCallback, useMemo, useState } from "react";

import {
  fixtureIssueCount,
  type AgentHarnessId,
  type IntegrationId,
  type IssuesChoiceId,
  type VcsHostId,
} from "./redesignFixtures";

export type RedesignSetupState = {
  workspaceName: string;
  inviteCopied: boolean;
  connected: Set<IntegrationId>;
  vcsHost: VcsHostId | null;
  selectedRepo: string | null;
  issuesRepo: string | null;
  issuesDiscovering: boolean;
  issuesDiscovered: boolean;
  issuesChoice: IssuesChoiceId | null;
  agent: AgentHarnessId | null;
  finished: boolean;
};

export function useRedesignSetupState(initialName = "") {
  const [workspaceName, setWorkspaceName] = useState(() => initialName.trim() || generateWorkspaceName());
  const [inviteCopied, setInviteCopied] = useState(false);
  const [connected, setConnected] = useState<Set<IntegrationId>>(() => new Set());
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
  const [finished, setFinished] = useState(false);

  const connectIntegration = useCallback((id: IntegrationId) => {
    setConnected((prev) => new Set(prev).add(id));
  }, []);

  const resetIssuesState = useCallback(() => {
    setIssuesRepo(null);
    setIssuesDiscovering(false);
    setIssuesDiscovered(false);
    setIssuesChoice(null);
    setIssuesCommitted(false);
  }, []);

  const selectVcsHost = useCallback(
    (host: VcsHostId) => {
      setVcsHost(host);
      setSelectedRepo(null);
      setRepoCommitted(false);
      resetIssuesState();
    },
    [resetIssuesState],
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

  const runIssuesDiscovery = useCallback((backlogRepo: string) => {
    setIssuesRepo(backlogRepo);
    setIssuesDiscovering(true);
    setIssuesDiscovered(false);
    window.setTimeout(() => {
      setIssuesDiscovering(false);
      setIssuesDiscovered(true);
      // Default to the connected host's Issues (GitHub or GitLab).
      setIssuesChoice((current) => current ?? "vcs");
    }, 900);
  }, []);

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
  const issueCount = backlogRepo ? fixtureIssueCount(backlogRepo) : 0;

  const nameReady = workspaceName.trim().length > 0;
  const repoReady = vcsHost !== null && connected.has(vcsHost) && selectedRepo !== null;
  const issuesReady =
    issuesChoice === "skip" ||
    issuesChoice === "vcs" ||
    (issuesChoice === "linear" && connected.has("linear")) ||
    (issuesChoice === "jira" && connected.has("jira"));
  const agentReady =
    agent !== null &&
    ((agent === "claude-code" && connected.has("claude")) ||
      (agent === "cursor" && connected.has("cursor")) ||
      (agent === "codex" && connected.has("openai")));

  const canFinish = nameReady && repoReady && agentReady;

  const summary = useMemo(
    () => ({
      workspaceName: workspaceName.trim(),
      vcsHost,
      selectedRepo,
      issuesRepo: backlogRepo,
      issuesChoice,
      agent,
      issueCount,
    }),
    [workspaceName, vcsHost, selectedRepo, backlogRepo, issuesChoice, agent, issueCount],
  );

  return {
    workspaceName,
    setWorkspaceName,
    inviteCopied,
    setInviteCopied,
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
    finished,
    setFinished,
    issueCount,
    nameReady,
    repoReady,
    issuesReady,
    agentReady,
    canFinish,
    summary,
  };
}

export type RedesignSetupApi = ReturnType<typeof useRedesignSetupState>;
