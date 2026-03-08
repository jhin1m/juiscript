package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func dbCmd(mgrs *Managers) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "db",
		Aliases: []string{"database"},
		Short:   "Manage databases and users",
	}
	cmd.AddCommand(
		dbListCmd(mgrs),
		dbCreateCmd(mgrs),
		dbDropCmd(mgrs),
		dbUserCreateCmd(mgrs),
		dbUserDropCmd(mgrs),
		dbResetPasswordCmd(mgrs),
		dbImportCmd(mgrs),
		dbExportCmd(mgrs),
	)
	return cmd
}

func dbListCmd(mgrs *Managers) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all databases",
		RunE: func(cmd *cobra.Command, args []string) error {
			dbs, err := mgrs.DB.ListDBs(context.Background())
			if err != nil {
				return err
			}
			if len(dbs) == 0 {
				fmt.Println("No databases found.")
				return nil
			}
			fmt.Fprintf(os.Stdout, "%-30s %-10s %-8s\n", "NAME", "SIZE (MB)", "TABLES")
			for _, d := range dbs {
				fmt.Fprintf(os.Stdout, "%-30s %-10.1f %-8d\n", d.Name, d.SizeMB, d.Tables)
			}
			return nil
		},
	}
}

func dbCreateCmd(mgrs *Managers) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.DB.CreateDB(context.Background(), name); err != nil {
				return err
			}
			fmt.Printf("Database created: %s\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Database name (required)")
	cmd.MarkFlagRequired("name")
	return cmd
}

func dbDropCmd(mgrs *Managers) *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "drop",
		Short: "Drop a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.DB.DropDB(context.Background(), name); err != nil {
				return err
			}
			fmt.Printf("Database dropped: %s\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Database name (required)")
	cmd.MarkFlagRequired("name")
	return cmd
}

func dbUserCreateCmd(mgrs *Managers) *cobra.Command {
	var (
		username string
		dbName   string
	)
	cmd := &cobra.Command{
		Use:   "user-create",
		Short: "Create a database user with privileges on a database",
		RunE: func(cmd *cobra.Command, args []string) error {
			password, err := mgrs.DB.CreateUser(context.Background(), username, dbName)
			if err != nil {
				return err
			}
			fmt.Printf("User created: %s\n", username)
			fmt.Printf("Password: %s\n", password)
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "Database username (required)")
	cmd.Flags().StringVar(&dbName, "database", "", "Database to grant access to (required)")
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("database")
	return cmd
}

func dbUserDropCmd(mgrs *Managers) *cobra.Command {
	var username string
	cmd := &cobra.Command{
		Use:   "user-drop",
		Short: "Drop a database user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.DB.DropUser(context.Background(), username); err != nil {
				return err
			}
			fmt.Printf("User dropped: %s\n", username)
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "Database username (required)")
	cmd.MarkFlagRequired("username")
	return cmd
}

func dbResetPasswordCmd(mgrs *Managers) *cobra.Command {
	var username string
	cmd := &cobra.Command{
		Use:   "reset-password",
		Short: "Reset password for a database user",
		RunE: func(cmd *cobra.Command, args []string) error {
			password, err := mgrs.DB.ResetPassword(context.Background(), username)
			if err != nil {
				return err
			}
			fmt.Printf("Password reset for: %s\n", username)
			fmt.Printf("Password: %s\n", password)
			return nil
		},
	}
	cmd.Flags().StringVar(&username, "username", "", "Database username (required)")
	cmd.MarkFlagRequired("username")
	return cmd
}

func dbImportCmd(mgrs *Managers) *cobra.Command {
	var (
		name     string
		filePath string
	)
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import a database from SQL dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.DB.Import(context.Background(), name, filePath); err != nil {
				return err
			}
			fmt.Printf("Import complete: %s <- %s\n", name, filePath)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Database name (required)")
	cmd.Flags().StringVar(&filePath, "file", "", "Path to SQL dump file (required)")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("file")
	return cmd
}

func dbExportCmd(mgrs *Managers) *cobra.Command {
	var (
		name       string
		outputPath string
	)
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export a database to SQL dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.DB.Export(context.Background(), name, outputPath); err != nil {
				return err
			}
			fmt.Printf("Export complete: %s -> %s\n", name, outputPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Database name (required)")
	cmd.Flags().StringVar(&outputPath, "output", "", "Output file path (required)")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("output")
	return cmd
}
