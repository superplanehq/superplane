import asyncio

from ai.stream_tracker import ActiveStreamTracker


class TestActiveStreamTracker:
    def test_track_stream_increments_and_decrements(self) -> None:
        async def run() -> None:
            tracker = ActiveStreamTracker(drain_timeout=5.0)
            assert tracker.active_count == 0

            async with tracker.track_stream():
                assert tracker.active_count == 1
                async with tracker.track_stream():
                    assert tracker.active_count == 2
                assert tracker.active_count == 1
            assert tracker.active_count == 0

        asyncio.run(run())

    def test_begin_shutdown_sets_flag(self) -> None:
        tracker = ActiveStreamTracker(drain_timeout=5.0)
        assert not tracker.is_shutting_down
        tracker.begin_shutdown()
        assert tracker.is_shutting_down

    def test_wait_for_drain_returns_immediately_when_no_streams(self) -> None:
        async def run() -> None:
            tracker = ActiveStreamTracker(drain_timeout=5.0)
            await tracker.wait_for_drain()

        asyncio.run(run())

    def test_wait_for_drain_waits_for_active_streams(self) -> None:
        async def run() -> None:
            tracker = ActiveStreamTracker(drain_timeout=5.0)
            finished_order: list[str] = []

            async def simulate_stream(delay: float, label: str) -> None:
                async with tracker.track_stream():
                    await asyncio.sleep(delay)
                    finished_order.append(label)

            stream1 = asyncio.create_task(simulate_stream(0.1, "fast"))
            stream2 = asyncio.create_task(simulate_stream(0.3, "slow"))

            await asyncio.sleep(0.01)
            assert tracker.active_count == 2

            tracker.begin_shutdown()
            await tracker.wait_for_drain()

            assert tracker.active_count == 0
            assert "fast" in finished_order
            assert "slow" in finished_order

        asyncio.run(run())

    def test_wait_for_drain_respects_timeout(self) -> None:
        async def run() -> None:
            tracker = ActiveStreamTracker(drain_timeout=0.2)

            async def stuck_stream() -> None:
                async with tracker.track_stream():
                    await asyncio.sleep(10.0)

            task = asyncio.create_task(stuck_stream())
            await asyncio.sleep(0.01)
            assert tracker.active_count == 1

            tracker.begin_shutdown()
            await tracker.wait_for_drain()

            assert tracker.active_count == 1
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass

        asyncio.run(run())

    def test_acquire_is_visible_before_generator_iterates(self) -> None:
        """Ensure eager acquire() is seen by drain even if the generator hasn't started."""

        async def run() -> None:
            tracker = ActiveStreamTracker(drain_timeout=1.0)

            await tracker.acquire()
            assert tracker.active_count == 1

            tracker.begin_shutdown()
            drained = asyncio.Event()

            async def drain() -> None:
                await tracker.wait_for_drain()
                drained.set()

            drain_task = asyncio.create_task(drain())
            await asyncio.sleep(0.05)
            assert not drained.is_set(), "drain should block while acquire is held"

            await tracker.release()
            await asyncio.wait_for(drain_task, timeout=1.0)
            assert drained.is_set()

        asyncio.run(run())
