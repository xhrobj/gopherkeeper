package model

import (
	"errors"
	"strings"
	"testing"
)

func TestCardPayload_Validate(t *testing.T) {
	month := 12
	year := 30
	valid := CardPayload{
		Number:      "4111111111111111",
		Cardholder:  "Joel Miller",
		ExpiryMonth: &month,
		ExpiryYear:  &year,
		CVV:         "014",
		Metadata:    "test card",
	}

	intPointer := func(value int) *int { return &value }
	tests := []struct {
		name    string
		mutate  func(*CardPayload)
		wantErr error
	}{
		{name: "valid"},
		{name: "minimal", mutate: func(value *CardPayload) {
			value.Cardholder = ""
			value.ExpiryMonth = nil
			value.ExpiryYear = nil
			value.CVV = ""
			value.Metadata = ""
		}},
		{name: "minimum number length", mutate: func(value *CardPayload) { value.Number = strings.Repeat("1", CardNumberMinSize) }},
		{name: "maximum number length", mutate: func(value *CardPayload) { value.Number = strings.Repeat("9", CardNumberMaxSize) }},
		{name: "zero year", mutate: func(value *CardPayload) { value.ExpiryYear = intPointer(0) }},
		{name: "maximum year", mutate: func(value *CardPayload) { value.ExpiryYear = intPointer(CardExpiryYearMax) }},
		{name: "Unicode cardholder at limit", mutate: func(value *CardPayload) {
			value.Cardholder = strings.Repeat("Я", CardholderMaxSize)
			value.Metadata = strings.Repeat("界", MetadataMaxSize)
		}},
		{name: "nil payload", mutate: nil, wantErr: ErrInvalidCardPayload},
		{name: "number too short", mutate: func(value *CardPayload) { value.Number = strings.Repeat("1", CardNumberMinSize-1) }, wantErr: ErrInvalidCardPayload},
		{name: "number too long", mutate: func(value *CardPayload) { value.Number = strings.Repeat("1", CardNumberMaxSize+1) }, wantErr: ErrInvalidCardPayload},
		{name: "number with spaces", mutate: func(value *CardPayload) { value.Number = "4111 1111 1111 1111" }, wantErr: ErrInvalidCardPayload},
		{name: "number with hyphen", mutate: func(value *CardPayload) { value.Number = "4111-1111-1111-1111" }, wantErr: ErrInvalidCardPayload},
		{name: "number with letter", mutate: func(value *CardPayload) { value.Number = "411111111111111A" }, wantErr: ErrInvalidCardPayload},
		{name: "number with Unicode digit", mutate: func(value *CardPayload) { value.Number = "411111111111111١" }, wantErr: ErrInvalidCardPayload},
		{name: "cardholder too long", mutate: func(value *CardPayload) { value.Cardholder = strings.Repeat("Я", CardholderMaxSize+1) }, wantErr: ErrInvalidCardPayload},
		{name: "cardholder control", mutate: func(value *CardPayload) { value.Cardholder = "Joel\nMiller" }, wantErr: ErrInvalidCardPayload},
		{name: "month without year", mutate: func(value *CardPayload) { value.ExpiryYear = nil }, wantErr: ErrInvalidCardPayload},
		{name: "year without month", mutate: func(value *CardPayload) { value.ExpiryMonth = nil }, wantErr: ErrInvalidCardPayload},
		{name: "month below range", mutate: func(value *CardPayload) { value.ExpiryMonth = intPointer(0) }, wantErr: ErrInvalidCardPayload},
		{name: "month above range", mutate: func(value *CardPayload) { value.ExpiryMonth = intPointer(13) }, wantErr: ErrInvalidCardPayload},
		{name: "year below range", mutate: func(value *CardPayload) { value.ExpiryYear = intPointer(-1) }, wantErr: ErrInvalidCardPayload},
		{name: "year above range", mutate: func(value *CardPayload) { value.ExpiryYear = intPointer(100) }, wantErr: ErrInvalidCardPayload},
		{name: "CVV too short", mutate: func(value *CardPayload) { value.CVV = "14" }, wantErr: ErrInvalidCardPayload},
		{name: "CVV too long", mutate: func(value *CardPayload) { value.CVV = "0140" }, wantErr: ErrInvalidCardPayload},
		{name: "CVV with letter", mutate: func(value *CardPayload) { value.CVV = "01A" }, wantErr: ErrInvalidCardPayload},
		{name: "CVV with Unicode digit", mutate: func(value *CardPayload) { value.CVV = "01١" }, wantErr: ErrInvalidCardPayload},
		{name: "metadata too long", mutate: func(value *CardPayload) { value.Metadata = strings.Repeat("я", MetadataMaxSize+1) }, wantErr: ErrInvalidCardPayload},
		{name: "metadata forbidden control", mutate: func(value *CardPayload) { value.Metadata = "test\x1bcard" }, wantErr: ErrInvalidCardPayload},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "nil payload" {
				var payload *CardPayload
				if err := payload.Validate(); !errors.Is(err, tt.wantErr) {
					t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			payload := valid
			if tt.mutate != nil {
				tt.mutate(&payload)
			}
			if err := payload.Validate(); !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
