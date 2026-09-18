import { Button } from "@/components/ui/button";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { FEATURE_FACTORIES } from "@/lib/experimentalFeatures";
import { cn } from "@/lib/utils";
import { ArrowRight } from "lucide-react";
import { useState } from "react";

import { FactorySetupPanel } from "./FactorySetupPanel";
import { getFactoryDefinition } from "./factories";
import { homePageSubtitleClassName, homePageTitleClassName } from "./homePageStyles";
import type { CanvasFolderData } from "./types";
import { useCreateApp } from "./useCreateApp";
import { useInstallFactory } from "./useInstallFactory";
import { useOrganizationId } from "@/hooks/useOrganizationId";

interface FreshOrgLandingProps {
  folder?: CanvasFolderData;
  folderContextPending?: boolean;
  title?: string;
}

/**
 * New-app landing. When factories are disabled, offers the template-based
 * "Setup Factory" onboarding. When enabled, only a blank app.
 */
export function FreshOrgLanding({
  folder,
  folderContextPending = false,
  title = "Create a new app",
}: FreshOrgLandingProps) {
  const organizationId = useOrganizationId() ?? undefined;
  const { has: hasExperimentalFeature } = useExperimentalFeature(organizationId);
  const showLegacyFactoryOnboarding = !hasExperimentalFeature(FEATURE_FACTORIES);
  const factory = getFactoryDefinition();
  const { createApp, isSaving } = useCreateApp({ folder });
  const { installFactory, isInstalling } = useInstallFactory({ folder });
  const [showFactorySetup, setShowFactorySetup] = useState(false);
  const busy = folderContextPending || isSaving || isInstalling;

  return (
    <div className="mx-auto w-full max-w-3xl px-8 py-14 lg:py-20">
      <h1 className={cn(homePageTitleClassName, "text-2xl text-gray-800")}>{title}</h1>
      <p className={cn(homePageSubtitleClassName, "mt-3 max-w-lg font-normal leading-normal text-gray-600")}>
        {showLegacyFactoryOnboarding
          ? "Set up a Software Factory to automate coding work with agents, from trigger to pull request. Or start from a blank app."
          : "Start from a blank app to map your first automation."}
      </p>
      {showLegacyFactoryOnboarding && !showFactorySetup && (
        <div className="mt-7">
          <Button
            type="button"
            size="lg"
            disabled={busy}
            onClick={() => {
              setShowFactorySetup(true);
            }}
          >
            Setup Factory
            <ArrowRight />
          </Button>
        </div>
      )}

      {showLegacyFactoryOnboarding && showFactorySetup && (
        <FactorySetupPanel
          factory={factory}
          busy={busy}
          onCancel={() => setShowFactorySetup(false)}
          onInstall={(result) => {
            if (busy) return;
            void installFactory(result);
          }}
        />
      )}

      {!showFactorySetup && (
        <div className={showLegacyFactoryOnboarding ? undefined : "mt-7"}>
          <CreateBlankAppLink
            busy={busy}
            onCreateBlank={() => {
              if (busy) return;
              void createApp(generateCanvasName());
            }}
          />
        </div>
      )}
    </div>
  );
}

function CreateBlankAppLink({ busy, onCreateBlank }: { busy: boolean; onCreateBlank: () => void }) {
  return (
    <p className="mt-8 text-sm font-normal text-gray-600 dark:text-gray-400">
      <Button
        type="button"
        variant="link"
        disabled={busy}
        onClick={onCreateBlank}
        className="h-auto p-0 text-sm font-normal text-gray-800 underline decoration-gray-300 underline-offset-4 dark:text-gray-200 dark:decoration-gray-600"
      >
        Create a blank app
      </Button>
    </p>
  );
}
