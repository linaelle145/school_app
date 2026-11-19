package main

import (
    "html/template"
    "log"
    "net/http"
)

var tpl = template.Must(
    template.New("").Funcs(template.FuncMap{
        "inc": func(i int) int {
            return i + 1
        },
    }).ParseGlob("templates/*.html"),
)

func main() {
    connectDB()
    defer db.Close()

    mux := http.NewServeMux()

    // Public pages
    mux.HandleFunc("/login", loginHandler)
    mux.HandleFunc("/logout", logoutHandler)

    // Any logged-in user
    mux.HandleFunc("GET /{$}", requireLogin(indexHandler))
    mux.HandleFunc("/create", requireLogin(createHandler))
    mux.HandleFunc("/edit", requireLogin(editHandler))

    // Admins only
    mux.HandleFunc("/delete", requireAdmin(deleteHandler))
    mux.HandleFunc("/users", requireAdmin(usersHandler))

    log.Println("Server running at http://localhost:8080")
    log.Fatal(http.ListenAndServe(":8080", mux))
}
