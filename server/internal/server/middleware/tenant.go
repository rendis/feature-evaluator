package middleware

import (
	"github.com/gin-gonic/gin"
)

// Context keys for tenant information.
const (
	CtxTenantID  = "tenantId"
	CtxCampusID  = "campusId"
	CtxProgramID = "programId"
)

// TenantExtractor extracts tenant context from request headers.
func TenantExtractor() gin.HandlerFunc {
	return func(c *gin.Context) {
		if tid := c.GetHeader("X-Tenant-ID"); tid != "" {
			c.Set(CtxTenantID, tid)
		}
		if cid := c.GetHeader("X-Campus-ID"); cid != "" {
			c.Set(CtxCampusID, cid)
		}
		if pid := c.GetHeader("X-Program-ID"); pid != "" {
			c.Set(CtxProgramID, pid)
		}
		c.Next()
	}
}

// GetTenantID returns the tenant ID from context.
func GetTenantID(c *gin.Context) string {
	if v, ok := c.Get(CtxTenantID); ok {
		return v.(string)
	}
	return ""
}

// GetCampusID returns the campus ID from context.
func GetCampusID(c *gin.Context) string {
	if v, ok := c.Get(CtxCampusID); ok {
		return v.(string)
	}
	return ""
}

// GetProgramID returns the program ID from context.
func GetProgramID(c *gin.Context) string {
	if v, ok := c.Get(CtxProgramID); ok {
		return v.(string)
	}
	return ""
}
