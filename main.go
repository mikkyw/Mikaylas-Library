package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Book struct {
	ID        int
	StartDate string
	EndDate   string
	Rating    int
	BookName  string
	Stars     string
}

var db *sql.DB

func main() {
	var err error
	err = godotenv.Load()
	if err != nil {
		log.Fatal(".env file not found")
	}

	connStr := os.Getenv("DB_CONN_STR")

	fmt.Println("Attempting to connect to the database...")
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Connection error:", err)
	}
	defer db.Close()
	if err = db.Ping(); err != nil {
		log.Fatal("Could not connect to the database:", err)
	}

	fmt.Println("Connected to the database!")
	http.HandleFunc("/", serveForm)
	http.HandleFunc("/insert", insertBook)
	fmt.Println("Server running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func serveForm(w http.ResponseWriter, r *http.Request) {
	books, err := getRecentBooks()
	if err != nil {
		http.Error(w, "Error fetching recent books", http.StatusInternalServerError)
		return
	}

	favbooks, err := getFavBooks()
	if err != nil {
		http.Error(w, "Error fetching favorite books", http.StatusInternalServerError)
		return
	}

	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	tmpl.Execute(w, struct {
		Books    []Book
		FavBooks []Book
	}{
		Books:    books,
		FavBooks: favbooks,
	})
}

func displayForm(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>Insert Book</title>
    </head>
    <body>
        <h2>Insert a New Book</h2>
        <form action="/insert" method="POST">
            <label for="start_date">Start Date:</label>
            <input type="date" id="start_date" name="start_date" required><br><br>

            <label for="end_date">End Date:</label>
            <input type="date" id="end_date" name="end_date" required><br><br>

            <label for="rating">Rating:</label>
            <input type="number" id="rating" name="rating" step="1" min="1" max="5" required><br><br>

            <label for="book_name">Book Name:</label>
            <input type="text" id="book_name" name="book_name" required><br><br>

            <button type="submit">Insert Book</button>
        </form>
    </body>
    </html>`)
}

func insertBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Request is invalid", http.StatusMethodNotAllowed)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Could not parse form", http.StatusInternalServerError)
		return
	}

	startDate := r.FormValue("start_date")
	endDate := r.FormValue("end_date")
	rating := r.FormValue("rating")
	bookName := r.FormValue("book_name")

	_, err = db.Exec("INSERT INTO books_read (start_date, end_date, rating, book_name) VALUES ($1, $2, $3, $4)", startDate, endDate, rating, bookName)
	if err != nil {
		http.Error(w, "Error inserting book", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func getRecentBooks() ([]Book, error) {
	rows, err := db.Query("SELECT id, start_date, end_date, rating, book_name FROM books_read ORDER BY end_date DESC LIMIT 5")

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book

	for rows.Next() {
		var book Book

		if err := rows.Scan(&book.ID, &book.StartDate, &book.EndDate, &book.Rating, &book.BookName); err != nil {
			return nil, err
		}
		fullStars := strings.Repeat("★", book.Rating)
		emptyStars := strings.Repeat("☆", 5-book.Rating)
		book.Stars = fullStars + emptyStars
		book.StartDate = FormatDate(book.StartDate)
		book.EndDate = FormatDate(book.EndDate)

		books = append(books, book)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return books, nil
}

func getFavBooks() ([]Book, error) {
	rows, err := db.Query("SELECT id, start_date, end_date, rating, book_name FROM books_read WHERE rating = 5 LIMIT 5")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var books []Book

	for rows.Next() {
		var book Book

		if err := rows.Scan(&book.ID, &book.StartDate, &book.EndDate, &book.Rating, &book.BookName); err != nil {
			return nil, err
		}
		book.Stars = "★★★★★"
		book.StartDate = FormatDate(book.StartDate)
		book.EndDate = FormatDate(book.EndDate)

		books = append(books, book)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return books, nil
}

func FormatDate(dateStr string) string {
	parsedTime, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return parsedTime.Format("January 2, 2006")
}
