import { Heading } from "@/components/Heading/heading";
import { Icon } from "@/components/Icon";
import { PersonalApiTokenDialogs, PersonalApiTokensTable } from "@/components/PersonalApiTokens";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { usePersonalTokensPanel } from "@/hooks/usePersonalTokensPanel";
import { settingsCardClassName } from "./settingsPageStyles";

interface ProfileApiTokensSectionProps {
  organizationId: string | null | undefined;
}

/**
 * Named personal API token management for the Profile page: create a token,
 * copy its secret once, and revoke a token from the row menu. The dialogs and
 * the table are shared with the factory General settings page.
 */
export function ProfileApiTokensSection({ organizationId }: ProfileApiTokensSectionProps) {
  const panel = usePersonalTokensPanel(organizationId);

  return (
    <>
      <div className="flex items-center justify-between gap-4">
        <div>
          <Heading level={2} className="text-lg text-left font-medium text-gray-800 dark:text-white mb-0">
            API Tokens
          </Heading>
          <Text className="text-gray-800 text-left dark:text-gray-400 text-sm">
            Use a personal API token to authenticate API requests to SuperPlane. Keep your tokens secure and do not
            share them.
          </Text>
        </div>
        <Button className="shrink-0" onClick={panel.openCreateDialog} data-testid="user-token-create-btn">
          <Icon name="plus" />
          Create token
        </Button>
      </div>

      <div className={settingsCardClassName}>
        <PersonalApiTokensTable
          tokens={panel.tokens}
          isLoading={panel.tokensLoading}
          onCreate={panel.openCreateDialog}
          onRevoke={panel.requestRevoke}
        />
      </div>

      <PersonalApiTokenDialogs panel={panel} />
    </>
  );
}
