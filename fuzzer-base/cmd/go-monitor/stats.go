package main

import (
	"bytes"
	"encoding/json"
	"sync"

	"maxfuzz/fuzzer-base/internal/helpers"

	sse "astuart.co/go-sse"
	"github.com/graphql-go/graphql"
)

type GoFuzzStats struct {
	Workers          int    `json:"Workers"`
	Corpus           int    `json:"Corpus"`
	Crashers         int    `json:"Crashers"`
	Execs            int    `json:"Execs"`
	Cover            int    `json:"Cover"`
	RestartsDenom    int    `json:"RestartsDenom"`
	LastNewInputTime string `json:"LastNewInputTime"`
	StartTime        string `json:"StartTime"`
	Uptime           string `json:"Uptime"`
}

var gqlStatsType = graphql.NewObject(graphql.ObjectConfig{
	Name: "Stats",
	Fields: graphql.Fields{
		"Workers": &graphql.Field{
			Type: graphql.Int,
		},
		"Corpus": &graphql.Field{
			Type: graphql.Int,
		},
		"Crashers": &graphql.Field{
			Type: graphql.Int,
		},
		"Execs": &graphql.Field{
			Type: graphql.Int,
		},
		"Cover": &graphql.Field{
			Type: graphql.Int,
		},
		"RestartsDenom": &graphql.Field{
			Type: graphql.Int,
		},
		"LastNewInputTime": &graphql.Field{
			Type: graphql.String,
		},
		"StartTime": &graphql.Field{
			Type: graphql.String,
		},
		"Uptime": &graphql.Field{
			Type: graphql.String,
		},
	},
})

var rootQuery = graphql.NewObject(graphql.ObjectConfig{
	Name: "RootQuery",
	Fields: graphql.Fields{
		"stats": &graphql.Field{
			Type:        gqlStatsType,
			Description: "Get fuzzer stats",
			Resolve:     resolveStats,
		},
	},
})

var schema, _ = graphql.NewSchema(graphql.SchemaConfig{
	Query: rootQuery,
})

var statsMutex = &sync.Mutex{}
var mainStats = &GoFuzzStats{}

func resolveStats(params graphql.ResolveParams) (interface{}, error) {
	statsMutex.Lock()
	returnStats := *mainStats
	statsMutex.Unlock()
	return &returnStats, nil
}

func updateStats(evCh chan *sse.Event) {
	e := <-evCh
	buf := new(bytes.Buffer)
	buf.ReadFrom(e.Data)
	s := buf.String()
	newStats := GoFuzzStats{}
	err := json.Unmarshal([]byte(s), &newStats)
	if err != nil {
		helpers.Check("Failed to unmarshal stats: %v", err)
	}
	statsMutex.Lock()
	mainStats = &newStats
	statsMutex.Unlock()
}
