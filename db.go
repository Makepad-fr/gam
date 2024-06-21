package gam

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// Expected table schema
var expectedSchema = map[string]string{
	"user_agent":                "String",
	"ip_address":                "String",
	"accept_language":           "String",
	"accept_encoding":           "String",
	"accept_charset":            "String",
	"accept":                    "String",
	"connection":                "String",
	"host":                      "String",
	"x_forwarded_for":           "String",
	"referer":                   "String",
	"cookie":                    "String",
	"dnt":                       "String",
	"upgrade_insecure_requests": "String",
	"cache_control":             "String",
	"pragma":                    "String",
	"via":                       "String",
	"forwarded":                 "String",
	"x_real_ip":                 "String",
	"x_forwarded_proto":         "String",
	"x_forwarded_host":          "String",
	"x_forwarded_port":          "String",
	"x_amz_date":                "String",
	"x_api_key":                 "String",
	"x_request_id":              "String",
	"authorization":             "String",
	"content_type":              "String",
	"content_length":            "Int64",
	"method":                    "String",
	"request_uri":               "String",
	"protocol":                  "String",
	"transfer_encoding":         "Array(String)",
	"tls_version":               "UInt16",
	"tls_cipher_suite":          "UInt16",
}

// ensureTableExistswithTheCorrectForm verifies if a table named with tableName exists and it has the correct schema.
// If the table does not exists it returns false, if the table exists and has the correct schema it returns true.
// If the table exists but it has a different schema, it returns false with an error
func ensureTableExistsWithTheCorrectForm(db *sql.DB, tableName string) (bool, error) {
	// Check if the table exists
	var tableCount int
	query := fmt.Sprintf("SELECT count() FROM system.tables WHERE name = '%s'", tableName)
	err := db.QueryRow(query).Scan(&tableCount)
	if err != nil {
		return false, err
	}

	if tableCount == 0 {
		// Table does not exist
		return false, nil
	}

	// Table exists, now check the schema
	query = fmt.Sprintf("SELECT name, type FROM system.columns WHERE table = '%s'", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	actualSchema := make(map[string]string)
	for rows.Next() {
		var columnName, columnType string
		if err := rows.Scan(&columnName, &columnType); err != nil {
			return false, err
		}
		actualSchema[columnName] = columnType
	}

	// Compare actual schema with expected schema
	for column, expectedType := range expectedSchema {
		actualType, exists := actualSchema[column]
		if !exists || !strings.EqualFold(actualType, expectedType) {
			return true, fmt.Errorf("schema mismatch for column '%s': expected '%s', got '%s'", column, expectedType, actualType)
		}
	}

	return true, nil
}

// createTable creates the table with the given name if it does not exists
func createTable(db *sql.DB, tableName string) error {
	expectedSchemaSlice := make([]string, 0, len(expectedSchema))
	var i = 0
	for key, value := range expectedSchema {
		expectedSchemaSlice = append(expectedSchemaSlice, fmt.Sprintf("%s %s", key, value))
		i++
	}
	createTableSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s ( %s ) ENGINE = MergeTree() ORDER BY (user_agent, ip_address);", tableName, strings.Join(expectedSchemaSlice, ","))
	_, err := db.Exec(createTableSQL)
	if err != nil {
		log.Printf("Error creating table: %v", err)
		return err
	}
	return nil
}
