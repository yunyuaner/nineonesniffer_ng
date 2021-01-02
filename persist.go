package nineonesniffer

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	_ "github.com/mattn/go-sqlite3"
)

type User struct {
	id         int
	username   string
	surname    string
	age        int
	university string
	//I created a struct with a struct to select the rows in the table and add data.
}

func addUser(db *sql.DB, username string, surname string, age int, university string) {
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("insert into testTable (username, surname, age, university) values (?,?,?,?)")
	_, err := stmt.Exec(username, surname, age, university)
	if err != nil {
		log.Fatal(err)
	}
	tx.Commit()
}

func getUsers(db *sql.DB, id2 int) User {
	rows, err := db.Query("select * from testTable")
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var tempUser User
		err = rows.Scan(&tempUser.id, &tempUser.username, &tempUser.surname,
			&tempUser.age, &tempUser.university)
		if err != nil {
			log.Fatal(err)
		}

		if tempUser.id == id2 {
			return tempUser
		}
	}
	return User{}
}

func updateUser(db *sql.DB, id2 int, username string, surname string, age int, university string) {
	sage := strconv.Itoa(age)
	sid := strconv.Itoa(id2)
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("update testTable set username=?,surname=?,age=?,university=? where id=?")
	_, err := stmt.Exec(username, surname, sage, university, sid)
	if err != nil {
		log.Fatal(err)
	}

	tx.Commit()
}

func deleteUser(db *sql.DB, id2 int) {
	sid := strconv.Itoa(id2)
	tx, _ := db.Begin()
	stmt, _ := tx.Prepare("delete from testTable where id=?")
	_, err := stmt.Exec(sid)
	if err != nil {
		log.Fatal(err)
	}

	tx.Commit()
}

func Persist() {
	db, _ := sql.Open("sqlite3", "nineone.db")
	db.Exec(`create table testTable if not exist (
				id integer primary key autoincrement,
				username text, 
				surname text,
				age Integer,
				university text)`)
	addUser(db, "fevzi omur ", "tekin", 24, "Sakarya University") // added data to database
	addUser(db, "Jia Guo", "", 38, "Huazhong University of Science and Technoloty")
	//updateUser(db, 2, "Ken", "Thompson", 75, "California university") //update data to database
	//deleteUser(db, 1)                                                 // delete data to database
	//fmt.Println(getUsers(db, 2)) // printing the user

	rows, err := db.Query("select * from testTable")
	if err != nil {
		log.Fatal(err)
	}

	for rows.Next() {
		var tempUser User
		err = rows.Scan(&tempUser.id, &tempUser.username, &tempUser.surname,
			&tempUser.age, &tempUser.university)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%d %s %s %d %s\n", tempUser.id, tempUser.username, tempUser.surname,
			tempUser.age, tempUser.university)
	}
}
