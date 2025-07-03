var $1254e5bb94ac8761$exports = require("./useEffectEvent.main.js");
var $81vbz$react = require("react");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useFormReset", () => $1f205e845604a423$export$5add1d006293d136);
/*
 * Copyright 2023 Adobe. All rights reserved.
 * This file is licensed to you under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License. You may obtain a copy
 * of the License at http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR REPRESENTATIONS
 * OF ANY KIND, either express or implied. See the License for the specific language
 * governing permissions and limitations under the License.
 */ 

function $1f205e845604a423$export$5add1d006293d136(ref, initialValue, onReset) {
    let resetValue = (0, $81vbz$react.useRef)(initialValue);
    let handleReset = (0, $1254e5bb94ac8761$exports.useEffectEvent)(()=>{
        if (onReset) onReset(resetValue.current);
    });
    (0, $81vbz$react.useEffect)(()=>{
        var _ref_current;
        let form = ref === null || ref === void 0 ? void 0 : (_ref_current = ref.current) === null || _ref_current === void 0 ? void 0 : _ref_current.form;
        form === null || form === void 0 ? void 0 : form.addEventListener('reset', handleReset);
        return ()=>{
            form === null || form === void 0 ? void 0 : form.removeEventListener('reset', handleReset);
        };
    }, [
        ref,
        handleReset
    ]);
}


//# sourceMappingURL=useFormReset.main.js.map
