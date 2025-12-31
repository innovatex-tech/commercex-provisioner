package registry

import "time"

type Client struct {
	ID             string    `json:"id"`
	Domain         string    `json:"domain"`
	BrandName      string    `json:"brand_name"`
	Status         string    `json:"status"`
	DBName         string    `json:"db_name"`
	AdminEmail     string    `json:"admin_email"`
	AdminPassword  string    `json:"admin_password"`
	CookieSecret   string    `json:"cookie_secret"`
	VendurePort    int       `json:"vendure_port"`
	StorefrontPort int       `json:"storefront_port"`
	CreatedAt      time.Time `json:"created_at"`
}
