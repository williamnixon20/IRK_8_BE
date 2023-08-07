-- Your initial migration SQL commands go here
CREATE DATABASE IF NOT EXISTS baru;
USE baru;

DROP TABLE IF EXISTS courses;
DROP TABLE IF EXISTS faculty;
DROP TABLE IF EXISTS major;

CREATE TABLE courses (
    id VARCHAR(500) NOT NULL,
    CourseName VARCHAR(255) NOT NULL,
    Grade INT,
    Workload INT,
    Faculty VARCHAR(255),
    MinimumSemester INT,
    PRIMARY KEY (id)
);

CREATE TABLE faculty (
    Faculty VARCHAR(255),
    PRIMARY KEY (Faculty)
);

CREATE TABLE major (
    Faculty VARCHAR(255),
    Major VARCHAR(255),
    PRIMARY KEY (Faculty, Major)
);
-- Add any other necessary initial migration steps
