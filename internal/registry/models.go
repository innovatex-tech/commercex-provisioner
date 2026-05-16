package registry

import "time"

type Client struct {
	ID        string `json:"id"`
	Domain    string `json:"domain"`
	BrandName string `json:"brand_name"`
	Status    string `json:"status"`

	// Database
	DBName     string `json:"db_name"`
	DBUsername string `json:"db_username"`
	DBPassword string `json:"db_password"`

	// Admin credentials
	AdminUsername string `json:"admin_username"`
	AdminPassword string `json:"admin_password"`

	// Secrets
	CookieSecret string `json:"cookie_secret"`

	// Ports
	AppPort        int `json:"app_port"`
	PostgresPort   int `json:"postgres_port"`
	StorefrontPort int `json:"storefront_port"`

	// Remote Server
	IsRemote    bool   `json:"is_remote"`
	ServerHost  string `json:"server_host,omitempty"`
	ServerUser  string `json:"server_user,omitempty"`
	SSHPassword string `json:"ssh_password,omitempty"`
	SSHKeyPath  string `json:"ssh_key_path,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}
