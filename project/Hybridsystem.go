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
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MySQLInstance  wraps a MySQL database connections
type MySQLInstance struct {
	db *sql.DB
}

// MongoDBInstance wraps a MongoDB client , Database , and lecturer collection.
type MongoDBInstance struct {
	Mongo    *mongo.Client
	DB       *mongo.Database
	Lecturer *mongo.Collection
}

// Redis Instance wraps a redis client.
type RedisInstance struct {
	Client *redis.Client
}

// HybridHandler aggregates MySQL , MongoDB , Redis instances along with a shared context.
type HybridHandler struct {
	MySQL   *MySQLInstance
	MongoDB *MongoDBInstance
	Redis   *RedisInstance
	Ctx     context.Context
}

// connectMySQL initilizes a MySQL connection using DSN from environment variables.
func ConnectMySQL() (*MySQLInstance, error) {
	db, err := sql.Open("mysql", os.Getenv("MYSQL_DSN"))
	if err != nil {
		log.Panic(err)
	}
	return &MySQLInstance{db: db}, nil
}

// ConnectMongoDB initilizes a MongoDB client and returns a MongoDB instance.
func ConnectMongoDB() (*MongoDBInstance, error) {
	ClientOptions := options.Client().ApplyURI(os.Getenv("MONGO_URI"))
	client, err := mongo.NewClient(ClientOptions)
	if err != nil {
		log.Panic(err)
	}

	// Context with timeout to avoid hanging connectiions
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Connect to mongoDB
	err = client.Connect(ctx)
	if err != nil {
		panic(err)
	}

	// select database and connections
	db := client.Database(os.Getenv("MONGO_DB"))
	return &MongoDBInstance{
		Mongo:    client,
		DB:       db,
		Lecturer: db.Collection("lecturers"),
	}, nil
}

// ConnectRedis initilizes a Redis Client environment variables
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

// main function
func CollegeManagementSystem() {

	// Load environment variables from .env file
	godotenv.Load()

	// Ensures JWT Secret  is set
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET is not set or empty")
	}
	SecretKey = []byte(secret)

	// Initilizes Redis
	redisinstance, err := ConnectRedis()
	if err != nil {
		panic(err)
	}

	// Initilizes MySQL
	mysqlinstance, err := ConnectMySQL()
	if err != nil {
		panic(err)
	}

	// Initilizes MongoDB
	mongodbinstance, err := ConnectMongoDB()
	if err != nil {
		panic(err)
	}

	// Create handler with all DB instanmces
	handler := &HybridHandler{Redis: redisinstance, MySQL: mysqlinstance, MongoDB: mongodbinstance, Ctx: context.Background()}

	// Setup HTTP routers
	r := mux.NewRouter()

	// Authentication routes
	r.HandleFunc("/login", LoginHandler).Methods("POST")
	r.HandleFunc("/refresh", RefreshHandler).Methods("POST")
	r.HandleFunc("/logout", LogoutHandler).Methods("POST")

	// Student CRUD routes
	r.HandleFunc("/students", handler.CreateStudentHandler).Methods("POST")
	r.HandleFunc("/students", handler.GetStudentHandler).Methods("GET")
	r.HandleFunc("/students/{id}", handler.GetstudentByIDHandler).Methods("GET")
	r.HandleFunc("/students/{id}", handler.UpdateStudentHandler).Methods("PUT")
	r.HandleFunc("/students/{id}", handler.DeleteStudentHandler).Methods("DELETE")

	// Lecturer CRUD routes
	r.HandleFunc("/lecturers", handler.CreateLecturerHandler).Methods("POST")
	r.HandleFunc("/lecturers", handler.GetLecturerHandler).Methods("GET")
	r.HandleFunc("/lecturers/{id}", handler.GetLecturerByIDHandler).Methods("GET")
	r.HandleFunc("/lecturers/{id}", handler.UpdateLecturerHandler).Methods("PUT")
	r.HandleFunc("/lecturers/{id}", handler.DeleteLecturerHandler).Methods("DELETE")

	// Library routes
	r.HandleFunc("/libraries", handler.CreateLibraryHandler).Methods("POST")
	r.HandleFunc("/libraries/{id}", handler.GetLibraryByIDHandler).Methods("GEt")

	// Borrow_records routes
	r.HandleFunc("/borrow", handler.Borrowbooks).Methods("POST")
	r.HandleFunc("/return", handler.ReturnBooksHandler).Methods("POST")

	fmt.Println("Server running on port:8080")
	http.ListenAndServe(":8080", r)
}
