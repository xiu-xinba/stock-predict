// Package handler 实现了 HTTP 请求处理器，负责将 HTTP 请求转发至领域服务并构造响应。
package handler

import (
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"time"

	funddomain "stock-predict-go/internal/domain/fund"
	database "stock-predict-go/internal/infrastructure/database"
	providers "stock-predict-go/internal/infrastructure/providers"
	"stock-predict-go/internal/platform/config"
	"stock-predict-go/internal/transport/http/response"
	httpclient "stock-predict-go/internal/platform/httpclient"

	"github.com/gin-gonic/gin"
)

// Handler 是所有 HTTP 请求处理器的容器，持有配置、服务注册表、数据仓储及日志等依赖。
type Handler struct {
	cfg       config.Config
	services  *providers.Registry
	store     funddomain.CoverageRepository
	searchIdx *database.SearchStore
	logger    *slog.Logger
	hsgtStore *database.HSGTFlowDailyStore
}

// New 创建并返回一个新的 Handler 实例，注入所有必要的依赖项。
func New(
	cfg config.Config,
	services *providers.Registry,
	store funddomain.CoverageRepository,
	searchIdx *database.SearchStore,
	logger *slog.Logger,
) *Handler {
	hsgtStore := database.NewHSGTFlowDailyStore(services.DB)
	return &Handler{
		cfg:       cfg,
		services:  services,
		store:     store,
		searchIdx: searchIdx,
		logger:    logger,
		hsgtStore: hsgtStore,
	}
}

// isSixDigitCode 判断给定字符串是否为恰好 6 位纯数字的代码。
func isSixDigitCode(value string) bool {
	return len(value) == 6 && httpclient.IsAllDigits(value)
}

// ────────────────────── HSGT 北向南向资金流向 ──────────────────────

// GetHSGTLatest 获取最新的北向南向资金数据
// GET /api/v1/hsgt/latest
func (h *Handler) GetHSGTLatest(c *gin.Context) {
	flows, err := h.hsgtStore.ListRecent(1)
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "failed to get latest HSGT data")
		return
	}
	
	if len(flows) == 0 {
		response.WriteSuccess(c, nil)
		return
	}
	
	response.WriteSuccess(c, flows[0])
}

// GetHSGTRecent 获取最近 N 天的数据
// GET /api/v1/hsgt/recent?days=30
func (h *Handler) GetHSGTRecent(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 || days > 365 {
		response.WriteError(c, http.StatusBadRequest, -1, "invalid days parameter (1-365)")
		return
	}
	
	flows, err := h.hsgtStore.ListRecent(days)
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "failed to get recent HSGT data")
		return
	}
	
	response.WriteSuccess(c, flows)
}

// GetHSGTRange 按日期范围查询
// GET /api/v1/hsgt/range?start=2024-01-01&end=2024-01-31
func (h *Handler) GetHSGTRange(c *gin.Context) {
	startDate := c.Query("start")
	endDate := c.Query("end")
	
	if startDate == "" || endDate == "" {
		response.WriteError(c, http.StatusBadRequest, -1, "start and end dates are required")
		return
	}
	
	flows, err := h.hsgtStore.ListRange(startDate, endDate)
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "failed to get HSGT data for range")
		return
	}
	
	response.WriteSuccess(c, flows)
}

// GetHSGTByDate 按特定日期查询
// GET /api/v1/hsgt/date/:date
func (h *Handler) GetHSGTByDate(c *gin.Context) {
	date := c.Param("date")
	
	if date == "" {
		response.WriteError(c, http.StatusBadRequest, -1, "date is required")
		return
	}
	
	flow, err := h.hsgtStore.GetByDate(date)
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "failed to get HSGT data")
		return
	}
	
	if flow == nil {
		response.WriteSuccess(c, nil)
		return
	}
	
	response.WriteSuccess(c, flow)
}

// GetHSGTStatistics 获取 HSGT 数据统计信息
// GET /api/v1/hsgt/stats
func (h *Handler) GetHSGTStatistics(c *gin.Context) {
	count, err := h.hsgtStore.Count()
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "failed to get statistics")
		return
	}
	
	recent, err := h.hsgtStore.ListRecent(1)
	if err != nil {
		response.WriteError(c, http.StatusInternalServerError, -1, "failed to get recent data")
		return
	}
	
	stats := map[string]interface{}{
		"totalRecords": count,
	}
	
	if len(recent) > 0 {
		stats["latestDate"] = recent[0].Date
		stats["latestUpdateTime"] = recent[0].UpdateTime
		stats["latestNorth"] = map[string]interface{}{
			"shBuy":    recent[0].NorthSHBuy,
			"szBuy":    recent[0].NorthSZBuy,
			"totalBuy": recent[0].NorthTotalBuy,
		}
		stats["latestSouth"] = map[string]interface{}{
			"hkBuy":    recent[0].SouthHKBuy,
			"shBuy":    recent[0].SouthSHBuy,
			"szBuy":    recent[0].SouthSZBuy,
			"totalBuy": recent[0].SouthTotalBuy,
		}
	}
	
	response.WriteSuccess(c, stats)
}

// RestartBackend 触发后端进程优雅重启。
// POST /api/v1/admin/restart
// 先返回成功响应，然后异步启动新进程并退出当前进程。
func (h *Handler) RestartBackend(c *gin.Context) {
	h.logger.Info("restart requested via API, scheduling graceful restart")

	response.WriteSuccess(c, map[string]string{"status": "restarting"})

	// 异步执行重启，确保响应先发送
	go func() {
		time.Sleep(500 * time.Millisecond) // 等待响应发送完毕

		// 获取当前可执行文件路径
		exePath, err := os.Executable()
		if err != nil {
			h.logger.Error("failed to get executable path", "error", err)
			return
		}

		h.logger.Info("spawning new process", "exe", exePath)
		cmd := exec.Command(exePath, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = os.Environ()
		if err := cmd.Start(); err != nil {
			h.logger.Error("failed to start new process", "error", err)
			return
		}

		h.logger.Info("new process started, exiting current process", "pid", cmd.Process.Pid)
		os.Exit(0)
	}()
}
