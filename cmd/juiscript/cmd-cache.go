package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func cacheCmd(mgrs *Managers) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cache",
		Short: "Manage cache services (Redis, Opcache)",
	}
	cmd.AddCommand(
		cacheStatusCmd(mgrs),
		cacheEnableRedisCmd(mgrs),
		cacheDisableRedisCmd(mgrs),
		cacheFlushCmd(mgrs),
		cacheOpcacheResetCmd(mgrs),
	)
	return cmd
}

func cacheStatusCmd(mgrs *Managers) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show cache service status",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := mgrs.Cache.Status(context.Background())
			if err != nil {
				return err
			}

			redisStr := "not running"
			if status.RedisRunning {
				redisStr = fmt.Sprintf("running (v%s, mem: %s)", status.RedisVersion, status.RedisMemory)
			}
			fmt.Fprintf(os.Stdout, "Redis: %s\n", redisStr)
			return nil
		},
	}
}

func cacheEnableRedisCmd(mgrs *Managers) *cobra.Command {
	var domain string
	var db int
	cmd := &cobra.Command{
		Use:   "enable-redis",
		Short: "Enable Redis for a site",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Cache.EnableRedis(context.Background(), domain, db); err != nil {
				return err
			}
			fmt.Printf("Redis enabled for %s (db: %d)\n", domain, db)
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.Flags().IntVar(&db, "db", 0, "Redis database number (0-15)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func cacheDisableRedisCmd(mgrs *Managers) *cobra.Command {
	var domain string
	cmd := &cobra.Command{
		Use:   "disable-redis",
		Short: "Disable Redis for a site",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := mgrs.Cache.DisableRedis(context.Background(), domain); err != nil {
				return err
			}
			fmt.Printf("Redis disabled for %s\n", domain)
			fmt.Println("Note: Update your app config (Laravel .env / WP wp-config.php) manually.")
			return nil
		},
	}
	cmd.Flags().StringVar(&domain, "domain", "", "Site domain (required)")
	cmd.MarkFlagRequired("domain")
	return cmd
}

func cacheFlushCmd(mgrs *Managers) *cobra.Command {
	var db int
	var all bool
	var force bool
	cmd := &cobra.Command{
		Use:   "flush",
		Short: "Flush Redis cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			if all {
				if !force {
					fmt.Fprint(os.Stderr, "WARNING: This will flush ALL Redis databases (sessions, queues, etc).\n")
					fmt.Fprint(os.Stderr, "Use --force to confirm.\n")
					return fmt.Errorf("flush-all requires --force flag")
				}
				if err := mgrs.Cache.FlushAll(ctx); err != nil {
					return err
				}
				fmt.Println("All Redis databases flushed")
				return nil
			}
			if err := mgrs.Cache.FlushDB(ctx, db); err != nil {
				return err
			}
			fmt.Printf("Redis database %d flushed\n", db)
			return nil
		},
	}
	cmd.Flags().IntVar(&db, "db", 0, "Redis database number")
	cmd.Flags().BoolVar(&all, "all", false, "Flush all Redis databases")
	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation for destructive operations")
	return cmd
}

func cacheOpcacheResetCmd(mgrs *Managers) *cobra.Command {
	var phpVersion string
	cmd := &cobra.Command{
		Use:   "opcache-reset",
		Short: "Reset PHP Opcache by restarting PHP-FPM",
		RunE: func(cmd *cobra.Command, args []string) error {
			ver := phpVersion
			if ver == "" {
				ver = mgrs.Cfg.PHP.DefaultVersion
			}
			if err := mgrs.Cache.ResetOpcache(context.Background(), ver); err != nil {
				return err
			}
			fmt.Printf("Opcache reset (PHP %s FPM restarted)\n", ver)
			return nil
		},
	}
	cmd.Flags().StringVar(&phpVersion, "php-version", "", "PHP version (default: from config)")
	return cmd
}
