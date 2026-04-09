import asyncio
from unittest.mock import MagicMock, patch

from ai.chat_retention import run_chat_retention_loop


def test_retention_loop_calls_delete_and_can_be_cancelled() -> None:
    store = MagicMock()
    store.delete_expired_chats = MagicMock(return_value=5)

    async def run() -> None:
        with patch("ai.chat_retention.CLEANUP_INTERVAL_SECONDS", 0.01):
            with patch("ai.chat_retention.config") as mock_config:
                mock_config.chat_retention_days = 14
                task = asyncio.create_task(run_chat_retention_loop(store))
                await asyncio.sleep(0.05)
                task.cancel()
                try:
                    await task
                except asyncio.CancelledError:
                    pass

    asyncio.run(run())
    store.delete_expired_chats.assert_called_with(14)
    assert store.delete_expired_chats.call_count >= 1


def test_retention_loop_disabled_when_zero_days() -> None:
    store = MagicMock()

    async def run() -> None:
        with patch("ai.chat_retention.config") as mock_config:
            mock_config.chat_retention_days = 0
            await run_chat_retention_loop(store)

    asyncio.run(run())
    store.delete_expired_chats.assert_not_called()


def test_retention_loop_continues_after_error() -> None:
    store = MagicMock()
    call_count = 0

    def side_effect(days: int) -> int:
        nonlocal call_count
        call_count += 1
        if call_count == 1:
            raise RuntimeError("db connection lost")
        return 0

    store.delete_expired_chats = MagicMock(side_effect=side_effect)

    async def run() -> None:
        with patch("ai.chat_retention.CLEANUP_INTERVAL_SECONDS", 0.01):
            with patch("ai.chat_retention.config") as mock_config:
                mock_config.chat_retention_days = 14
                task = asyncio.create_task(run_chat_retention_loop(store))
                await asyncio.sleep(0.05)
                task.cancel()
                try:
                    await task
                except asyncio.CancelledError:
                    pass

    asyncio.run(run())
    assert store.delete_expired_chats.call_count >= 2
