define(['./_setup', './_unmethodize'], function (_setup, _unmethodize) {

	var apply = _unmethodize(_setup.apply);

	return apply;

});
