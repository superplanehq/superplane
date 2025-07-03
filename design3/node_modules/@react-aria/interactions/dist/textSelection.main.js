var $20aJV$reactariautils = require("@react-aria/utils");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "disableTextSelection", () => $f7e14e656343df57$export$16a4697467175487);
$parcel$export(module.exports, "restoreTextSelection", () => $f7e14e656343df57$export$b0d6fa1ab32e3295);
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
// Note that state only matters here for iOS. Non-iOS gets user-select: none applied to the target element
// rather than at the document level so we just need to apply/remove user-select: none for each pressed element individually
let $f7e14e656343df57$var$state = 'default';
let $f7e14e656343df57$var$savedUserSelect = '';
let $f7e14e656343df57$var$modifiedElementMap = new WeakMap();
function $f7e14e656343df57$export$16a4697467175487(target) {
    if ((0, $20aJV$reactariautils.isIOS)()) {
        if ($f7e14e656343df57$var$state === 'default') {
            const documentObject = (0, $20aJV$reactariautils.getOwnerDocument)(target);
            $f7e14e656343df57$var$savedUserSelect = documentObject.documentElement.style.webkitUserSelect;
            documentObject.documentElement.style.webkitUserSelect = 'none';
        }
        $f7e14e656343df57$var$state = 'disabled';
    } else if (target instanceof HTMLElement || target instanceof SVGElement) {
        // If not iOS, store the target's original user-select and change to user-select: none
        // Ignore state since it doesn't apply for non iOS
        let property = 'userSelect' in target.style ? 'userSelect' : 'webkitUserSelect';
        $f7e14e656343df57$var$modifiedElementMap.set(target, target.style[property]);
        target.style[property] = 'none';
    }
}
function $f7e14e656343df57$export$b0d6fa1ab32e3295(target) {
    if ((0, $20aJV$reactariautils.isIOS)()) {
        // If the state is already default, there's nothing to do.
        // If it is restoring, then there's no need to queue a second restore.
        if ($f7e14e656343df57$var$state !== 'disabled') return;
        $f7e14e656343df57$var$state = 'restoring';
        // There appears to be a delay on iOS where selection still might occur
        // after pointer up, so wait a bit before removing user-select.
        setTimeout(()=>{
            // Wait for any CSS transitions to complete so we don't recompute style
            // for the whole page in the middle of the animation and cause jank.
            (0, $20aJV$reactariautils.runAfterTransition)(()=>{
                // Avoid race conditions
                if ($f7e14e656343df57$var$state === 'restoring') {
                    const documentObject = (0, $20aJV$reactariautils.getOwnerDocument)(target);
                    if (documentObject.documentElement.style.webkitUserSelect === 'none') documentObject.documentElement.style.webkitUserSelect = $f7e14e656343df57$var$savedUserSelect || '';
                    $f7e14e656343df57$var$savedUserSelect = '';
                    $f7e14e656343df57$var$state = 'default';
                }
            });
        }, 300);
    } else if (target instanceof HTMLElement || target instanceof SVGElement) // If not iOS, restore the target's original user-select if any
    // Ignore state since it doesn't apply for non iOS
    {
        if (target && $f7e14e656343df57$var$modifiedElementMap.has(target)) {
            let targetOldUserSelect = $f7e14e656343df57$var$modifiedElementMap.get(target);
            let property = 'userSelect' in target.style ? 'userSelect' : 'webkitUserSelect';
            if (target.style[property] === 'none') target.style[property] = targetOldUserSelect;
            if (target.getAttribute('style') === '') target.removeAttribute('style');
            $f7e14e656343df57$var$modifiedElementMap.delete(target);
        }
    }
}


//# sourceMappingURL=textSelection.main.js.map
