package cmd

import (
	"context"
	"database/sql"
	"os"
	"strings"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/chatstorage"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/infrastructure/whatsapp"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/pkg/sqlite"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	waLog "go.mau.fi/whatsmeow/util/log"
)

var (
	importSQLitePath      string
	importChatStoragePath string
	importProxyFile       string
)

// importSessionsCmd consolidates another GOWA into this one: the paired devices of its
// SQLite session store (and its registry, with webhooks) move into this instance's
// Postgres store, so numbers that used to need a container per proxy can live in one
// instance with a proxy per device. Stop the source GOWA first, then restart this one.
var importSessionsCmd = &cobra.Command{
	Use:   "import-sessions",
	Short: "Import the paired devices of another SQLite store into this Postgres store",
	Run: func(_ *cobra.Command, _ []string) {
		uri := strings.Trim(config.DBURI, `"'`)
		if !strings.HasPrefix(uri, "postgres:") {
			logrus.Fatal("import-sessions needs --db-uri / DB_URI pointing at Postgres")
		}
		pg, err := sql.Open("postgres", uri)
		if err != nil {
			logrus.Fatalf("open postgres: %v", err)
		}
		defer pg.Close()
		ctx := context.Background()

		devices, err := whatsapp.ImportSQLiteSessions(ctx, importSQLitePath, pg, waLog.Stdout("Database", config.WhatsappLogLevel, true))
		if err != nil {
			logrus.Fatalf("import sessions: %v", err)
		}
		logrus.Infof("[DB_IMPORT] %d device session(s) imported from %s", devices, importSQLitePath)

		if importChatStoragePath == "" {
			return
		}
		registry, ok := chatStorageRepo.(*chatstorage.PostgresDeviceRegistry)
		if !ok {
			logrus.Fatal("import-sessions: the device registry is not in Postgres")
		}
		src, err := sql.Open(sqlite.DriverName, sqlite.FormatChatStorageURI("file:"+importChatStoragePath, true, true))
		if err != nil {
			logrus.Fatalf("open source chat storage: %v", err)
		}
		defer src.Close()
		source := chatstorage.NewStorageRepository(src)
		records, err := source.ListDeviceRecords()
		if err != nil {
			logrus.Fatalf("read source registry: %v", err)
		}
		if _, err := registry.ImportRecords(source, records); err != nil {
			logrus.Fatalf("import registry: %v", err)
		}

		if importProxyFile == "" {
			return
		}
		raw, err := os.ReadFile(importProxyFile)
		if err != nil {
			logrus.Fatalf("read proxy file: %v", err)
		}
		proxyURL := strings.TrimSpace(string(raw))
		if err := whatsapp.ValidateProxyURL(proxyURL); err != nil {
			logrus.Fatalf("proxy file: %v", err)
		}
		for _, rec := range records {
			if err := registry.SetDeviceProxy(rec.DeviceID, &proxyURL); err != nil {
				logrus.Fatalf("set proxy of %s: %v", rec.DeviceID, err)
			}
		}
		logrus.Infof("[DB_IMPORT] %d device record(s) imported with their proxy", len(records))
	},
}

func init() {
	importSessionsCmd.Flags().StringVar(&importSQLitePath, "sqlite", "", "path to the source whatsapp.db (required)")
	importSessionsCmd.Flags().StringVar(&importChatStoragePath, "chat-storage", "", "path to the source chatstorage.db, to import the device registry and webhooks")
	importSessionsCmd.Flags().StringVar(&importProxyFile, "proxy-file", "", "file holding the proxy URL to set on the imported devices (kept out of argv)")
	_ = importSessionsCmd.MarkFlagRequired("sqlite")
	rootCmd.AddCommand(importSessionsCmd)
}
