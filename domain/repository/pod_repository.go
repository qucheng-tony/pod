package repository

import (
	"github.com/qucheng-tony/pod/domain/model"
	"gorm.io/gorm"
)

// IpodRepository 创建需要实现的接口
type IpodRepository interface {
	// InitTable 初始化表
	InitTable() error
	FindPodByID(int64) (*model.Pod, error)
	// CreatePod 创建一条POD
	CreatePod(*model.Pod) (int64, error)
	// DeletePod 根据ID删除一条POD数据
	DeletePod(int64) error
	// UpdatePod 修改数据
	UpdatePod(*model.Pod) error
	// FindAll 查找所有
	FindAll() ([]model.Pod, error)
}

func NewPodRepository(db *gorm.DB) IpodRepository {
	return &PodRepository{
		db,
	}
}

type PodRepository struct {
	mysqlDb *gorm.DB
}

func (u *PodRepository) InitTable() error {
	return u.mysqlDb.AutoMigrate(
		model.Pod{},
		model.PodPort{},
		model.PodEnv{},
	)
}

func (u *PodRepository) FindPodByID(podId int64) (pod *model.Pod, err error) {
	pod = &model.Pod{}
	return pod, u.mysqlDb.Preload("PodEnv").Preload("PodPort").First(pod, podId).Error
}

// CreatePod 创建一条POD
func (u *PodRepository) CreatePod(pod *model.Pod) (int64, error) {
	return pod.ID, u.mysqlDb.Create(pod).Error

}

// DeletePod 根据ID删除一条POD数据
func (u *PodRepository) DeletePod(podID int64) error {
	tx := u.mysqlDb.Begin()
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
		}
	}()
	if tx.Error != nil {
		return tx.Error
	}
	if err := u.mysqlDb.Where("id = ?", podID).Delete(&model.Pod{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := u.mysqlDb.Where("pod_id = ?", podID).Delete(&model.PodEnv{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := u.mysqlDb.Where("pod_id = ?", podID).Delete(&model.PodPort{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return u.mysqlDb.Commit().Error
}

// UpdatePod 修改数据
func (u *PodRepository) UpdatePod(pod *model.Pod) error {
	return u.mysqlDb.Model(pod).Updates(pod).Error
}

// FindAll 查找所有
func (u *PodRepository) FindAll() (podAll []model.Pod, err error) {
	return nil, u.mysqlDb.Find(&podAll).Error
}
