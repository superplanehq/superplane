import React from "react";
import { Text } from "@/components/Text/text";
import { Button } from "@/components/ui/button";
import { ErrorBanner } from "./ErrorBanner";

type SmtpPromptStepProps = {
  loading: boolean;
  error: string | null;
  onBack: () => void;
  onEnableSMTP: () => void;
  onSkipSMTP: () => void;
};

export const SmtpPromptStep: React.FC<SmtpPromptStepProps> = ({ loading, error, onBack, onEnableSMTP, onSkipSMTP }) => (
  <div className="space-y-6">
    <ErrorBanner message={error} />

    <div className="text-left">
      <h4 className="mb-2 text-lg font-medium text-gray-800 dark:text-white">Set up email delivery?</h4>
      <Text className="text-gray-800 dark:text-gray-300">
        Configure SMTP now to receive notifications. You can skip and set it up later.
      </Text>
    </div>

    <div className="flex flex-wrap items-center gap-3">
      <Button type="button" variant="outline" disabled={loading} onClick={onBack}>
        Back
      </Button>

      <div className="ml-auto flex flex-wrap justify-end gap-3">
        <Button type="button" disabled={loading} onClick={onEnableSMTP}>
          Set up SMTP
        </Button>
        <Button type="button" variant="outline" disabled={loading} onClick={onSkipSMTP}>
          Do this later
        </Button>
      </div>
    </div>
  </div>
);
