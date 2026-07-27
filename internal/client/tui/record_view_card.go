package tui

import (
	"fmt"
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func buildCardPreviewLines(t theme, payload *recordmodel.CardPayload, reveal bool, width int) []recordViewLine {
	cardholder := strings.ToUpper(strings.TrimSpace(payload.Cardholder))
	expiry := ""
	if payload.ExpiryMonth != nil && payload.ExpiryYear != nil {
		expiry = fmt.Sprintf("%02d/%02d", *payload.ExpiryMonth, *payload.ExpiryYear%100)
	}
	cvv := ""
	if payload.CVV != "" {
		cvv = visibleCVV(payload.CVV, reveal)
	}
	number := formattedVisibleCardNumber(payload.Number, reveal)

	cardWidth := clamp(width*2/5, 32, 34)
	stripeFillWidth := max(1, cardWidth-2)
	cardTextWidth := max(1, cardWidth-4)

	cardFace := t.recordCard
	cardLabel := t.recordCardMuted
	cardStripe := t.recordCardStripe

	padToWindow := func(card string) string {
		used := lipgloss.Width(card)
		if used >= width {
			return card
		}
		return card + t.windowBody.Width(width-used).Render("")
	}

	renderCardLine := func(content string) string {
		content = fitSingleLinePreserve(content, cardTextWidth)
		padding := max(0, cardTextWidth-lipgloss.Width(content))
		card := cardFace.Render("  " + content + strings.Repeat(" ", padding) + "  ")
		return padToWindow(card)
	}

	renderBlank := func() string {
		return renderCardLine("")
	}

	renderStripe := func() string {
		content := " " + strings.Repeat("▒", stripeFillWidth) + " "
		card := cardStripe.Render(content)
		return padToWindow(card)
	}

	renderFooter := func() string {
		left := cardLabel.Render("exp") + cardFace.Render(" ") + cardFace.Render(expiry)
		cvvText := fitSingleLinePreserve(cvv, recordmodel.CardCVVSize)
		right := cardFace.Render("[") + cardLabel.Render("cvv") + cardFace.Render(" "+cvvText) + cardFace.Render("]")
		used := lipgloss.Width(left) + lipgloss.Width(right)
		gap := max(1, cardTextWidth-used)
		card := cardFace.Render("  ") + left + cardFace.Render(strings.Repeat(" ", gap)) + right + cardFace.Render("  ")
		return padToWindow(card)
	}

	return []recordViewLine{
		{rendered: renderBlank()},
		{rendered: renderStripe()},
		{rendered: renderBlank()},
		{rendered: renderCardLine(number)},
		{rendered: renderBlank()},
		{rendered: renderCardLine(cardholder)},
		{rendered: renderBlank()},
		{rendered: renderFooter()},
		{rendered: renderBlank()},
		{rendered: renderBlank()},
	}
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	return strings.Repeat("•", max(4, len([]rune(value))))
}

func visibleSecret(value string, reveal bool) string {
	if reveal {
		return value
	}
	return maskSecret(value)
}

func visibleCVV(value string, reveal bool) string {
	if reveal {
		return value
	}

	if value == "" {
		return ""
	}

	return strings.Repeat("•", len([]rune(value)))
}

func formattedVisibleCardNumber(value string, reveal bool) string {
	clean := cardDigits(value)

	if clean == "" {
		return ""
	}

	if !reveal {
		clean = maskCardNumber(clean)
	}

	return groupCardDigits(clean)
}

func cardDigits(value string) string {
	var b strings.Builder

	for _, r := range value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}

func groupCardDigits(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	parts := make([]string, 0, (len(runes)+3)/4)

	for len(runes) > 0 {
		chunk := min(4, len(runes))
		parts = append(parts, string(runes[:chunk]))
		runes = runes[chunk:]
	}

	return strings.Join(parts, " ")
}

func maskCardNumber(value string) string {
	r := []rune(value)
	digits := 0

	for _, c := range r {
		if unicode.IsDigit(c) {
			digits++
		}
	}

	keep := 4
	seen := 0
	out := make([]rune, len(r))

	for i, c := range r {
		if unicode.IsDigit(c) {
			seen++
			if seen <= digits-keep {
				out[i] = '•'
			} else {
				out[i] = c
			}
		} else {
			out[i] = c
		}
	}

	return string(out)
}
