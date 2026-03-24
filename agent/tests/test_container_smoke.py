import os


def test_runs_inside_container() -> None:
    assert os.getenv("IN_DOCKER") == "1"
