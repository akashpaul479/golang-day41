package project

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MySQLInstance struct {
	db *sql.DB
}

type MongoDBInstance struct {
	Mongo    *mongo.Client
	DB       *mongo.Database
	Lecturer *mongo.Collection
}

type RedisInstance struct {
	Client *redis.Client
}

type HybridHandler struct {
	MySQL   *MySQLInstance
	MongoDB *MongoDBInstance
	Redis   *RedisInstance
	Ctx     context.Context
}

func ConnectMySQL() (*MySQLInstance, error) {
	db, err := sql.Open("mysql", os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Panic(err)
	}
	return &MySQLInstance{db: db}, nil
}
func ConnectMongoDB() (*MongoDBInstance, error) {
	ClientOptions := options.Client().ApplyURI(os.Getenv("MONGO_URI"))
	client, err := mongo.NewClient(ClientOptions)
	if err != nil {
		log.Panic(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	err = client.Connect(ctx)
	if err != nil {
		panic(err)
	}
	db := client.Database(os.Getenv("MONGO_DB"))
	return &MongoDBInstance{
		Mongo:    client,
		DB:       db,
		Lecturer: db.Collection("lecturers"),
	}, nil
}

func ConnectRedis() (*RedisInstance, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr: os.Getenv("REDIS_ADDR"),
		DB:   0,
	})
	return &RedisInstance{Client: rdb}, nil
}

// Background Utilities
// Logging (gouroutine safe)
func LogActivity(action, actor string) {
	log.Printf("[LOG] %s by %s at %s\n", action, actor, time.Now())
}

// Audit trail
func AuditLog(action, entity string, id any, actor string) {
	log.Printf("[AUDIT] action=%s entity=%s id=%v actor=%s time=%s\n", action, entity, id, actor, time.Now())
}

func CollegeManagementSystem() {
	godotenv.Load()
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set or empty")
	}
	SecretKey = []byte(secret)

	redisinstance, err := ConnectRedis()
	if err != nil {
		panic(err)
	}
	mysqlinstance, err := ConnectMySQL()
	if err != nil {
		panic(err)
	}
	mongodbinstance, err := ConnectMongoDB()
	if err != nil {
		panic(err)
	}

	handler := &HybridHandler{Redis: redisinstance, MySQL: mysqlinstance, MongoDB: mongodbinstance, Ctx: context.Background()}

	r := mux.NewRouter()

	r.HandleFunc("/login", LoginHandler).Methods("POST")
	r.HandleFunc("/refresh", RefreshHandler).Methods("POST")
	r.HandleFunc("/logout", LogoutHandler).Methods("POST")

	r.HandleFunc("/students", handler.CreateStudentHandler).Methods("POST")
	r.HandleFunc("/students/{id}", handler.GetstudentHandler).Methods("GET")
	r.HandleFunc("/students/{id}", handler.UpdateStudentHandler).Methods("PUT")
	r.HandleFunc("/students/{id}", handler.DeleteStudentHandler).Methods("DELETE")

	r.HandleFunc("/lecturers", handler.CreateLecturerHandler).Methods("POST")
	r.HandleFunc("/lecturers/{id}", handler.GetLecturerHandler).Methods("GET")
	r.HandleFunc("/lecturers/{id}", handler.UpdateLecturerHandler).Methods("PUT")
	r.HandleFunc("/lecturers/{id}", handler.DeleteLecturerHandler).Methods("DELETE")

	r.HandleFunc("/libraries", handler.CreateLibraryHandler).Methods("POST")

	fmt.Println("Server running on port:8080")
	http.ListenAndServe(":8080", r)
}
