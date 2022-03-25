package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type Server struct {
	Db *sql.DB
	Router *http.ServeMux
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	s := runServer()

	db, err := sql.Open("sqlite3", "movies.db")

	if err != nil {
		return err
	}
	// assign the database before continuing
	s.Db = db
	if err := s.migrate(); err != nil {
		return err
	}
	if err := s.seed(); err != nil {
		return err
	}
	// routes get initilized last so we have all the dependencies setup
	// we could change this and pass in the db directly to the handler
	// but this feels weird
	if err := s.routes(); err != nil {
		return err
	}
	if err := http.ListenAndServe(":4000", s); err != nil {
		return err
	}
	return nil
}

func (s *Server) migrate() error {
	sqlStmt := `
	CREATE TABLE IF NOT EXISTS main.movies (
		id  INTEGER PRIMARY KEY NOT NULL,
		title STRING NOT NULL,
		rating REAL NOT NULL
	);
	`
	_, err := s.Db.Exec(sqlStmt)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) seed() error {
	// no need to seed if we already have data
	row := s.Db.QueryRow("SELECT * FROM movies LIMIT 1;")
	var title string
	row.Scan(&title)
	if title != "" {
		return nil
	}

	stmt, err := os.ReadFile("seed.sql")
	if err != nil {
		return err
	}
	// return error if problem seeding
	if _, err := s.Db.Exec(string(stmt)); err != nil {
		return err
	}
	// should be all good bebbe ;)
	return nil
}

func runServer() *Server {
	s := &Server{
		Router: &http.ServeMux{},
	}
	return s
}

func (s *Server)ServeHTTP(w http.ResponseWriter, r *http.Request){
	s.Router.ServeHTTP(w,r)
}

func (s *Server)routes() error {
	s.Router.HandleFunc("/", s.handleHello())
	s.Router.HandleFunc("/movies", s.handleMovies())
	return nil
}

func (s *Server) handleHello() http.HandlerFunc {
	type Human struct {
		Name string `json:"name"`
	}
	h := Human{Name: "Eddie"}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(h)
		fmt.Fprintln(w, &buf)
	}
}

func (s *Server) handleMovies() http.HandlerFunc {
	type Movie struct {
		Title string `json:"title"`
		Rating float64 `json:"rating"`
	}

	movies := []Movie{}
	rows, err := s.Db.Query("SELECT title, rating FROM movies;")
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		m := Movie{}
		err = rows.Scan(&m.Title, &m.Rating)
		if err != nil {
			log.Fatal(err)
		}
		movies = append(movies, m)
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}	

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var buf bytes.Buffer
		json.NewEncoder(&buf).Encode(movies)
		fmt.Fprintln(w, &buf)
	}
}