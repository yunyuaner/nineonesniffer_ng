package nineonesniffer

import (
	"fmt"
	"log"
	"time"
)

type VideoItem struct {
	Title                string
	Author               string
	Duration             time.Duration
	UploadTime           time.Time
	VideoDetailedPageURL string
	VideoDescripterURL   string
	VideoSource          string
	ViewKey              string
	ThumbnailId          int
	ThumbnailURL         string
	ThumbnailName        string
}

type VideoDataSet map[string]*VideoItem

func (vds *VideoDataSet) save(persister *nineonePersister) {
	var newlyAdded int

	vds.iterate(func(item *VideoItem) bool {
		err := persister.videoListTableInsert(item.ViewKey,
			item.VideoDetailedPageURL,
			item.Title,
			item.ThumbnailURL,
			item.ThumbnailId,
			item.Author,
			item.Duration.String())
		if err == nil {
			fmt.Printf("title - %s, author - %s\n", item.Title, item.Author)
			newlyAdded++
		}
		return true
	})

	fmt.Printf("%d new items added\n", newlyAdded)
}

func (vds *VideoDataSet) sync(persister *nineonePersister) {
	vds.iterate(func(item *VideoItem) bool {
		persister.updateVideoDuration(item)
		return true
	})
}

func (vds *VideoDataSet) loadAll(persister *nineonePersister) {
	rows, err := persister.db.Query("select title, viewkey, url, thumbnail, thumbnail_id from VideoListTable")
	if err != nil {
		log.Fatal(err)
	}

	count := 0

	for rows.Next() {
		var item VideoItem
		err = rows.Scan(&item.Title, &item.ViewKey, &item.VideoDetailedPageURL, &item.ThumbnailURL, &item.ThumbnailId)
		if err != nil {
			log.Print(err)
			continue
		}
		item.ThumbnailName = fmt.Sprintf("%d.jpg", item.ThumbnailId)
		vds.append(item.ViewKey, &item)

		count++
		fmt.Printf("\r%6d item added", count)
	}

	fmt.Printf("\rGot %d items \n", vds.size())
}

func (vds *VideoDataSet) append(key string, item *VideoItem) *VideoItem {
	(*vds)[key] = item
	return item
}

func (vds *VideoDataSet) remove(item *VideoItem) *VideoItem {
	delete(*vds, item.ViewKey)
	return item
}

func (vds *VideoDataSet) has(key string) bool {
	_, ok := (*vds)[key]
	return ok
}

func (vds *VideoDataSet) get(key string) (*VideoItem, bool) {
	item, ok := (*vds)[key]
	return item, ok
}

func (vds *VideoDataSet) iterate(visitor func(item *VideoItem) bool) {
	for _, info := range *vds {
		if ret := visitor(info); !ret {
			return
		}
	}
}

func (vds *VideoDataSet) size() int {
	return len(*vds)
}
