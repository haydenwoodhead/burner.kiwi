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
)

const inMemory = "memory"
const postgreSQL = "postgres"
const dynamoDB = "dynamo"
const sqLite3 = "sqlite3"

const mailgunProvider = "mailgun"

func mustParseNewServerInput() burner.NewInput {
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

	emailType := parseStringVarWithDefault("EMAIL_TYPE", mailgunProvider)

	var email burner.EmailProvider

	switch emailType {
	case mailgunProvider:
		email = mailgunmail.NewMailgunProvider(mustParseStringVar("MG_DOMAIN"), mustParseStringVar("MG_KEY"))
	}

	return burner.NewInput{
		Key:                mustParseStringVar("KEY"),
		URL:                mustParseStringVar("WEBSITE_URL"),
		StaticURL:          mustParseStringVar("STATIC_URL"),
		Developing:         parseBoolVarWithDefault("DEVELOPING", false),
		Domains:            mustParseSliceVar("DOMAINS"),
		UsingLambda:        parseBoolVarWithDefault("LAMBDA", false),
		RestoreRealIP:      parseBoolVarWithDefault("RESTOREREALIP", false),
		BlacklistedDomains: parseSliceVar("BLACKLISTED"),
		Database:           db,
		Email:              email,
	}
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
