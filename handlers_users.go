package main

import (
    "log"
    "net/http"
    "strings"

    "golang.org/x/crypto/bcrypt"
)

type UserRow struct {
    ID       int
    Username string
    Role     string
}

type UsersPage struct {
    User    Session
    Users   []UserRow
    Message string
    Success bool
}

func usersHandler(w http.ResponseWriter, r *http.Request, s Session) {
    page := UsersPage{User: s}

    if r.Method == http.MethodPost {
        username := strings.TrimSpace(r.FormValue("username"))
        password := r.FormValue("password")
        role := r.FormValue("role")

        switch {
        case username == "":
            page.Message = "Username is required."

        case len(password) < 6:
            page.Message = "Password must be at least 6 characters."

        case role != "admin" && role != "staff":
            page.Message = "Invalid role."

        default:
            hash, err := bcrypt.GenerateFromPassword(
                []byte(password),
                bcrypt.DefaultCost,
            )

            if err != nil {
                page.Message = "Password is too long."
                break
            }

            _, err = db.Exec(
                "INSERT INTO users (username, password, role) VALUES (?, ?, ?)",
                username,
                string(hash),
                role,
            )

            switch {
            case err == nil:
                page.Message = "User added successfully."
                page.Success = true

            case mysqlErrNo(err) == 1062:
                page.Message = "Username is already taken."

            default:
                log.Println(err)
                page.Message = "Could not add the user."
            }
        }
    }

    rows, err := db.Query(
        "SELECT id, username, role FROM users ORDER BY id",
    )

    if err != nil {
        log.Println(err)
        http.Error(w, "Server error", http.StatusInternalServerError)
        return
    }
    defer rows.Close()

    for rows.Next() {
        var u UserRow

        if err := rows.Scan(
            &u.ID,
            &u.Username,
            &u.Role,
        ); err != nil {
            log.Println(err)
            http.Error(w, "Server error", http.StatusInternalServerError)
            return
        }

        page.Users = append(page.Users, u)
    }

    tpl.ExecuteTemplate(w, "users.html", page)
}
