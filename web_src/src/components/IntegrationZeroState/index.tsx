import { useState } from "react";
import { Button } from "../ui/button";
import { IntegrationModal } from "./IntegrationModal";

interface IntegrationZeroStateProps {
  integrationType: string;
  label: string;
  canvasId: string;
  organizationId: string;
  onSuccess?: (integrationId: string) => void;
  hasError?: boolean;
}

const IntegrationZeroState = ({
  integrationType,
  label,
  canvasId,
  organizationId,
  onSuccess,
  hasError = false,
}: IntegrationZeroStateProps) => {
  const [showModal, setShowModal] = useState(false);

  return (
    <>
      <div
        className={`text-center m-4 py-8 rounded-md ${
          hasError
            ? "bg-red-50 dark:bg-red-900/20 border-2 border-red-400 dark:border-red-200"
            : "bg-zinc-50 dark:bg-zinc-700 border border-gray-200 dark:border-gray-700"
        }`}
      >
        <div className="text-gray-500 dark:text-zinc-400 mb-3 font-[400] max-w-[20rem] mx-auto">
          Looks like you haven't connected any {label} yet
        </div>
        <Button onClick={() => setShowModal(true)}>Create integration</Button>
      </div>

      <IntegrationModal
        open={showModal}
        onClose={() => setShowModal(false)}
        integrationType={integrationType}
        canvasId={canvasId}
        organizationId={organizationId}
        onSuccess={onSuccess}
      />
    </>
  );
};

export default IntegrationZeroState;
