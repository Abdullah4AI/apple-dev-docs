package asc

import (
	"encoding/json"
	"testing"
)

func TestStateDetailDecodeMirrorsDescriptionIntoMessage(t *testing.T) {
	testCases := []struct {
		name            string
		payload         string
		wantCode        string
		wantDescription string
		wantMessage     string
	}{
		{
			name:            "description only",
			payload:         `{"code":"ERROR_ITMS_90XXX","description":"Invalid Bundle. The bundle is missing a minimum OS version."}`,
			wantCode:        "ERROR_ITMS_90XXX",
			wantDescription: "Invalid Bundle. The bundle is missing a minimum OS version.",
			wantMessage:     "Invalid Bundle. The bundle is missing a minimum OS version.",
		},
		{
			name:            "message only",
			payload:         `{"code":"IMAGE_INCORRECT_DIMENSIONS","message":"The image dimensions are invalid."}`,
			wantCode:        "IMAGE_INCORRECT_DIMENSIONS",
			wantDescription: "The image dimensions are invalid.",
			wantMessage:     "The image dimensions are invalid.",
		},
		{
			name:            "both present keeps each value",
			payload:         `{"code":"CODE","description":"described","message":"messaged"}`,
			wantCode:        "CODE",
			wantDescription: "described",
			wantMessage:     "messaged",
		},
		{
			name:            "explicit empty description is preserved",
			payload:         `{"code":"CODE","description":"","message":"messaged"}`,
			wantCode:        "CODE",
			wantDescription: "",
			wantMessage:     "messaged",
		},
		{
			name:            "explicit empty message is preserved",
			payload:         `{"code":"CODE","description":"described","message":""}`,
			wantCode:        "CODE",
			wantDescription: "described",
			wantMessage:     "",
		},
		{
			name:     "code only",
			payload:  `{"code":"90062"}`,
			wantCode: "90062",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			var detail StateDetail
			if err := json.Unmarshal([]byte(testCase.payload), &detail); err != nil {
				t.Fatalf("unmarshal state detail: %v", err)
			}
			if detail.Code != testCase.wantCode {
				t.Errorf("expected code %q, got %q", testCase.wantCode, detail.Code)
			}
			if detail.Description != testCase.wantDescription {
				t.Errorf("expected description %q, got %q", testCase.wantDescription, detail.Description)
			}
			if detail.Message != testCase.wantMessage {
				t.Errorf("expected message %q, got %q", testCase.wantMessage, detail.Message)
			}
		})
	}
}

func TestStateDetailDecodeResetsOmittedFieldsOnReusedReceiver(t *testing.T) {
	detail := StateDetail{
		Code:        "STALE_CODE",
		Description: "stale description",
		Message:     "stale message",
	}

	if err := json.Unmarshal([]byte(`{"code":"NEW_CODE"}`), &detail); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if detail.Code != "NEW_CODE" {
		t.Fatalf("expected decoded code, got %q", detail.Code)
	}
	if detail.Description != "" {
		t.Fatalf("expected omitted description to reset, got %q", detail.Description)
	}
	if detail.Message != "" {
		t.Fatalf("expected omitted message to reset, got %q", detail.Message)
	}
}

func TestStateDetailEncodeKeepsAppleWireFormat(t *testing.T) {
	var detail StateDetail
	if err := json.Unmarshal([]byte(`{"code":"CODE","description":"described"}`), &detail); err != nil {
		t.Fatalf("unmarshal state detail: %v", err)
	}

	encoded, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("marshal state detail: %v", err)
	}
	if got, want := string(encoded), `{"code":"CODE","description":"described"}`; got != want {
		t.Fatalf("expected re-encoded detail %s, got %s", want, got)
	}
}

func TestAppMediaAssetStateDecodesDescriptionsForAllSeverities(t *testing.T) {
	payload := `{"state":"FAILED",
		"errors":[{"code":"ERR","description":"error text"}],
		"warnings":[{"code":"WARN","description":"warning text"}],
		"infos":[{"code":"INFO","description":"info text"}]}`

	var state AppMediaAssetState
	if err := json.Unmarshal([]byte(payload), &state); err != nil {
		t.Fatalf("unmarshal asset state: %v", err)
	}

	if len(state.Errors) != 1 || state.Errors[0].Message != "error text" {
		t.Fatalf("expected error message %q, got %+v", "error text", state.Errors)
	}
	if len(state.Warnings) != 1 || state.Warnings[0].Message != "warning text" {
		t.Fatalf("expected warning message %q, got %+v", "warning text", state.Warnings)
	}
	if len(state.Infos) != 1 || state.Infos[0].Message != "info text" {
		t.Fatalf("expected info message %q, got %+v", "info text", state.Infos)
	}
}
