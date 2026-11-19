
package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// ---------- Data structures ----------

type StudentRow struct {
	ID        int
	NIM       string
	Name      string
	MajorName string
}

type Major struct {
	ID   int
	Name string
}

type StudentForm struct {
	ID      int
	NIM     string
	Name    string
	MajorID int
}

type IndexPage struct {
	User     Session
	Students []StudentRow
}

type FormPage struct {
	User    Session
	Student StudentForm
	Majors  []Major
	Error   string
}

// ---------- Dashboard with JOIN ----------

func indexHandler(w http.ResponseWriter, r *http.Request, s Session) {
	rows, err := db.Query(`
		SELECT s.id, s.nim, s.name, m.name AS major_name
		FROM students s
		JOIN majors m ON s.major_id = m.id
		ORDER BY s.id DESC`)

	if err != nil {
		log.Println(err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	page := IndexPage{User: s}

	for rows.Next() {
		var st StudentRow

		if err := rows.Scan(
			&st.ID,
			&st.NIM,
			&st.Name,
			&st.MajorName,
		); err != nil {
			log.Println(err)
			http.Error(w, "Server error", http.StatusInternalServerError)
			return
		}

		page.Students = append(page.Students, st)
	}

	tpl.ExecuteTemplate(w, "index.html", page)
}

// ---------- Helpers shared by create and edit ----------

func listMajors() ([]Major, error) {
	rows, err := db.Query(
		"SELECT id, name FROM majors ORDER BY name",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Major

	for rows.Next() {
		var m Major

		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			return nil, err
		}

		list = append(list, m)
	}

	return list, rows.Err()
}

func isDigits(s string) bool {
	if s == "" {
		return false
	}

	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}

	return true
}

func studentFromForm(r *http.Request) StudentForm {
	majorID, _ := strconv.Atoi(r.FormValue("major_id"))

	return StudentForm{
		NIM:     strings.TrimSpace(r.FormValue("nim")),
		Name:    strings.TrimSpace(r.FormValue("name")),
		MajorID: majorID,
	}
}

func validateStudent(f StudentForm) string {
	switch {
	case !isDigits(f.NIM):
		return "NIM must be a valid number!"
	case f.Name == "":
		return "Name is required."
	case f.MajorID == 0:
		return "Please choose a major."
	}

	return ""
}

// ---------- Database error handling ----------

func saveError(err error) string {
	if mysqlErrNo(err) == 1452 {
		return "That major does not exist."
	}

	log.Println(err)
	return "Could not save the student."
}

// ---------- Create ----------

func createHandler(w http.ResponseWriter, r *http.Request, s Session) {
	majors, err := listMajors()

	if err != nil {
		log.Println(err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	page := FormPage{
		User:   s,
		Majors: majors,
	}

	if r.Method == http.MethodPost {
		page.Student = studentFromForm(r)
		page.Error = validateStudent(page.Student)

		if page.Error == "" {
			_, err := db.Exec(
				"INSERT INTO students (nim, name, major_id) VALUES (?, ?, ?)",
				page.Student.NIM,
				page.Student.Name,
				page.Student.MajorID,
			)

			if err == nil {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}

			page.Error = saveError(err)
		}
	}

	tpl.ExecuteTemplate(w, "create.html", page)
}

// ---------- Edit ----------

func editHandler(w http.ResponseWriter, r *http.Request, s Session) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	var current StudentForm

	err := db.QueryRow(
		"SELECT id, nim, name, major_id FROM students WHERE id = ?",
		id,
	).Scan(
		&current.ID,
		&current.NIM,
		&current.Name,
		&current.MajorID,
	)

	if errors.Is(err, sql.ErrNoRows) {
		http.Error(w, "Student not found.", http.StatusNotFound)
		return
	} else if err != nil {
		log.Println(err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	majors, err := listMajors()

	if err != nil {
		log.Println(err)
		http.Error(w, "Server error", http.StatusInternalServerError)
		return
	}

	page := FormPage{
		User:    s,
		Student: current,
		Majors:  majors,
	}

	if r.Method == http.MethodPost {
		page.Student = studentFromForm(r)
		page.Student.ID = id
		page.Error = validateStudent(page.Student)

		if page.Error == "" {
			_, err := db.Exec(
				"UPDATE students SET nim = ?, name = ?, major_id = ? WHERE id = ?",
				page.Student.NIM,
				page.Student.Name,
				page.Student.MajorID,
				id,
			)

			if err == nil {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}

			page.Error = saveError(err)
		}
	}

	tpl.ExecuteTemplate(w, "edit.html", page)
}

// ---------- Delete ----------

func deleteHandler(w http.ResponseWriter, r *http.Request, s Session) {
	id, _ := strconv.Atoi(r.URL.Query().Get("id"))

	if _, err := db.Exec(
		"DELETE FROM students WHERE id = ?",
		id,
	); err != nil {
		log.Println(err)
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
