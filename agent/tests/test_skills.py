from pathlib import Path

import pytest

from ai import skills as skills_mod
from ai.skills import get_agent_skill, skill_index_markdown


def test_skill_index_and_get_empty_dir(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(skills_mod, "_skills_root", lambda: tmp_path)
    assert skill_index_markdown() == ""
    assert get_agent_skill("anything") is None


def test_skill_round_trip(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    monkeypatch.setattr(skills_mod, "_skills_root", lambda: tmp_path)
    (tmp_path / "demo.md").write_text(
        "---\ndescription: Short summary for index\n---\n\n# Ignored title\n\nBody **here**.\n",
        encoding="utf-8",
    )
    idx = skill_index_markdown()
    assert "## Available skills" in idx
    assert "**demo**" in idx
    assert "Short summary for index" in idx

    loaded = get_agent_skill("demo")
    assert loaded is not None
    assert loaded["id"] == "demo"
    assert "Body **here**." in loaded["content"]
    assert "# Ignored title" in loaded["content"]
