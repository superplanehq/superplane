import { cleanup } from "@testing-library/react";
import { afterAll, afterEach, vi } from "vitest";

const SHARED_ENV_MOCKS = [
  "@/api-client",
  "@/components/AgentSidebar/widgets/MarkdownCode",
  "@/contexts/useAccount",
  "@/contexts/usePermissions",
  "@/contexts/useTheme",
  "@/hooks/useAccountOrganizations",
  "@/hooks/useAgentChats",
  "@/hooks/useAgentSessionWebsocket",
  "@/hooks/useApiKeys",
  "@/hooks/useBindGitHubInstallation",
  "@/hooks/useCanvasData",
  "@/hooks/useCanvasWebsocket",
  "@/hooks/useComponentData",
  "@/hooks/useExperimentalFeature",
  "@/hooks/useFactoryData",
  "@/hooks/useFactoryIntakeData",
  "@/hooks/useFactoryLineRunnerModels",
  "@/hooks/useFactoryPRFeedbackData",
  "@/hooks/useFactoryUsage",
  "@/hooks/useFactoryVelocity",
  "@/hooks/useHostedLLMModels",
  "@/hooks/useIntegrations",
  "@/hooks/useLLMModelAllowlists",
  "@/hooks/useMe",
  "@/hooks/useNotificationSettings",
  "@/hooks/useOrgUserLookup",
  "@/hooks/useOrganizationData",
  "@/hooks/useOrganizationWorkspaceUsage",
  "@/hooks/usePageTitle",
  "@/hooks/useRecheckGitHubInstallRequest",
  "@/hooks/useReportPageReady",
  "@/hooks/useSecrets",
  "@/hooks/useSelectableLLMModels",
  "@/hooks/useUserTokens",
  "@/hooks/useWorkOrderCardActions",
  "@/hooks/useWorkOrderChecks",
  "@/lib/accountSettings",
  "@/lib/browserAction",
  "@/lib/integrationSetupReturn",
  "@/lib/toast",
  "@/pages/app/lib/workflow-spec-files",
  "@/pages/factories/WorkOrderDescriptionEditor",
  "@/pages/factories/agent/FactoryRichMessage",
  "@/pages/factories/layout/factoriesLayoutContext",
  "@/pages/factories/pages/LineVelocityPanel",
  "@/pages/factories/pages/ProductiveIntakeSetupDialog",
  "@/pages/factories/pages/onboarding/AgentStep",
  "@/pages/factories/pages/onboarding/useGithubAppAvailability",
  "@/pages/factories/pages/planningSessionClient",
  "@/pages/factories/pages/settings/DeleteAccountDangerZone",
  "@/pages/factories/pages/work-order-split-run/useSplitRunFooterActions",
  "@/pages/factories/pages/workspaceNextStepTransition",
  "@/posthog",
  "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream",
  "@/ui/IntegrationCreateDialog",
  "@/ui/componentSidebar/integrationIcons",
  "@monaco-editor/react",
  "posthog-js",
  "react-router",
  "sonner",
] as const;

const originalLocation = window.location;
const originalClipboard = navigator.clipboard;
const originalHtmlElementFocus = HTMLElement.prototype.focus;

function ensureHtmlElementFocusIsWritable() {
  Object.defineProperty(HTMLElement.prototype, "focus", {
    configurable: true,
    writable: true,
    value: originalHtmlElementFocus,
  });
}

ensureHtmlElementFocusIsWritable();

function clearDocumentCookies() {
  for (const part of document.cookie.split(";")) {
    const name = part.split("=")[0]?.trim();
    if (name) {
      document.cookie = `${name}=; Max-Age=0; Path=/`;
    }
  }
}

afterEach(async () => {
  cleanup();

  const [
    { resetApiInterceptor },
    { resetBacklogAnalysisPending },
    { resetFactoriesThemeClass },
    { resetCreateWithAgentUnmountEnds },
  ] = await Promise.all([
    import("@/lib/api-interceptor"),
    import("@/pages/factories/lib/backlogAnalysis"),
    import("@/pages/factories/lib/useFactoriesThemeClass"),
    import("@/pages/factories/pages/planningSessionUnmountEnds"),
  ]);
  resetApiInterceptor();
  resetBacklogAnalysisPending();
  resetFactoriesThemeClass();
  resetCreateWithAgentUnmountEnds();

  vi.useRealTimers();
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
  localStorage.clear();
  sessionStorage.clear();
  if (window.location !== originalLocation) {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: originalLocation,
    });
  }
  if (navigator.clipboard !== originalClipboard) {
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: originalClipboard,
    });
  }
  window.history.replaceState({}, "", "/");
  document.title = "";
  clearDocumentCookies();
  ensureHtmlElementFocusIsWritable();
});

afterAll(() => {
  for (const specifier of SHARED_ENV_MOCKS) {
    vi.doUnmock(specifier);
  }
  vi.resetModules();
});
