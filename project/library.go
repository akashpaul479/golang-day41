package project

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

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

type Borrow_records struct {
	Borrowid   int        `json:"borrow_id"`
	Userid     int        `json:"user_id"`
	Usertype   string     `json:"usertype"`
	Bookid     int        `json:"book_id"`
	Borrowdate *time.Time `json:"bowwow_date"`
	Returndate *time.Time `json:"return_date"`
}

func ValidateLibrary(library *Library) error {
	if strings.TrimSpace(library.Title) == "" {
		return fmt.Errorf("title cannot be empty")
	}
	if len(library.Book) == 0 {
		return fmt.Errorf("atleast one book is required")
	}
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

func ValidateBorrowRecords(BR Borrow_records) error {
	if BR.Bookid <= 0 {
		return fmt.Errorf("invalid Borrow_id")
	}
	if BR.Userid <= 0 {
		return fmt.Errorf("invalid user_id")
	}
	if BR.Usertype == "" {
		return fmt.Errorf("user type cannot be empty")
	}
	if BR.Bookid <= 0 {
		return fmt.Errorf("invalid book_id")
	}
	if BR.Borrowdate == nil {
		return fmt.Errorf("borrow_date is required")
	}
	if BR.Returndate != nil && BR.Returndate.Before(*BR.Borrowdate) {
		return fmt.Errorf("return_date cannot be before borrow_date")
	}
	return nil
}

func (a *HybridHandler) CreateLibraryHandler(w http.ResponseWriter, r *http.Request) {
	var libraries Library
	if err := json.NewDecoder(r.Body).Decode(&libraries); err != nil {
		http.Error(w, "failed to decode response", http.StatusInternalServerError)
		return
	}
	if err := ValidateLibrary(&libraries); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
	}
	tx, err := a.MySQL.db.Begin()
	if err != nil {
		http.Error(w, "failed to start transcation", http.StatusInternalServerError)
		return
	}

	res, err := tx.Exec("INSERT INTO libraries (title , availablecopies) VALUES (? , ?)", libraries.Title, libraries.Availablecopies)
	if err != nil {
		tx.Rollback()
		http.Error(w, "failed to insert libraries", http.StatusInternalServerError)
		return
	}
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
	if err := tx.Commit(); err != nil {
		http.Error(w, "failed to commit transaction", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "library created succesfully", "library_id": libraryID,
	})
}
