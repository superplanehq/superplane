// Internal helper that wraps an `iteratee` to call it with the
// property of a closed-over `object`. Useful when iterating over
// an array of keys of `object`.
function applyProperty(iteratee, object) {
  return function(key) {
    return iteratee(object[key], key, object);
  };
}

module.exports = applyProperty;
