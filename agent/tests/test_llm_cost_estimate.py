"""Tests for Claude USD estimates and Anthropic prompt-cache token semantics."""

from pydantic_ai.usage import RunUsage

from ai.llm_cost_estimate import ClaudeUsdPerMillion, estimate_claude_cost_usd

_SONNET = ClaudeUsdPerMillion(input_base=3.0, output=15.0)


def test_estimate_claude_cost_inclusive_input_tokens_with_cache() -> None:
    """When input_tokens sums uncached + cache read + cache write, do not full-rate bill cache."""
    usage = RunUsage(
        input_tokens=39404,
        output_tokens=1173,
        cache_read_tokens=26184,
        cache_write_tokens=6546,
    )
    cost = estimate_claude_cost_usd(usage, _SONNET)
    uncached = 39404 - 26184 - 6546
    expected = (
        uncached * 3.0 / 1_000_000
        + 1173 * 15.0 / 1_000_000
        + 26184 * (0.1 * 3.0) / 1_000_000
        + 6546 * (1.25 * 3.0) / 1_000_000
    )
    assert abs(cost - expected) < 1e-9
    assert cost < 0.12


def test_estimate_claude_cost_disjoint_input_tokens_with_cache() -> None:
    """Raw-style usage: input_tokens is uncached-only; remainder would be negative if subtracted."""
    usage = RunUsage(
        input_tokens=6674,
        output_tokens=100,
        cache_read_tokens=26184,
        cache_write_tokens=6546,
    )
    cost = estimate_claude_cost_usd(usage, _SONNET)
    expected = (
        6674 * 3.0 / 1_000_000
        + 100 * 15.0 / 1_000_000
        + 26184 * (0.1 * 3.0) / 1_000_000
        + 6546 * (1.25 * 3.0) / 1_000_000
    )
    assert abs(cost - expected) < 1e-9


def test_estimate_claude_cost_no_cache() -> None:
    usage = RunUsage(input_tokens=10_000, output_tokens=500)
    cost = estimate_claude_cost_usd(usage, _SONNET)
    assert abs(cost - (10_000 * 3.0 / 1_000_000 + 500 * 15.0 / 1_000_000)) < 1e-9
