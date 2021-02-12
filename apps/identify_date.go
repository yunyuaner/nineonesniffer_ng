package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var fileMap map[int]time.Time

func main() {
	f, err := os.Open("data/images/new")
	if err != nil {
		log.Fatal(err)
	}

	info, err := f.Readdir(0)
	if err != nil {
		log.Fatal(err)
	}

	fileMap = make(map[int]time.Time)

	for _, fileInfo := range info {
		//fmt.Printf("file - %s, date - %s\n", fileInfo.Name(), fileInfo.ModTime().Format("2006-01-02 15:04:05"))

		videoID, _ := strconv.Atoi(fileInfo.Name()[:len(fileInfo.Name())-4])

		fileMap[videoID] = fileInfo.ModTime()
	}

	db, _ := sql.Open("sqlite3", "nineone.db")

	fileMapSize := len(fileMap)
	var counter int

	for k, v := range fileMap {
		//fmt.Printf("videoID - %d, date - %s\n", k, v.Format("2006-01-02 15:04:05"))
		tx, _ := db.Begin()
		stmt, _ := tx.Prepare("update VideoListTable set upload_date=?  where thumbnail_id=?")
		_, err := stmt.Exec(v.Format("2006-01-02 15:04:05"), strconv.Itoa(k))
		if err != nil {
			tx.Rollback()
			continue
		}

		counter++
		fmt.Printf("\r[%6d of %d] updated", counter, fileMapSize)

		tx.Commit()
	}

	fmt.Printf("\nDone\n")
}
