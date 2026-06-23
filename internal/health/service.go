// Package health 是一个最小的业务模块示例:演示 transport → service 分层。
// 按你的真实领域,把它复制成 order / user / … 等模块,保持同样的分层与依赖方向。
package health

// Status 是健康检查的领域模型(domain model)。
// 领域模型集中在 service 层,不掺杂 HTTP/JSON 之外的传输细节。
type Status struct {
	OK      bool   `json:"ok"`
	Version string `json:"version"`
}

// Service 持有业务逻辑。transport(handler)依赖它,禁止反向依赖。
// 后续接入数据库时,Service 依赖 repository 接口,而非具体实现。
type Service struct {
	version string
}

// NewService 构造 Service。依赖通过构造函数注入,便于测试替换。
func NewService(version string) *Service {
	return &Service{version: version}
}

// Check 返回当前服务状态。纯逻辑、无副作用,易于单测。
func (s *Service) Check() Status {
	return Status{OK: true, Version: s.version}
}
