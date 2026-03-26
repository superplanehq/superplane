"""Rough USD cost estimates from LLM token usage.

Rates for **Claude 4.6 / 4.5** family models only (Opus, Sonnet, Haiku) from the public 1P table
(USD per million tokens).
Source: https://platform.claude.com/docs/en/about-claude/pricing

This is an **approximation**:
- Assumes ``cache_write_tokens`` are billed at the 5-minute cache write multiplier (1.25× base
  input) when non-zero; 1-hour writes cost more in reality.
- ``cache_read_tokens`` use the cache hit rate (0.1× base input), per Anthropic's table.
- Other providers and OpenRouter-style model strings are not priced here (returns ``None``).
- Batch, long-context premiums, fast mode, server-side tools, etc. are not modeled.
"""

from __future__ import annotations

from dataclasses import dataclass

from pydantic_ai.usage import RunUsage

_PRICING_DOC_URL = "https://platform.claude.com/docs/en/about-claude/pricing"


@dataclass(frozen=True)
class ClaudeUsdPerMillion:
    """USD per 1M tokens at listed 1P ``Base Input`` / ``Output`` rates."""

    input_base: float
    output: float


# Substrings to match against normalized model ids (provider prefix stripped, lowercased).
# Only Claude **4.6** and **4.5** tiers (Opus/Sonnet; Haiku 4.5 only — no Haiku 4.6 in table yet).
_CLAUDE_RATES: tuple[tuple[str, ClaudeUsdPerMillion], ...] = (
    ("claude-opus-4-6", ClaudeUsdPerMillion(5.0, 25.0)),
    ("opus-4-6", ClaudeUsdPerMillion(5.0, 25.0)),
    ("claude-opus-4-5", ClaudeUsdPerMillion(5.0, 25.0)),
    ("opus-4-5", ClaudeUsdPerMillion(5.0, 25.0)),
    ("claude-sonnet-4-6", ClaudeUsdPerMillion(3.0, 15.0)),
    ("sonnet-4-6", ClaudeUsdPerMillion(3.0, 15.0)),
    ("claude-sonnet-4-5", ClaudeUsdPerMillion(3.0, 15.0)),
    ("sonnet-4-5", ClaudeUsdPerMillion(3.0, 15.0)),
    ("claude-haiku-4-5", ClaudeUsdPerMillion(1.0, 5.0)),
    ("haiku-4-5", ClaudeUsdPerMillion(1.0, 5.0)),
)


def pricing_reference_url() -> str:
    return _PRICING_DOC_URL


def normalize_model_id(model: str) -> str:
    s = model.strip().lower()
    if ":" in s:
        s = s.split(":", 1)[1]
    return s.replace("_", "-")


def matched_claude_pricing_key(model: str) -> str | None:
    """First table key that matches the normalized model id (for transparency in reports)."""
    if not model.strip() or model.strip().lower() == "test":
        return None
    mid = normalize_model_id(model)
    for needle, _rates in _CLAUDE_RATES:
        if needle in mid:
            return needle
    return None


def claude_rates_for_model(model: str) -> ClaudeUsdPerMillion | None:
    """Return Claude 1P rates if ``model`` matches a 4.5/4.6 tier id; else ``None``."""
    if not model.strip() or model.strip().lower() == "test":
        return None
    mid = normalize_model_id(model)
    for needle, rates in _CLAUDE_RATES:
        if needle in mid:
            return rates
    return None


def estimate_claude_cost_usd(usage: RunUsage, rates: ClaudeUsdPerMillion) -> float:
    """Estimate USD for one run using base input/output and Anthropic cache multipliers."""
    m = 1_000_000.0
    # Base input and output (standard tiers from pricing table).
    cost = usage.input_tokens * rates.input_base / m
    cost += usage.output_tokens * rates.output / m
    # Prompt caching: cache read ≈ 0.1× base input; 5m cache write ≈ 1.25× base input per MTok.
    if usage.cache_read_tokens:
        cost += usage.cache_read_tokens * (0.1 * rates.input_base) / m
    if usage.cache_write_tokens:
        cost += usage.cache_write_tokens * (1.25 * rates.input_base) / m
    return cost


def estimate_cost_usd_for_model(model: str, usage: RunUsage) -> float | None:
    """Best-effort USD estimate; ``None`` if model is not covered (e.g. OpenAI or unknown)."""
    rates = claude_rates_for_model(model)
    if rates is None:
        return None
    return estimate_claude_cost_usd(usage, rates)
