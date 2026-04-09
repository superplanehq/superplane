"""Checks agent token usage limits against the saas usage service."""

from __future__ import annotations

from datetime import UTC, datetime
from typing import Protocol

import grpc  # type: ignore[import-untyped]

import usage_pb2


class AgentTokenLimitExceeded(Exception):
    pass


class AgentUsageLimitChecker(Protocol):
    def check_agent_token_limit(self, organization_id: str) -> None:
        """Raise AgentTokenLimitExceeded if the org has exceeded its agent token budget."""
        ...

    def close(self) -> None: ...


_DESCRIBE_USAGE_METHOD = "/superplane.usage.v1.Usage/DescribeOrganizationUsage"


def _format_next_decrease_hint(next_leak_at_unix: int) -> str:
    if next_leak_at_unix <= 0:
        return ""

    next_at = datetime.fromtimestamp(next_leak_at_unix, tz=UTC)
    return f" Usage will decrease at {next_at.strftime('%b %d, %Y at %I:%M %p UTC')}."


class UsageLimitChecker:
    """Checks agent token limits via the saas usage gRPC service."""

    def __init__(self, usage_grpc_url: str) -> None:
        self._channel = grpc.insecure_channel(usage_grpc_url)
        self._call = self._channel.unary_unary(
            _DESCRIBE_USAGE_METHOD,
            request_serializer=usage_pb2.DescribeOrganizationUsageRequest.SerializeToString,  # type: ignore[attr-defined]
            response_deserializer=usage_pb2.DescribeOrganizationUsageResponse.FromString,  # type: ignore[attr-defined]
        )

    def check_agent_token_limit(self, organization_id: str) -> None:
        try:
            response = self._call(
                usage_pb2.DescribeOrganizationUsageRequest(organization_id=organization_id),  # type: ignore[attr-defined]
                timeout=5,
            )
        except grpc.RpcError as error:
            print(f"[web] usage limit check failed, allowing request: {error}", flush=True)
            return

        usage = response.usage
        if usage is None:
            return

        capacity = usage.agent_token_bucket_capacity

        # unlimited capacity
        if capacity <= 0:
            return

        if usage.agent_token_bucket_level >= capacity:
            next_leak_at = usage.next_agent_token_bucket_leak_at_unix_seconds
            hint = _format_next_decrease_hint(next_leak_at)
            raise AgentTokenLimitExceeded(f"Agent token limit exceeded.{hint}")

    def close(self) -> None:
        self._channel.close()


class NoopUsageLimitChecker:
    """No-op checker used when USAGE_GRPC_URL is not configured."""

    def check_agent_token_limit(self, organization_id: str) -> None:
        pass

    def close(self) -> None:
        pass
