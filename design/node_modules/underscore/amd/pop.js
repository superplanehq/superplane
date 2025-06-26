define(['./_unmethodize', './_setup'], function (_unmethodize, _setup) {

	var pop = _unmethodize(_setup.ArrayProto.pop);

	return pop;

});
