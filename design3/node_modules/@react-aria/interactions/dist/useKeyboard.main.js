var $951fbcbbca8db6ce$exports = require("./createEventHandler.main.js");


function $parcel$export(e, n, v, s) {
  Object.defineProperty(e, n, {get: v, set: s, enumerable: true, configurable: true});
}

$parcel$export(module.exports, "useKeyboard", () => $892d64db2a3c53b0$export$8f71654801c2f7cd);
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
function $892d64db2a3c53b0$export$8f71654801c2f7cd(props) {
    return {
        keyboardProps: props.isDisabled ? {} : {
            onKeyDown: (0, $951fbcbbca8db6ce$exports.createEventHandler)(props.onKeyDown),
            onKeyUp: (0, $951fbcbbca8db6ce$exports.createEventHandler)(props.onKeyUp)
        }
    };
}


//# sourceMappingURL=useKeyboard.main.js.map
