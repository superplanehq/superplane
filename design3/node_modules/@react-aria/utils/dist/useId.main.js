var $78605a5d7424e31b$exports = require("./useLayoutEffect.main.js");
var $19a2307bfabafaf1$exports = require("./useValueEffect.main.js");
var $dG5aF$react = require("react");
var $dG5aF$reactariassr = require("@react-aria/ssr");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useId", () => $8c61827343eed941$export$f680877a34711e37);
$parcel$export(module.exports, "mergeIds", () => $8c61827343eed941$export$cd8c9cb68f842629);
$parcel$export(module.exports, "useSlotId", () => $8c61827343eed941$export$b4cc09c592e8fdb8);
/*
 * Copyright 2020 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 



// copied from SSRProvider.tsx to reduce exports, if needed again, consider sharing
let $8c61827343eed941$var$canUseDOM = Boolean(typeof window !== 'undefined' && window.document && window.document.createElement);
let $8c61827343eed941$export$d41a04c74483c6ef = new Map();
// This allows us to clean up the idsUpdaterMap when the id is no longer used.
// Map is a strong reference, so unused ids wouldn't be cleaned up otherwise.
// This can happen in suspended components where mount/unmount is not called.
let $8c61827343eed941$var$registry;
if (typeof FinalizationRegistry !== 'undefined') $8c61827343eed941$var$registry = new FinalizationRegistry((heldValue)=>{
    $8c61827343eed941$export$d41a04c74483c6ef.delete(heldValue);
});
function $8c61827343eed941$export$f680877a34711e37(defaultId) {
    let [value, setValue] = (0, $dG5aF$react.useState)(defaultId);
    let nextId = (0, $dG5aF$react.useRef)(null);
    let res = (0, $dG5aF$reactariassr.useSSRSafeId)(value);
    let cleanupRef = (0, $dG5aF$react.useRef)(null);
    if ($8c61827343eed941$var$registry) $8c61827343eed941$var$registry.register(cleanupRef, res);
    if ($8c61827343eed941$var$canUseDOM) {
        const cacheIdRef = $8c61827343eed941$export$d41a04c74483c6ef.get(res);
        if (cacheIdRef && !cacheIdRef.includes(nextId)) cacheIdRef.push(nextId);
        else $8c61827343eed941$export$d41a04c74483c6ef.set(res, [
            nextId
        ]);
    }
    (0, $78605a5d7424e31b$exports.useLayoutEffect)(()=>{
        let r = res;
        return ()=>{
            // In Suspense, the cleanup function may be not called
            // when it is though, also remove it from the finalization registry.
            if ($8c61827343eed941$var$registry) $8c61827343eed941$var$registry.unregister(cleanupRef);
            $8c61827343eed941$export$d41a04c74483c6ef.delete(r);
        };
    }, [
        res
    ]);
    // This cannot cause an infinite loop because the ref is always cleaned up.
    // eslint-disable-next-line
    (0, $dG5aF$react.useEffect)(()=>{
        let newId = nextId.current;
        if (newId) setValue(newId);
        return ()=>{
            if (newId) nextId.current = null;
        };
    });
    return res;
}
function $8c61827343eed941$export$cd8c9cb68f842629(idA, idB) {
    if (idA === idB) return idA;
    let setIdsA = $8c61827343eed941$export$d41a04c74483c6ef.get(idA);
    if (setIdsA) {
        setIdsA.forEach((ref)=>ref.current = idB);
        return idB;
    }
    let setIdsB = $8c61827343eed941$export$d41a04c74483c6ef.get(idB);
    if (setIdsB) {
        setIdsB.forEach((ref)=>ref.current = idA);
        return idA;
    }
    return idB;
}
function $8c61827343eed941$export$b4cc09c592e8fdb8(depArray = []) {
    let id = $8c61827343eed941$export$f680877a34711e37();
    let [resolvedId, setResolvedId] = (0, $19a2307bfabafaf1$exports.useValueEffect)(id);
    let updateId = (0, $dG5aF$react.useCallback)(()=>{
        setResolvedId(function*() {
            yield id;
            yield document.getElementById(id) ? id : undefined;
        });
    }, [
        id,
        setResolvedId
    ]);
    (0, $78605a5d7424e31b$exports.useLayoutEffect)(updateId, [
        id,
        updateId,
        ...depArray
    ]);
    return resolvedId;
}


//# sourceMappingURL=useId.main.js.map
