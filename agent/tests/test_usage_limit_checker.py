from unittest.mock import MagicMock, patch

import grpc  # type: ignore[import-untyped]
import pytest

from ai.usage_limit_checker import (
    AgentTokenLimitExceeded,
    NoopUsageLimitChecker,
    UsageLimitChecker,
)


def _make_checker(stub: MagicMock) -> UsageLimitChecker:
    with patch("ai.usage_limit_checker.grpc") as mock_grpc:
        mock_channel = MagicMock()
        mock_grpc.insecure_channel.return_value = mock_channel
        mock_channel.unary_unary.return_value = stub
        return UsageLimitChecker("localhost:50051")


def test_check_allows_when_within_limit() -> None:
    usage = MagicMock()
    usage.agent_token_bucket_level = 500
    usage.agent_token_bucket_capacity = 100000
    stub = MagicMock(return_value=MagicMock(usage=usage))

    checker = _make_checker(stub)
    checker.check_agent_token_limit("org-123")


def test_check_blocks_when_at_capacity() -> None:
    usage = MagicMock()
    usage.agent_token_bucket_level = 100000
    usage.agent_token_bucket_capacity = 100000
    usage.next_agent_token_bucket_leak_at_unix_seconds = 1743580800
    stub = MagicMock(return_value=MagicMock(usage=usage))

    checker = _make_checker(stub)
    with pytest.raises(AgentTokenLimitExceeded, match="Agent token limit exceeded"):
        checker.check_agent_token_limit("org-123")


def test_check_includes_next_decrease_time_in_error() -> None:
    usage = MagicMock()
    usage.agent_token_bucket_level = 100000
    usage.agent_token_bucket_capacity = 100000
    usage.next_agent_token_bucket_leak_at_unix_seconds = 1743580800
    stub = MagicMock(return_value=MagicMock(usage=usage))

    checker = _make_checker(stub)
    with pytest.raises(
        AgentTokenLimitExceeded, match="Usage will decrease at Apr 02, 2025 at 08:00 AM UTC"
    ):
        checker.check_agent_token_limit("org-123")


def test_check_allows_unlimited_capacity() -> None:
    usage = MagicMock()
    usage.agent_token_bucket_level = 999999
    usage.agent_token_bucket_capacity = -1
    stub = MagicMock(return_value=MagicMock(usage=usage))

    checker = _make_checker(stub)
    checker.check_agent_token_limit("org-123")


def test_check_allows_on_grpc_error(capsys: pytest.CaptureFixture[str]) -> None:
    stub = MagicMock(side_effect=grpc.RpcError())

    checker = _make_checker(stub)
    checker.check_agent_token_limit("org-123")

    captured = capsys.readouterr()
    assert "usage limit check failed, allowing request" in captured.out


def test_noop_checker_does_nothing() -> None:
    checker = NoopUsageLimitChecker()
    checker.check_agent_token_limit("org-123")
    checker.close()
