var _concat =
/*#__PURE__*/
require("./internal/_concat.js");

var _curry1 =
/*#__PURE__*/
require("./internal/_curry1.js");

var curryN =
/*#__PURE__*/
require("./curryN.js");
/**
 * As with `addIndex`, `addIndexRight` creates a new list iteration function
 * from an existing one by adding two new parameters to its callback function:
 * the current index, and the entire list.
 *
 * Unlike `addIndex`, `addIndexRight` iterates from the right to the left.
 *
 * @func
 * @memberOf R
 * @since v0.29.0
 * @category Function
 * @category List
 * @sig ((a ... -> b) ... -> [a] -> *) -> (a ..., Int, [a] -> b) ... -> [a] -> *)
 * @param {Function} fn A list iteration function that does not pass index or list to its callback
 * @return {Function} An altered list iteration function that passes (item, index, list) to its callback
 * @example
 *
 *      const revmap = (fn, ary) => R.map(fn, R.reverse(ary));
 *      const revmapIndexed = R.addIndexRight(revmap);
 *      revmapIndexed((val, idx) => idx + '-' + val, ['f', 'o', 'o', 'b', 'a', 'r']);
 *      //=> [ '5-r', '4-a', '3-b', '2-o', '1-o', '0-f' ]
 */


var addIndexRight =
/*#__PURE__*/
_curry1(function addIndex(fn) {
  return curryN(fn.length, function () {
    var origFn = arguments[0];
    var list = arguments[arguments.length - 1];
    var idx = list.length - 1;
    var args = Array.prototype.slice.call(arguments, 0);

    args[0] = function () {
      var result = origFn.apply(this, _concat(arguments, [idx, list]));
      idx -= 1;
      return result;
    };

    return fn.apply(this, args);
  });
});

module.exports = addIndexRight;