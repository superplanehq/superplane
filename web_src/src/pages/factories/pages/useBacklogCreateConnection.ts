import { useContext } from "react";

import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryOwnerEligibility } from "@/hooks/useFactoryOwnerEligibility";
import { showErrorToast } from "@/lib/toast";
import { CodexActionContext } from "../codex/codexActionContext";

export function useBacklogCreateConnection(organizationId: string, factoryId: string) {
  const { currentUserId } = usePermissions();
  const eligibility = useFactoryOwnerEligibility(organizationId, factoryId);
  const connect = useContext(CodexActionContext);

  return async () => {
    const result = await eligibility.refetch();
    const member = result.data?.members?.find((member) => member.userId === currentUserId);
    if (!result.isError && result.data?.subscription === false) return true;
    if (!result.isError && member?.eligible === true) return true;
    if (!result.isError && member?.eligible === false && connect) return connect(organizationId, factoryId);
    showErrorToast("SuperPlane could not check your ChatGPT connection. Try again.");
    return false;
  };
}
