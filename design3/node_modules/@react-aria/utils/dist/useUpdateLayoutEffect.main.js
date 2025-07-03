var $78605a5d7424e31b$exports = require("./useLayoutEffect.main.js");
var $ah9Dz$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useUpdateLayoutEffect", () => $0fa310503218f75f$export$72ef708ab07251f1);
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

function $0fa310503218f75f$export$72ef708ab07251f1(effect, dependencies) {
    const isInitialMount = (0, $ah9Dz$react.useRef)(true);
    const lastDeps = (0, $ah9Dz$react.useRef)(null);
    (0, $78605a5d7424e31b$exports.useLayoutEffect)(()=>{
        isInitialMount.current = true;
        return ()=>{
            isInitialMount.current = false;
        };
    }, []);
    (0, $78605a5d7424e31b$exports.useLayoutEffect)(()=>{
        if (isInitialMount.current) isInitialMount.current = false;
        else if (!lastDeps.current || dependencies.some((dep, i)=>!Object.is(dep, lastDeps[i]))) effect();
        lastDeps.current = dependencies;
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, dependencies);
}


//# sourceMappingURL=useUpdateLayoutEffect.main.js.map
