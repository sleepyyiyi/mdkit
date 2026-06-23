package health

import (
	"encoding/json"
	"net/http"
)

// Handler 是传输层(transport):只负责 HTTP 编解码,业务委托给 Service。
// 禁止在 handler 内写业务逻辑或直接访问数据库/外部 IO(见 CLAUDE.md 工具约束)。
func Handler(svc *Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := svc.Check()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(status); err != nil {
			// 对外只给通用错误,不泄露内部堆栈/实现细节(见 review-checklist 安全项)。
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
	}
}
