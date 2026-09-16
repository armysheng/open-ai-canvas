package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/protocol"
)

func TestHydrateXAIVideoLocalImageBuildsOfficialRequest(t *testing.T) {
	t.Setenv("CANVAS_PUBLIC_BASE_URL", "")
	for _, operation := range []string{"image_to_video", "reference_to_video"} {
		t.Run(operation, func(t *testing.T) {
			svc := newResourceTestService(t)
			_, imageBytes, err := decodeProviderDataURL(testGeminiReferenceImageDataURL)
			if err != nil {
				t.Fatal(err)
			}
			resource := model.Resource{ID: "local-image", UserID: "user-1", Kind: "image", Status: model.ResourceStatusReady, Provider: "local", ObjectKey: "reference.png", MimeType: "image/png"}
			if err := os.MkdirAll(filepath.Join(svc.dataDir, "resources", filepath.Dir(resource.ObjectKey)), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(svc.dataDir, "resources", resource.ObjectKey), imageBytes, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := svc.repo.CreateResource(&resource); err != nil {
				t.Fatal(err)
			}
			input := canvasGenerationInput{
				Mode: "video", Prompt: "animate the image", Config: providerConfig{Model: "grok-imagine-video-1.5", InterfaceType: "xai-video"},
				ReferenceImages: []providerMedia{{ID: "frame", StorageKey: "resource:local-image", URL: "/api/resources/local-image/file", DataURL: "data:image/png;base64,c3RhbGU="}},
				Metadata:        map[string]interface{}{"videoEditOperation": operation, "videoStartFrameNodeId": "frame"},
			}
			policy := providerMediaHydrationPolicyFor(context.Background(), input)
			if err := svc.hydrateGenerationMedia("user-1", &input, policy); err != nil {
				t.Fatalf("hydrate local image without a public server: %v", err)
			}
			media := input.ReferenceImages[0]
			if media.URL != "" || media.DataURL != testGeminiReferenceImageDataURL || media.Bytes != int64(len(imageBytes)) {
				t.Fatalf("hydrated image does not match owned resource: %#v", media)
			}
			body := officialVideoCreateBody(t, input)
			image := body["image"]
			if operation == "reference_to_video" {
				refs, ok := body["reference_images"].([]any)
				if !ok || len(refs) != 1 {
					t.Fatalf("reference_images = %#v", body["reference_images"])
				}
				image = refs[0]
			}
			mappedImage, ok := image.(map[string]any)
			if !ok || mappedImage["url"] != testGeminiReferenceImageDataURL {
				t.Fatalf("official request image = %#v", image)
			}
		})
	}
}

func TestHydrateXAIVideoLocalImagePreservesResourceGuards(t *testing.T) {
	t.Setenv("CANVAS_PUBLIC_BASE_URL", "")
	for _, tc := range []struct {
		name, protocol, userID, kind, wantError string
		status                                  model.ResourceStatus
		oversized                               bool
	}{
		{name: "another owner", protocol: "xai-video", userID: "other-user", kind: "image", status: model.ResourceStatusReady, wantError: "读取任务参考资源失败"},
		{name: "unfinished upload", protocol: "xai-video", userID: "user-1", kind: "image", status: model.ResourceStatusPending, wantError: "尚未上传完成"},
		{name: "oversized image", protocol: "xai-video", userID: "user-1", kind: "image", status: model.ResourceStatusReady, oversized: true, wantError: "超过 1MB"},
		{name: "video still needs URL", protocol: "xai-video", userID: "user-1", kind: "video", status: model.ResourceStatusReady, wantError: "CANVAS_PUBLIC_BASE_URL"},
		{name: "audio still needs URL", protocol: "xai-video", userID: "user-1", kind: "audio", status: model.ResourceStatusReady, wantError: "CANVAS_PUBLIC_BASE_URL"},
		{name: "minimax still needs URL", protocol: "minimax-video", userID: "user-1", kind: "image", status: model.ResourceStatusReady, wantError: "CANVAS_PUBLIC_BASE_URL"},
		{name: "newapi still needs URL", protocol: "newapi-channel-1", userID: "user-1", kind: "image", status: model.ResourceStatusReady, wantError: "CANVAS_PUBLIC_BASE_URL"},
		{name: "ark still needs URL", protocol: "volcengine-ark-video", userID: "user-1", kind: "image", status: model.ResourceStatusReady, wantError: "CANVAS_PUBLIC_BASE_URL"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := newResourceTestService(t)
			resource := model.Resource{ID: "local-resource", UserID: "user-1", Kind: tc.kind, Status: tc.status, Provider: "local", ObjectKey: "reference.png", MimeType: "image/png"}
			if err := svc.repo.CreateResource(&resource); err != nil {
				t.Fatal(err)
			}
			if tc.oversized {
				policy := defaultRuntimePolicy()
				policy.Resource.ResourceUploadMB = 1
				encoded, err := json.Marshal(policy)
				if err != nil {
					t.Fatal(err)
				}
				if err := svc.repo.SaveSystemSetting(&model.SystemSetting{Key: runtimePolicySettingKey, ValueJSON: string(encoded)}); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(filepath.Join(svc.dataDir, "resources", filepath.Dir(resource.ObjectKey)), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(svc.dataDir, "resources", resource.ObjectKey), make([]byte, megabytes(1)+1), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			input := canvasGenerationInput{Mode: "video", Config: providerConfig{InterfaceType: tc.protocol}}
			policy := providerMediaHydrationPolicyFor(context.Background(), input)
			err := svc.hydrateProviderMedia(tc.userID, &providerMedia{StorageKey: "resource:local-resource"}, policy)
			if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("hydrate error = %v, want %q", err, tc.wantError)
			}
		})
	}
	policy := providerMediaHydrationPolicyFor(context.Background(), canvasGenerationInput{Mode: "video", Config: providerConfig{InterfaceType: "xai-video"}})
	svc := newResourceTestService(t)
	if err := svc.hydrateProviderMedia("user-1", &providerMedia{DataURL: testGeminiReferenceImageDataURL}, policy); err == nil {
		t.Fatal("unbacked inline input should still be rejected")
	}
}

func TestQueryXAIVideoDownloadsNestedResultWithScopedAuthentication(t *testing.T) {
	t.Setenv("CANVAS_ALLOWED_PRIVATE_UPSTREAM_HOSTS", "127.0.0.1")
	for _, location := range []string{"same-origin relative", "same-origin absolute", "external absolute"} {
		t.Run(location, func(t *testing.T) {
			var resultURL string
			polls, downloads := 0, 0
			serveVideo := func(w http.ResponseWriter, r *http.Request, authenticated bool) {
				downloads++
				wantAuthorization, wantCustom := "", ""
				if authenticated {
					wantAuthorization, wantCustom = "Bearer fixture-key", "fixture-header"
				}
				if r.Header.Get("Authorization") != wantAuthorization || r.Header.Get("X-Provider-Test") != wantCustom {
					t.Errorf("download auth = %q/%q, want %q/%q", r.Header.Get("Authorization"), r.Header.Get("X-Provider-Test"), wantAuthorization, wantCustom)
				}
				w.Header().Set("Content-Type", "video/mp4")
				_, _ = w.Write([]byte("fixture-video-bytes"))
			}
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("recovery issued %s, want GET only", r.Method)
				}
				switch r.URL.Path {
				case "/v1/videos/request-1":
					polls++
					if r.Header.Get("Authorization") != "Bearer fixture-key" {
						t.Errorf("poll authorization = %q", r.Header.Get("Authorization"))
					}
					w.Header().Set("Content-Type", "application/json")
					_, _ = fmt.Fprintf(w, `{"status":"done","video":{"duration":10,"url":%q}}`, resultURL)
				case "/v1/videos/request-1/content":
					serveVideo(w, r, true)
				default:
					t.Errorf("unexpected provider path %s", r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer provider.Close()
			resultURL = "/v1/videos/request-1/content"
			if location == "same-origin absolute" {
				resultURL = provider.URL + resultURL
			} else if location == "external absolute" {
				cdn := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					serveVideo(w, r, false)
				}))
				defer cdn.Close()
				resultURL = cdn.URL + "/clip.mp4"
			}
			adapter, _ := protocol.Builtins().Get("xai-video")
			input := canvasGenerationInput{Mode: "video", Config: providerConfig{
				BaseURL: provider.URL + "/v1", APIKey: "fixture-key", InterfaceType: "xai-video",
				Headers: []OutboundHeader{{Name: "X-Provider-Test", Value: "fixture-header"}},
			}}
			result, status, err := queryProtocolAdapterVideoTask(context.Background(), input, adapter, "request-1")
			if err != nil {
				t.Fatal(err)
			}
			if status != "succeeded" || polls != 1 || downloads != 1 {
				t.Fatalf("status=%q polls=%d downloads=%d", status, polls, downloads)
			}
			video, ok := result["video"].(map[string]interface{})
			if !ok {
				t.Fatalf("video result = %#v", result)
			}
			mimeType, data, err := decodeProviderDataURL(video["dataUrl"].(string))
			if err != nil || mimeType != "video/mp4" || string(data) != "fixture-video-bytes" {
				t.Fatalf("download result: MIME=%q bytes=%q error=%v", mimeType, data, err)
			}
		})
	}
}
