var $4gIVO$react = require("react");
var $4gIVO$reactariassr = require("@react-aria/ssr");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useViewportSize", () => $8b24bab62f5c65ad$export$d699905dd57c73ca);
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

let $8b24bab62f5c65ad$var$visualViewport = typeof document !== 'undefined' && window.visualViewport;
function $8b24bab62f5c65ad$export$d699905dd57c73ca() {
    let isSSR = (0, $4gIVO$reactariassr.useIsSSR)();
    let [size, setSize] = (0, $4gIVO$react.useState)(()=>isSSR ? {
            width: 0,
            height: 0
        } : $8b24bab62f5c65ad$var$getViewportSize());
    (0, $4gIVO$react.useEffect)(()=>{
        // Use visualViewport api to track available height even on iOS virtual keyboard opening
        let onResize = ()=>{
            setSize((size)=>{
                let newSize = $8b24bab62f5c65ad$var$getViewportSize();
                if (newSize.width === size.width && newSize.height === size.height) return size;
                return newSize;
            });
        };
        if (!$8b24bab62f5c65ad$var$visualViewport) window.addEventListener('resize', onResize);
        else $8b24bab62f5c65ad$var$visualViewport.addEventListener('resize', onResize);
        return ()=>{
            if (!$8b24bab62f5c65ad$var$visualViewport) window.removeEventListener('resize', onResize);
            else $8b24bab62f5c65ad$var$visualViewport.removeEventListener('resize', onResize);
        };
    }, []);
    return size;
}
function $8b24bab62f5c65ad$var$getViewportSize() {
    return {
        width: $8b24bab62f5c65ad$var$visualViewport && ($8b24bab62f5c65ad$var$visualViewport === null || $8b24bab62f5c65ad$var$visualViewport === void 0 ? void 0 : $8b24bab62f5c65ad$var$visualViewport.width) || window.innerWidth,
        height: $8b24bab62f5c65ad$var$visualViewport && ($8b24bab62f5c65ad$var$visualViewport === null || $8b24bab62f5c65ad$var$visualViewport === void 0 ? void 0 : $8b24bab62f5c65ad$var$visualViewport.height) || window.innerHeight
    };
}


//# sourceMappingURL=useViewportSize.main.js.map
