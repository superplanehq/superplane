import type { FactoriesFactory } from "@/api-client";
import type { ReactElement } from "react";
import { Navigate } from "react-router-dom";
import { factoryDetailPath, factoryListPath } from "./lib/factoryPagePaths";

export function factoryListRedirect(organizationId: string) {
  return <Navigate to={factoryListPath(organizationId)} replace />;
}

export function factoryDetailRedirect(organizationId: string, factoryId: string) {
  return <Navigate to={factoryDetailPath(organizationId, factoryId)} replace />;
}

export function resolveFactoryLoadRedirect(
  factoryLoading: boolean,
  factoryError: unknown,
  organizationId: string,
): ReactElement | null {
  if (!factoryLoading && factoryError) {
    return factoryListRedirect(organizationId);
  }
  return null;
}

export function resolveFactoryLineEditRedirect(input: {
  canUpdate: boolean;
  factoryLoading: boolean;
  factoryError: unknown;
  isCreate: boolean;
  factory: FactoriesFactory | undefined;
  line: { id?: string } | null;
  organizationId: string;
  factoryId: string;
}): ReactElement | null {
  if (!input.canUpdate) {
    return factoryDetailRedirect(input.organizationId, input.factoryId);
  }

  if (!input.factoryLoading && input.factoryError) {
    return factoryListRedirect(input.organizationId);
  }

  if (!input.isCreate && !input.factoryLoading && input.factory && !input.line) {
    return factoryDetailRedirect(input.organizationId, input.factoryId);
  }

  return null;
}

export function resolveWorkOrderDetailRedirect(input: {
  factoryLoading: boolean;
  factoryError: unknown;
  orderLoading: boolean;
  orderError: unknown;
  organizationId: string;
  factoryId: string;
}): ReactElement | null {
  if (!input.factoryLoading && input.factoryError) {
    return factoryListRedirect(input.organizationId);
  }

  if (!input.orderLoading && input.orderError) {
    return factoryDetailRedirect(input.organizationId, input.factoryId);
  }

  return null;
}

export function isFactoryLineEditPageFailed(input: {
  factoryError: unknown;
  isCreate: boolean;
  factoryLoading: boolean;
  line: { id?: string } | null;
}) {
  return Boolean(input.factoryError) || (!input.isCreate && !input.factoryLoading && !input.line);
}
