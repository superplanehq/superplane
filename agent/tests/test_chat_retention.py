import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

from ai.chat_retention import run_cleanup, start_chat_retention_scheduler


def _make_limit_checker(retention_days: int | None = None) -> AsyncMock:
    checker = AsyncMock()
    checker.get_org_retention_days = AsyncMock(return_value=retention_days)
    return checker


def test_run_cleanup_deletes_per_org() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(return_value=["org-1", "org-2"])
    store.delete_expired_chats_for_org = MagicMock(return_value=5)
    checker = _make_limit_checker(retention_days=30)

    with patch("ai.chat_retention.config") as mock_config:
        mock_config.chat_retention_days = 14
        asyncio.run(run_cleanup(store, checker))

    assert store.delete_expired_chats_for_org.call_count == 2
    store.delete_expired_chats_for_org.assert_any_call("org-1", 30)
    store.delete_expired_chats_for_org.assert_any_call("org-2", 30)


def test_run_cleanup_falls_back_to_config_when_usage_unavailable() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(return_value=["org-1"])
    store.delete_expired_chats_for_org = MagicMock(return_value=0)
    checker = _make_limit_checker(retention_days=None)

    with patch("ai.chat_retention.config") as mock_config:
        mock_config.chat_retention_days = 14
        asyncio.run(run_cleanup(store, checker))

    store.delete_expired_chats_for_org.assert_called_with("org-1", 14)


def test_run_cleanup_skips_orgs_when_fallback_zero_and_usage_unset() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(return_value=["org-1"])
    store.delete_expired_chats_for_org = MagicMock(return_value=0)
    checker = _make_limit_checker(retention_days=None)

    with patch("ai.chat_retention.config") as mock_config:
        mock_config.chat_retention_days = 0
        asyncio.run(run_cleanup(store, checker))

    store.list_distinct_org_ids.assert_called()
    store.delete_expired_chats_for_org.assert_not_called()


def test_run_cleanup_uses_usage_service_even_when_fallback_zero() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(return_value=["org-1"])
    store.delete_expired_chats_for_org = MagicMock(return_value=3)
    checker = _make_limit_checker(retention_days=30)

    with patch("ai.chat_retention.config") as mock_config:
        mock_config.chat_retention_days = 0
        asyncio.run(run_cleanup(store, checker))

    store.delete_expired_chats_for_org.assert_called_with("org-1", 30)


def test_run_cleanup_does_not_propagate_errors() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(side_effect=RuntimeError("db connection lost"))
    store.delete_expired_chats_for_org = MagicMock(return_value=0)
    checker = _make_limit_checker(retention_days=None)

    with patch("ai.chat_retention.config") as mock_config:
        mock_config.chat_retention_days = 14
        asyncio.run(run_cleanup(store, checker))

    store.delete_expired_chats_for_org.assert_not_called()


def test_start_chat_retention_scheduler_returns_running_scheduler() -> None:
    store = MagicMock()
    store.list_distinct_org_ids = MagicMock(return_value=[])
    checker = _make_limit_checker()

    async def run() -> None:
        with patch("ai.chat_retention.config") as mock_config:
            mock_config.chat_retention_days = 14
            scheduler = start_chat_retention_scheduler(store, checker)
            assert scheduler.running
            scheduler.shutdown(wait=False)

    asyncio.run(run())
