package main

import (
	"fmt"
	"os"

	"github.com/innovatex/provisioner/internal/core"
	"github.com/innovatex/provisioner/internal/db"
	"github.com/innovatex/provisioner/internal/registry"
	"github.com/spf13/cobra"
)

var config = &core.Config{
	WorkDir:        "./data/clients",
	TemplateDir:    "./templates",
	StorefrontRepo: "https://github.com/The-Coding-Kiddo/clothing-storefront.git",
	DBHost:         "localhost",
	DBPort:         6543,
	DBUser:         "vendure",
	DBPassword:     "XTE9YTewFVAY2hvXK9-MUg",
	AdminDB:        "vendure",
	BasePort:       8000,
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "innovatex",
		Short: "InnovateX multi-client e-commerce provisioner",
	}

	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(statusCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func createCmd() *cobra.Command {
	var clientID, domain, brandName, adminEmail string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new client",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := registry.NewStore("./data/registry.json")
			dbProv := db.NewProvisioner(config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.AdminDB)
			prov := core.NewProvisioner(config, reg, dbProv)

			req := &core.CreateRequest{
				ClientID:   clientID,
				Domain:     domain,
				BrandName:  brandName,
				AdminEmail: adminEmail,
			}

			client, err := prov.Create(req)
			if err != nil {
				return err
			}

			fmt.Printf("\n")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			fmt.Printf("  ✓ Client Created Successfully!\n")
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
			fmt.Printf("Client ID:   %s\n", client.ID)
			fmt.Printf("Brand Name:  %s\n", client.BrandName)
			fmt.Printf("Database:    %s\n\n", client.DBName)
			fmt.Printf("URLs:\n")
			fmt.Printf("  Vendure:     http://localhost:%d\n", client.VendurePort)
			fmt.Printf("  Storefront:  http://localhost:%d\n\n", client.StorefrontPort)
			fmt.Printf("Admin Login:\n")
			fmt.Printf("  Email:       %s\n", client.AdminEmail)
			fmt.Printf("  Password:    %s\n\n", client.AdminPassword)
			fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID (required)")
	cmd.Flags().StringVarP(&domain, "domain", "d", "", "Domain (required)")
	cmd.Flags().StringVarP(&brandName, "brand", "b", "", "Brand name (required)")
	cmd.Flags().StringVarP(&adminEmail, "email", "e", "", "Admin email (required)")

	cmd.MarkFlagRequired("id")
	cmd.MarkFlagRequired("domain")
	cmd.MarkFlagRequired("brand")
	cmd.MarkFlagRequired("email")

	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all clients",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := registry.NewStore("./data/registry.json")
			clients, err := reg.List()
			if err != nil {
				return err
			}

			if len(clients) == 0 {
				fmt.Println("No clients found")
				return nil
			}

			fmt.Printf("\nTotal clients: %d\n\n", len(clients))
			fmt.Printf("%-15s %-25s %-10s %-12s\n", "ID", "BRAND", "STATUS", "PORTS")
			fmt.Printf("%-15s %-25s %-10s %-12s\n", "───", "─────", "──────", "─────")

			for _, c := range clients {
				ports := fmt.Sprintf("%d/%d", c.VendurePort, c.StorefrontPort)
				fmt.Printf("%-15s %-25s %-10s %-12s\n", c.ID, c.BrandName, c.Status, ports)
			}
			fmt.Println()

			return nil
		},
	}
}

func deleteCmd() *cobra.Command {
	var clientID string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a client",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := registry.NewStore("./data/registry.json")
			dbProv := db.NewProvisioner(config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.AdminDB)
			prov := core.NewProvisioner(config, reg, dbProv)

			if err := prov.Delete(clientID); err != nil {
				return err
			}

			fmt.Printf("\n✓ Client %s deleted successfully\n\n", clientID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID (required)")
	cmd.MarkFlagRequired("id")

	return cmd
}

func statusCmd() *cobra.Command {
	var clientID string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show client status",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := registry.NewStore("./data/registry.json")
			client, err := reg.Get(clientID)
			if err != nil {
				return err
			}

			fmt.Printf("\n")
			fmt.Printf("Client: %s\n", client.ID)
			fmt.Printf("Brand:  %s\n", client.BrandName)
			fmt.Printf("Status: %s\n", client.Status)
			fmt.Printf("DB:     %s\n\n", client.DBName)
			fmt.Printf("Vendure:    http://localhost:%d\n", client.VendurePort)
			fmt.Printf("Storefront: http://localhost:%d\n\n", client.StorefrontPort)
			fmt.Printf("Admin:      %s / %s\n\n", client.AdminEmail, client.AdminPassword)

			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID (required)")
	cmd.MarkFlagRequired("id")

	return cmd
}
