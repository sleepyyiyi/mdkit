package health

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// table-driven 测试:覆盖关键路径 + 边界。新增分支时往 tests 里加 case。
func TestHandler(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		wantStatus  int
		wantOK      bool
		wantVersion string
	}{
		{
			name:        "healthz 返回 ok",
			version:     "0.1.0",
			wantStatus:  http.StatusOK,
			wantOK:      true,
			wantVersion: "0.1.0",
		},
		{
			name:        "空版本号也应正常返回(边界)",
			version:     "",
			wantStatus:  http.StatusOK,
			wantOK:      true,
			wantVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
			rec := httptest.NewRecorder()

			Handler(NewService(tt.version))(rec, req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			var got Status
			if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if got.OK != tt.wantOK {
				t.Errorf("ok = %v, want %v", got.OK, tt.wantOK)
			}
			if got.Version != tt.wantVersion {
				t.Errorf("version = %q, want %q", got.Version, tt.wantVersion)
			}
		})
	}
}
