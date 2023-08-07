package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
)

func connectDB() (*sql.DB, error) {
	dbUser := "root"
	dbPassword := "baru"
	dbHost := "mysql"
	dbPort := "3306"
	dbName := "baru"

	dataSourceName := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return db, nil
}

func insertCourse(course Course) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
    INSERT INTO courses (id, CourseName, Grade, Workload, Faculty, MinimumSemester)
    VALUES (?, ?, ?, ?, ?, ?)
    ON DUPLICATE KEY UPDATE
    CourseName = VALUES(CourseName),
    Grade = VALUES(Grade),
    Workload = VALUES(Workload),
    Faculty = VALUES(Faculty),
    MinimumSemester = VALUES(MinimumSemester)
    `, course.Id, course.CourseName, course.Grade, course.Workload, course.Faculty, course.MinimumSemester)

	if err != nil {
		fmt.Println("Failed to insert course:", err)
		return err
	}

	return nil
}

func deleteCourse(courseId string) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM courses WHERE id = ?", courseId)
	if err != nil {
		return err
	}

	return nil
}

func addMajor(faculty string, major string) error {
	db, err := connectDB()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO major values (?,?)", faculty, major)
	if err != nil {
		return err
	}

	return nil
}


type Course struct {
	Id              string   `json:"Id"`
	CourseName      string `json:"Course Name"`
	Grade           float32    `json:"Grade"`
	Workload        int    `json:"Workload"`
	Faculty         string `json:"Faculty"`
	MinimumSemester int    `json:"Minimum Semester"`
}

type RequestBody struct {
	Courses         []Course `json:"courses"`
	CurrentSemester int      `json:"current_semester"`
	Faculty         string   `json:"faculty"`
	MaxWorkload     int      `json:"max_workload"`
	MinWorkload     int      `json:"min_workload"`
}

type ResponseBody struct {
	MaxGrade           float32      `json:"max_grade"`
	SelectedCourses    []string `json:"selected_courses"`
	TotalWorkload      int      `json:"total_workload"`
}

func findFacultyByMajor(majorName string) (string, error) {
    db, err := connectDB()
    if err != nil {
        return "", err
    }
    defer db.Close()

    rows, err := db.Query("SELECT faculty FROM major WHERE major = ?", majorName)
    if err != nil {
        return "", err
    }
    defer rows.Close()

    var faculty string
    for rows.Next() {
        err := rows.Scan(&faculty)
        if err != nil {
            return "", err
        }
        // Assuming you expect only one row for a given major
        break
    }

    return faculty, nil
}

func maximizeGrade(courses []Course, faculty string, currentSemester int, maxWorkload int, minWorkload int) (float32, []string, int) {
	var validCourses []Course
	var faculty_real, _ = findFacultyByMajor(faculty);
	for _, course := range courses {
		var faculty_real2, _ = findFacultyByMajor(course.Faculty);
		if (course.Faculty == faculty || (faculty == faculty_real2 && faculty_real2 != "") || (course.Faculty == faculty_real && faculty_real != "") || (faculty_real2 == faculty_real && faculty_real2 != "")) && course.MinimumSemester <= currentSemester {
			validCourses = append(validCourses, course)
		}
	}
	fmt.Println(validCourses)

	n := len(validCourses)
	dp := make([][]float32, n+1)
	selectedCourses := make([][][]string, n+1)
	for i := 0; i <= n; i++ {
		dp[i] = make([]float32, maxWorkload+1)
		selectedCourses[i] = make([][]string, maxWorkload+1)
	}

	for w := 1; w <= maxWorkload; w++ {
		dp[0][w] = -9999999
	}

	for i := 1; i <= n; i++ {
		course := validCourses[i-1]
		courseId := course.Id
		grade := course.Grade
		workload := course.Workload

		for w := 0; w <= maxWorkload; w++ {
			if (w-workload >= 0) {
				cum_1 := dp[i-1][w] 
				cum_2 := dp[i-1][w-workload]+(grade*float32(workload))
				if cum_1 > cum_2 {
					dp[i][w] = dp[i-1][w]
					selectedCourses[i][w] = selectedCourses[i-1][w][:]
				} else {
					dp[i][w] = dp[i-1][w-workload]+(grade*float32(workload))
					selectedCourses[i][w] = append(selectedCourses[i-1][w-course.Workload][:], courseId)
				}
			} else {
				dp[i][w] = dp[i-1][w]
				selectedCourses[i][w] = selectedCourses[i-1][w][:]
			}
		}
	}

	maxGrade := float32(-1.0)
	selectedCourseArrangement := []string{}
	totalWorkload := 0
	for w:= minWorkload; w <= maxWorkload; w++ {
		fmt.Println("w", w, "dp[n][maxWorkload]:" , dp[n][w], selectedCourses[n][w])
		if dp[n][w]/float32(w) > maxGrade || (math.Abs(float64((dp[n][w]/float32(w) - maxGrade))) < 0.001 && w > totalWorkload){
			maxGrade = dp[n][w]/float32(w)
			totalWorkload = w
			selectedCourseArrangement = selectedCourses[n][w]
		}
	}
	selectedCourseNames := []string{}
	for _, course := range courses {
		for _, selectedCourse := range selectedCourseArrangement {
			if course.Id == selectedCourse {
				selectedCourseNames = append(selectedCourseNames, course.CourseName)
			}
		}
	}

	return maxGrade, selectedCourseNames, totalWorkload
}


func calculateHandler(w http.ResponseWriter, r *http.Request) {
	var reqBody RequestBody
	err := json.NewDecoder(r.Body).Decode(&reqBody)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	maxGrade, selectedCourses, totalWorkload := maximizeGrade(reqBody.Courses, reqBody.Faculty, reqBody.CurrentSemester, reqBody.MaxWorkload, reqBody.MinWorkload)

	respBody := ResponseBody{
		MaxGrade:        maxGrade,
		SelectedCourses: selectedCourses,
		TotalWorkload:   totalWorkload,
	}

	respJSON, err := json.Marshal(respBody)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respJSON)
}

func createCourseHandler(w http.ResponseWriter, r *http.Request) {
    var course Course
    err := json.NewDecoder(r.Body).Decode(&course)
    if err != nil {
        http.Error(w, "Invalid JSON", http.StatusBadRequest)
        return
    }

    err = insertCourse(course)
    if err != nil {
        http.Error(w, "Failed to insert course", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    w.Write([]byte("Course created successfully"))
}

type Major struct {
	Faculty string   `json:"Faculty"`
	Major   []string `json:"Major"`
}

func createMajor(w http.ResponseWriter, r *http.Request) {
	var majorData []Major

	// Decode the JSON request body into a slice of Major structs
	err := json.NewDecoder(r.Body).Decode(&majorData)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	for _, majorItem := range majorData {
		for _, majorName := range majorItem.Major {
			err = addMajor(majorItem.Faculty, majorName)
			if err != nil {
				http.Error(w, "Failed to insert major", http.StatusInternalServerError)
				return
			}
		}
	}

	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Majors created successfully"))
}



func getMajors(w http.ResponseWriter, r *http.Request) {
	db, err := connectDB()
    if err != nil {
        http.Error(w, "Database connection error", http.StatusInternalServerError)
        return
    }
    defer db.Close()

	rows, err := db.Query("SELECT * FROM major")
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to retrieve majors", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var majors []Major
	for rows.Next() {
		var faculty string
		var majorName string

		err := rows.Scan(&faculty, &majorName)
		if err != nil {
			http.Error(w, "Failed to scan row", http.StatusInternalServerError)
			return
		}

		existingMajorIndex := findMajorIndex(majors, faculty)
		if existingMajorIndex != -1 {
			majors[existingMajorIndex].Major = append(majors[existingMajorIndex].Major, majorName)
		} else {
			majors = append(majors, Major{Faculty: faculty, Major: []string{majorName}})
		}
	}

	jsonResponse, err := json.Marshal(majors)
	if err != nil {
		http.Error(w, "Failed to marshal JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(jsonResponse)
}

func findMajorIndex(majors []Major, faculty string) int {
	for i, major := range majors {
		if major.Faculty == faculty {
			return i
		}
	}
	return -1
}

func getCoursesHandler(w http.ResponseWriter, r *http.Request) {
    db, err := connectDB()
    if err != nil {
        http.Error(w, "Database connection error", http.StatusInternalServerError)
        return
    }
    defer db.Close()

    rows, err := db.Query("SELECT * FROM courses")
    if err != nil {
        http.Error(w, "Database query error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    var courses []Course
    for rows.Next() {
        var course Course
        err := rows.Scan(&course.Id, &course.CourseName, &course.Grade, &course.Workload, &course.Faculty, &course.MinimumSemester)
        if err != nil {
            http.Error(w, "Database row scan error", http.StatusInternalServerError)
            return
        }
        courses = append(courses, course)
    }

    response, err := json.Marshal(courses)
    if err != nil {
        http.Error(w, "JSON encoding error", http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusOK)
    w.Write(response)
}

func deleteCourseHandler(w http.ResponseWriter, r *http.Request) {
    courseId := r.URL.Query().Get("Id")
    if courseId == "" {
        http.Error(w, "Missing course_id parameter", http.StatusBadRequest)
        return
    }

    err := deleteCourse(courseId)
    if err != nil {
        http.Error(w, "Failed to delete course", http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Course deleted successfully"))
}


func main() {
	http.HandleFunc("/api/calculate", calculateHandler)
	http.HandleFunc("/api/create", createCourseHandler) // POST endpoint to create a course
    http.HandleFunc("/api/get", getCoursesHandler)     // GET endpoint to retrieve courses
    http.HandleFunc("/api/delete", deleteCourseHandler) // DELETE endpoint to delete a course
	http.HandleFunc("/api/major", createMajor) // POST endpoint to create a major
	http.HandleFunc("/api/majors", getMajors) // GET endpoint to retrieve majors

	fmt.Println("Server is running on port 8080...")
	http.ListenAndServe(":8080", nil)
}
