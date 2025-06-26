function sequence(iteratee, start, stop, step) {
  if (stop == null) {
    stop = start || 0;
    start = 0;
  }
  if (!step) {
    step = stop < start ? -1 : 1;
  }
  var rest = (stop - start) % step;
  stop += (rest && step - rest);
  for ( ; start != stop ; start += step) if (iteratee(start)) return start;
}

module.exports = sequence;
