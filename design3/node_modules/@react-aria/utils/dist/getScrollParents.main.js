var $9a54e9cd5db23b5d$exports = require("./isScrollable.main.js");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "getScrollParents", () => $49f0d9486c2408aa$export$94ed1c92c7beeb22);
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
function $49f0d9486c2408aa$export$94ed1c92c7beeb22(node, checkForOverflow) {
    const scrollParents = [];
    while(node && node !== document.documentElement){
        if ((0, $9a54e9cd5db23b5d$exports.isScrollable)(node, checkForOverflow)) scrollParents.push(node);
        node = node.parentElement;
    }
    return scrollParents;
}


//# sourceMappingURL=getScrollParents.main.js.map
