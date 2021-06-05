# 91Porn Site sniffer
Download videos from 91Pxxn

Usage: sniffer -mode [prefetch|fetch|parse|dl_desc|dl_video|sync|load|identify_date] [url] [dir] [count] [persist] [thumbnail] [help]

Get the newest video list
	-mode prefetch -count num [-proxy]
Parse the newest video list items and persit into datastore
	-mode parse -dir dirname -persist
Download video descriptor
	-mode dl_desc -url video_page_url [-presist]
Download video files using per-downloaded video descriptors
	-mode dl_video [-url video_page_url] [-transcode] [-persist]
Sync the lastest video list ( prefetch + parse )
	-mode sync -count num [-proxy] [-keep]
Download thumbnails
	-mode load -thumbnail [-script]
Identify video uploaded date according to thumbnails
	-mode identify_date
