package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
)

func main() {

	dataSourceName := os.Getenv("HAKARU_DATASOURCENAME")
	if dataSourceName == "" {
		dataSourceName = "root:hakaru-pass@tcp(127.0.0.1:13306)/hakaru-db"
	}
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		panic(err.Error())
	}
	defer db.Close()

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(10)

	hakaru := HakaruHandler{DB: db}

	http.Handle("/hakaru", hakaru)
	http.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) })

	// start server
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

type HakaruHandler struct {
	DB *sql.DB
}

func (h HakaruHandler)ServeHTTP(w http.ResponseWriter, r *http.Request) {
	stmt, e := h.DB.Prepare("INSERT INTO eventlog(at, name, value) values(NOW(), ?, ?)")
	if e != nil {
		panic(e.Error())
	}

	defer stmt.Close()

	name := r.URL.Query().Get("name")
	value := r.URL.Query().Get("value")

	_, _ = stmt.Exec(name, value)

	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
}
