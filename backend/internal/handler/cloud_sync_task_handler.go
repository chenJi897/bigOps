package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/response"
	"github.com/bigops/platform/internal/repository"
)

var _ model.CloudSyncTask // swag import

type CloudSyncTaskHandler struct {
	repo *repository.CloudSyncTaskRepository
}

func NewCloudSyncTaskHandler() *CloudSyncTaskHandler {
	return &CloudSyncTaskHandler{repo: repository.NewCloudSyncTaskRepository()}
}

// List 同步任务日志列表。
// @Summary 同步任务日志列表
// @Description 分页查询云同步任务记录，支持按状态/触发类型/云账号筛选
// @Tags 同步日志
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(20)
// @Param status query string false "状态" Enums(running,success,failed)
// @Param trigger_type query string false "触发类型" Enums(manual,schedule)
// @Param cloud_account_id query int false "云账号ID"
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.CloudSyncTask}} "同步日志列表"
// @Failure 500 {object} response.Response "查询失败"
// @Router /sync-tasks [get]
func (h *CloudSyncTaskHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))
	status := c.Query("status")
	triggerType := c.Query("trigger_type")
	accountID, _ := strconv.ParseInt(c.Query("cloud_account_id"), 10, 64)

	tasks, total, err := h.repo.List(page, size, status, triggerType, accountID)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, tasks, total, page, size)
}

// GetByAccountID 查询某云账号的同步历史。
// @Summary 云账号同步历史
// @Description 查询指定云账号的同步任务记录
// @Tags 同步日志
// @Produce json
// @Security BearerAuth
// @Param id path int true "云账号ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页条数" default(10)
// @Success 200 {object} response.Response{data=response.PageData{list=[]model.CloudSyncTask}} "同步历史"
// @Router /cloud-accounts/{id}/sync-tasks [get]
func (h *CloudSyncTaskHandler) GetByAccountID(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "10"))

	tasks, total, err := h.repo.ListByAccountID(id, page, size)
	if err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}
	response.Page(c, tasks, total, page, size)
}
