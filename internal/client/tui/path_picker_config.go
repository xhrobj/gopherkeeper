package tui

func configBrowseTarget(focus configFocus) (pathPickerTarget, bool) {
	for target, mode := range pathPickerModes {
		if mode.browseFocus != 0 && focus == mode.browseFocus {
			return target, true
		}
	}

	return 0, false
}

func configTargetFieldFocus(target pathPickerTarget) configFocus {
	return target.mode().configFocus
}

func configTargetFieldIndex(target pathPickerTarget) int {
	return target.mode().configIndex
}
