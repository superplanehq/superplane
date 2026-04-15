import asyncio
import logging
from datetime import UTC, datetime

from apscheduler.schedulers.asyncio import AsyncIOScheduler

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


async def run_cleanup(store: SessionStore, limit_checker: AgentUsageLimitChecker) -> None:
    fallback_days = config.chat_retention_days
    try:
        org_ids = await asyncio.to_thread(store.list_distinct_org_ids)
    except Exception:
        logger.exception("agent chat retention: failed to list orgs")
        return

    total = 0
    for org_id in org_ids:
        try:
            retention_days = await _resolve_retention_days(limit_checker, org_id, fallback_days)
            if retention_days <= 0:
                continue
            deleted = await asyncio.to_thread(
                store.delete_expired_chats_for_org, org_id, retention_days
            )
            total += deleted
        except Exception:
            logger.exception("agent chat retention: error cleaning up org %s", org_id)

    if total > 0:
        logger.info("agent chat retention: deleted %d expired chat(s)", total)


def start_chat_retention_scheduler(
    store: SessionStore, limit_checker: AgentUsageLimitChecker
) -> AsyncIOScheduler:
    scheduler = AsyncIOScheduler()
    scheduler.add_job(
        run_cleanup,
        "interval",
        id="chat-retention-cleanup",
        seconds=CLEANUP_INTERVAL_SECONDS,
        args=[store, limit_checker],
        next_run_time=datetime.now(UTC),  # run once immediately at startup
        misfire_grace_time=None,
    )
    scheduler.start()

    logger.info(
        "agent chat retention started: fallback %d days, checking every %ds",
        config.chat_retention_days,
        CLEANUP_INTERVAL_SECONDS,
    )

    return scheduler
