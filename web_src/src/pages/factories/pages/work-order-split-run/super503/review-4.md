· [web_src/src/hooks/useSpokenPhraseDictation.ts](https://github.com/superplanehq/superplane/pull/7771#discussion_r4082508398)
<a href="#"><img alt="P1" src="https://greptile-static-assets.s3.amazonaws.com/badges/p1.svg?v=9" align="top"></a> **Partial finalization duplicates pending words**

A final result does not necessarily finish the entire saved interim phrase. For example, let the description receive interim results `"hello "` and `"world"`, then switch to the title. If the next event finalizes `"hello"` but leaves `"world"` interim, `finishSegment()` advances the counter. Returning to the description commits all of `"hello world"`, so the next interim `"world today"` produces `"Notes hello world world today"` instead of replacing the pending words.

Track completion of individual recognition results so that restoring a snapshot commits only completed words and keeps the remaining words replaceable. Add a regression test with mixed final and interim results across a field switch.
