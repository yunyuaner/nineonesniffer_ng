package nineonesniffer

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbFileName = "nineone.db"
)

type nineonePersister struct {
	db      *sql.DB
	sniffer *NineOneSniffer
}

func (persister *nineonePersister) init() {
	var isDatabaseFirstCreated bool
	confmgr := persister.sniffer.confmgr

	database := filepath.Join(confmgr.config.sqliteDir, dbFileName)
	if _, err := os.Stat(database); os.IsNotExist(err) {
		file, err := os.Create(database)
		if err != nil {
			log.Fatal(err.Error())
		}

		file.Close()
		isDatabaseFirstCreated = true
	}

	persister.db, _ = sql.Open("sqlite3", database)
	if isDatabaseFirstCreated {
		persister.videoListTableCreate()
	}

}

func (persister *nineonePersister) finalize() {
	persister.db.Close()
}

func (persister *nineonePersister) videoListTableCreate() {
	createVideoListTableSQL := `CREATE TABLE IF NOT EXISTS "VideoInfoTable" (
		"id"	INTEGER,
		"title"	TEXT,
		"author"	TEXT,
		"viewkey"	TEXT NOT NULL UNIQUE,
		"thumbnail"	TEXT,
		"duration"	INTEGER,
		"uploadTime"	INTEGER,
		"source"	TEXT,
		"content"	TEXT,
		PRIMARY KEY("id" AUTOINCREMENT)
	);`
	statement, err := persister.db.Prepare(createVideoListTableSQL)
	if err != nil {
		log.Fatal(err.Error())
	}
	statement.Exec()
}

func (persister *nineonePersister) videoListTableInsert(viewkey string, url string,
	title string, thumbnail string, thumbnailID int, author string, duration string) error {
	/* for sql statement, check https://stackoverflow.com/questions/40157049/sqlite-case-statement-insert-if-not-exists */
	//sql := `insert into VideoListTable(viewkey, url)
	//			select viewkey, url
	//			from (select ? as vk, ? as url) t
	//			where not exists (select 1 from VideoListTable where VideoListTable.viewkey = t.vk)`
	tx, _ := persister.db.Begin()
	stmt, _ := tx.Prepare("insert into VideoListTable (title, viewkey, url, thumbnail, thumbnail_id, date, author, duration) values (?,?,?,?,?,?,?,?)")
	_, err := stmt.Exec(title, viewkey, url, thumbnail, thumbnailID, time.Now().Format("2006-01-02 15:04:05"), author, duration)
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (persister *nineonePersister) updateVideoDuration(item *VideoItem) error {
	tx, _ := persister.db.Begin()
	stmt, _ := tx.Prepare("update VideoListTable set author=?, duration=? where thumbnail_id=?")
	_, err := stmt.Exec(item.Author, item.Duration.String(), strconv.Itoa(item.ThumbnailId))
	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (persister *nineonePersister) updateVideoUploadDate(uploaded_date time.Time, thumbnail_id int) error {
	tx, _ := persister.db.Begin()
	stmt, _ := tx.Prepare("update VideoListTable set upload_date=?  where thumbnail_id=?")
	_, err := stmt.Exec(uploaded_date.Format("2006-01-02 15:04:05"), thumbnail_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (persister *nineonePersister) updateVideoDescriptorURL(url string, thumbnail_id int) error {
	tx, _ := persister.db.Begin()
	stmt, _ := tx.Prepare("update VideoListTable set descriptor_url=?  where thumbnail_id=?")
	_, err := stmt.Exec(url, thumbnail_id)
	if err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

func (persister *nineonePersister) queryVideoItemsWithoutUploadDate() (partialist []partial, err error) {
	rows, err := persister.db.Query("select thumbnail_id, thumbnail from VideoListTable where upload_date is null")
	if err != nil {
		return nil, err
	}

	for rows.Next() {
		var thumbnailID int
		var thumbnail string

		err = rows.Scan(&thumbnailID, &thumbnail)
		if err != nil {
			log.Print(err)
			continue
		}
		partialist = append(partialist, partial{thumbnail_id: thumbnailID, thumbnail: thumbnail})
	}

	return partialist, nil
}
