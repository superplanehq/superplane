import {isScrollable as $cc38e7bd3fc7b213$export$2bb74740c4e19def} from "./isScrollable.mjs";

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
function $62d8ded9296f3872$export$cfa2225e87938781(node, checkForOverflow) {
    let scrollableNode = node;
    if ((0, $cc38e7bd3fc7b213$export$2bb74740c4e19def)(scrollableNode, checkForOverflow)) scrollableNode = scrollableNode.parentElement;
    while(scrollableNode && !(0, $cc38e7bd3fc7b213$export$2bb74740c4e19def)(scrollableNode, checkForOverflow))scrollableNode = scrollableNode.parentElement;
    return scrollableNode || document.scrollingElement || document.documentElement;
}


export {$62d8ded9296f3872$export$cfa2225e87938781 as getScrollParent};
//# sourceMappingURL=getScrollParent.module.js.map
