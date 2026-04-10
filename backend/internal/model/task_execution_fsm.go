package model

// 任务执行（TaskExecution）状态机约定：仅允许以下终态。
// pending -> running -> success | partial_fail | failed | canceled
const (
	TaskExecStatusPending     = "pending"
	TaskExecStatusRunning     = "running"
	TaskExecStatusSuccess     = "success"
	TaskExecStatusPartialFail = "partial_fail"
	TaskExecStatusFailed      = "failed"
	TaskExecStatusCanceled    = "canceled"
)

// TaskExecTerminal 是否为终态（用于聚合完成判断）。
func TaskExecTerminal(status string) bool {
	switch status {
	case TaskExecStatusSuccess, TaskExecStatusPartialFail, TaskExecStatusFailed, TaskExecStatusCanceled:
		return true
	default:
		return false
	}
}

// CanTransitionTaskExecution 校验执行记录状态流转是否合法。
func CanTransitionTaskExecution(from, to string) bool {
	if from == to {
		return true
	}
	switch from {
	case TaskExecStatusPending:
		// 无可用 Agent 等场景：pending 可直接失败结束
		return to == TaskExecStatusRunning || to == TaskExecStatusFailed || to == TaskExecStatusCanceled
	case TaskExecStatusRunning:
		return to == TaskExecStatusSuccess || to == TaskExecStatusPartialFail || to == TaskExecStatusFailed || to == TaskExecStatusCanceled
	default:
		return false
	}
}
