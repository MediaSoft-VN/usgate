package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/middleware"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

type tenantDNSRecord struct {
	CNAMETarget string `json:"cname_target"`
	ARecordIP   string `json:"a_record_ip"`
}

type tenantRequest struct {
	Slug        string `json:"slug"`
	PartnerCode string `json:"partner_code"`
	Name        string `json:"name"`
	OwnerUserID int    `json:"owner_user_id"`
}
type domainRequest struct {
	Hostname string `json:"hostname"`
}

func tenantIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("tenant_id"))
	return id, err == nil && id > 0
}

func GetTenants(c *gin.Context) {
	var tenants []model.Tenant
	query := model.DB.Where("tenants.deleted_at IS NULL")
	if c.GetInt("role") < common.RoleAdminUser {
		query = query.Joins("JOIN tenant_memberships ON tenant_memberships.tenant_id = tenants.id").Where("tenant_memberships.user_id = ? AND tenant_memberships.status = ?", c.GetInt("id"), model.MembershipActive)
	}
	err := query.Find(&tenants).Error
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tenants)
}

func PromoteTenant(c *gin.Context) {
	var request tenantRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.OwnerUserID <= 0 {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	tenant, err := service.PromoteTenant(request.Slug, request.Name, request.PartnerCode, request.OwnerUserID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tenant)
}

func GetTenant(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok || !middleware.RequireTenantAccess(c, id, "read") {
		return
	}
	var tenant model.Tenant
	if err := model.DB.Where("id = ? AND deleted_at IS NULL", id).First(&tenant).Error; err != nil {
		common.ApiError(c, model.ErrTenantNotFound)
		return
	}
	common.ApiSuccess(c, tenant)
}

func UpdateTenant(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok || !middleware.RequireTenantAccess(c, id, "tenant_update") {
		return
	}
	var request tenantRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	var tenant model.Tenant
	if err := model.DB.First(&tenant, id).Error; err != nil {
		common.ApiError(c, model.ErrTenantNotFound)
		return
	}
	if request.Name != "" {
		tenant.Name = request.Name
	}
	tenant.UpdatedAt = common.GetTimestamp()
	if err := model.DB.Save(&tenant).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, tenant)
}

func ListTenantMembers(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok || !middleware.RequireTenantAccess(c, id, "read") {
		return
	}
	var members []model.TenantMembership
	if err := model.DB.Where("tenant_id = ?", id).Find(&members).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, members)
}

func ListTenantDomains(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok || !middleware.RequireTenantAccess(c, id, "read") {
		return
	}
	var domains []model.TenantDomain
	if err := model.DB.Where("tenant_id = ?", id).Find(&domains).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, domains)
}

func GetTenantDomainDNSRecord(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok || !middleware.RequireTenantAccess(c, id, "read") {
		return
	}
	common.OptionMapRWMutex.RLock()
	record := tenantDNSRecord{
		CNAMETarget: common.Interface2String(common.OptionMap["TenantDomainDNSRecordCNAME"]),
		ARecordIP:   common.Interface2String(common.OptionMap["TenantDomainDNSRecordA"]),
	}
	common.OptionMapRWMutex.RUnlock()
	common.ApiSuccess(c, record)
}

func CreateTenantDomain(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok || !middleware.RequireTenantAccess(c, id, "domain_admin") {
		return
	}
	var req domainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	domain, err := service.CreateDomain(id, req.Hostname)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"domain": domain})
}

func UpdateTenantDomain(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok || !middleware.RequireTenantAccess(c, id, "domain_admin") {
		return
	}
	domainID, err := strconv.Atoi(c.Param("domain_id"))
	if err != nil || domainID <= 0 {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	var req domainRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	domain, err := service.UpdateDomain(domainID, id, req.Hostname)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"domain": domain})
}

func VerifyTenantDomain(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok || !middleware.RequireTenantAccess(c, id, "domain_admin") {
		return
	}
	domainID, err := strconv.Atoi(c.Param("domain_id"))
	if err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	if err = service.VerifyDomain(domainID, id); err != nil {
		common.ApiErrorT(c, "Domain verification failed")
		return
	}
	common.ApiSuccess(c, nil)
}

func DeleteTenantDomain(c *gin.Context) {
	id, ok := tenantIDParam(c)
	if !ok || !middleware.RequireTenantAccess(c, id, "domain_admin") {
		return
	}
	domainID, err := strconv.Atoi(c.Param("domain_id"))
	if err != nil {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	if err = service.DeleteDomain(domainID, id); err != nil {
		common.ApiError(c, errors.New("domain not found"))
		return
	}
	common.ApiSuccess(c, nil)
}

func TenantContextInfo(c *gin.Context) {
	if tenant, ok := c.Get(string(constant.ContextKeyTenant)); ok {
		common.ApiSuccess(c, tenant)
		return
	}
	common.ApiSuccess(c, nil)
}

func SelectTenantContext(c *gin.Context) {
	var request struct {
		TenantID int `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || request.TenantID <= 0 {
		common.ApiErrorT(c, "Invalid parameters")
		return
	}
	if !middleware.RequireTenantAccess(c, request.TenantID, "read") {
		return
	}
	var tenant model.Tenant
	if err := model.DB.First(&tenant, request.TenantID).Error; err != nil {
		common.ApiError(c, model.ErrTenantNotFound)
		return
	}
	common.ApiSuccess(c, tenant)
}
