import {getScrollParent as $62d8ded9296f3872$export$cfa2225e87938781} from "./getScrollParent.mjs";
import {useEffectEvent as $8ae05eaa5c114e9c$export$7f54fc3180508a52} from "./useEffectEvent.mjs";
import {useLayoutEffect as $f0a04ccd8dbdd83b$export$e5c5a5f917a5871c} from "./useLayoutEffect.mjs";
import {useRef as $7FoZl$useRef} from "react";

/*
 * Copyright 2024 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 



function $a5fa973c1850dd36$export$90a12e6abf95cbe0(props, ref) {
    let { collection: collection, onLoadMore: onLoadMore, scrollOffset: scrollOffset = 1 } = props;
    let sentinelObserver = (0, $7FoZl$useRef)(null);
    let triggerLoadMore = (0, $8ae05eaa5c114e9c$export$7f54fc3180508a52)((entries)=>{
        // Use "isIntersecting" over an equality check of 0 since it seems like there is cases where
        // a intersection ratio of 0 can be reported when isIntersecting is actually true
        for (let entry of entries)// Note that this will be called if the collection changes, even if onLoadMore was already called and is being processed.
        // Up to user discretion as to how to handle these multiple onLoadMore calls
        if (entry.isIntersecting && onLoadMore) onLoadMore();
    });
    (0, $f0a04ccd8dbdd83b$export$e5c5a5f917a5871c)(()=>{
        if (ref.current) {
            // Tear down and set up a new IntersectionObserver when the collection changes so that we can properly trigger additional loadMores if there is room for more items
            // Need to do this tear down and set up since using a large rootMargin will mean the observer's callback isn't called even when scrolling the item into view beause its visibility hasn't actually changed
            // https://codesandbox.io/p/sandbox/magical-swanson-dhgp89?file=%2Fsrc%2FApp.js%3A21%2C21
            sentinelObserver.current = new IntersectionObserver(triggerLoadMore, {
                root: (0, $62d8ded9296f3872$export$cfa2225e87938781)(ref === null || ref === void 0 ? void 0 : ref.current),
                rootMargin: `0px ${100 * scrollOffset}% ${100 * scrollOffset}% ${100 * scrollOffset}%`
            });
            sentinelObserver.current.observe(ref.current);
        }
        return ()=>{
            if (sentinelObserver.current) sentinelObserver.current.disconnect();
        };
    }, [
        collection,
        triggerLoadMore,
        ref,
        scrollOffset
    ]);
}


export {$a5fa973c1850dd36$export$90a12e6abf95cbe0 as UNSTABLE_useLoadMoreSentinel};
//# sourceMappingURL=useLoadMoreSentinel.module.js.map
