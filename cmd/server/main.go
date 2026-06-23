// Command server 启动 mdkit 的 HTTP 服务。
// 入口只做装配(wire up):构造 service、注册路由、起监听;不写业务逻辑。
package main

import (
	"log"
	"net/http"

	"mdkit/internal/health"
	"mdkit/internal/markdown"
)

func main() {
	mux := http.NewServeMux()

	// transport 层依赖 service 层;依赖单向,禁止反向(见 CLAUDE.md 架构要点)。
	healthSvc := health.NewService("0.1.0")
	mux.HandleFunc("GET /healthz", health.Handler(healthSvc))

	mdSvc := markdown.NewService(markdown.NewMockLLM())
	markdown.RegisterRoutes(mux, mdSvc)

	const addr = ":8080"
	log.Printf("mdkit listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server exited: %v", err)
	}
}
