import { useState } from "react";
import type { MeUserApiToken } from "@/api-client/types.gen";
import { useUserTokens, useCreateUserToken, useRevokeUserToken } from "@/hooks/useUserTokens";
import { getApiErrorMessage } from "@/lib/errors";
import { showSuccessToast } from "@/lib/toast";

export interface RevealedPersonalToken {
  id: string;
  name: string;
  plaintext: string;
}

export interface PersonalTokenTarget {
  id: string;
  name: string;
}

export type PersonalTokensPanel = ReturnType<typeof usePersonalTokensPanel>;

/**
 * State behind the personal API tokens panel on Profile and on the factory
 * General settings page. Each step of the flow owns a dialog: name the
 * token, copy the secret one time, then confirm a revoke. Both pages render
 * their own table chrome around this state and share the dialogs.
 */
export function usePersonalTokensPanel(organizationId: string | null | undefined) {
  const orgId = organizationId || "";
  const { data: tokens = [], isLoading: tokensLoading } = useUserTokens(orgId);
  const createMutation = useCreateUserToken(orgId);
  const revokeMutation = useRevokeUserToken(orgId);

  const [isCreateOpen, setIsCreateOpen] = useState(false);
  const [newTokenName, setNewTokenName] = useState("");
  const [createError, setCreateError] = useState<string | null>(null);
  const [revealedToken, setRevealedToken] = useState<RevealedPersonalToken | null>(null);
  const [revokeTarget, setRevokeTarget] = useState<PersonalTokenTarget | null>(null);
  const [revokeError, setRevokeError] = useState<string | null>(null);

  const openCreateDialog = () => {
    setNewTokenName("");
    setCreateError(null);
    setIsCreateOpen(true);
  };

  const closeCreateDialog = () => {
    if (createMutation.isPending) return;
    setIsCreateOpen(false);
    setCreateError(null);
  };

  const createToken = async () => {
    const name = newTokenName.trim();
    if (!name) return;

    try {
      setCreateError(null);
      const response = await createMutation.mutateAsync({ name });
      if (!response?.plaintext) {
        setCreateError("SuperPlane did not return a token. Try again.");
        return;
      }

      setRevealedToken({
        id: response.token?.id || "",
        name: response.token?.name || name,
        plaintext: response.plaintext,
      });
      setNewTokenName("");
      setIsCreateOpen(false);
    } catch (err) {
      setCreateError(`Failed to create the token: ${getApiErrorMessage(err)}`);
    }
  };

  const dismissRevealedToken = () => setRevealedToken(null);

  const requestRevoke = (token: MeUserApiToken) => {
    setRevokeError(null);
    setRevokeTarget({ id: token.id || "", name: token.name || "Unnamed" });
  };

  const cancelRevoke = () => {
    if (revokeMutation.isPending) return;
    setRevokeTarget(null);
    setRevokeError(null);
  };

  const confirmRevoke = async () => {
    if (!revokeTarget) return;

    try {
      setRevokeError(null);
      await revokeMutation.mutateAsync(revokeTarget.id);
      if (revealedToken?.id === revokeTarget.id) {
        setRevealedToken(null);
      }
      setRevokeTarget(null);
      showSuccessToast("Token revoked.");
    } catch (err) {
      setRevokeError(`Failed to revoke the token: ${getApiErrorMessage(err)}`);
    }
  };

  return {
    tokens,
    tokensLoading,
    isCreateOpen,
    openCreateDialog,
    closeCreateDialog,
    newTokenName,
    setNewTokenName,
    createToken,
    isCreating: createMutation.isPending,
    createError,
    revealedToken,
    dismissRevealedToken,
    revokeTarget,
    requestRevoke,
    cancelRevoke,
    confirmRevoke,
    isRevoking: revokeMutation.isPending,
    revokeError,
  };
}
