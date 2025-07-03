var _curry2 =
/*#__PURE__*/
require("./internal/_curry2.js");

var _filter =
/*#__PURE__*/
require("./internal/_filter.js");

var _Set =
/*#__PURE__*/
require("./internal/_Set.js");

var uniq =
/*#__PURE__*/
require("./uniq.js");
/**
 * Combines two lists into a set (i.e. no duplicates) composed of those
 * elements common to both lists.
 *
 * @func
 * @memberOf R
 * @since v0.1.0
 * @category Relation
 * @sig [*] -> [*] -> [*]
 * @param {Array} list1 The first list.
 * @param {Array} list2 The second list.
 * @return {Array} The list of elements found in both `list1` and `list2`.
 * @see R.innerJoin
 * @example
 *
 *      R.intersection([1,2,3,4], [7,6,5,4,3]); //=> [4, 3]
 */


var intersection =
/*#__PURE__*/
_curry2(function intersection(list1, list2) {
  var toKeep = new _Set();

  for (var i = 0; i < list1.length; i += 1) {
    toKeep.add(list1[i]);
  }

  return uniq(_filter(toKeep.has.bind(toKeep), list2));
});

module.exports = intersection;