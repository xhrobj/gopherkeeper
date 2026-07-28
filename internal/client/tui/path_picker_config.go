package tui

type configPathPickerBinding struct {
	target      pathPickerTarget
	fieldFocus  configFocus
	browseFocus configFocus
}

var configPathPickerBindings = [...]configPathPickerBinding{
	{target: pathPickerCACert, fieldFocus: configCACertFile, browseFocus: configCACertBrowse},
	{target: pathPickerSessionDirectory, fieldFocus: configSessionDir, browseFocus: configSessionBrowse},
	{target: pathPickerCacheDirectory, fieldFocus: configCacheDir, browseFocus: configCacheBrowse},
}

func configBrowseTarget(focus configFocus) (pathPickerTarget, bool) {
	for _, binding := range configPathPickerBindings {
		if focus == binding.browseFocus {
			return binding.target, true
		}
	}

	return 0, false
}

func configTargetFieldFocus(target pathPickerTarget) (configFocus, bool) {
	for _, binding := range configPathPickerBindings {
		if target == binding.target {
			return binding.fieldFocus, true
		}
	}

	return 0, false
}
