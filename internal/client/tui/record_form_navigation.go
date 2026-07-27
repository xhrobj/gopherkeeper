package tui

func (form *recordForm) move(step int) {
	controls := form.controls()
	if len(controls) == 0 {
		form.focus = 0
		return
	}
	form.focus = (form.focus + step + len(controls)) % len(controls)
	form.moveCursorToEnd()
}

func (form *recordForm) setFocus(control recordFormControl) {
	for index, candidate := range form.controls() {
		if candidate == control {
			form.focus = index
			form.moveCursorToEnd()
			return
		}
	}
}

func (form *recordForm) field(control recordFormControl) *textField {
	switch control {
	case recordFormTitle:
		return &form.title
	case recordFormLogin:
		return &form.login
	case recordFormPassword:
		return &form.password
	case recordFormURL:
		return &form.url
	case recordFormNumber:
		return &form.number
	case recordFormCardholder:
		return &form.cardholder
	case recordFormExpiryMonth:
		return &form.expiryMonth
	case recordFormExpiryYear:
		return &form.expiryYear
	case recordFormCVV:
		return &form.cvv
	case recordFormMetadata:
		return &form.metadata
	default:
		return nil
	}
}

func (form *recordForm) activeField() *textField {
	return form.field(form.activeControl())
}

func (form recordForm) textAreaActive() bool {
	return form.activeControl() == recordFormText
}

func (form *recordForm) insert(value string) {
	if form.textAreaActive() {
		form.mutateText(func() { form.text.insert(value) })
		return
	}

	if field := form.activeField(); field != nil {
		field.insert(value)
	}
}

func (form *recordForm) insertKey(key string) bool {
	if form.textAreaActive() {
		changed := false
		form.mutateText(func() { changed = form.text.insertKey(key) })
		return changed
	}

	if field := form.activeField(); field != nil {
		return field.insertKey(key)
	}

	return false
}

func (form *recordForm) backspace() {
	if form.textAreaActive() {
		form.mutateText(func() { form.text.backspace() })
		return
	}

	if field := form.activeField(); field != nil {
		field.backspace()
	}
}

func (form *recordForm) delete() {
	if form.textAreaActive() {
		form.mutateText(func() { form.text.delete() })
		return
	}

	if field := form.activeField(); field != nil {
		field.delete()
	}
}

func (form *recordForm) insertNewline() {
	if !form.textAreaActive() {
		return
	}

	form.mutateText(func() { form.text.insertNewline() })
	form.text.ensureVisible(recordTextAreaHeight)
}

func (form *recordForm) mutateText(change func()) {
	before := form.text.value
	change()

	if form.editing && form.text.value != before {
		form.textDirty = true
	}
}

func recordFormControlIsButton(control recordFormControl) bool {
	switch control {
	case recordFormFilePath, recordFormSubmit, recordFormReveal, recordFormCancel:
		return true
	default:
		return false
	}
}

func (form *recordForm) moveCursor(step int) {
	if form.textAreaActive() {
		form.text.moveCursor(step)
		form.text.ensureVisible(recordTextAreaHeight)
		return
	}

	if field := form.activeField(); field != nil {
		field.moveCursor(step)
	}
}

func (form *recordForm) moveLine(step int) {
	if form.textAreaActive() {
		form.text.moveLine(step)
		form.text.ensureVisible(recordTextAreaHeight)
		return
	}

	form.move(step)
}

func (form *recordForm) movePage(step int) {
	if form.textAreaActive() {
		form.text.movePage(step, recordTextAreaHeight)
		form.text.ensureVisible(recordTextAreaHeight)
	}
}

func (form *recordForm) moveCursorToStart() {
	if form.textAreaActive() {
		form.text.moveCursorToLineStart()
		return
	}

	if field := form.activeField(); field != nil {
		field.moveCursorToStart()
	}
}

func (form *recordForm) moveCursorToEnd() {
	if form.textAreaActive() {
		form.text.moveCursorToLineEnd()
		form.text.ensureVisible(recordTextAreaHeight)
		return
	}

	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}
