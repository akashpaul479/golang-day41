package project

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

// Student 	represent a student entity stored in mysql
type Student struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
	Dept  string `json:"dept"`
}

// Validatestudent validates incoming student data
func ValidateStudent(student Student) error {
	// Name validation
	if strings.TrimSpace(student.Name) == "" {
		return fmt.Errorf("Empty name or invalid name")
	}
	// Email validation
	if strings.TrimSpace(student.Email) == "" {
		return fmt.Errorf("Empty email or invalid email")
	}
	if !strings.HasSuffix(student.Email, "@gmail.com") {
		return fmt.Errorf("email is invalid and does not contains @gmail.com")
	}
	prefix := strings.TrimSuffix(student.Email, "@gmail.com")
	if prefix == "" {
		return fmt.Errorf("email must contains a prefix before @gmail.com")
	}
	// Department validation
	if strings.TrimSpace(student.Dept) == "" {
		return fmt.Errorf("Empty dept or invalid dept")
	}
	// Age validation
	if student.Age <= 0 {
		return fmt.Errorf("Age is less than 0 ")
	}
	if student.Age >= 100 {
		return fmt.Errorf("Age is grater than 0")
	}
	return nil
}

// CreateStudentHandler handles creation of a new student
func (a *HybridHandler) CreateStudentHandler(w http.ResponseWriter, r *http.Request) {

	// Decode incoming JSON request body
	var students Student
	if err := json.NewDecoder(r.Body).Decode(&students); err != nil {
		http.Error(w, "Failed to decode response", http.StatusInternalServerError)
		return
	}

	// validate requests payload
	if err := ValidateStudent(students); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
		return
	}

	// Insert student record into MySQL database
	res, err := a.MySQL.db.Exec("INSERT INTO students (name , age , email , dept) VALUES (? , ? , ? , ?)", students.Name, students.Age, students.Email, students.Dept)
	if err != nil {
		http.Error(w, "Unable to insert", http.StatusInternalServerError)
		return
	}

	// auto_generated id
	id, err := res.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	students.Id = int(id)

	// Lod activity and Audit trail
	go LogActivity("CREATE_EMPLOYEE", "system")
	go AuditLog("CREATE", "EMPLOYEE", students.Id, "system")

	// send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(students)
}

// GetStudentHandler to get all students
func (a *HybridHandler) GetStudentHandler(w http.ResponseWriter, r *http.Request) {
	rows, err := a.MySQL.db.Query("SELECT id , name , age , email , dept FROM students")
	if err != nil {
		http.Error(w, "unable to fetch students", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var students []Student
	for rows.Next() {
		var s Student
		if err := rows.Scan(&s.Id, &s.Name, &s.Age, &s.Email, &s.Dept); err != nil {
			http.Error(w, "rows scan failed", http.StatusInternalServerError)
			return
		}
		students = append(students, s)

	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}

// GetStudentByIDHandler retrives a student by id
func (a *HybridHandler) GetstudentByIDHandler(w http.ResponseWriter, r *http.Request) {
	// Extract student id from URl
	vars := mux.Vars(r)
	id := vars["id"]

	// Log Get activity
	go LogActivity("GET_EMPLOYEE", "system")

	// Attempt to fetch from Redis cache first
	value, err := a.Redis.Client.Get(a.Ctx, id).Result()
	if err == nil {
		log.Println("Cache Hit...")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(value))
		return
	}
	// cache miss fetching from MySQL database
	fmt.Println("cache miss querying MySQL...")
	row := a.MySQL.db.QueryRow("SELECT name , age , email , dept FROM students WHERE id=?", id)

	var students Student
	if err := row.Scan(&students.Name, &students.Age, &students.Email, &students.Dept); err != nil {
		http.Error(w, "student not found ", http.StatusNotFound)
		return
	}

	// Marshal student data for caching
	jsonData, err := json.Marshal(students)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// store result in a redis cache (short TTL)
	go a.Redis.Client.Set(a.Ctx, id, jsonData, 10*time.Second)

	//  send response
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// UpdateStudentHandler updates a exsisting student
func (a *HybridHandler) UpdateStudentHandler(w http.ResponseWriter, r *http.Request) {

	// Decode request Body
	var students Student
	if err := json.NewDecoder(r.Body).Decode(&students); err != nil {
		http.Error(w, "Failed to decode response", http.StatusInternalServerError)
		return
	}

	// validate updated data
	if err := ValidateStudent(students); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
		return
	}

	// Execute updated query
	res, err := a.MySQL.db.Exec("UPDATE students SET name=? , age=? , email=? , dept=? WHERE id=?", students.Name, students.Age, students.Email, students.Dept, students.Id)
	if err != nil {
		http.Error(w, "unable to update", http.StatusInternalServerError)
		return
	}

	//  check if record exsists
	rows, err := res.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == 0 {
		http.Error(w, "user not found ", http.StatusNotFound)
		return
	}

	// update redis cache
	jsonData, err := json.Marshal(students)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go a.Redis.Client.Set(a.Ctx, fmt.Sprint(students.Id), jsonData, 10*time.Second)

	// Log update actions
	go LogActivity("UPDATE_STUDENT", "system")
	go AuditLog("UPDATE", "STUDENT", students.Id, "system")

	//  send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

// DeleteStudentHandler deletes a student by ID
func (a *HybridHandler) DeleteStudentHandler(w http.ResponseWriter, r *http.Request) {

	// Extract id from URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Convert id to integer
	idINT, _ := strconv.Atoi(id)

	// Execute delete query
	res, err := a.MySQL.db.Exec("DELETE FROM students WHERE id=?", idINT)
	if err != nil {
		http.Error(w, "unable to delete", http.StatusInternalServerError)
		return
	}

	// Check if student exsists
	rows, err := res.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if rows == 0 {
		http.Error(w, "student not found", http.StatusNotFound)
		return
	}

	// Remove cache entry
	go a.Redis.Client.Del(a.Ctx, id)

	// Log delete response
	go LogActivity("DELETE_STUDENTS", "system")
	go AuditLog("DELETE", "STUDENT", idINT, "system")

	// Send success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("student deleted!"))
}
