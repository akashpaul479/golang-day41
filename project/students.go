package project

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

type Student struct {
	Id    int    `json:"id"`
	Name  string `json:"name"`
	Age   int    `json:"age"`
	Email string `json:"email"`
	Dept  string `json:"dept"`
}

// Validation
func ValidateStudent(student Student) error {
	if strings.TrimSpace(student.Name) == "" {
		return fmt.Errorf("Empty name or invalid name")
	}
	if strings.TrimSpace(student.Email) == "" {
		return fmt.Errorf("Empty email or invalid email")
	}
	if strings.TrimSpace(student.Dept) == "" {
		return fmt.Errorf("Empty dept or invalid dept")
	}
	if !strings.HasSuffix(student.Email, "@gmail.com") {
		return fmt.Errorf("email is invalid and does not contains @gmail.com")
	}
	prefix := strings.TrimSuffix(student.Email, "@gmail.com")
	if prefix == "" {
		return fmt.Errorf("email must contains a prefix before @gmail.com")
	}
	if student.Age <= 0 {
		return fmt.Errorf("Age is less than 0 ")
	}
	if student.Age >= 100 {
		return fmt.Errorf("Age is grater than 0")
	}
	return nil
}

// Create student
func (a *HybridHandler) CreateStudentHandler(w http.ResponseWriter, r *http.Request) {
	var students Student
	if err := json.NewDecoder(r.Body).Decode(&students); err != nil {
		http.Error(w, "Failed to decode response", http.StatusInternalServerError)
		return
	}
	if err := ValidateStudent(students); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
		return
	}
	res, err := a.MySQL.db.Exec("INSERT INTO students (name , age , email , dept) VALUES (? , ? , ? , ?)", students.Name, students.Age, students.Email, students.Dept)
	if err != nil {
		http.Error(w, "Unable to insert", http.StatusInternalServerError)
		return
	}
	id, err := res.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	students.Id = int(id)

	go LogActivity("CREATE_EMPLOYEE", "system")
	go AuditLog("CREATE", "EMPLOYEE", students.Id, "system")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(students)
}

func (a *HybridHandler) GetstudentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	go LogActivity("GET_EMPLOYEE", "system")

	value, err := a.Redis.Client.Get(a.Ctx, id).Result()
	if err == nil {
		log.Println("Cache Hit...")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(value))
		return
	}
	fmt.Println("cache miss querying MySQL...")
	row := a.MySQL.db.QueryRow("SELECT name , age , email , dept FROM students WHERE id=?", id)

	var students Student
	if err := row.Scan(&students.Name, &students.Age, &students.Email, &students.Dept); err != nil {
		http.Error(w, "student not found ", http.StatusNotFound)
		return
	}
	jsonData, err := json.Marshal(students)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go a.Redis.Client.Set(a.Ctx, id, jsonData, 10*time.Second)

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

func (a *HybridHandler) UpdateStudentHandler(w http.ResponseWriter, r *http.Request) {
	var students Student
	if err := json.NewDecoder(r.Body).Decode(&students); err != nil {
		http.Error(w, "Failed to decode response", http.StatusInternalServerError)
		return
	}
	if err := ValidateStudent(students); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
		return
	}
	res, err := a.MySQL.db.Exec("UPDATE students SET name=? , age=? , email=? , dept=? WHERE id=?", students.Name, students.Age, students.Email, students.Dept, students.Id)
	if err != nil {
		http.Error(w, "unable to update", http.StatusInternalServerError)
		return
	}
	rows, err := res.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rows == 0 {
		http.Error(w, "user not found ", http.StatusNotFound)
		return
	}
	jsonData, err := json.Marshal(students)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	go a.Redis.Client.Set(a.Ctx, fmt.Sprint(students.Id), jsonData, 10*time.Second)

	go LogActivity("UPDATE_STUDENT", "system")
	go AuditLog("UPDATE", "STUDENT", students.Id, "system")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(jsonData)
}

func (a *HybridHandler) DeleteStudentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	idINT, _ := strconv.Atoi(id)

	res, err := a.MySQL.db.Exec("DELETE FROM students WHERE id=?", idINT)
	if err != nil {
		http.Error(w, "unable to delete", http.StatusInternalServerError)
		return
	}
	rows, err := res.RowsAffected()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if rows == 0 {
		http.Error(w, "student not found", http.StatusNotFound)
		return
	}
	go a.Redis.Client.Del(a.Ctx, id)

	go LogActivity("DELETE_STUDENTS", "system")
	go AuditLog("DELETE", "STUDENT", idINT, "system")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("student deleted!"))
}
