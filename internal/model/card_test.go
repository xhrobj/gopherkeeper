package model

import (
	"errors"
	"strings"
	"testing"
)

func TestCardPayload_Validate(t *testing.T) {
	january := 1
	december := 12
	year := 2038
	zeroMonth := 0
	lateMonth := 13

	tests := []struct {
		name    string
		payload *CardPayload
		wantErr error
	}{
		{
			name: "minimal payload",
			payload: &CardPayload{
				Number: "2013 0614 2020 0619",
			},
		},
		{
			name: "full payload with first month",
			payload: &CardPayload{
				Number:      "2013 0614 2020 0619",
				Cardholder:  "Joel Miller",
				ExpiryMonth: &january,
				ExpiryYear:  &year,
				CVV:         "014",
				Metadata:    "test card",
			},
		},
		{
			name: "last month",
			payload: &CardPayload{
				Number:      "2013 0614 2020 0619",
				ExpiryMonth: &december,
				ExpiryYear:  &year,
			},
		},
		{
			name:    "nil payload",
			payload: nil,
			wantErr: ErrInvalidCardPayload,
		},
		{
			name:    "empty number",
			payload: &CardPayload{},
			wantErr: ErrInvalidCardPayload,
		},
		{
			name: "blank number",
			payload: &CardPayload{
				Number: " \t\n",
			},
			wantErr: ErrInvalidCardPayload,
		},
		{
			name: "expiry month without year",
			payload: &CardPayload{
				Number:      "2013 0614 2020 0619",
				ExpiryMonth: &january,
			},
			wantErr: ErrInvalidCardPayload,
		},
		{
			name: "expiry year without month",
			payload: &CardPayload{
				Number:     "2013 0614 2020 0619",
				ExpiryYear: &year,
			},
			wantErr: ErrInvalidCardPayload,
		},
		{
			name: "month below range",
			payload: &CardPayload{
				Number:      "2013 0614 2020 0619",
				ExpiryMonth: &zeroMonth,
				ExpiryYear:  &year,
			},
			wantErr: ErrInvalidCardPayload,
		},
		{
			name: "month above range",
			payload: &CardPayload{
				Number:      "2013 0614 2020 0619",
				ExpiryMonth: &lateMonth,
				ExpiryYear:  &year,
			},
			wantErr: ErrInvalidCardPayload,
		},
		{
			name: "invalid UTF-8",
			payload: &CardPayload{
				Number: string([]byte{0xff}),
			},
			wantErr: ErrInvalidCardPayload,
		},
		{
			name: "metadata too large",
			payload: &CardPayload{
				Number:   "2013 0614 2020 0619",
				Metadata: strings.Repeat("a", MetadataMaxSize+1),
			},
			wantErr: ErrPayloadTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
