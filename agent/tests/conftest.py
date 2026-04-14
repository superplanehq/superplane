"""Shared test fixtures and SQLite type adapters for agent tests.

SQLite does not natively support timezone-aware datetimes or PostgreSQL-specific
types (``UUID``, ``JSONB``). The adapters and monkeypatches below bridge these
gaps so that ``SessionStore`` integration tests can run against an in-memory
SQLite database without modifying production code.
"""

import uuid
from collections.abc import Generator
from datetime import UTC, datetime

import pytest
from sqlalchemy import create_engine
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.dialects.postgresql import UUID as PG_UUID
from sqlalchemy.engine import Engine
from sqlalchemy.ext.compiler import compiles
from sqlalchemy.orm import Session

from db.db import build_session_factory
from db.models import Base
from ai.session_store import SessionStore, SessionStoreConfig


@compiles(PG_UUID, "sqlite")
def _compile_pg_uuid_sqlite(type_: PG_UUID[uuid.UUID], compiler: object, **kw: object) -> str:
    return "VARCHAR(36)"


@compiles(JSONB, "sqlite")
def _compile_jsonb_sqlite(type_: JSONB, compiler: object, **kw: object) -> str:
    return "JSON"


def _from_db_time_sqlite(value: datetime) -> datetime:
    """SQLite returns naive datetimes; PostgreSQL returns aware ones.

    This replaces ``_from_db_time`` in the ``sqlite_store`` fixture so that
    production code can assume aware datetimes unconditionally."""
    if value.tzinfo is None:
        return value.replace(tzinfo=UTC)
    return value.astimezone(UTC)


@pytest.fixture()
def sqlite_engine() -> Generator[Engine, None, None]:
    engine = create_engine("sqlite:///:memory:")
    Base.metadata.create_all(engine)
    yield engine
    engine.dispose()


@pytest.fixture()
def sqlite_session(sqlite_engine: Engine) -> Generator[Session, None, None]:
    factory = build_session_factory(sqlite_engine)
    session = factory()
    yield session
    session.close()


@pytest.fixture()
def sqlite_store(sqlite_engine: Engine, monkeypatch: pytest.MonkeyPatch) -> SessionStore:
    factory = build_session_factory(sqlite_engine)
    monkeypatch.setattr("ai.session_store.build_engine", lambda **kwargs: sqlite_engine)
    monkeypatch.setattr("ai.session_store.build_session_factory", lambda e: factory)
    monkeypatch.setattr("ai.session_store._from_db_time", _from_db_time_sqlite)
    return SessionStore(
        SessionStoreConfig(
            host="localhost",
            port=5432,
            dbname="test",
            user="test",
            password="test",
            sslmode="disable",
            application_name="test",
        )
    )
