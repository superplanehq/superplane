import asyncio
import logging

from ai.config import config
from ai.session_store import SessionStore
from ai.usage_limit_checker import AgentUsageLimitChecker

logger = logging.getLogger("agent.chat_retention")

CLEANUP_INTERVAL_SECONDS = 3600


async def _resolve_retention_days(
    limit_checker: AgentUsageLimitChecker, org_id: str, fallback: int
) -> int:
    days = await limit_checker.get_org_retention_days(org_id)
    if days is not None:
        return days
    return fallback


async def run_chat_retention_loop(
    store: SessionStore, limit_checker: AgentUsageLimitChecker
) -> None:
    fallback_days = config.chat_retention_days

    logger.info(
        "agent chat retention started: fallback %d days, checking every %ds",
        fallback_days,
        CLEANUP_INTERVAL_SECONDS,
    )

    while True:
        try:
            org_ids = await asyncio.to_thread(store.list_distinct_org_ids)
            total = 0
            for org_id in org_ids:
                retention_days = await _resolve_retention_days(limit_checker, org_id, fallback_days)
                if retention_days <= 0:
                    continue
                deleted = await asyncio.to_thread(
                    store.delete_expired_chats_for_org, org_id, retention_days
                )
                total += deleted
            if total > 0:
                logger.info("agent chat retention: deleted %d expired chat(s)", total)
        except asyncio.CancelledError:
            raise
        except Exception:
            logger.exception("agent chat retention: error during cleanup")

        await asyncio.sleep(CLEANUP_INTERVAL_SECONDS)
