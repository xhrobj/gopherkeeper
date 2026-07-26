package tui

// updateRecordForm обрабатывает общий ввод формы записи.
// Возвращает true, если пользователь активировал текущую кнопку клавишей Enter.
func updateRecordForm(form *recordForm, key string) bool {
	switch key {
	case "tab":
		form.move(1)
	case "shift+tab":
		form.move(-1)
	case "up":
		form.moveLine(-1)
	case "down":
		form.moveLine(1)
	case "left":
		if recordFormControlIsButton(form.activeControl()) {
			form.move(-1)
		} else {
			form.moveCursor(-1)
		}
	case "right":
		if recordFormControlIsButton(form.activeControl()) {
			form.move(1)
		} else {
			form.moveCursor(1)
		}
	case "home":
		form.moveCursorToStart()
	case "end":
		form.moveCursorToEnd()
	case "pgup":
		form.movePage(-1)
	case "pgdown":
		form.movePage(1)
	case "backspace":
		form.backspace()
	case "delete":
		form.delete()
	case "enter":
		if form.textAreaActive() {
			form.insertNewline()
			return false
		}
		return true
	default:
		form.insertKey(key)
	}

	return false
}
