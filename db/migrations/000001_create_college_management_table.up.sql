USE college_management_system;

CREATE TABLE IF NOT EXISTS students(
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    age INT NOT NULL,
    email VARCHAR(100) NOT NULL,
    dept VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS lecturers (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    age INT NOT NULL,
    email VARCHAR(100) NOT NULL,
    designation VARCHAR(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS libraries (
    library_id INT AUTO_INCREMENT PRIMARY KEY,
    title  VARCHAR(100) NOT NULL,
    available_copies INT NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS books (
    book_id INT PRIMARY KEY,
    book_name VARCHAR(100) NOT NULL,
    library_id INT NOT NULL,
    FOREIGN KEY (library_id) REFERENCES libraries(library_id) ON DELETE CASCADE

);

CREATE TABLE IF  NOT EXISTS authors (
    author_id INT PRIMARY KEY ,
    author_name VARCHAR(100) NOT NULL ,
    library_id INT NOT NULL,
   FOREIGN KEY (library_id) REFERENCES libraries(library_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS borrow_records (
     borrow_id   INT AUTO_INCREMENT PRIMARY KEY,
    user_id     INT NOT NULL,
    usertype    VARCHAR(100) NOT NULL,
    book_id     INT NOT NULL,
    borrow_date DATE NOT NULL,
    return_date DATE,
    FOREIGN KEY (book_id) REFERENCES books(book_id)
        ON DELETE CASCADE

);