package service

import (
	"encoding/json"
	"meetBack/internal/model"
	"testing"
)

func TestBuildAttendanceNudgePayloadUsesDefaultBody(t *testing.T) {
	payload, err := buildAttendanceNudgePayload(
		&model.Meeting{
			ID:      35,
			ShortId: "m35",
			Title:   "알림 테스트 미팅",
		},
		&model.AttendanceNudge{
			NudgeId: "nudge_123",
		},
	)
	if err != nil {
		t.Fatalf("buildAttendanceNudgePayload returned error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if body["title"] != "알림 테스트 미팅" {
		t.Fatalf("unexpected title: %v", body["title"])
	}
	if body["body"] != "아직 가능한 시간을 선택하지 않았어요. 지금 미팅 응답을 남겨주세요." {
		t.Fatalf("unexpected default body: %v", body["body"])
	}

	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("payload data is missing: %v", body["data"])
	}
	if data["url"] != "/m/m35" {
		t.Fatalf("unexpected url: %v", data["url"])
	}
	if data["type"] != "attendance-nudge" {
		t.Fatalf("unexpected type: %v", data["type"])
	}
}

func TestBuildAttendanceNudgePayloadUsesMessageOverride(t *testing.T) {
	payload, err := buildAttendanceNudgePayload(
		&model.Meeting{
			ID:      35,
			ShortId: "m35",
			Title:   "알림 테스트 미팅",
		},
		&model.AttendanceNudge{
			NudgeId:         "nudge_456",
			MessageOverride: "최종 응답 부탁드려요",
		},
	)
	if err != nil {
		t.Fatalf("buildAttendanceNudgePayload returned error: %v", err)
	}

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if body["body"] != "최종 응답 부탁드려요" {
		t.Fatalf("unexpected override body: %v", body["body"])
	}
}
