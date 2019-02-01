package main

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/haydenwoodhead/burner.kiwi/data/dynamodb"

	"github.com/haydenwoodhead/burner.kiwi/data/inmemory"
	"github.com/haydenwoodhead/burner.kiwi/data/postgresql"

	"github.com/haydenwoodhead/burner.kiwi/data"

	"github.com/haydenwoodhead/burner.kiwi/server"
)

const inMemory = "memory"
const postgreSQL = "postgres"
const dynamoDB = "dynamo"

func mustParseNewServerInput() server.NewServerInput {
	dbType := parseStringVarWithDefault("DB_TYPE", inMemory)

	var db data.Database

	switch dbType {
	case inMemory:
		db = inmemory.GetInMemoryDB()
	case postgreSQL:
		db = postgresql.GetPostgreSQLDB(mustParseStringVar("DATABASE_URL"))
	case dynamoDB:
		db = dynamodb.GetNewDynamoDB(mustParseStringVar("DYNAMO_TABLE"))
	}

	return server.NewServerInput{
		Key:         mustParseStringVar("KEY"),
		URL:         mustParseStringVar("WEBSITE_URL"),
		StaticURL:   mustParseStringVar("STATIC_URL"),
		MGKey:       mustParseStringVar("MG_KEY"),
		MGDomain:    mustParseStringVar("MG_DOMAIN"),
		Developing:  parseBoolVarWithDefault("DEVELOPING", false),
		Domains:     mustParseSliceVar("DOMAINS"),
		UsingLambda: parseBoolVarWithDefault("LAMBDA", false),
		Database:    db,
	}
}

func parseStringVar(key string) string {
	return os.Getenv(key)
}

func parseBoolVar(key string) (bool, error) {
	val := mustParseStringVar(key)
	return strconv.ParseBool(val)
}

func mustParseStringVar(key string) (v string) {
	v = parseStringVar(key)
	if strings.Compare(v, "") == 0 {
		log.Fatalf("Env var %v cannot be empty", key)
	}

	return
}

func mustParseSliceVar(key string) (v []string) {
	val := mustParseStringVar(key)
	split := strings.Split(val, ",")

	for _, s := range split {
		v = append(v, strings.TrimSpace(s))
	}

	return
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
