package gam

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"

	"github.com/ClickHouse/clickhouse-go"
)

type gam struct {
	tableName string
	db        *sql.DB
}

// Init creates a new gam structure by initialising the database connection
func Init(username, password, hostname, port, databaseName, tableName string, ensureTableExistanceAndSchema, createTableIfNotExists bool) (gam, error) {
	connectionString := fmt.Sprintf("tcp://%s:%s?username=%s&password=%s&database=%s", hostname, port, username, password, databaseName)
	db, err := sql.Open("clickhouse", connectionString)
	if err != nil {
		return gam{}, err
	}
	// Ping the database to verify the connection
	if err := db.Ping(); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			return gam{}, fmt.Errorf("[%d]: %s \n%s", exception.Code, exception.Message, exception.StackTrace)
		} else {
			return gam{}, err
		}
	}
	if ensureTableExistanceAndSchema {
		exists, err := ensureTableExistsWithTheCorrectForm(db, tableName)
		if err != nil {
			// Table exists but has a different schema
			return gam{}, err
		}
		if exists {
			// Table exists and has the correct schema
			return gam{tableName: tableName, db: db}, nil
		}
	}
	if createTableIfNotExists {
		err := createTable(db, tableName)
		if err != nil {
			// Error while creating the table with the expected schema
			return gam{}, err
		}
	}
	return gam{tableName: tableName, db: db}, nil
}

type analyticsData struct {
	UserAgent               string
	IPAddress               string
	AcceptLanguage          string
	AcceptEncoding          string
	AcceptCharset           string
	Accept                  string
	Connection              string
	Host                    string
	XForwardedFor           string
	Referer                 string
	Cookie                  string
	DNT                     string
	UpgradeInsecureRequests string
	CacheControl            string
	Pragma                  string
	Via                     string
	Forwarded               string
	XRealIP                 string
	XForwardedProto         string
	XForwardedHost          string
	XForwardedPort          string
	XAmzDate                string
	XApiKey                 string
	XRequestID              string
	Authorization           string
	ContentType             string
	ContentLength           int64
	Method                  string
	RequestURI              string
	Protocol                string
	TransferEncoding        []string
	TLSVersion              uint16
	TLSCipherSuite          uint16
}

// Close closes the active database connection of the current gam instance
func (g gam) Close() error {
	return g.db.Close()
}

// extractAnalyticsData extracts the analytics data from given *http.Request and returns the extracted AnalyseData
func extractAnalyticsData(r *http.Request) analyticsData {
	// Extract additional TLS information if available
	var tlsVersion uint16
	var tlsCipherSuite uint16
	if r.TLS != nil {
		tlsVersion = r.TLS.Version
		tlsCipherSuite = r.TLS.CipherSuite
	}

	return analyticsData{
		UserAgent:               r.UserAgent(),
		IPAddress:               r.RemoteAddr,
		AcceptLanguage:          r.Header.Get("Accept-Language"),
		AcceptEncoding:          r.Header.Get("Accept-Encoding"),
		AcceptCharset:           r.Header.Get("Accept-Charset"),
		Accept:                  r.Header.Get("Accept"),
		Connection:              r.Header.Get("Connection"),
		Host:                    r.Host,
		XForwardedFor:           r.Header.Get("X-Forwarded-For"),
		Referer:                 r.Referer(),
		Cookie:                  r.Header.Get("Cookie"),
		DNT:                     r.Header.Get("DNT"),
		UpgradeInsecureRequests: r.Header.Get("Upgrade-Insecure-Requests"),
		CacheControl:            r.Header.Get("Cache-Control"),
		Pragma:                  r.Header.Get("Pragma"),
		Via:                     r.Header.Get("Via"),
		Forwarded:               r.Header.Get("Forwarded"),
		XRealIP:                 r.Header.Get("X-Real-IP"),
		XForwardedProto:         r.Header.Get("X-Forwarded-Proto"),
		XForwardedHost:          r.Header.Get("X-Forwarded-Host"),
		XForwardedPort:          r.Header.Get("X-Forwarded-Port"),
		XAmzDate:                r.Header.Get("X-Amz-Date"),
		XApiKey:                 r.Header.Get("X-Api-Key"),
		XRequestID:              r.Header.Get("X-Request-ID"),
		Authorization:           r.Header.Get("Authorization"),
		ContentType:             r.Header.Get("Content-Type"),
		ContentLength:           r.ContentLength,
		Method:                  r.Method,
		RequestURI:              r.RequestURI,
		Protocol:                r.Proto,
		TransferEncoding:        r.TransferEncoding,
		TLSVersion:              tlsVersion,
		TLSCipherSuite:          tlsCipherSuite,
	}
}

// Middleware extracts the analytics data from the http.Request object and upload them to the database.
// If everything goes as expected it passes to the next handler, otherwise it returns an internal server error 500
func (g gam) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract fingerprint
		fingerprint := extractAnalyticsData(r)
		err := g.addAnalyticsDataToDB(fingerprint)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// addAnalyticsDataToDB upload the AnalyticsData passed in parameters to the database
func (g gam) addAnalyticsDataToDB(data analyticsData) error {
	tx, err := g.db.Begin()
	if err != nil {
		log.Printf("Error while starting transaction %v", err)
		return err
	}
	// Insert fingerprint into ClickHouse
	_, err = tx.Exec(fmt.Sprintf(`
INSERT INTO %s (
	user_agent, ip_address, accept_language, accept_encoding, 
	accept_charset, accept, connection, host, x_forwarded_for, 
	referer, cookie, dnt, upgrade_insecure_requests, cache_control, 
	pragma, via, forwarded, x_real_ip, x_forwarded_proto, x_forwarded_host,
	x_forwarded_port, x_amz_date, x_api_key, x_request_id, authorization, 
	content_type, content_length, method, request_uri, protocol, 
	transfer_encoding, tls_version, tls_cipher_suite
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, g.tableName),
		data.UserAgent, data.IPAddress, data.AcceptLanguage,
		data.AcceptEncoding, data.AcceptCharset, data.Accept,
		data.Connection, data.Host, data.XForwardedFor,
		data.Referer, data.Cookie, data.DNT,
		data.UpgradeInsecureRequests, data.CacheControl, data.Pragma,
		data.Via, data.Forwarded, data.XRealIP, data.XForwardedProto,
		data.XForwardedHost, data.XForwardedPort, data.XAmzDate,
		data.XApiKey, data.XRequestID, data.Authorization,
		data.ContentType, data.ContentLength, data.Method,
		data.RequestURI, data.Protocol, data.TransferEncoding,
		data.TLSVersion, data.TLSCipherSuite,
	)
	if err != nil {
		tx.Rollback()
		log.Printf("Error inserting fingerprint: %v", err)
		return err
	}
	err = tx.Commit()
	if err != nil {
		log.Printf("Error while committing transaction %v", err)
		return err
	}
	return nil
}

// func main() {
// 	db, err := sql.Open("clickhouse", "tcp://localhost:9000?database=default")
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	mux := http.NewServeMux()
// 	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		w.Write([]byte("Hello, World!"))
// 	}))

// 	// Use the middleware
// 	loggedMux := analyticsMiddleware(db, mux)

// 	log.Fatal(http.ListenAndServe(":8080", loggedMux))
// }
