from datetime import datetime
from uuid import UUID

from sqlalchemy import BigInteger, DateTime, ForeignKey, Index, Text, desc, func
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.dialects.postgresql import UUID as PG_UUID
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


class Base(DeclarativeBase):
    pass


class AgentChat(Base):
    __tablename__ = "agent_chats"
    __table_args__ = (
        Index(
            "idx_agent_chats_owner_canvas_created",
            "org_id",
            "user_id",
            "canvas_id",
            desc("created_at"),
        ),
    )

    id: Mapped[UUID] = mapped_column(PG_UUID(as_uuid=True), primary_key=True)
    org_id: Mapped[UUID] = mapped_column(PG_UUID(as_uuid=True), nullable=False)
    user_id: Mapped[UUID] = mapped_column(PG_UUID(as_uuid=True), nullable=False)
    canvas_id: Mapped[UUID] = mapped_column(PG_UUID(as_uuid=True), nullable=False)
    initial_message: Mapped[str | None] = mapped_column(Text, nullable=True)
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False, server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False, server_default=func.now())
    total_input_tokens: Mapped[int] = mapped_column(BigInteger, nullable=False, server_default="0")
    total_output_tokens: Mapped[int] = mapped_column(BigInteger, nullable=False, server_default="0")
    total_tokens: Mapped[int] = mapped_column(BigInteger, nullable=False, server_default="0")

    messages: Mapped[list["AgentChatMessage"]] = relationship(
        back_populates="chat", cascade="all, delete-orphan"
    )
    runs: Mapped[list["AgentChatRun"]] = relationship(
        back_populates="chat", cascade="all, delete-orphan"
    )


class AgentChatMessage(Base):
    __tablename__ = "agent_chat_messages"
    __table_args__ = (
        Index(
            "idx_agent_chat_messages_chat_id_message_index",
            "chat_id",
            "message_index",
            unique=True,
        ),
    )

    id: Mapped[UUID] = mapped_column(PG_UUID(as_uuid=True), primary_key=True)
    chat_id: Mapped[UUID] = mapped_column(
        PG_UUID(as_uuid=True),
        ForeignKey("agent_chats.id", ondelete="CASCADE"),
        nullable=False,
    )
    run_id: Mapped[UUID | None] = mapped_column(
        PG_UUID(as_uuid=True),
        ForeignKey("agent_chat_runs.id", ondelete="SET NULL"),
        nullable=True,
    )
    message_index: Mapped[int] = mapped_column(nullable=False)
    message: Mapped[dict] = mapped_column(JSONB, nullable=False)  # type: ignore[type-arg]
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False, server_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False, server_default=func.now())

    chat: Mapped["AgentChat"] = relationship(back_populates="messages")


class AgentChatRun(Base):
    __tablename__ = "agent_chat_runs"
    __table_args__ = (Index("idx_agent_chat_runs_chat_id", "chat_id"),)

    id: Mapped[UUID] = mapped_column(PG_UUID(as_uuid=True), primary_key=True)
    chat_id: Mapped[UUID] = mapped_column(
        PG_UUID(as_uuid=True),
        ForeignKey("agent_chats.id", ondelete="CASCADE"),
        nullable=False,
    )
    model: Mapped[str] = mapped_column(Text, nullable=False, server_default="")
    input_tokens: Mapped[int] = mapped_column(BigInteger, nullable=False, server_default="0")
    output_tokens: Mapped[int] = mapped_column(BigInteger, nullable=False, server_default="0")
    cache_read_tokens: Mapped[int] = mapped_column(BigInteger, nullable=False, server_default="0")
    cache_write_tokens: Mapped[int] = mapped_column(BigInteger, nullable=False, server_default="0")
    total_tokens: Mapped[int] = mapped_column(BigInteger, nullable=False, server_default="0")
    created_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False, server_default=func.now())

    chat: Mapped["AgentChat"] = relationship(back_populates="runs")


class AgentCanvasMarkdownMemory(Base):
    __tablename__ = "agent_canvas_markdown_memory"

    canvas_id: Mapped[UUID] = mapped_column(PG_UUID(as_uuid=True), primary_key=True)
    markdown_body: Mapped[str] = mapped_column(Text, nullable=False)
    updated_at: Mapped[datetime] = mapped_column(DateTime(timezone=True), nullable=False, server_default=func.now())
