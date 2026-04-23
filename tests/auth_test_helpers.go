package tests

import (
	"ad-platform/internal/admin"

	"github.com/gin-gonic/gin"
)

func injectPermissions(permissions ...string) gin.HandlerFunc {
	role := admin.Role{
		Name:        "test_role",
		Permissions: buildPermissions(permissions...),
	}
	user := &admin.User{
		ID:     1,
		Status: true,
		Roles:  []admin.Role{role},
	}

	return func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	}
}

func buildPermissions(names ...string) []admin.Permission {
	perms := make([]admin.Permission, 0, len(names))
	for _, name := range names {
		perms = append(perms, admin.Permission{Name: name})
	}
	return perms
}
