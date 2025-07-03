var $e77252a287ef94ab$exports = require("./useFocusVisible.main.js");
var $fCG8z$reactariautils = require("@react-aria/utils");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "focusSafely", () => $2833058fcd3993f5$export$80f3e147d781571c);
/*
 * Copyright 2020 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the 'License');
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an 'AS IS' BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 

function $2833058fcd3993f5$export$80f3e147d781571c(element) {
    // If the user is interacting with a virtual cursor, e.g. screen reader, then
    // wait until after any animated transitions that are currently occurring on
    // the page before shifting focus. This avoids issues with VoiceOver on iOS
    // causing the page to scroll when moving focus if the element is transitioning
    // from off the screen.
    const ownerDocument = (0, $fCG8z$reactariautils.getOwnerDocument)(element);
    const activeElement = (0, $fCG8z$reactariautils.getActiveElement)(ownerDocument);
    if ((0, $e77252a287ef94ab$exports.getInteractionModality)() === 'virtual') {
        let lastFocusedElement = activeElement;
        (0, $fCG8z$reactariautils.runAfterTransition)(()=>{
            // If focus did not move and the element is still in the document, focus it.
            if ((0, $fCG8z$reactariautils.getActiveElement)(ownerDocument) === lastFocusedElement && element.isConnected) (0, $fCG8z$reactariautils.focusWithoutScrolling)(element);
        });
    } else (0, $fCG8z$reactariautils.focusWithoutScrolling)(element);
}


//# sourceMappingURL=focusSafely.main.js.map
