var $581a96d6eb128c1b$exports = require("./useFocusRing.main.js");
var $hE1Ku$clsx = require("clsx");
var $hE1Ku$reactariautils = require("@react-aria/utils");
var $hE1Ku$react = require("react");


function $parcel$interopDefault(a) {
  return a && a.__esModule ? a.default : a;
}

function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "FocusRing", () => $dfd8c70b928eb1b3$export$1a38b4ad7f578e1d);
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



function $dfd8c70b928eb1b3$export$1a38b4ad7f578e1d(props) {
    let { children: children, focusClass: focusClass, focusRingClass: focusRingClass } = props;
    let { isFocused: isFocused, isFocusVisible: isFocusVisible, focusProps: focusProps } = (0, $581a96d6eb128c1b$exports.useFocusRing)(props);
    let child = (0, ($parcel$interopDefault($hE1Ku$react))).Children.only(children);
    return /*#__PURE__*/ (0, ($parcel$interopDefault($hE1Ku$react))).cloneElement(child, (0, $hE1Ku$reactariautils.mergeProps)(child.props, {
        ...focusProps,
        className: (0, ($parcel$interopDefault($hE1Ku$clsx)))({
            [focusClass || '']: isFocused,
            [focusRingClass || '']: isFocusVisible
        })
    }));
}


//# sourceMappingURL=FocusRing.main.js.map
