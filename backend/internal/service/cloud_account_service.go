package service

import (
	"errors"

	"github.com/bigops/platform/internal/model"
	"github.com/bigops/platform/internal/pkg/config"
	"github.com/bigops/platform/internal/pkg/crypto"
	"github.com/bigops/platform/internal/repository"
)

type CloudAccountService struct {
	repo *repository.CloudAccountRepository
}

func NewCloudAccountService() *CloudAccountService {
	return &CloudAccountService{repo: repository.NewCloudAccountRepository()}
}

func (s *CloudAccountService) getEncryptKey() string {
	return config.Get().Encrypt.Key
}

func (s *CloudAccountService) Create(name, provider, accessKey, secretKey, region string) error {
	key := s.getEncryptKey()
	encAK, err := crypto.AESEncrypt(accessKey, key)
	if err != nil {
		return errors.New("加密 AccessKey 失败")
	}
	encSK, err := crypto.AESEncrypt(secretKey, key)
	if err != nil {
		return errors.New("加密 SecretKey 失败")
	}
	account := &model.CloudAccount{
		Name:      name,
		Provider:  provider,
		AccessKey: encAK,
		SecretKey: encSK,
		Region:    region,
		Status:    1,
	}
	return s.repo.Create(account)
}

func (s *CloudAccountService) Update(id int64, name, region string, status int8) error {
	account, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("云账号不存在")
	}
	account.Name = name
	account.Region = region
	account.Status = status
	return s.repo.Update(account)
}

func (s *CloudAccountService) UpdateKeys(id int64, accessKey, secretKey string) error {
	account, err := s.repo.GetByID(id)
	if err != nil {
		return errors.New("云账号不存在")
	}
	key := s.getEncryptKey()
	encAK, err := crypto.AESEncrypt(accessKey, key)
	if err != nil {
		return errors.New("加密 AccessKey 失败")
	}
	encSK, err := crypto.AESEncrypt(secretKey, key)
	if err != nil {
		return errors.New("加密 SecretKey 失败")
	}
	account.AccessKey = encAK
	account.SecretKey = encSK
	return s.repo.Update(account)
}

func (s *CloudAccountService) Delete(id int64) error {
	return s.repo.Delete(id)
}

func (s *CloudAccountService) GetByID(id int64) (*model.CloudAccount, error) {
	return s.repo.GetByID(id)
}

func (s *CloudAccountService) List(page, size int) ([]*model.CloudAccount, int64, error) {
	return s.repo.List(page, size)
}

// GetDecryptedKeys 获取解密后的 AK/SK（仅内部同步使用，不暴露给前端）。
func (s *CloudAccountService) GetDecryptedKeys(id int64) (accessKey, secretKey string, err error) {
	account, err := s.repo.GetByID(id)
	if err != nil {
		return "", "", errors.New("云账号不存在")
	}
	key := s.getEncryptKey()
	accessKey, err = crypto.AESDecrypt(account.AccessKey, key)
	if err != nil {
		return "", "", errors.New("解密 AccessKey 失败")
	}
	secretKey, err = crypto.AESDecrypt(account.SecretKey, key)
	if err != nil {
		return "", "", errors.New("解密 SecretKey 失败")
	}
	return accessKey, secretKey, nil
}

// UpdateSyncStatus 更新同步状态。
func (s *CloudAccountService) UpdateSyncStatus(id int64, status, message string, syncTime *model.LocalTime) error {
	account, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	account.LastSyncStatus = status
	account.LastSyncMessage = message
	account.LastSyncAt = syncTime
	return s.repo.Update(account)
}
