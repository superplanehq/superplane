import type { FactoriesFactory } from "@/api-client";
import type { IntegrationSelections } from "@/pages/home/InstallIntegrationsSection";
import { useEffect, useMemo, useRef, useState } from "react";

import { loadSavedInstallationName, selectionsWithSavedVcsInstallation } from "./githubIntegrationSelection";
import { AGENT_PROVIDER_IDS } from "./onboardingAgentReadiness";
import type { IntegrationId } from "./onboardingFixtures";
import { initialOnboardingSelections, onboardingVcsHost } from "./onboardingStatus";

const SAVED_INSTALLATION_FOLLOW_UP_MS = 15_000;

function installationNameIsKnown(selection: IntegrationSelections[string] | undefined, integrationId: string): boolean {
  if (!selection || selection.id !== integrationId || !selection.ready) return false;
  const name = selection.name.trim();
  return name !== "" && name !== integrationId;
}

function connectedFromSelections(selections: IntegrationSelections): Set<IntegrationId> {
  const ready = new Set<IntegrationId>();
  if (selections.github?.ready) ready.add("github");
  if (selections.bitbucket?.ready) ready.add("bitbucket");
  if (selections.jira?.ready) ready.add("jira");
  if (selections.linear?.ready) ready.add("linear");
  for (const name of AGENT_PROVIDER_IDS) {
    if (selections[name]?.ready) ready.add(name);
  }
  return ready;
}

export function useOnboardingIntegrationSelections(organizationId: string, onboarding: FactoriesFactory["onboarding"]) {
  const [selections, setSelections] = useState<IntegrationSelections>(() => initialOnboardingSelections(onboarding));
  const selectionsRef = useRef(selections);
  selectionsRef.current = selections;
  const [lookupNonce, setLookupNonce] = useState(0);

  useEffect(() => {
    const refresh = () => {
      if (document.visibilityState === "hidden") return;
      setLookupNonce((value) => value + 1);
    };
    window.addEventListener("focus", refresh);
    document.addEventListener("visibilitychange", refresh);
    return () => {
      window.removeEventListener("focus", refresh);
      document.removeEventListener("visibilitychange", refresh);
    };
  }, []);

  useEffect(() => {
    const host = onboardingVcsHost(onboarding);
    const id = onboarding?.vcsIntegrationId?.trim() ?? "";
    if (!organizationId || !host || !id) return;
    if (installationNameIsKnown(selectionsRef.current[host], id)) return;

    let cancelled = false;
    let followUp: ReturnType<typeof setTimeout> | undefined;
    const load = () => {
      void loadSavedInstallationName(organizationId, id, { cancelled: () => cancelled }).then((name) => {
        if (cancelled) return;
        if (!name) {
          followUp = setTimeout(load, SAVED_INSTALLATION_FOLLOW_UP_MS);
          return;
        }
        setSelections((latest) => selectionsWithSavedVcsInstallation(onboarding, latest, name));
      });
    };
    load();
    return () => {
      cancelled = true;
      clearTimeout(followUp);
    };
  }, [organizationId, onboarding, lookupNonce]);

  const connected = useMemo(() => connectedFromSelections(selections), [selections]);
  return { selections, connected, setSelections };
}
