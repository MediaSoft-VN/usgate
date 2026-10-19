package model

import (
	"errors"
	"strings"
	"time"
)

const (
	ProviderPartnerActive   = 1
	ProviderPartnerDisabled = 0
)

var (
	ErrInvalidProviderPartner = errors.New("invalid provider partner")
	ErrProviderPartnerExists  = errors.New("provider partner already exists")
)

// ProviderPartner is the business partner that owns provider channels. A
// partner may have channels for multiple upstream AI providers and models.
type ProviderPartner struct {
	ID        int64  `json:"id" gorm:"primaryKey"`
	Code      string `json:"code" gorm:"type:varchar(64);not null;uniqueIndex"`
	Name      string `json:"name" gorm:"type:varchar(255);not null;index"`
	Status    int    `json:"status" gorm:"not null;index"`
	CreatedAt int64  `json:"created_at" gorm:"not null;index"`
	UpdatedAt int64  `json:"updated_at" gorm:"not null;index"`
}

func normalizeProviderPartner(partner *ProviderPartner) error {
	if partner == nil {
		return ErrInvalidProviderPartner
	}
	partner.Code = strings.ToLower(strings.TrimSpace(partner.Code))
	partner.Name = strings.TrimSpace(partner.Name)
	if partner.Code == "" || len(partner.Code) > 64 || partner.Name == "" || len(partner.Name) > 255 {
		return ErrInvalidProviderPartner
	}
	if partner.Status != ProviderPartnerActive && partner.Status != ProviderPartnerDisabled {
		partner.Status = ProviderPartnerActive
	}
	return nil
}

func CreateProviderPartner(partner *ProviderPartner) error {
	if err := normalizeProviderPartner(partner); err != nil {
		return err
	}
	partner.CreatedAt = time.Now().Unix()
	partner.UpdatedAt = partner.CreatedAt
	if err := DB.Create(partner).Error; err != nil {
		return ErrProviderPartnerExists
	}
	return nil
}

func ListProviderPartners() ([]ProviderPartner, error) {
	var partners []ProviderPartner
	err := DB.Order("name ASC, id ASC").Find(&partners).Error
	return partners, err
}

func GetProviderPartner(id int64) (*ProviderPartner, error) {
	if id <= 0 {
		return nil, ErrInvalidProviderPartner
	}
	partner := &ProviderPartner{}
	if err := DB.Where("id = ?", id).First(partner).Error; err != nil {
		return nil, err
	}
	return partner, nil
}

func SetChannelProviderPartner(channelID int, partnerID *int64) error {
	if channelID <= 0 {
		return ErrInvalidProviderPartner
	}
	var channel Channel
	if err := DB.Where("id = ?", channelID).First(&channel).Error; err != nil {
		return err
	}
	if partnerID != nil {
		if _, err := GetProviderPartner(*partnerID); err != nil {
			return err
		}
	}
	return DB.Model(&channel).Update("provider_partner_id", partnerID).Error
}
