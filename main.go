package main

import (
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/valyala/fasthttp"

	"os"

	_ "github.com/go-sql-driver/mysql"
)

type Value struct {
	Name  string    `db:"name"`
	Value string    `db:"value"`
	Now   time.Time `db:"now"`
}

var (
	db    *sqlx.DB
	queCh = make(chan Value, 1000)
)

func inserter() {
	ticker := time.NewTicker(1 * time.Second)
	valueQue := make([]Value, 0, 1000)
	for {
		select {
		case <-ticker.C:
			if len(valueQue) == 0 {
				continue
			}
			query := "INSERT INTO eventlog(at, name, value) values(?, ?, ?)" + strings.Repeat(", (?, ?, ?)", len(valueQue)-1)
			stmt, e := db.Prepare(query)
			if e != nil {
				panic(e.Error())
			}

			defer stmt.Close()

			args := make([]interface{}, 3*len(valueQue))
			for i, que := range valueQue {
				args[i] = que.Now
				args[i+1] = que.Name
				args[i+2] = que.Value
			}
			_, _ = stmt.Exec(args...)
			valueQue = make([]Value, 0, 1000)

		case que := <-queCh:
			valueQue = append(valueQue, que)

		}
	}
}
func hakaruHandler(ctx *fasthttp.RequestCtx) {
	now := time.Now()
	name := string(ctx.QueryArgs().Peek("name"))
	value := string(ctx.URI().QueryArgs().Peek("value"))

	queCh <- Value{
		Now:   now,
		Name:  name,
		Value: value,
	}

	origin := string(ctx.Request.Header.Peek("Origin")[0])
	if origin != "" {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", origin)
		ctx.Response.Header.Set("Access-Control-Allow-Credentials", "true")
	} else {
		ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	}
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET")
}

func main() {
	dataSourceName := os.Getenv("HAKARU_DATASOURCENAME")
	if dataSourceName == "" {
		dataSourceName = "root:hakaru-pass@tcp(127.0.0.1:13306)/hakaru-db"

	}

	_db, err := sqlx.Open("mysql", dataSourceName)
	if err != nil {
		panic(err.Error())
	}
	defer _db.Close()

	db = _db

	db.SetMaxIdleConns(20)
	db.SetMaxOpenConns(20)
	router := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/ok":
			ctx.SetStatusCode(200)
		case "/hakaru":
			hakaruHandler(ctx)
		default:
			ctx.Error("not found", fasthttp.StatusNotFound)
		}
	}

	// start server
	if err := fasthttp.ListenAndServe(":8081", router); err != nil {
		log.Fatal(err)
	}
}
