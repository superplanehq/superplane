import asyncio
import logging

from ai.config import config
from ai.session_store import SessionStore

logger = logging.getLogger("agent.chat_retention")

CLEANUP_INTERVAL_SECONDS = 3600


async def run_chat_retention_loop(store: SessionStore) -> None:
    retention_days = config.chat_retention_days
    if retention_days <= 0:
        logger.info("agent chat retention is disabled (AGENT_CHAT_RETENTION_DAYS=0)")
        return

    logger.info(
        "agent chat retention started: deleting chats older than %d days, checking every %ds",
        retention_days,
        CLEANUP_INTERVAL_SECONDS,
    )

    while True:
        try:
            deleted = await asyncio.to_thread(store.delete_expired_chats, retention_days)
            if deleted > 0:
                logger.info("agent chat retention: deleted %d expired chat(s)", deleted)
        except asyncio.CancelledError:
            raise
        except Exception:
            logger.exception("agent chat retention: error during cleanup")

        await asyncio.sleep(CLEANUP_INTERVAL_SECONDS)
