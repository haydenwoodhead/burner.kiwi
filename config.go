package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/haydenwoodhead/burner.kiwi/burner"
	"github.com/haydenwoodhead/burner.kiwi/data/dynamodb"
	"github.com/haydenwoodhead/burner.kiwi/data/inmemory"
	"github.com/haydenwoodhead/burner.kiwi/data/postgresql"
	"github.com/haydenwoodhead/burner.kiwi/data/sqlite3"
	"github.com/haydenwoodhead/burner.kiwi/email/mailgunmail"
	"github.com/haydenwoodhead/burner.kiwi/email/smtpmail"
)

const inMemory = "memory"
const postgreSQL = "postgres"
const dynamoDB = "dynamo"
const sqLite3 = "sqlite3"

const mailgunProvider = "mailgun"
const smtpProvider = "smtp"

func mustParseConfig() (burner.Config, burner.Database, burner.EmailProvider, string) {
	dbType := parseStringVarWithDefault("DB_TYPE", inMemory)

	var db burner.Database

	switch dbType {
	case inMemory:
		db = inmemory.GetInMemoryDB()
	case postgreSQL:
		db = postgresql.GetPostgreSQLDB(mustParseStringVar("DATABASE_URL"))
	case dynamoDB:
		db = dynamodb.GetNewDynamoDB(mustParseStringVar("DYNAMO_TABLE"))
	case sqLite3:
		db = sqlite3.GetSQLite3DB(mustParseStringVar("DATABASE_URL"))
	}

	emailType := mustParseStringVar("EMAIL_TYPE")

	var email burner.EmailProvider

	switch emailType {
	case mailgunProvider:
		email = mailgunmail.NewMailProvider(mustParseStringVar("MG_DOMAIN"), mustParseStringVar("MG_KEY"))
	case smtpProvider:
		email = smtpmail.NewMailProvider(parseStringVarWithDefault("SMTP_LISTEN", ":25"))
	}

	listenAddr := parseStringVarWithDefault("LISTEN", ":8080")

	return burner.Config{
		Key:                mustParseStringVar("KEY"),
		URL:                mustParseStringVar("WEBSITE_URL"),
		StaticURL:          mustParseStringVar("STATIC_URL"),
		Developing:         parseBoolVarWithDefault("DEVELOPING", false),
		Domains:            mustParseSliceVar("DOMAINS"),
		UsingLambda:        parseBoolVarWithDefault("LAMBDA", false),
		RestoreRealIP:      parseBoolVarWithDefault("RESTOREREALIP", false),
		BlacklistedDomains: parseSliceVar("BLACKLISTED"),
	}, db, email, listenAddr
}

func parseStringVar(key string) string {
	return os.Getenv(key)
}

func parseBoolVar(key string) (bool, error) {
	val := parseStringVar(key)
	return strconv.ParseBool(val)
}

func mustParseStringVar(key string) (v string) {
	v = parseStringVar(key)
	if v == "" {
		log.Fatalf("Env var %v cannot be empty", key)
	}
	return
}

func parseSliceVar(key string) (v []string) {
	val := parseStringVar(key)
	split := strings.Split(val, ",")

	for _, s := range split {
		v = append(v, strings.TrimSpace(s))
	}

	return
}

func mustParseSliceVar(key string) []string {
	v := parseSliceVar(key)
	if len(v) == 0 {
		log.Fatalf("Env var %v cannot be empty", key)
	}
	return v
}

func parseBoolVarWithDefault(key string, def bool) bool {
	v, err := parseBoolVar(key)
	if err != nil {
		return def
	}
	return v
}

func parseStringVarWithDefault(key, def string) string {
	v := parseStringVar(key)
	if v == "" {
		return def
	}
	return v
}
