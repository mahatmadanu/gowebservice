package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"sosmed/lib"
	"strconv"
	"strings"
	"sync"
)

type API struct {
	Message string "json:message"
}
type User struct {
	ID        int    "json:id"
	Username  string "json:username"
	FirstName string "json:firstname"
	LastName  string "json:lastname"
	Email     string "json:email"
}
type Users struct {
	Users []User `json:"Users"`
}
type CreateResponse struct {
	Error string "json:error"
}
type UpdateResponse struct {
	Error     string "json:error"
	ErrorCode int    "json:code"
}
type ErrMsg struct {
	ErrCode    int
	StatusCode int
	Msg        string
}

const (
	ServerName   = "localhost"
	SSLPort      = ":443"
	HttpPort     = ":8002"
	SSLProtocol  = "https://"
	HttpProtocol = "http://"
)

var database *sql.DB

func main() {
	gorillaRoute := mux.NewRouter()
	gorillaRoute.HandleFunc("/api/users", UserRetrieve).Methods("GET")
	gorillaRoute.HandleFunc("/api/users", UserCreate).Methods("POST")
	gorillaRoute.HandleFunc("/api/users/{id:[0-9]+}", UserUpdate).Methods("PUT")
	gorillaRoute.HandleFunc("/api/users", UserInfo).Methods("OPTIONS")
	http.Handle("/", gorillaRoute)
	db, err := sql.Open("mysql", "root:@/social_network")
	if err != nil {
		panic("DB connection failed")
	}
	database = db
	wg := sync.WaitGroup{}
	log.Println("Starting Redirection server, try to access @http:")
	wg.Add(1)
	go func() {
		http.ListenAndServe(HttpPort, http.HandlerFunc(redirectNonSecure))
		wg.Done()
	}()
	go func() {

		http.ListenAndServeTLS(SSLPort, "cert.pem", "key.pem", nil)
		wg.Done()
	}()
	wg.Wait()
	// http.ListenAndServe(":8002", nil)

}
func UserInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Allow", "DELETE, GET, HEAD, OPTIONS, POST, PUT")
}
func UserCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	NewUser := User{}
	Response := CreateResponse{}
	NewUser.Username = r.FormValue("username")
	NewUser.FirstName = r.FormValue("firstname")
	NewUser.LastName = r.FormValue("lastname")
	NewUser.Email = r.FormValue("email")
	fmt.Println(r.FormValue("password"))
	salt, hpass := password.ReturnPassword(r.FormValue("password"))

	output, err := json.Marshal(NewUser)

	fmt.Println(string(output))
	if err != nil {
		fmt.Println("Something Goes Wrong")
	}
	query := "INSERT INTO users set user_nickname ='" + NewUser.Username + "',user_password='" + hpass + "',user_salt='" + salt + "', user_first='" + NewUser.FirstName +
		"', user_last='" + NewUser.LastName + "', user_email='" + NewUser.Email + "'"
	q, err := database.Exec(query)
	if err != nil {
		_, code := dbErrorParse(err.Error())
		errMgs := ErrorMessages(code)
		Response.Error = errMgs.Msg
	}
	fmt.Println(q)
	createOutput, err := json.Marshal(Response)
	fmt.Fprintln(w, string(createOutput))
}
func UserGet(w http.ResponseWriter, r *http.Request) {
	urlParams := mux.Vars(r)
	id := urlParams["id"]
	readuser := User{}
	query := "SELECT * FROM users WHERE user_id='" + id + "'"
	err := database.QueryRow(query).Scan(&readuser.ID, &readuser.Username, &readuser.FirstName, &readuser.LastName, &readuser.Email)
	switch {
	case err == sql.ErrNoRows:
		fmt.Printf("No user with that ID.")
	case err != nil:
		log.Fatal(err)
	default:
		output, _ := json.Marshal(readuser)
		fmt.Fprint(w, string(output))
	}

}
func UserRetrieve(w http.ResponseWriter, r *http.Request) {
	query := "SELECT user_id, user_nickname, user_first, user_last, user_email FROM users"
	rows, _ := database.Query(query)

	response := Users{}
	for rows.Next() {
		user := User{}
		rows.Scan(&user.ID, &user.Username, &user.FirstName, &user.LastName, &user.Email)
		fmt.Println(user)
		response.Users = append(response.Users, user)
	}

	output, err := json.Marshal(response)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Fprint(w, string(output))
}
func UserUpdate(w http.ResponseWriter, r *http.Request) {
	urlParams := mux.Vars(r)
	userid := urlParams["id"]
	email := r.FormValue("email")
	response := UpdateResponse{}
	var userCount int
	qusercount := "SELECT COUNT(user_id) FROM users WHERE user_id ='" + userid + "'"
	err := database.QueryRow(qusercount).Scan(&userCount)
	if userCount == 0 {
		errormsg := ErrorMessages(404)
		log.Println(errormsg.Msg)
		log.Println(errormsg.ErrCode)
		log.Println(errormsg.StatusCode)

	} else if err != nil {
		log.Println(err)
	} else {
		qupdate := "UPDATE users SET user_email='" + email + "' WHERE user_id='" + userid + "'"
		_, err := database.Exec(qupdate)
		if err != nil {
			log.Println(err)
		} else {
			response.Error = "Success"
			response.ErrorCode = 0
			output, err := json.Marshal(response)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Fprint(w, string(output))
		}

	}

}
func dbErrorParse(err string) (string, int64) {
	Parts := strings.Split(err, ":")
	Code := strings.Split(Parts[0], "Error ")
	ErrorMessage := Parts[1]
	ErrorCode, _ := strconv.ParseInt(Code[1], 10, 64)
	return ErrorMessage, ErrorCode
}
func ErrorMessages(err int64) ErrMsg {
	var em = ErrMsg{}
	ErrorMessage := ""
	ErrorCode := 0
	StatusCode := 200
	fmt.Println(err)
	switch err {
	default:
		ErrorMessage = http.StatusText(int(err))
		ErrorCode = 0
		StatusCode = int(err)
	case 1062:
		ErrorMessage = "Duplicate Entry"
		ErrorCode = 10
		StatusCode = 409
	}
	em.ErrCode = ErrorCode
	em.StatusCode = StatusCode
	em.Msg = ErrorMessage
	return em
}
func secureRequest(w http.ResponseWriter, r *http.Request) {
	fmt.Println(w, "You have arrived at port 443 and secure")
}
func redirectNonSecure(w http.ResponseWriter, r *http.Request) {
	log.Println("Non-secure request initiated, redirecting")
	redirect := SSLProtocol + ServerName + r.RequestURI
	http.Redirect(w, r, redirect, http.StatusOK)
}
