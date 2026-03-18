package scheduler

import (
	"github.com/drama-generator/backend/application/services"
	"github.com/drama-generator/backend/pkg/logger"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

type ResourceTransferScheduler struct {
	cron            *cron.Cron
	transferService *services.ResourceTransferService
	db              *gorm.DB
	log             *logger.Logger
	running         bool
}

func NewResourceTransferScheduler(
	transferService *services.ResourceTransferService,
	db *gorm.DB,
	log *logger.Logger,
) *ResourceTransferScheduler {
	return &ResourceTransferScheduler{
		cron:            cron.New(cron.WithSeconds()),
		transferService: transferService,
		db:              db,
		log:             log,
		running:         false,
	}
}

// Start 启动定时任务
func (s *ResourceTransferScheduler) Start() error {
	if s.running {
		s.log.Warn("Resource transfer scheduler already running")
		return nil
	}

	s.log.Info("Starting resource transfer scheduler...")

	// 每小时执行一次资源转存任务
	_, err := s.cron.AddFunc("0 0 * * * *", func() {
		s.log.Info("Starting scheduled resource transfer task")
		s.transferPendingResources()
	})
	if err != nil {
		return err
	}

	// 每天凌晨2点执行完整扫描
	_, err = s.cron.AddFunc("0 0 2 * * *", func() {
		s.log.Info("Starting daily full resource scan and transfer")
		s.transferAllPendingResources()
	})
	if err != nil {
		return err
	}

	s.cron.Start()
	s.running = true
	s.log.Info("Resource transfer scheduler started successfully")

	return nil
}

// Stop 停止定时任务
func (s *ResourceTransferScheduler) Stop() {
	if !s.running {
		return
	}

	s.log.Info("Stopping resource transfer scheduler...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.running = false
	s.log.Info("Resource transfer scheduler stopped")
}

// transferPendingResources 转存最近生成的待转存资源（最近24小时） // Modified: MinIO implementation removed
func (s *ResourceTransferScheduler) transferPendingResources() {
	s.log.Info("Scanning for pending resources to transfer (skipped, minio removed)...")
}

// transferAllPendingResources 转存所有待转存的资源（全量扫描）
func (s *ResourceTransferScheduler) transferAllPendingResources() {
	s.log.Info("Starting full scan for all pending resources (skipped, minio removed)...")
}

// RunNow 立即执行一次转存任务（用于手动触发）
func (s *ResourceTransferScheduler) RunNow() {
	s.log.Info("Manually triggering resource transfer task...")
	go s.transferPendingResources()
}

// RunFullScan 立即执行一次全量扫描（用于手动触发）
func (s *ResourceTransferScheduler) RunFullScan() {
	s.log.Info("Manually triggering full resource scan...")
	go s.transferAllPendingResources()
}
