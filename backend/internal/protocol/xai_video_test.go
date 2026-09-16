package protocol

import (
	"context"
	"testing"
)

func TestXAIVideoPackageReadsRequestID(t *testing.T) {
	adapter := officialPackageAdapter(t, "xai-video.yingce-plugin", "xai-video")
	for _, payload := range []string{`{"request_id":"request-1"}`, `{"data":{"request_id":"request-1"}}`} {
		created, err := adapter.ParseCreate(context.Background(), []byte(payload))
		if err != nil || created.TaskID != "request-1" || created.Status != StatusPending {
			t.Errorf("create = %#v, error = %v", created, err)
		}
	}
}

func TestXAIVideoPackageReadsNestedCompletedVideo(t *testing.T) {
	adapter := officialPackageAdapter(t, "xai-video.yingce-plugin", "xai-video")
	payload := `{"model":"grok-imagine-video-1.5","progress":100,"status":"done","usage":{"cost_in_usd_ticks":8000000000},"video":{"duration":10,"respect_moderation":true,"url":"/v1/videos/request-1/content"}}`
	result, err := adapter.ParsePoll(context.Background(), PollContext{TaskID: "request-1"}, []byte(payload))
	if err != nil || result.Status != StatusSucceeded || result.TaskID != "request-1" {
		t.Fatalf("poll = %#v, error = %v", result, err)
	}
	if result.Result == nil || len(result.Result.Videos) != 1 {
		t.Fatalf("expected one completed video, got %#v", result.Result)
	}
	video := result.Result.Videos[0]
	if video.URL != "/v1/videos/request-1/content" || video.Kind != "video" || !video.Ephemeral {
		t.Fatalf("video = %#v", video)
	}
}

func TestXAIVideoPollNestedResult(t *testing.T) {
	adapter, ok := Builtins().Get("xai-video")
	if !ok {
		t.Fatal("xAI video adapter missing")
	}
	cases := []struct {
		name    string
		baseURL string
		payload string
		wantURL string
	}{
		{
			name:    "relative content URL from completed provider task",
			baseURL: "https://provider.example/v1",
			payload: `{"model":"grok-imagine-video-1.5","progress":100,"status":"done","usage":{"cost_in_usd_ticks":8000000000},"video":{"duration":10,"respect_moderation":true,"url":"/v1/videos/request-1/content"}}`,
			wantURL: "/v1/videos/request-1/content",
		},
		{
			name:    "absolute CDN URL stays on CDN",
			baseURL: "https://provider.example/v1",
			payload: `{"status":"done","video":{"url":"https://cdn.example/video.mp4?signature=fixture"}}`,
			wantURL: "https://cdn.example/video.mp4?signature=fixture",
		},
		{
			name:    "relative URL preserves query for host resolution",
			baseURL: "https://provider.example:8443/gateway/v1",
			payload: `{"status":"completed","video":{"url":"/v1/videos/request-1/content?download=1"}}`,
			wantURL: "/v1/videos/request-1/content?download=1",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := adapter.ParsePoll(context.Background(), PollContext{BaseURL: tc.baseURL, TaskID: "request-1"}, []byte(tc.payload))
			if err != nil {
				t.Fatal(err)
			}
			if result.TaskID != "request-1" || result.Status != StatusSucceeded {
				t.Fatalf("poll identity/status = %#v", result)
			}
			if result.Result == nil || len(result.Result.Videos) != 1 {
				t.Fatalf("expected one completed video, got %#v", result.Result)
			}
			if video := result.Result.Videos[0]; video.URL != tc.wantURL || video.Kind != "video" {
				t.Fatalf("video = %#v, want URL %q and kind video", video, tc.wantURL)
			}
		})
	}
}

func TestXAIVideoPollPreservesStatusAndExistingResultShapes(t *testing.T) {
	adapter, _ := Builtins().Get("xai-video")
	cases := []struct {
		name       string
		payload    string
		wantStatus Status
		wantURL    string
	}{
		{"pending", `{"status":"pending"}`, StatusPending, ""},
		{"failed", `{"status":"failed","error":"moderation rejected"}`, StatusFailed, ""},
		{"missing result remains missing", `{"status":"done"}`, StatusSucceeded, ""},
		{"flat compatibility result", `{"status":"completed","video_url":"https://cdn.example/flat.mp4"}`, StatusSucceeded, "https://cdn.example/flat.mp4"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := adapter.ParsePoll(context.Background(), PollContext{BaseURL: "https://provider.example", TaskID: "request-1"}, []byte(tc.payload))
			if err != nil || result.Status != tc.wantStatus {
				t.Fatalf("poll = %#v, error = %v", result, err)
			}
			if tc.wantURL == "" {
				if result.Result != nil && len(result.Result.Videos) != 0 {
					t.Fatalf("unexpected video result: %#v", result.Result)
				}
				return
			}
			if result.Result == nil || len(result.Result.Videos) != 1 || result.Result.Videos[0].URL != tc.wantURL {
				t.Fatalf("video result = %#v, want %q", result.Result, tc.wantURL)
			}
		})
	}
}
