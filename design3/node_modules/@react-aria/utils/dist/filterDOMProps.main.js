
function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "filterDOMProps", () => $8d15d0e1797d4238$export$457c3d6518dd4c6f);
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
 */ const $8d15d0e1797d4238$var$DOMPropNames = new Set([
    'id'
]);
const $8d15d0e1797d4238$var$labelablePropNames = new Set([
    'aria-label',
    'aria-labelledby',
    'aria-describedby',
    'aria-details'
]);
// See LinkDOMProps in dom.d.ts.
const $8d15d0e1797d4238$var$linkPropNames = new Set([
    'href',
    'hrefLang',
    'target',
    'rel',
    'download',
    'ping',
    'referrerPolicy'
]);
const $8d15d0e1797d4238$var$propRe = /^(data-.*)$/;
function $8d15d0e1797d4238$export$457c3d6518dd4c6f(props, opts = {}) {
    let { labelable: labelable, isLink: isLink, propNames: propNames } = opts;
    let filteredProps = {};
    for(const prop in props)if (Object.prototype.hasOwnProperty.call(props, prop) && ($8d15d0e1797d4238$var$DOMPropNames.has(prop) || labelable && $8d15d0e1797d4238$var$labelablePropNames.has(prop) || isLink && $8d15d0e1797d4238$var$linkPropNames.has(prop) || (propNames === null || propNames === void 0 ? void 0 : propNames.has(prop)) || $8d15d0e1797d4238$var$propRe.test(prop))) filteredProps[prop] = props[prop];
    return filteredProps;
}


//# sourceMappingURL=filterDOMProps.main.js.map
