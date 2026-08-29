import { useState } from "react";
import { useUserTokens, useCreateUserToken, useRevokeUserToken } from "@/hooks/useUserTokens";
import { getApiErrorMessage } from "@/lib/errors";

export interface RevealedPersonalToken {
  id: string;
  plaintext: string;
}

/**
 * Encapsulates the state and actions behind the personal API tokens panel
 * shown on Profile and the factory General settings page: the create form,
 * the create-once secret reveal, and per-row revoke. Both pages render their
 * own markup around this shared state so each can match its surrounding
 * design language.
 */
export function usePersonalTokensPanel(organizationId: string | null | undefined) {
  const orgId = organizationId || "";
  const { data: tokens = [], isLoading: tokensLoading } = useUserTokens(orgId);
  const createMutation = useCreateUserToken(orgId);
  const revokeMutation = useRevokeUserToken(orgId);

  const [newTokenName, setNewTokenName] = useState("");
  const [revealedToken, setRevealedToken] = useState<RevealedPersonalToken | null>(null);
  const [tokenVisible, setTokenVisible] = useState(false);
  const [revokingId, setRevokingId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const createToken = async () => {
    const name = newTokenName.trim();
    if (!name) return;

    try {
      setActionError(null);
      const response = await createMutation.mutateAsync({ name });
      if (response?.token && response.plaintext) {
        setRevealedToken({ id: response.token.id || "", plaintext: response.plaintext });
        setTokenVisible(false);
      }
      setNewTokenName("");
    } catch (err) {
      setActionError(`Failed to create token: ${getApiErrorMessage(err)}`);
    }
  };

  const revokeToken = async (id: string, name: string) => {
    if (!confirm(`Revoke token "${name || "Unnamed"}"? Anything using it stops working immediately.`)) return;

    try {
      setActionError(null);
      setRevokingId(id);
      await revokeMutation.mutateAsync(id);
      if (revealedToken?.id === id) {
        setRevealedToken(null);
      }
    } catch (err) {
      setActionError(`Failed to revoke token: ${getApiErrorMessage(err)}`);
    } finally {
      setRevokingId(null);
    }
  };

  return {
    tokens,
    tokensLoading,
    newTokenName,
    setNewTokenName,
    revealedToken,
    tokenVisible,
    setTokenVisible,
    revokingId,
    actionError,
    isCreating: createMutation.isPending,
    createToken,
    revokeToken,
  };
}
