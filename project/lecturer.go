package project

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Lecturer represents a lecturer entity in MongoDB and exchange via json in API requests/responses
type Lecturer struct {
	Id          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name"`
	Age         int                `json:"age" bson:"age"`
	Email       string             `json:"email" bson:"email"`
	Designation string             `json:"designation" bson:"designation"`
}

// Validatelecturer validates lecturer input before DB operations
func ValidateLecturer(lecturer Lecturer) error {
	// name validation
	if strings.TrimSpace(lecturer.Name) == "" {
		return fmt.Errorf("empty name or invalid name")
	}
	// email validation
	if strings.TrimSpace(lecturer.Email) == "" {
		return fmt.Errorf("empty email or invalid email")
	}
	if !strings.HasSuffix(lecturer.Email, "@gmail.com") {
		return fmt.Errorf("email is invalid and does not contain @gmail.com")
	}
	prefix := strings.TrimSuffix(lecturer.Email, "@gmail.com")
	if prefix == "" {
		return fmt.Errorf("email must contains prefix before @gmail.com")
	}
	// Age validation
	if lecturer.Age <= 0 {
		return fmt.Errorf("age is less than 0")
	}
	if lecturer.Age >= 100 {
		return fmt.Errorf("Age is grater than 100")
	}
	if strings.TrimSpace(lecturer.Designation) == "" {
		return fmt.Errorf("empty designation or invalid designation")
	}
	return nil
}

// CreateLecturerHandler handles lecturers creation
func (a *HybridHandler) CreateLecturerHandler(w http.ResponseWriter, r *http.Request) {
	// Decode incoming json request
	var lecturers Lecturer
	if err := json.NewDecoder(r.Body).Decode(&lecturers); err != nil {
		http.Error(w, "failed to decode respoonse", http.StatusInternalServerError)
		return
	}

	// validate lecturer data
	if err := ValidateLecturer(lecturers); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
		return
	}

	// create context with timeout to avoid hanging DB calls
	ctx, cancel := context.WithTimeout(a.Ctx, 15*time.Second)
	defer cancel()

	// Insert lecturer into MongoDB
	res, err := a.MongoDB.Lecturer.InsertOne(ctx, lecturers)
	if err != nil {
		http.Error(w, "unable to connect mongoDB", http.StatusInternalServerError)
		return
	}

	// Assign generated MongoDB ID
	lecturers.Id = res.InsertedID.(primitive.ObjectID)

	// Marshal lecturer data for redis caching
	jsonData, err := json.Marshal(lecturers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// store lecturer in redis
	go a.Redis.Client.Set(a.Ctx, lecturers.Id.Hex(), jsonData, 10*time.Minute)

	//  Logactivity and audit trail
	go LogActivity("CREATE_LECTURER", "system")
	go AuditLog("CREATE", "LECTURER", lecturers.Id.Hex(), "system")

	// Set success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(lecturers)
}

// GetLecturerHandler Fetches lecturers by ID
func (a *HybridHandler) GetLecturerHandler(w http.ResponseWriter, r *http.Request) {
	// Extract id from URL
	vars := mux.Vars(r)
	id := vars["id"]

	// Get LOG activity
	go LogActivity("GET_LECTURER", "system")

	// Attempt to fetch from redis cache
	value, err := a.Redis.Client.Get(a.Ctx, id).Result()
	if err == nil {
		log.Println("cache Hit...")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(value))
		return
	}
	// Cache miss fetch from MongoDB
	fmt.Println("cache miss querying mongodb...")
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}

	var lecturers Lecturer
	ctx, cancel := context.WithTimeout(a.Ctx, 10*time.Second)
	defer cancel()

	// Fetch lecturer from MongoDB
	err = a.MongoDB.Lecturer.FindOne(ctx, bson.M{"_id": objectID}).Decode(&lecturers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// cache lecturer in redis
	jsonData, err := json.Marshal(lecturers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go a.Redis.Client.Set(a.Ctx, id, jsonData, 10*time.Minute)

	// Return lecturer
	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonData)
}

// UpdateLecturerHandler updates lecturer details
func (a *HybridHandler) UpdateLecturerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var lecturers Lecturer
	if err := json.NewDecoder(r.Body).Decode(&lecturers); err != nil {
		http.Error(w, "Failed to decode response", http.StatusInternalServerError)
		return
	}
	// validate updated data
	if err := ValidateLecturer(lecturers); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"err": err.Error()})
		return
	}
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
	}
	ctx, cancel := context.WithTimeout(a.Ctx, 10*time.Minute)
	defer cancel()

	// update feilds
	update := bson.M{
		"$set": bson.M{
			"name":        lecturers.Name,
			"age":         lecturers.Age,
			"email":       lecturers.Email,
			"designation": lecturers.Designation,
		},
	}

	res, err := a.MongoDB.Lecturer.UpdateOne(ctx, bson.M{"_id": objID}, update)
	if err != nil {
		http.Error(w, "unable to update", http.StatusInternalServerError)
		return
	}

	// Handle record not found
	if res.MatchedCount == 0 {
		http.Error(w, "lecturer not found", http.StatusNotFound)
		return
	}
	lecturers.Id = objID
	// update redis cache
	jsonData, err := json.Marshal(lecturers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	go a.Redis.Client.Set(a.Ctx, id, jsonData, 10*time.Minute)

	// Log update activity
	go LogActivity("UPDATE_LECTURER", "system")
	go AuditLog("UPDATE", "LECTURER", lecturers.Id.Hex(), "system")

	// send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lecturers)
}

// DeleteLecturerHandler deletes lecturer by id
func (a *HybridHandler) DeleteLecturerHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		http.Error(w, "invalid id format", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(a.Ctx, 10*time.Minute)
	defer cancel()

	res, err := a.MongoDB.Lecturer.DeleteOne(ctx, bson.M{"_id": objID})
	if err != nil {
		http.Error(w, "unable to delete", http.StatusInternalServerError)
		return
	}
	// Handle record not found
	if res.DeletedCount == 0 {
		http.Error(w, "Lecturer not found", http.StatusNotFound)
		return
	}
	// remove cache entry
	a.Redis.Client.Del(a.Ctx, id)

	// Log delete activity
	go LogActivity("DELETE_LECTURER", "system")
	go AuditLog("DELETE", "LECTURER", objID.Hex(), "system")

	w.Header().Set("Content-Type", "system")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Lecturer deleted!"))
}
