package project

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// Library represents a library entity
type Library struct {
	Libraryid int `json:"library_id"`
	Book      []struct {
		Bookid   int    `json:"book_id"`
		Bookname string `json:"book_name"`
	} `json:"book"`
	Title  string `json:"title"`
	Author []struct {
		Authorid   int    `json:"author_id"`
		Authorname string `json:"author_name"`
	} `json:"author"`
	Availablecopies int `json:"available_copies"`
}

// borrowrecords represents borrowing transactions
type Borrowrecords struct {
	Borrowid   int        `json:"borrow_id"`
	Userid     int        `json:"user_id"`
	Usertype   string     `json:"usertype"`
	Bookid     int        `json:"book_id"`
	Borrowdate *time.Time `json:"bowwow_date"`
	Returndate *time.Time `json:"return_date"`
}

// validate library ensures that library input data is valid before DB operations
func ValidateLibrary(library *Library) error {
	// validate title
	if strings.TrimSpace(library.Title) == "" {
		return fmt.Errorf("title cannot be empty")
	}
	// validate book
	if len(library.Book) == 0 {
		return fmt.Errorf("atleast one book is required")
	}
	// validate availablecopies
	if library.Availablecopies < 0 {
		return fmt.Errorf("availabe copies cannot be negative")
	}
	for _, b := range library.Book {
		if b.Bookid <= 0 {
			return fmt.Errorf("invalid Book_id: %d", b.Bookid)
		}
		if b.Bookname == "" {
			return fmt.Errorf("book name cannot be empty")
		}
	}
	for _, a := range library.Author {
		if a.Authorid <= 0 {
			return fmt.Errorf("invalid author_id: %d", a.Authorid)
		}
		if a.Authorname == "" {
			return fmt.Errorf("author_name cannot be empty")
		}
	}
	return nil
}

// validateBorrowRecords ensures borrow record input is valid
func ValidateBorrowRecords(BR Borrowrecords) error {
	if BR.Bookid <= 0 {
		return fmt.Errorf("invalid Borrow_id")
	}
	if BR.Userid <= 0 {
		return fmt.Errorf("invalid user_id")
	}
	if BR.Usertype == "" {
		return fmt.Errorf("user type cannot be empty")
	}

	if BR.Borrowdate == nil {
		return fmt.Errorf("borrow_date is required")
	}
	if BR.Returndate != nil && BR.Returndate.Before(*BR.Borrowdate) {
		return fmt.Errorf("return_date cannot be before borrow_date")
	}
	return nil
}

// createlibraryhandler handles creation of a new library
func (a *HybridHandler) CreateLibraryHandler(w http.ResponseWriter, r *http.Request) {

	// Decode incoming JSON requests body
	var libraries Library
	if err := json.NewDecoder(r.Body).Decode(&libraries); err != nil {
		http.Error(w, "failed to decode response", http.StatusInternalServerError)
		return
	}
	// validate input
	if err := ValidateLibrary(&libraries); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
		return
	}

	// Begin transaction
	tx, err := a.MySQL.db.Begin()
	if err != nil {
		http.Error(w, "failed to start transcation", http.StatusInternalServerError)
		return
	}

	// Insert library records
	res, err := tx.Exec("INSERT INTO libraries (title , availablecopies) VALUES (? , ?)", libraries.Title, libraries.Availablecopies)
	if err != nil {
		tx.Rollback()
		http.Error(w, "failed to insert libraries", http.StatusInternalServerError)
		return
	}

	// Get generated library ID
	libraryID, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		http.Error(w, "Failed to fetch libraryID", http.StatusInternalServerError)
		return
	}
	// Insert BOOKS
	for _, b := range libraries.Book {
		_, err := tx.Exec("INSERT INTO books (book_id, book_name , library_id) VALUES (? , ? , ?)", b.Bookid, b.Bookname, libraryID)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to insert books", http.StatusInternalServerError)
			return
		}
	}
	// Insert authors
	for _, a := range libraries.Author {
		_, err := tx.Exec("INSERT INTO authors (author_id , author_name , library_id) VALUES (? , ? ,?)", a.Authorid, a.Authorname, libraryID)
		if err != nil {
			tx.Rollback()
			http.Error(w, "Failed to insert authors", http.StatusInternalServerError)
			return
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}

	// success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "library created succesfully", "library_id": libraryID,
	})
}

// GetLibraryHandler
func (a *HybridHandler) GetLibraryByIDHandler(w http.ResponseWriter, r *http.Request) {

	// Extract id from URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Try redis cache
	value, err := a.Redis.Client.Get(a.Ctx, id).Result()
	if err == nil {
		log.Println("cache Hit...")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(value))
		return
	}

	// cache miss querying MySQL
	fmt.Println("cache miss querying MySQL...")
	var lib Library

	// Fetch library details
	err = a.MySQL.db.QueryRow("SELECT title , availablecopies FROM libraries WHERE library_id=?", id).Scan(&lib.Title, &lib.Availablecopies)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "library not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to fetch library", http.StatusInternalServerError)
		return
	}
	// Fetch Books
	booksRows, err := a.MySQL.db.Query("SELECT book_id , book_name FROM books WHERE library_id", id)
	if err != nil {
		http.Error(w, "failed to fetch books", http.StatusInternalServerError)
		return
	}
	defer booksRows.Close()

	for booksRows.Next() {
		var b struct {
			Bookid   int    `json:"book_id"`
			Bookname string `json:"book_name"`
		}
		if err := booksRows.Scan(&b.Bookid, &b.Bookname); err != nil {
			http.Error(w, "Failed to scan books", http.StatusInternalServerError)
			return
		}
		lib.Book = append(lib.Book, b)
	}
	// Fetch authors
	authorrows, err := a.MySQL.db.Query("SELECT author_id , author_name FROM authors WHERE library_id=?", id)
	if err != nil {
		http.Error(w, "failed to fetch authors", http.StatusInternalServerError)
		return
	}
	defer authorrows.Close()

	for authorrows.Next() {
		var aObj struct {
			Authorid   int    `json:"author_id"`
			Authorname string `json:"author_name"`
		}
		if err := authorrows.Scan(&aObj.Authorid, &aObj.Authorname); err != nil {
			http.Error(w, "Failed to scan author", http.StatusInternalServerError)
			return
		}
		lib.Author = append(lib.Author, aObj)

	}
	//  marshal results to json

	responseJson, err := json.Marshal(lib)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}
	// cache in redis
	a.Redis.Client.Set(a.Ctx, id, responseJson, 10*time.Minute)

	// return response
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseJson)
}

// Borrow books
func (a *HybridHandler) Borrowbooks(w http.ResponseWriter, r *http.Request) {

	// Decode incoming JSON requests body
	var records Borrowrecords
	if err := json.NewDecoder(r.Body).Decode(&records); err != nil {
		http.Error(w, "failed to decode response", http.StatusInternalServerError)
		return
	}
	// validate
	if records.Usertype != "student" && records.Usertype != "lecturer" {
		http.Error(w, "Invalid user type , user must be student or lecturer", http.StatusBadRequest)
		return
	}
	//  check book is available
	var available int
	err := a.MySQL.db.QueryRow("SELECT available_copies FROM books WHERE book_id=?", records.Bookid).Scan(&available)
	if err != nil {
		http.Error(w, "Book not found", http.StatusNotFound)
		return
	}
	if available <= 0 {
		http.Error(w, "book not available", http.StatusBadRequest)
		return
	}
	// Insert borrow records
	_, err = a.MySQL.db.Exec("INSERT INTO borrow_records(user_id , usertype , book_id , borrow_date) VALUES (? , ? , ? , ?),CURDATE()", records.Userid, records.Usertype, records.Bookid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Decrement availables copies
	_, err = a.MySQL.db.Exec("UPDATE books available_copies= available_copies-1 WHERE book_id=?", records.Bookid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "book borrowed"})
}

// Return books
func (a *HybridHandler) ReturnBooksHandler(w http.ResponseWriter, r *http.Request) {

	// Decode incoming JSON requests body
	var records Borrowrecords
	if err := json.NewDecoder(r.Body).Decode(&records); err != nil {
		http.Error(w, "failed to decode response", http.StatusInternalServerError)
		return
	}

	// validate
	if records.Usertype != "student" && records.Usertype != "lecturer" {
		http.Error(w, "invalid user type", http.StatusInternalServerError)
		return
	}

	// Execute update query
	res, err := a.MySQL.db.Exec("UPDATE borrow_records SET return_date=CURDATE() WHERE user_id=? AND book_id=? AND return_date is NULL", records.Userid, records.Bookid)
	if err != nil {
		http.Error(w, "failed to update", http.StatusInternalServerError)
		return
	}
	rows, _ := res.LastInsertId()
	if rows == 0 {
		http.Error(w, "no borrow records found", http.StatusInternalServerError)
		return
	}

	// increment available copies
	_, err = a.MySQL.db.Exec("UPDATE books SET available_copies=available_copies+1 WHERE book_id=?", records.Bookid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "book returned"})
}
