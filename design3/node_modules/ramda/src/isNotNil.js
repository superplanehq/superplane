var isNil =
/*#__PURE__*/
require("./isNil.js");

var _curry1 =
/*#__PURE__*/
require("./internal/_curry1.js");
/**
 * Checks if the input value is not `null` and not `undefined`.
 *
 * @func
 * @memberOf R
 * @since v0.29.0
 * @category Type
 * @sig * -> Boolean
 * @param {*} x The value to test.
 * @return {Boolean} `true` if `x` is not `undefined` or not `null`, otherwise `false`.
 * @example
 *
 *      R.isNotNil(null); //=> false
 *      R.isNotNil(undefined); //=> false
 *      R.isNotNil(0); //=> true
 *      R.isNotNil([]); //=> true
 */


var isNotNil =
/*#__PURE__*/
_curry1(function isNotNil(x) {
  return !isNil(x);
});

module.exports = isNotNil;