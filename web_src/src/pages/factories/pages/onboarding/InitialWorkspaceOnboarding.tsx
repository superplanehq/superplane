import { useEffect, useRef, useState } from "react";
import { useLocation } from "react-router";

import type { FactoriesFactory } from "@/api-client";
import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { useAccount } from "@/contexts/useAccount";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactories, useFactory } from "@/hooks/useFactoryData";
import { useRecordLastLocation } from "@/hooks/useRecordLastLocation";
import { recordLastVisitedOrganization } from "@/lib/lastVisitedOrganization";

import { resolveFactoryByKey, type FactoryResolutionStatus } from "../../lib/factoryKeyResolution";
import { useFactoriesThemeClass } from "../../lib/useFactoriesThemeClass";
import { FactoriesLayoutError, FactoriesLayoutLoading } from "../../layout/FactoriesLayout";
import { FactoriesLayoutContext } from "../../layout/factoriesLayoutContext";
import {
  decideInitialWorkspaceOnboarding,
  type InitialWorkspaceOnboardingDecision,
  type InitialWorkspaceLookup,
} from "./initialWorkspaceOnboardingScreen";
import { OnboardingPage } from "./OnboardingPage";
import { onboardingResumePath } from "./onboardingResumePath";
import type { OnboardingWorkspaceResolution } from "./onboardingWorkspaceResolutionContext";
import { useOnboardingWorkspaceResolution } from "./useOnboardingWorkspaceResolution";

const ignoreOpenCreateWorkOrder = () => undefined;

type OpenWorkspace = {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
};

type RefetchableQuery = {
  refetch: () => Promise<unknown>;
};

function lookupFromQueries(
  describeId: string,
  list: { isError: boolean },
  resolution: FactoryResolutionStatus,
  described: { data?: FactoriesFactory; isError: boolean },
  workspaceOpen: boolean,
): Omit<InitialWorkspaceLookup, "reresolveRunning" | "reresolveAttempted"> {
  return {
    listError: list.isError,
    resolution,
    describeError: Boolean(describeId) && described.isError,
    describeReady: Boolean(described.data) && !described.isError,
    workspaceOpen,
  };
}

function refetchInitialWorkspaceLookup(describeId: string, list: RefetchableQuery, described: RefetchableQuery) {
  const tasks = [list.refetch()];
  if (describeId) tasks.push(described.refetch());
  return Promise.all(tasks).then(() => undefined);
}

function rememberOpenWorkspace(
  openWorkspace: { current: OpenWorkspace | null },
  phase: InitialWorkspaceOnboardingDecision["phase"],
  next: {
    organizationId: string;
    factoryKey: string;
    factoryId?: string;
    factory?: FactoriesFactory;
    factories?: FactoriesFactory[];
  },
) {
  if (phase !== "ready" || !next.factoryId || !next.factory) return;
  openWorkspace.current = {
    organizationId: next.organizationId,
    factoryId: next.factoryId,
    factoryKey: next.factoryKey,
    factory: next.factory,
    factories: next.factories ?? [],
  };
}

/** Renders the existing workspace wizard while the provisional organization stays outside the browser URL. */
export function InitialWorkspaceOnboarding({
  organizationId,
  factoryKey,
}: {
  organizationId: string;
  factoryKey: string;
}) {
  useFactoriesThemeClass();
  const screen = useInitialWorkspaceScreen(organizationId, factoryKey);
  if (screen.view === "error") {
    return <FactoriesLayoutError organizationId={organizationId} />;
  }
  if (!screen.factory?.id) {
    return <FactoriesLayoutLoading />;
  }

  return (
    <PermissionsProvider organizationId={organizationId}>
      <ResolvedInitialWorkspaceOnboarding
        organizationId={organizationId}
        factoryId={screen.factory.id}
        factoryKey={screen.factoryKey}
        factory={screen.factory}
        factories={screen.factories}
      />
    </PermissionsProvider>
  );
}

type InitialWorkspaceScreen = {
  view: InitialWorkspaceOnboardingDecision["view"];
  factory: FactoriesFactory | undefined;
  factoryKey: string;
  factories: FactoriesFactory[];
};

function useInitialWorkspaceScreen(organizationId: string, factoryKey: string): InitialWorkspaceScreen {
  const factoriesQuery = useFactories(organizationId);
  const resolution = resolveFactoryByKey(
    factoriesQuery.data ?? [],
    factoryKey,
    factoriesQuery.isLoading || factoriesQuery.isFetching,
  );
  const openWorkspace = useRef<OpenWorkspace | null>(null);
  const openHere = openWorkspaceFor(openWorkspace.current, organizationId, factoryKey);
  const describeId = resolution.factory?.id ?? openHere?.factoryId ?? "";
  const factoryQuery = useFactory(organizationId, describeId);
  const decision = useInitialWorkspaceLookupRecovery(
    lookupFromQueries(describeId, factoriesQuery, resolution.status, factoryQuery, Boolean(openHere)),
    () => refetchInitialWorkspaceLookup(describeId, factoriesQuery, factoryQuery),
  );
  const factory = factoryQuery.data ?? openHere?.factory;
  rememberOpenWorkspace(openWorkspace, decision.phase, {
    organizationId,
    factoryKey,
    factoryId: resolution.factory?.id,
    factory: factoryQuery.data,
    factories: factoriesQuery.data,
  });

  return {
    view: decision.view,
    factory: decision.view === "setup" ? factory : undefined,
    factoryKey: resolution.factory?.key ?? openHere?.factoryKey ?? factoryKey,
    factories: factoriesForOpenWorkspace(factoriesQuery.data, openHere),
  };
}

function useInitialWorkspaceLookupRecovery(
  lookup: Omit<InitialWorkspaceLookup, "reresolveRunning" | "reresolveAttempted">,
  refetch: () => Promise<unknown>,
): InitialWorkspaceOnboardingDecision {
  const reresolve = useOnboardingWorkspaceResolution();
  const reresolveRef = useRef<OnboardingWorkspaceResolution | null>(reresolve);
  const refetchRef = useRef(refetch);
  reresolveRef.current = reresolve;
  refetchRef.current = refetch;
  const [running, setRunning] = useState(false);
  const [attempted, setAttempted] = useState(false);
  const started = useRef(false);
  const attempt = useRef(0);
  const decision = decideInitialWorkspaceOnboarding({
    ...lookup,
    reresolveRunning: running,
    reresolveAttempted: attempted,
  });

  useEffect(() => {
    if (decision.phase !== "ready") return;
    attempt.current += 1;
    started.current = false;
    setRunning(false);
    setAttempted(false);
  }, [decision.phase]);

  useEffect(() => {
    if (decision.phase !== "reresolve" || started.current) return;
    started.current = true;
    const current = attempt.current;
    setRunning(true);
    void recoverInitialWorkspaceLookup(reresolveRef.current, () => refetchRef.current()).finally(() => {
      if (attempt.current !== current) return;
      setRunning(false);
      setAttempted(true);
    });
  }, [decision.phase]);

  return decision;
}

async function recoverInitialWorkspaceLookup(
  reresolve: OnboardingWorkspaceResolution | null,
  refetch: () => Promise<unknown>,
) {
  await reresolve?.().catch(() => undefined);
  await refetch().catch(() => undefined);
}

function openWorkspaceFor(
  open: OpenWorkspace | null,
  organizationId: string,
  factoryKey: string,
): OpenWorkspace | null {
  if (!open || open.organizationId !== organizationId || open.factoryKey !== factoryKey) return null;
  return open;
}

function factoriesForOpenWorkspace(
  listed: FactoriesFactory[] | undefined,
  open: OpenWorkspace | null,
): FactoriesFactory[] {
  if (!open) return listed ?? [];
  if (listed?.some((factory) => factory.id === open.factoryId)) return listed;
  return open.factories;
}

function ResolvedInitialWorkspaceOnboarding({
  organizationId,
  factoryId,
  factoryKey,
  factory,
  factories,
}: {
  organizationId: string;
  factoryId: string;
  factoryKey: string;
  factory: FactoriesFactory;
  factories: FactoriesFactory[];
}) {
  const permissions = usePermissions();
  if (permissions.isLoading) {
    return <FactoriesLayoutLoading />;
  }

  return (
    <FactoriesLayoutContext.Provider
      value={{
        organizationId,
        factoryId,
        factoryKey,
        factory,
        factories,
        openCreateWorkOrder: ignoreOpenCreateWorkOrder,
      }}
    >
      <RecordOnboardingLastLocation organizationSlug={organizationId} factoryKey={factoryKey} />
      <main className="h-dvh overflow-y-auto bg-background">
        <OnboardingPage />
      </main>
    </FactoriesLayoutContext.Provider>
  );
}

function RecordOnboardingLastLocation({
  organizationSlug,
  factoryKey,
}: {
  organizationSlug: string;
  factoryKey: string;
}) {
  const { account } = useAccount();
  const { search } = useLocation();
  const { canAct, isLoading } = usePermissions();
  const resumePath = onboardingResumePath(organizationSlug, factoryKey, search);
  const canRecord = !isLoading && (canAct("factories", "read") || canAct("canvases", "read"));

  useEffect(() => {
    if (!canRecord || !account?.id) {
      return;
    }
    recordLastVisitedOrganization(account.id, organizationSlug);
  }, [account?.id, canRecord, organizationSlug]);

  useRecordLastLocation(canRecord ? organizationSlug : null, account?.id, resumePath);
  return null;
}
