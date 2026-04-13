import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

from ai.chat_retention import run_chat_retention_loop


def _make_limit_checker(retention_days: int | None = None) -> AsyncMock:
    checker = AsyncMock()
    checker.get_org_retention_days = AsyncMock(return_value=retention_days)
    return checker


def test_retention_loop_deletes_per_org_and_can_be_cancelled() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(return_value=["org-1", "org-2"])
    store.delete_expired_chats_for_org = MagicMock(return_value=5)
    checker = _make_limit_checker(retention_days=30)

    async def run() -> None:
        with patch("ai.chat_retention.CLEANUP_INTERVAL_SECONDS", 0.01):
            with patch("ai.chat_retention.config") as mock_config:
                mock_config.chat_retention_days = 14
                task = asyncio.create_task(run_chat_retention_loop(store, checker))
                await asyncio.sleep(0.05)
                task.cancel()
                try:
                    await task
                except asyncio.CancelledError:
                    pass

    asyncio.run(run())
    store.list_distinct_org_ids.assert_called()
    assert store.delete_expired_chats_for_org.call_count >= 2
    store.delete_expired_chats_for_org.assert_any_call("org-1", 30)
    store.delete_expired_chats_for_org.assert_any_call("org-2", 30)


def test_retention_loop_falls_back_to_config_when_usage_unavailable() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(return_value=["org-1"])
    store.delete_expired_chats_for_org = MagicMock(return_value=0)
    checker = _make_limit_checker(retention_days=None)

    async def run() -> None:
        with patch("ai.chat_retention.CLEANUP_INTERVAL_SECONDS", 0.01):
            with patch("ai.chat_retention.config") as mock_config:
                mock_config.chat_retention_days = 14
                task = asyncio.create_task(run_chat_retention_loop(store, checker))
                await asyncio.sleep(0.05)
                task.cancel()
                try:
                    await task
                except asyncio.CancelledError:
                    pass

    asyncio.run(run())
    store.delete_expired_chats_for_org.assert_called_with("org-1", 14)


def test_retention_loop_skips_orgs_when_fallback_zero_and_usage_unset() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(return_value=["org-1"])
    store.delete_expired_chats_for_org = MagicMock(return_value=0)
    checker = _make_limit_checker(retention_days=None)

    async def run() -> None:
        with patch("ai.chat_retention.CLEANUP_INTERVAL_SECONDS", 0.01):
            with patch("ai.chat_retention.config") as mock_config:
                mock_config.chat_retention_days = 0
                task = asyncio.create_task(run_chat_retention_loop(store, checker))
                await asyncio.sleep(0.05)
                task.cancel()
                try:
                    await task
                except asyncio.CancelledError:
                    pass

    asyncio.run(run())
    store.list_distinct_org_ids.assert_called()
    store.delete_expired_chats_for_org.assert_not_called()


def test_retention_loop_uses_usage_service_even_when_fallback_zero() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(return_value=["org-1"])
    store.delete_expired_chats_for_org = MagicMock(return_value=3)
    checker = _make_limit_checker(retention_days=30)

    async def run() -> None:
        with patch("ai.chat_retention.CLEANUP_INTERVAL_SECONDS", 0.01):
            with patch("ai.chat_retention.config") as mock_config:
                mock_config.chat_retention_days = 0
                task = asyncio.create_task(run_chat_retention_loop(store, checker))
                await asyncio.sleep(0.05)
                task.cancel()
                try:
                    await task
                except asyncio.CancelledError:
                    pass

    asyncio.run(run())
    store.delete_expired_chats_for_org.assert_called_with("org-1", 30)


def test_retention_loop_continues_after_error() -> None:
    store = MagicMock()
    call_count = 0

    def list_orgs_side_effect() -> list[str]:
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            raise RuntimeError("db connection lost")
        return ["org-1"]

    store.list_distinct_org_ids = MagicMock(side_effect=list_orgs_side_effect)
    store.delete_expired_chats_for_org = MagicMock(return_value=0)
    checker = _make_limit_checker(retention_days=None)

    async def run() -> None:
        with patch("ai.chat_retention.CLEANUP_INTERVAL_SECONDS", 0.01):
            with patch("ai.chat_retention.config") as mock_config:
                mock_config.chat_retention_days = 14
                task = asyncio.create_task(run_chat_retention_loop(store, checker))
                await asyncio.sleep(0.05)
                task.cancel()
                try:
                    await task
                except asyncio.CancelledError:
                    pass

    asyncio.run(run())
    assert store.list_distinct_org_ids.call_count >= 2
