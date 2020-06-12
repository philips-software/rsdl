package main

import (
	"crypto/subtle"
	"fmt"
	"github.com/cloudfoundry-community/gautocloud"
	"github.com/philips-software/gautocloud-connectors/hsdp"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/lib/pq"
	"github.com/loafoe/sqltocsv"
	"github.com/spf13/viper"
	"net/http"
	"os"
	"time"
)

var GitCommit = "deadbeaf"

func main() {
	log.Infof("rsdl version: %s", GitCommit)

	// Config
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.SetEnvPrefix("rsdl")
	viper.SetDefault("password", "")
	viper.SetDefault("schema", "")
	viper.SetDefault("gzip", "true")
	viper.AutomaticEnv()
	viper.AddConfigPath(".")
	viper.ReadInConfig()

	password := viper.GetString("password")
	if password == "" { // Never without a password
		log.Error("missing password")
		return
	}
	schemaName := viper.GetString("schema")
	if schemaName == "" {
		log.Error("missing schema")
		return
	}
	useCompression := viper.GetBool("gzip")
	usePort := os.Getenv("PORT")
	if usePort == "" {
		usePort = "8080"
	}

	// Database
	var rs *hsdp.PostgresSQLClient
	err := gautocloud.Inject(&rs)
	if err != nil {
		log.Errorf("database error: %v", err)
		return
	}

	// Webapp
	e := echo.New()
	e.Logger.SetLevel(log.INFO)
	if useCompression {
		e.Use(middleware.GzipWithConfig(middleware.GzipConfig{
			Level: 5,
		}))
	}
	e.Use(middleware.BasicAuth(authCheck))
	e.Use(middleware.Logger())
	e.GET("/redshift/:tableName/:type", downloader(rs, schemaName))
	e.GET("/redshift/:schema/:tableName/:type", downloader(rs, schemaName))
	log.Fatal(e.Start(":" + usePort))
	return
}

// authCheck verifies basic auth. Username hardcoded to `redshift`
func authCheck(username, password string, c echo.Context) (bool, error) {
	if subtle.ConstantTimeCompare([]byte(username), []byte("redshift")) == 1 &&
		subtle.ConstantTimeCompare([]byte(password), []byte(password)) == 1 {
		return true, nil
	}
	return false, nil
}

// downloader queries and streams TAB separated CSV of a pg table
func downloader(rs *hsdp.PostgresSQLClient, schemaName string) echo.HandlerFunc {
	return func(e echo.Context) error {
		schema := e.Param("schema")
		tableName := e.Param("tableName")
		downloadType := e.Param("type")

		quotedTable := pq.QuoteIdentifier(tableName)
		quotedSchema := schemaName // Default
		if schema != "" {
			quotedSchema = pq.QuoteIdentifier(schema)
		}

		rows, err := rs.QueryContext(e.Request().Context(),
			fmt.Sprintf("SELECT * FROM %s.%s", quotedSchema, quotedTable))

		if err != nil {
			e.String(http.StatusBadRequest, err.Error())
			return err
		}
		defer rows.Close()

		switch downloadType {
		case "full.csv":
		default:
			e.String(http.StatusBadRequest, "unsupported download type")
			return nil
		}

		resp := e.Response()
		header := resp.Header()
		header.Set(echo.HeaderContentType, echo.MIMEOctetStream)
		header.Set(echo.HeaderContentDisposition, "attachment; filename="+"full.csv")
		header.Set("Content-Transfer-Encoding", "binary")
		header.Set("Expires", "0")

		h, _ := rows.Columns()

		var count = 0
		csvConverter := sqltocsv.New(rows)
		csvConverter.Delimiter = '\t'
		csvConverter.Headers = h
		csvConverter.SetRowPreProcessor(func(row []string, names []string) (outputRow bool, processedRow []string) {
			count++
			return true, row
		})
		csvConverter.SetContext(e.Request().Context())
		done := make(chan bool)
		log := e.Logger()
		go func() {
			for {
				select {
				case <-done:
					log.Infof("done with request: %d rows total", count)
					return
				case <-time.After(2 * time.Second):
					log.Infof("processed %d rows", count)
				}
			}
		}()
		err = csvConverter.Write(resp.Writer)
		done <- true
		return err
	}
}
