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

type tenantRequest struct { Slug string `json:"slug"`; Name string `json:"name"` }
type membershipRequest struct { UserID int `json:"user_id"`; Role string `json:"role_key"` }
type domainRequest struct { Hostname string `json:"hostname"` }
type domainVerifyRequest struct { Token string `json:"token"` }

func tenantIDParam(c *gin.Context) (int, bool) {
	id, err := strconv.Atoi(c.Param("tenant_id")); return id, err == nil && id > 0
}

func GetTenants(c *gin.Context) {
	var tenants []model.Tenant
	err := model.DB.Joins("JOIN tenant_memberships ON tenant_memberships.tenant_id = tenants.id").Where("tenant_memberships.user_id = ? AND tenant_memberships.status = ? AND tenants.deleted_at IS NULL", c.GetInt("id"), model.MembershipActive).Find(&tenants).Error
	if err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, tenants)
}

func CreateTenant(c *gin.Context) {
	var request tenantRequest
	if err := c.ShouldBindJSON(&request); err != nil { common.ApiErrorT(c, "Invalid parameters"); return }
	tenant, err := service.CreateTenant(request.Slug, request.Name, c.GetInt("id"))
	if err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, tenant)
}

func GetTenant(c *gin.Context) {
	id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "read") { return }
	var tenant model.Tenant
	if err := model.DB.Where("id = ? AND deleted_at IS NULL", id).First(&tenant).Error; err != nil { common.ApiError(c, model.ErrTenantNotFound); return }; common.ApiSuccess(c, tenant)
}

func UpdateTenant(c *gin.Context) {
	id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "tenant_update") { return }
	var request tenantRequest; if err := c.ShouldBindJSON(&request); err != nil { common.ApiErrorT(c, "Invalid parameters"); return }
	var tenant model.Tenant; if err := model.DB.First(&tenant, id).Error; err != nil { common.ApiError(c, model.ErrTenantNotFound); return }
	if request.Name != "" { tenant.Name = request.Name }; tenant.UpdatedAt = common.GetTimestamp()
	if err := model.DB.Save(&tenant).Error; err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, tenant)
}

func ListTenantMembers(c *gin.Context) { id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "read") { return }; var members []model.TenantMembership; if err := model.DB.Where("tenant_id = ?", id).Find(&members).Error; err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, members) }

func UpsertTenantMember(c *gin.Context) {
	id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "member_admin") { return }
	var request membershipRequest; if err := c.ShouldBindJSON(&request); err != nil || request.UserID <= 0 { common.ApiErrorT(c, "Invalid parameters"); return }
	if request.Role == "" { request.Role = model.TenantRoleMember }
	if request.Role != model.TenantRoleOwner && request.Role != model.TenantRoleAdmin && request.Role != model.TenantRoleMember && request.Role != model.TenantRoleViewer { common.ApiErrorT(c, "Invalid parameters"); return }
	membership := model.TenantMembership{TenantId: id, UserId: request.UserID, RoleKey: request.Role, Status: model.MembershipActive, CreatedAt: common.GetTimestamp(), UpdatedAt: common.GetTimestamp()}
	err := model.DB.Where("tenant_id = ? AND user_id = ?", id, request.UserID).Assign(membership).FirstOrCreate(&membership).Error
	if err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, membership)
}

func UpdateTenantMember(c *gin.Context) { id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "member_admin") { return }; userID, err := strconv.Atoi(c.Param("user_id")); if err != nil { common.ApiErrorT(c, "Invalid parameters"); return }; var req membershipRequest; if err = c.ShouldBindJSON(&req); err != nil { common.ApiErrorT(c, "Invalid parameters"); return }; if err = model.ChangeMembership(model.DB, id, userID, req.Role, model.MembershipActive); err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, gin.H{"tenant_id": id, "user_id": userID, "role_key": req.Role}) }

func RemoveTenantMember(c *gin.Context) { id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "member_admin") { return }; userID, err := strconv.Atoi(c.Param("user_id")); if err != nil { common.ApiErrorT(c, "Invalid parameters"); return }; if err = model.ChangeMembership(model.DB, id, userID, model.TenantRoleMember, model.MembershipRemoved); err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, nil) }

func TransferTenantOwner(c *gin.Context) { common.ApiErrorT(c, "Ownership transfer requires recent authentication"); c.Abort() }

func ListTenantDomains(c *gin.Context) { id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "read") { return }; var domains []model.TenantDomain; if err := model.DB.Where("tenant_id = ?", id).Find(&domains).Error; err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, domains) }

func CreateTenantDomain(c *gin.Context) { id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "domain_admin") { return }; var req domainRequest; if err := c.ShouldBindJSON(&req); err != nil { common.ApiErrorT(c, "Invalid parameters"); return }; domain, token, err := service.CreateDomain(id, req.Hostname); if err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, gin.H{"domain": domain, "verification_token": token}) }

func VerifyTenantDomain(c *gin.Context) { id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "domain_admin") { return }; domainID, err := strconv.Atoi(c.Param("domain_id")); if err != nil { common.ApiErrorT(c, "Invalid parameters"); return }; var req domainVerifyRequest; if err = c.ShouldBindJSON(&req); err != nil { common.ApiErrorT(c, "Invalid parameters"); return }; if err = service.VerifyDomain(domainID, id, req.Token); err != nil { common.ApiError(c, err); return }; common.ApiSuccess(c, nil) }

func DisableTenantDomain(c *gin.Context) { id, ok := tenantIDParam(c); if !ok || !middleware.RequireTenantAccess(c, id, "domain_admin") { return }; domainID, err := strconv.Atoi(c.Param("domain_id")); if err != nil { common.ApiErrorT(c, "Invalid parameters"); return }; result := model.DB.Model(&model.TenantDomain{}).Where("id = ? AND tenant_id = ?", domainID, id).Updates(map[string]any{"status": model.DomainDisabled, "updated_at": common.GetTimestamp()}); if result.Error != nil || result.RowsAffected != 1 { common.ApiError(c, errors.New("domain not found")); return }; common.ApiSuccess(c, nil) }

func TenantContextInfo(c *gin.Context) { if tenant, ok := c.Get(string(constant.ContextKeyTenant)); ok { common.ApiSuccess(c, tenant); return }; common.ApiSuccess(c, nil) }

func SelectTenantContext(c *gin.Context) { var request struct { TenantID int `json:"tenant_id"` }; if err := c.ShouldBindJSON(&request); err != nil || request.TenantID <= 0 { common.ApiErrorT(c, "Invalid parameters"); return }; if !middleware.RequireTenantAccess(c, request.TenantID, "read") { return }; var tenant model.Tenant; if err := model.DB.First(&tenant, request.TenantID).Error; err != nil { common.ApiError(c, model.ErrTenantNotFound); return }; common.ApiSuccess(c, tenant) }
