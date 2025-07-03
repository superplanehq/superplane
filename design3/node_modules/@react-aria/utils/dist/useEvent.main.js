var $1254e5bb94ac8761$exports = require("./useEffectEvent.main.js");
var $8UjJN$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useEvent", () => $2a8c0bb1629926c8$export$90fc3a17d93f704c);
/*
 * Copyright 2021 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 

function $2a8c0bb1629926c8$export$90fc3a17d93f704c(ref, event, handler, options) {
    let handleEvent = (0, $1254e5bb94ac8761$exports.useEffectEvent)(handler);
    let isDisabled = handler == null;
    (0, $8UjJN$react.useEffect)(()=>{
        if (isDisabled || !ref.current) return;
        let element = ref.current;
        element.addEventListener(event, handleEvent, options);
        return ()=>{
            element.removeEventListener(event, handleEvent, options);
        };
    }, [
        ref,
        event,
        options,
        isDisabled,
        handleEvent
    ]);
}


//# sourceMappingURL=useEvent.main.js.map
