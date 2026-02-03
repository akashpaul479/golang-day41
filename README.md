# golang-day41

#JWT Authentication in Go

This project implements **jwt-based authentication** in Go.
It provides secure login , refresh , logout handlers and middleware for protecting routes with access and refesh tokens stored in cookies.

## Features
- Login with email/password
-Issue **Access Token** (15 minutes TTL) and **refresh token** (7 days TTL)
-refresh endpoint to renew access tokens 
-middleware to project routes
-logout handler to clear cookies
-secure cookie handling ('HTTP-only','samesite')

## Dependencies
-[github.com/golang-jwt/jwt/v5]
Install with: ```bash
go get github.com/golang-jwt/jwt/v5


🎓 College Management System – Golang Backend

A hybrid backend REST API built using Go (Golang) that manages Students, Lecturers, and Library operations using MySQL, MongoDB, Redis, and JWT authentication.

This project demonstrates real-world backend architecture, combining SQL & NoSQL databases with caching, validation, transactions, and clean API design.

🚀 Features

Student CRUD operations (MySQL + Redis)

Lecturer CRUD operations (MongoDB + Redis)

Library management (Books, Authors)

Borrow & Return books functionality

Redis caching for faster reads

JWT Authentication (Login / Refresh / Logout)

Input validation & proper error handling

Background logging & audit trail

RESTful API using Gorilla Mux

🏗️ Architecture Overview
Client
  |
  v
REST API (Gorilla Mux)
  |
HybridHandler
 ├── MySQL     → Students, Libraries, Borrow Records
 ├── MongoDB   → Lecturers
 └── Redis     → Caching Layer

🧰 Tech Stack
Component	Technology
Language	Go (Golang)
Router	Gorilla Mux
SQL DB	MySQL
NoSQL DB	MongoDB
Cache	Redis
Auth	JWT
Env Config	godotenv
📁 Project Structure
project/
├── main.go
├── student.go
├── lecturer.go
├── library.go
├── auth.go
├── middleware.go
├── utils.go
└── .env

🔐 Authentication

JWT-based authentication is implemented.

Auth Endpoints
Method	Endpoint	Description
POST	/login	Login user
POST	/refresh	Refresh token
POST	/logout	Logout user

JWT secret is loaded from environment variables.

🎓 Student Module (MySQL + Redis)
Entity
{
  "id": 1,
  "name": "Akash",
  "age": 22,
  "email": "akash@gmail.com",
  "dept": "CSE"
}

Endpoints
Method	Endpoint	Description
POST	/students	Create student
GET	/students	Get all students
GET	/students/{id}	Get student by ID
PUT	/students/{id}	Update student
DELETE	/students/{id}	Delete student
Caching

Key: student_id

TTL: 10 seconds

Cache-aside strategy

👨‍🏫 Lecturer Module (MongoDB + Redis)
Entity
{
  "id": "ObjectID",
  "name": "John",
  "age": 40,
  "email": "john@gmail.com",
  "designation": "Professor"
}

Endpoints
Method	Endpoint	Description
POST	/lecturers	Create lecturer
GET	/lecturers	Get all lecturers
GET	/lecturers/{id}	Get lecturer by ID
PUT	/lecturers/{id}	Update lecturer
DELETE	/lecturers/{id}	Delete lecturer
Caching

Individual lecturer cached by ObjectID

All lecturers cached with key all_lecturers

TTL: 10 minutes

📚 Library Module (MySQL + Transactions)
Entity
{
  "library_id": 1,
  "title": "Central Library",
  "available_copies": 10,
  "book": [],
  "author": []
}

Endpoints
Method	Endpoint	Description
POST	/libraries	Create library
GET	/libraries/{id}	Get library by ID
Key Concepts Used

SQL transactions (BEGIN → COMMIT → ROLLBACK)

One-to-many relationships

Redis caching for library lookup

📖 Borrow & Return Books
Borrow Book
POST /borrow


Rules:

User must be student or lecturer

Book availability is checked

Available copies are decremented

Return Book
POST /return


Rules:

Return date is updated

Available copies incremented

Prevents invalid returns

✅ Validation

Each module has its own validation logic:

ValidateStudent

ValidateLecturer

ValidateLibrary

ValidateBorrowRecords

Ensures:

Clean API input

Data integrity

Meaningful error responses

⚡ Redis Caching Summary
Data	Key	TTL
Student	student_id	10 sec
Lecturer	lecturer_id	10 min
All Lecturers	all_lecturers	10 min
Library	library_id	10 min
🧾 Logging & Audit Trail

Background goroutines handle logs:

go LogActivity("ACTION", "system")
go AuditLog("ACTION", "ENTITY", id, "system")


Used for:

Debugging

Monitoring

Audit tracking

⚙️ Environment Variables (.env)
MYSQL_DSN=root:password@tcp(localhost:3306)/college
MONGO_URI=mongodb://localhost:27017
MONGO_DB=college
REDIS_ADDR=localhost:6379
JWT_SECRET=supersecretkey

▶️ Running the Application
go mod tidy
go run main.go


Server runs on:

http://localhost:8080

