var $1254e5bb94ac8761$exports = require("./useEffectEvent.main.js");
var $78605a5d7424e31b$exports = require("./useLayoutEffect.main.js");
var $fymK9$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useValueEffect", () => $19a2307bfabafaf1$export$14d238f342723f25);
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

function $19a2307bfabafaf1$export$14d238f342723f25(defaultValue) {
    let [value, setValue] = (0, $fymK9$react.useState)(defaultValue);
    let effect = (0, $fymK9$react.useRef)(null);
    // Store the function in a ref so we can always access the current version
    // which has the proper `value` in scope.
    let nextRef = (0, $1254e5bb94ac8761$exports.useEffectEvent)(()=>{
        if (!effect.current) return;
        // Run the generator to the next yield.
        let newValue = effect.current.next();
        // If the generator is done, reset the effect.
        if (newValue.done) {
            effect.current = null;
            return;
        }
        // If the value is the same as the current value,
        // then continue to the next yield. Otherwise,
        // set the value in state and wait for the next layout effect.
        if (value === newValue.value) nextRef();
        else setValue(newValue.value);
    });
    (0, $78605a5d7424e31b$exports.useLayoutEffect)(()=>{
        // If there is an effect currently running, continue to the next yield.
        if (effect.current) nextRef();
    });
    let queue = (0, $1254e5bb94ac8761$exports.useEffectEvent)((fn)=>{
        effect.current = fn(value);
        nextRef();
    });
    return [
        value,
        queue
    ];
}


//# sourceMappingURL=useValueEffect.main.js.map
