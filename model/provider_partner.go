package model

import (
	"encoding/base64"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	ProviderPartnerActive   = 1
	ProviderPartnerDisabled = 0
)

var (
	ErrInvalidProviderPartner = errors.New("invalid provider partner")
	ErrProviderPartnerExists  = errors.New("provider partner already exists")
	ErrProviderPartnerInUse   = errors.New("provider partner is in use")
	providerPartnerCreateMu   sync.Mutex
)

// ProviderPartner is the business partner that owns provider channels. A
// partner may have channels for multiple upstream AI providers and models.
type ProviderPartner struct {
	ID               int64  `json:"id" gorm:"primaryKey"`
	Code             string `json:"code" gorm:"type:varchar(64);not null;uniqueIndex"`
	Name             string `json:"name" gorm:"type:varchar(255);not null;index"`
	ImageURL         string `json:"image_url" gorm:"type:text"`
	WebsiteURL       string `json:"website_url" gorm:"type:varchar(1024)"`
	APIKey           string `json:"-" gorm:"type:varchar(512)"`
	APIEndpoint      string `json:"api_endpoint" gorm:"type:varchar(1024)"`
	APIKeyConfigured bool   `json:"api_key_configured" gorm:"-"`
	Status           int    `json:"status" gorm:"not null;index"`
	CreatedAt        int64  `json:"created_at" gorm:"not null;index"`
	UpdatedAt        int64  `json:"updated_at" gorm:"not null;index"`
}

func normalizeProviderPartner(partner *ProviderPartner) error {
	if partner == nil {
		return ErrInvalidProviderPartner
	}
	partner.Code = strings.ToLower(strings.TrimSpace(partner.Code))
	partner.Name = strings.TrimSpace(partner.Name)
	partner.ImageURL = strings.TrimSpace(partner.ImageURL)
	partner.WebsiteURL = strings.TrimSpace(partner.WebsiteURL)
	partner.APIKey = strings.TrimSpace(partner.APIKey)
	partner.APIEndpoint = strings.TrimSpace(partner.APIEndpoint)
	if partner.Code == "" && partner.Name != "" {
		partner.Code = providerPartnerCode(partner.Name)
	}
	if partner.Code == "" || len(partner.Code) > 64 || partner.Name == "" || len(partner.Name) > 255 || partner.APIKey == "" || partner.APIEndpoint == "" || len(partner.ImageURL) > 512*1024 || len(partner.WebsiteURL) > 1024 || len(partner.APIKey) > 512 || len(partner.APIEndpoint) > 1024 {
		return ErrInvalidProviderPartner
	}
	if partner.ImageURL != "" && !validProviderPartnerImage(partner.ImageURL) {
		return ErrInvalidProviderPartner
	}
	if partner.APIEndpoint != "" {
		endpoint, err := url.ParseRequestURI(partner.APIEndpoint)
		if err != nil || (endpoint.Scheme != "http" && endpoint.Scheme != "https") || endpoint.Host == "" {
			return ErrInvalidProviderPartner
		}
	}
	if partner.WebsiteURL != "" {
		website, err := url.ParseRequestURI(partner.WebsiteURL)
		if err != nil || (website.Scheme != "http" && website.Scheme != "https") || website.Host == "" {
			return ErrInvalidProviderPartner
		}
	}
	if partner.Status != ProviderPartnerActive && partner.Status != ProviderPartnerDisabled {
		partner.Status = ProviderPartnerActive
	}
	return nil
}

func providerPartnerCode(name string) string {
	var builder strings.Builder
	for _, character := range strings.ToLower(name) {
		switch {
		case character >= 'a' && character <= 'z', character >= '0' && character <= '9':
			builder.WriteRune(character)
		case builder.Len() > 0 && !strings.HasSuffix(builder.String(), "-"):
			builder.WriteByte('-')
		}
	}
	code := strings.Trim(builder.String(), "-")
	if code == "" {
		return "partner"
	}
	return code
}

func validProviderPartnerImage(imageData string) bool {
	media, payload, ok := strings.Cut(strings.TrimPrefix(imageData, "data:"), ";base64,")
	if !strings.HasPrefix(imageData, "data:") || !ok || (media != "image/png" && media != "image/jpeg") {
		return false
	}
	decoded, err := base64.StdEncoding.Strict().DecodeString(payload)
	if err != nil || len(decoded) == 0 {
		return false
	}
	return (media == "image/png" && strings.HasPrefix(string(decoded), "\x89PNG\r\n\x1a\n")) ||
		(media == "image/jpeg" && len(decoded) > 2 && decoded[0] == 0xff && decoded[1] == 0xd8)
}

func CreateProviderPartner(partner *ProviderPartner) error {
	providerPartnerCreateMu.Lock()
	defer providerPartnerCreateMu.Unlock()
	if err := normalizeProviderPartner(partner); err != nil {
		return err
	}
	var count int64
	if err := DB.Model(&ProviderPartner{}).Where("name = ?", partner.Name).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrProviderPartnerExists
	}
	partner.CreatedAt = time.Now().Unix()
	partner.UpdatedAt = partner.CreatedAt
	baseCode := partner.Code
	for suffix := 1; ; suffix++ {
		var count int64
		if err := DB.Model(&ProviderPartner{}).Where("code = ?", partner.Code).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			break
		}
		partner.Code = baseCode + "-" + strconv.Itoa(suffix+1)
	}
	if err := DB.Create(partner).Error; err != nil {
		return ErrProviderPartnerExists
	}
	partner.APIKeyConfigured = partner.APIKey != ""
	return nil
}

func ListProviderPartners() ([]ProviderPartner, error) {
	var partners []ProviderPartner
	assignedPartnerIDs := DB.Model(&Channel{}).
		Select("provider_partner_id").
		Where("provider_partner_id IS NOT NULL")
	err := DB.Where("id IN (?)", assignedPartnerIDs).
		Order("name ASC, id ASC").Find(&partners).Error
	for i := range partners {
		partners[i].APIKeyConfigured = partners[i].APIKey != ""
	}
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

func UpdateProviderPartner(partner *ProviderPartner) error {
	providerPartnerCreateMu.Lock()
	defer providerPartnerCreateMu.Unlock()
	if partner == nil || partner.ID <= 0 {
		return ErrInvalidProviderPartner
	}
	existing, err := GetProviderPartner(partner.ID)
	if err != nil {
		return err
	}
	if partner.APIKey == "" {
		partner.APIKey = existing.APIKey
	}
	if partner.Code == "" {
		partner.Code = existing.Code
	}
	if err = normalizeProviderPartner(partner); err != nil {
		return err
	}
	var count int64
	if err = DB.Model(&ProviderPartner{}).Where("name = ? AND id <> ?", partner.Name, partner.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrProviderPartnerExists
	}
	partner.UpdatedAt = time.Now().Unix()
	partner.APIKeyConfigured = partner.APIKey != ""
	return DB.Model(existing).Updates(map[string]any{
		"name":         partner.Name,
		"image_url":    partner.ImageURL,
		"website_url":  partner.WebsiteURL,
		"api_key":      partner.APIKey,
		"api_endpoint": partner.APIEndpoint,
		"status":       partner.Status,
		"updated_at":   partner.UpdatedAt,
	}).Error
}

func DeleteProviderPartner(id int64) error {
	if id <= 0 {
		return ErrInvalidProviderPartner
	}
	var channelCount, fundingCount int64
	if err := DB.Model(&Channel{}).Where("provider_partner_id = ?", id).Count(&channelCount).Error; err != nil {
		return err
	}
	if err := DB.Model(&ProviderFunding{}).Where("provider_partner_id = ?", id).Count(&fundingCount).Error; err != nil {
		return err
	}
	if channelCount > 0 || fundingCount > 0 {
		return ErrProviderPartnerInUse
	}
	result := DB.Delete(&ProviderPartner{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrInvalidProviderPartner
	}
	return nil
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
