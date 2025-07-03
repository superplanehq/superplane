var _curry2 =
/*#__PURE__*/
require("./internal/_curry2.js");

var ap =
/*#__PURE__*/
require("./ap.js");

var map =
/*#__PURE__*/
require("./map.js");

var prepend =
/*#__PURE__*/
require("./prepend.js");

var reduceRight =
/*#__PURE__*/
require("./reduceRight.js");

var identity =
/*#__PURE__*/
require("./internal/_identity.js");
/**
 * Transforms a [Traversable](https://github.com/fantasyland/fantasy-land#traversable)
 * of [Applicative](https://github.com/fantasyland/fantasy-land#applicative) into an
 * Applicative of Traversable.
 *
 * Dispatches to the `"fantasy-land/traverse"` or the `traverse` method of the second argument, if present.
 *
 * @func
 * @memberOf R
 * @since v0.19.0
 * @category List
 * @sig fantasy-land/of :: TypeRep f => f ~> a -> f a
 * @sig (Applicative f, Traversable t) => TypeRep f -> t (f a) -> f (t a)
 * @sig (Applicative f, Traversable t) => (a -> f a) -> t (f a) -> f (t a)
 * @param {Object|Function} TypeRepresentative with an `of` or `fantasy-land/of` method
 * @param {*} traversable
 * @return {*}
 * @see R.traverse
 * @example
 *
 *      R.sequence(Maybe.of, [Just(1), Just(2), Just(3)]);   //=> Just([1, 2, 3])
 *      R.sequence(Maybe.of, [Just(1), Just(2), Nothing()]); //=> Nothing()
 *
 *      R.sequence(R.of(Array), Just([1, 2, 3])); //=> [Just(1), Just(2), Just(3)]
 *      R.sequence(R.of(Array), Nothing());       //=> [Nothing()]
 */


var sequence =
/*#__PURE__*/
_curry2(function sequence(F, traversable) {
  var of = typeof F['fantasy-land/of'] === 'function' ? F['fantasy-land/of'] : typeof F.of === 'function' ? F.of : F;
  var TypeRep = {
    'fantasy-land/of': of
  };
  return typeof traversable['fantasy-land/traverse'] === 'function' ? traversable['fantasy-land/traverse'](TypeRep, identity) : typeof traversable.traverse === 'function' ? traversable.traverse(TypeRep, identity) : reduceRight(function (x, acc) {
    return ap(map(prepend, x), acc);
  }, of([]), traversable);
});

module.exports = sequence;